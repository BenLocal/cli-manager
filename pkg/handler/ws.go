package handler

import (
	"context"
	"errors"
	"fmt"
	"io"
	"net/http"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strings"
	"sync"
	"time"

	"github.com/creack/pty"
	"github.com/philippseith/signalr"
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
}

type terminalSession struct {
	id           string
	nodeID       string
	connectionID string
	process      string
	workspace    string
	createdAt    string
	cmd          *exec.Cmd
	ptyFile      *os.File
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

func NewRootHandler() (http.Handler, error) {
	uiHandler, err := NewAppHandler()
	if err != nil {
		return nil, err
	}

	manager := newTerminalSessionManager()
	server, err := signalr.NewServer(
		context.Background(),
		signalr.HubFactory(func() signalr.HubInterface {
			return &terminalHub{manager: manager}
		}),
		signalr.HTTPTransports(signalr.TransportWebSockets),
	)
	if err != nil {
		return nil, err
	}

	manager.server = server

	mux := http.NewServeMux()
	server.MapHTTP(signalr.WithHTTPServeMux(mux), "/hub/terminal")
	mux.Handle("/", uiHandler)
	return mux, nil
}

func newTerminalSessionManager() *terminalSessionManager {
	return &terminalSessionManager{
		sessions:           make(map[string]*terminalSession),
		connectionSessions: make(map[string]map[string]struct{}),
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
		process = defaultShell()
	}

	workspace = strings.TrimSpace(workspace)
	if workspace == "" {
		workspace = defaultWorkspace()
	}
	workspace = filepath.Clean(workspace)

	info, err := os.Stat(workspace)
	if err != nil {
		return sessionCreatedPayload{}, fmt.Errorf("workspace unavailable: %w", err)
	}
	if !info.IsDir() {
		return sessionCreatedPayload{}, fmt.Errorf("workspace is not a directory: %s", workspace)
	}

	args := strings.Fields(process)
	if len(args) == 0 {
		return sessionCreatedPayload{}, errors.New("process is required")
	}

	cmd := exec.Command(args[0], args[1:]...)
	cmd.Dir = workspace
	cmd.Env = append(os.Environ(), "TERM=xterm-256color")

	ptyFile, err := pty.StartWithSize(cmd, &pty.Winsize{Cols: 120, Rows: 32})
	if err != nil {
		return sessionCreatedPayload{}, fmt.Errorf("start pty session: %w", err)
	}

	session := &terminalSession{
		id:           fmt.Sprintf("sess-%d", time.Now().UnixNano()),
		nodeID:       nodeID,
		connectionID: connectionID,
		process:      process,
		workspace:    workspace,
		createdAt:    time.Now().Format("15:04:05"),
		cmd:          cmd,
		ptyFile:      ptyFile,
	}

	m.mu.Lock()
	m.sessions[session.id] = session
	if _, ok := m.connectionSessions[connectionID]; !ok {
		m.connectionSessions[connectionID] = make(map[string]struct{})
	}
	m.connectionSessions[connectionID][session.id] = struct{}{}
	m.mu.Unlock()

	go m.streamSession(session)
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
	buffer := make([]byte, 4096)
	for {
		n, err := session.ptyFile.Read(buffer)
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
	if err := session.cmd.Wait(); err != nil {
		reason = err.Error()
	}
	_ = m.closeSession(session.id, reason, true)
}

func (m *terminalSessionManager) write(sessionID, data string) error {
	session, err := m.getSession(sessionID)
	if err != nil {
		return err
	}
	_, err = io.WriteString(session.ptyFile, data)
	return err
}

func (m *terminalSessionManager) resize(sessionID string, cols, rows int) error {
	session, err := m.getSession(sessionID)
	if err != nil {
		return err
	}
	return pty.Setsize(session.ptyFile, &pty.Winsize{
		Cols: uint16(cols),
		Rows: uint16(rows),
	})
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

		_ = session.ptyFile.Close()
		if session.cmd.Process != nil {
			_ = session.cmd.Process.Kill()
		}

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

func defaultShell() string {
	if runtime.GOOS == "windows" {
		return "powershell"
	}
	if _, err := exec.LookPath("bash"); err == nil {
		return "bash"
	}
	return "sh"
}

func defaultWorkspace() string {
	if home, err := os.UserHomeDir(); err == nil && home != "" {
		return home
	}
	return "/tmp"
}
