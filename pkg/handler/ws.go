package handler

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"io"
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
	stdout       io.Reader
	stderr       io.Reader
	closeOnce    sync.Once
}

type sessionCreatedPayload struct {
	SessionID string `json:"sessionId"`
	NodeID    string `json:"nodeId"`
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

func ws(h *chttp.RegistryContext, router *route.Engine) {
	sp := "/api/signalr"
	manager := newTerminalSessionManager(h.Database())
	server, err := signalr.NewServer(
		context.Background(),
		signalr.HubFactory(func() signalr.HubInterface {
			return &terminalHub{manager: manager}
		}),
		signalr.HTTPTransports(signalr.TransportWebSockets),
	)
	if err != nil {
		panic(err)
	}
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
		database:           database,
	}
}

func (h *terminalHub) OnDisconnected(connectionID string) {
	h.manager.closeConnection(connectionID)
}

func (h *terminalHub) CreateSession(nodeID, process, workspace string) {
	payload, err := h.manager.createLocalSession(h.ConnectionID(), nodeID, process, workspace)
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

func (h *terminalHub) CloseSession(sessionID string) {
	if err := h.manager.closeSession(sessionID, "session closed by client", true); err != nil {
		h.sendError(sessionErrorPayload{SessionID: sessionID, Message: err.Error()})
	}
}

func (h *terminalHub) sendError(payload sessionErrorPayload) {
	h.Clients().Caller().Send("SessionError", payload)
}

func (m *terminalSessionManager) createLocalSession(connectionID, nodeID, process, workspace string) (sessionCreatedPayload, error) {
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
	stdout, err := sshSession.StdoutPipe()
	if err != nil {
		_ = sshSession.Close()
		_ = client.Close()
		return sessionCreatedPayload{}, err
	}
	stderr, err := sshSession.StderrPipe()
	if err != nil {
		_ = sshSession.Close()
		_ = client.Close()
		return sessionCreatedPayload{}, err
	}

	record, err := m.database.CreateSession(db.SessionInput{
		NodeID:    nodeKey,
		Name:      process,
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
		stdout:       stdout,
		stderr:       stderr,
	}

	m.mu.Lock()
	m.sessions[session.id] = session
	if _, ok := m.connectionSessions[connectionID]; !ok {
		m.connectionSessions[connectionID] = make(map[string]struct{})
	}
	m.connectionSessions[connectionID][session.id] = struct{}{}
	m.mu.Unlock()

	go m.streamOutput(session, session.stdout)
	go m.streamOutput(session, session.stderr)
	go m.watchSession(session)

	return sessionCreatedPayload{
		SessionID: session.id,
		NodeID:    nodeID,
		Process:   process,
		Workspace: workspace,
		CreatedAt: session.createdAt,
	}, nil
}

func (m *terminalSessionManager) streamSession(session *terminalSession) {
	panic("unused")
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
		reason = err.Error()
	}
	if sessionID, err := strconv.ParseInt(session.id, 10, 64); err == nil {
		_ = m.database.SetSessionStatus(sessionID, db.SessionStatusClosed)
	}
	_ = m.closeSession(session.id, reason, true)
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

func (m *terminalSessionManager) closeConnection(connectionID string) {
	m.mu.RLock()
	sessionSet := m.connectionSessions[connectionID]
	sessionIDs := make([]string, 0, len(sessionSet))
	for sessionID := range sessionSet {
		sessionIDs = append(sessionIDs, sessionID)
	}
	m.mu.RUnlock()

	for _, sessionID := range sessionIDs {
		_ = m.closeSession(sessionID, "browser disconnected", false)
	}
}

func (m *terminalSessionManager) closeSession(sessionID, reason string, notify bool) error {
	session, err := m.getSession(sessionID)
	if err != nil {
		return err
	}

	session.closeOnce.Do(func() {
		m.mu.Lock()
		delete(m.sessions, sessionID)
		if sessionSet, ok := m.connectionSessions[session.connectionID]; ok {
			delete(sessionSet, sessionID)
			if len(sessionSet) == 0 {
				delete(m.connectionSessions, session.connectionID)
			}
		}
		m.mu.Unlock()

		_ = session.stdin.Close()
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
