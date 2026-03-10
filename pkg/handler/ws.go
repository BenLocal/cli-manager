package handler

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"io"
	"log"
	"net/http"
	"os"
	"strconv"
	"strings"
	"sync"
	"time"

	"github.com/benlocal/cli-manager/pkg/db"
	chttp "github.com/benlocal/cli-manager/pkg/http"
	"github.com/cloudwego/hertz/pkg/common/adaptor"
	"github.com/cloudwego/hertz/pkg/route"
	"github.com/philippseith/signalr"
	"golang.org/x/crypto/ssh"
)

type terminalHub struct {
	signalr.Hub
	manager *terminalSessionManager
}

type signalRServer interface {
	MapHTTP(routerFactory func() signalr.MappableRouter, path string)
	HubClients() signalr.HubClients
}

type terminalSessionManager struct {
	mu                 sync.RWMutex
	sessions           map[string]*terminalSession
	connectionSessions map[string]map[string]struct{}
	pendingCloses      map[string]*time.Timer
	server             signalRServer
	database           *db.DB
}

type terminalSession struct {
	id           string
	nodeID       string
	connectionID string
	process      string
	workspace    string
	createdAt    string
	client       *ssh.Client
	session      *ssh.Session
	stdin        io.WriteCloser
	output       io.Reader
	outputWriter io.Closer
	closeOnce    sync.Once
}

type sessionCreatedPayload struct {
	SessionID string `json:"sessionId"`
	NodeID    string `json:"nodeId"`
	Name      string `json:"name"`
	Process   string `json:"process"`
	Workspace string `json:"workspace"`
	CreatedAt string `json:"createdAt"`
}

type sessionOutputPayload struct {
	SessionID string `json:"sessionId"`
	NodeID    string `json:"nodeId"`
	Data      string `json:"data"`
}

type sessionClosedPayload struct {
	SessionID string `json:"sessionId"`
	NodeID    string `json:"nodeId"`
	Reason    string `json:"reason"`
}

type sessionErrorPayload struct {
	SessionID string `json:"sessionId,omitempty"`
	NodeID    string `json:"nodeId,omitempty"`
	Message   string `json:"message"`
}

type signalRFilterLogger struct{}

func ws(h *chttp.RegistryContext, router *route.Engine) {
	sp := "/api/hub/terminal"
	manager := newTerminalSessionManager(h.Database())
	baseOpts := []func(signalr.Party) error{
		signalr.HubFactory(func() signalr.HubInterface {
			return &terminalHub{manager: manager}
		}),
		signalr.HTTPTransports(signalr.TransportWebSockets, signalr.TransportServerSentEvents),
		signalr.KeepAliveInterval(10 * time.Second),
		signalr.TimeoutInterval(30 * time.Second),
		signalr.HandshakeTimeout(15 * time.Second),
		signalr.InsecureSkipVerify(true),
		signalr.Logger(signalRFilterLogger{}, false),
	}
	server, err := signalr.NewServer(context.Background(), baseOpts...)
	if err != nil {
		panic(err)
	}
	manager.server = server
	mux := http.NewServeMux()
	server.MapHTTP(signalr.WithHTTPServeMux(mux), sp)
	handler := adaptor.HertzHandler(mux)
	router.Any(sp, handler)
	router.Any(sp+"/*wsPath", handler)
}

func newTerminalSessionManager(database *db.DB) *terminalSessionManager {
	return &terminalSessionManager{
		sessions:           make(map[string]*terminalSession),
		connectionSessions: make(map[string]map[string]struct{}),
		pendingCloses:      make(map[string]*time.Timer),
		database:           database,
	}
}

func (h *terminalHub) OnConnected(string) {
}

func (h *terminalHub) OnDisconnected(connectionID string) {
	h.manager.scheduleConnectionClose(connectionID)
}

func (h *terminalHub) CreateSession(nodeID, name, process, workspace string) {
	payload, err := h.manager.createLocalSession(h.ConnectionID(), nodeID, name, process, workspace)
	if err != nil {
		h.sendError(sessionErrorPayload{NodeID: nodeID, Message: err.Error()})
		return
	}

	h.Clients().Caller().Send("SessionCreated", payload)
}

func (h *terminalHub) Input(sessionID, data string) {
	if err := h.manager.write(sessionID, data); err != nil {
		h.sendError(sessionErrorPayload{SessionID: sessionID, Message: err.Error()})
	}
}

func (h *terminalHub) Resize(sessionID string, cols, rows int) {
	if err := h.manager.resize(sessionID, cols, rows); err != nil {
		h.sendError(sessionErrorPayload{SessionID: sessionID, Message: err.Error()})
	}
}

func (h *terminalHub) ReconnectSession(sessionID string) {
	if err := h.manager.reattachSession(h.ConnectionID(), sessionID); err != nil {
		h.sendError(sessionErrorPayload{SessionID: sessionID, Message: err.Error()})
	}
}

func (h *terminalHub) CloseSession(sessionID string) {
	if err := h.manager.closeSession(sessionID, "session closed by client", true); err != nil {
		if strings.Contains(err.Error(), "session not found:") {
			return
		}
		h.sendError(sessionErrorPayload{SessionID: sessionID, Message: err.Error()})
	}
}

func (h *terminalHub) sendError(payload sessionErrorPayload) {
	h.Clients().Caller().Send("SessionError", payload)
}

func (m *terminalSessionManager) createLocalSession(connectionID, nodeID, name, process, workspace string) (sessionCreatedPayload, error) {
	name = strings.TrimSpace(name)
	if name == "" {
		return sessionCreatedPayload{}, errors.New("session name is required")
	}

	process = strings.TrimSpace(process)
	if process == "" {
		process = "bash"
	}

	workspace = strings.TrimSpace(workspace)
	if workspace == "" {
		workspace = "/root"
	}

	nodeKey, err := strconv.ParseInt(nodeID, 10, 64)
	if err != nil {
		return sessionCreatedPayload{}, errors.New("invalid node id")
	}

	node, err := m.database.GetNode(nodeKey)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return sessionCreatedPayload{}, errors.New("node not found")
		}
		return sessionCreatedPayload{}, err
	}

	client, err := ssh.Dial("tcp", fmt.Sprintf("%s:%d", node.IP, node.Port), &ssh.ClientConfig{
		User:            node.User,
		Auth:            []ssh.AuthMethod{ssh.Password(node.Password)},
		HostKeyCallback: ssh.InsecureIgnoreHostKey(),
		Timeout:         8 * time.Second,
	})
	if err != nil {
		return sessionCreatedPayload{}, fmt.Errorf("ssh dial failed: %w", err)
	}

	sshSession, err := client.NewSession()
	if err != nil {
		_ = client.Close()
		return sessionCreatedPayload{}, fmt.Errorf("ssh session failed: %w", err)
	}

	if err := sshSession.RequestPty("xterm-256color", 32, 120, ssh.TerminalModes{
		ssh.ECHO:          1,
		ssh.TTY_OP_ISPEED: 14400,
		ssh.TTY_OP_OSPEED: 14400,
	}); err != nil {
		_ = sshSession.Close()
		_ = client.Close()
		return sessionCreatedPayload{}, fmt.Errorf("request remote pty failed: %w", err)
	}

	stdin, err := sshSession.StdinPipe()
	if err != nil {
		_ = sshSession.Close()
		_ = client.Close()
		return sessionCreatedPayload{}, err
	}
	outputReader, outputWriter := io.Pipe()
	sshSession.Stdout = outputWriter
	sshSession.Stderr = outputWriter

	record, err := m.database.CreateSession(db.SessionInput{
		NodeID:    nodeKey,
		Name:      name,
		Process:   process,
		Workspace: workspace,
		Status:    db.SessionStatusLive,
	})
	if err != nil {
		_ = sshSession.Close()
		_ = client.Close()
		return sessionCreatedPayload{}, err
	}

	command := fmt.Sprintf("cd %s && exec %s", shellQuote(workspace), process)
	if err := sshSession.Start(command); err != nil {
		_ = m.database.SetSessionStatus(record.ID, db.SessionStatusClosed)
		_ = outputWriter.Close()
		_ = sshSession.Close()
		_ = client.Close()
		return sessionCreatedPayload{}, fmt.Errorf("start remote command failed: %w", err)
	}

	session := &terminalSession{
		id:           strconv.FormatInt(record.ID, 10),
		nodeID:       nodeID,
		connectionID: connectionID,
		process:      process,
		workspace:    workspace,
		createdAt:    record.CreatedAt.Format("15:04:05"),
		client:       client,
		session:      sshSession,
		stdin:        stdin,
		output:       outputReader,
		outputWriter: outputWriter,
	}

	m.mu.Lock()
	if timer, ok := m.pendingCloses[session.id]; ok {
		timer.Stop()
		delete(m.pendingCloses, session.id)
	}
	m.sessions[session.id] = session
	if _, ok := m.connectionSessions[connectionID]; !ok {
		m.connectionSessions[connectionID] = make(map[string]struct{})
	}
	m.connectionSessions[connectionID][session.id] = struct{}{}
	m.mu.Unlock()

	go m.streamOutput(session, session.output)
	go m.watchSession(session)

	return sessionCreatedPayload{
		SessionID: session.id,
		NodeID:    nodeID,
		Name:      name,
		Process:   process,
		Workspace: workspace,
		CreatedAt: session.createdAt,
	}, nil
}

func (m *terminalSessionManager) streamOutput(session *terminalSession, reader io.Reader) {
	buffer := make([]byte, 4096)
	for {
		n, err := reader.Read(buffer)
		if n > 0 {
			m.emitToConnection(session.connectionID, "SessionOutput", sessionOutputPayload{
				SessionID: session.id,
				NodeID:    session.nodeID,
				Data:      string(buffer[:n]),
			})
		}
		if err != nil {
			if !errors.Is(err, io.EOF) && !errors.Is(err, os.ErrClosed) {
				m.emitToConnection(session.connectionID, "SessionError", sessionErrorPayload{
					SessionID: session.id,
					NodeID:    session.nodeID,
					Message:   err.Error(),
				})
			}
			return
		}
	}
}

func (m *terminalSessionManager) watchSession(session *terminalSession) {
	reason := "session closed"
	if err := session.session.Wait(); err != nil {
		reason = formatSessionCloseReason(err)
	}
	if sessionID, err := strconv.ParseInt(session.id, 10, 64); err == nil {
		_ = m.database.SetSessionStatus(sessionID, db.SessionStatusClosed)
	}
	_ = m.closeSession(session.id, reason, true)
}

func formatSessionCloseReason(err error) string {
	if err == nil {
		return "session closed"
	}

	message := strings.TrimSpace(err.Error())
	lower := strings.ToLower(message)

	switch {
	case strings.Contains(lower, "exit status"):
		return "remote process exited"
	case strings.Contains(lower, "exit signal"):
		return "remote process terminated by signal"
	case strings.Contains(lower, "without exit status or exit signal"):
		return "remote process ended unexpectedly"
	default:
		return message
	}
}

func (m *terminalSessionManager) write(sessionID, data string) error {
	session, err := m.getSession(sessionID)
	if err != nil {
		return err
	}
	_, err = io.WriteString(session.stdin, data)
	return err
}

func (m *terminalSessionManager) resize(sessionID string, cols, rows int) error {
	session, err := m.getSession(sessionID)
	if err != nil {
		return err
	}
	return session.session.WindowChange(rows, cols)
}

func (m *terminalSessionManager) scheduleConnectionClose(connectionID string) {
	m.mu.RLock()
	sessionSet := m.connectionSessions[connectionID]
	sessionIDs := make([]string, 0, len(sessionSet))
	for sessionID := range sessionSet {
		sessionIDs = append(sessionIDs, sessionID)
	}
	m.mu.RUnlock()

	for _, sessionID := range sessionIDs {
		m.scheduleSessionClose(sessionID, "browser disconnected")
	}
}

func (m *terminalSessionManager) scheduleSessionClose(sessionID, reason string) {
	const disconnectGracePeriod = 20 * time.Second

	m.mu.Lock()
	if timer, ok := m.pendingCloses[sessionID]; ok {
		timer.Stop()
	}
	m.pendingCloses[sessionID] = time.AfterFunc(disconnectGracePeriod, func() {
		_ = m.closeSession(sessionID, reason, false)
	})
	m.mu.Unlock()
}

func (m *terminalSessionManager) reattachSession(connectionID, sessionID string) error {
	m.mu.Lock()
	defer m.mu.Unlock()

	session, ok := m.sessions[sessionID]
	if !ok {
		return fmt.Errorf("session not found: %s", sessionID)
	}

	if timer, ok := m.pendingCloses[sessionID]; ok {
		timer.Stop()
		delete(m.pendingCloses, sessionID)
	}

	if session.connectionID != "" && session.connectionID != connectionID {
		if sessionSet, ok := m.connectionSessions[session.connectionID]; ok {
			delete(sessionSet, sessionID)
			if len(sessionSet) == 0 {
				delete(m.connectionSessions, session.connectionID)
			}
		}
	}

	session.connectionID = connectionID
	if _, ok := m.connectionSessions[connectionID]; !ok {
		m.connectionSessions[connectionID] = make(map[string]struct{})
	}
	m.connectionSessions[connectionID][sessionID] = struct{}{}
	return nil
}

func (m *terminalSessionManager) closeSession(sessionID, reason string, notify bool) error {
	session, err := m.getSession(sessionID)
	if err != nil {
		return err
	}

	session.closeOnce.Do(func() {
		m.mu.Lock()
		if timer, ok := m.pendingCloses[sessionID]; ok {
			timer.Stop()
			delete(m.pendingCloses, sessionID)
		}
		delete(m.sessions, sessionID)
		if sessionSet, ok := m.connectionSessions[session.connectionID]; ok {
			delete(sessionSet, sessionID)
			if len(sessionSet) == 0 {
				delete(m.connectionSessions, session.connectionID)
			}
		}
		m.mu.Unlock()

		_ = session.stdin.Close()
		_ = session.outputWriter.Close()
		_ = session.session.Close()
		_ = session.client.Close()

		if notify {
			m.emitToConnection(session.connectionID, "SessionClosed", sessionClosedPayload{
				SessionID: session.id,
				NodeID:    session.nodeID,
				Reason:    reason,
			})
		}
	})

	return nil
}

func (m *terminalSessionManager) getSession(sessionID string) (*terminalSession, error) {
	m.mu.RLock()
	defer m.mu.RUnlock()

	session, ok := m.sessions[sessionID]
	if !ok {
		return nil, fmt.Errorf("session not found: %s", sessionID)
	}
	return session, nil
}

func (m *terminalSessionManager) emitToConnection(connectionID, method string, payload any) {
	m.mu.RLock()
	server := m.server
	m.mu.RUnlock()
	if server == nil {
		return
	}
	server.HubClients().Client(connectionID).Send(method, payload)
}

func shellQuote(value string) string {
	return "'" + strings.ReplaceAll(value, "'", `'\''`) + "'"
}

func (signalRFilterLogger) Log(keyVals ...interface{}) error {
	if shouldIgnoreSignalREvent(keyVals) {
		return nil
	}
	log.Println(keyVals...)
	return nil
}

func shouldIgnoreSignalREvent(keyVals []interface{}) bool {
	for index := 0; index < len(keyVals)-1; index += 2 {
		key, ok := keyVals[index].(string)
		if !ok {
			continue
		}
		if key != "error" {
			continue
		}

		message := strings.ToLower(fmt.Sprint(keyVals[index+1]))
		switch {
		case strings.Contains(message, "failed to read frame header: eof"):
			return true
		case strings.Contains(message, "statusgoingaway"):
			return true
		case strings.Contains(message, "closed network connection"):
			return true
		}
	}
	return false
}
