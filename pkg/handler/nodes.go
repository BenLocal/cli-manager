package handler

import (
	"database/sql"
	"encoding/json"
	"errors"
	"net/http"
	"strconv"
	"strings"

	"github.com/benlocal/cli-manager/pkg/db"
)

type nodePayload struct {
	ID               string `json:"id,omitempty"`
	Name             string `json:"name"`
	IP               string `json:"ip"`
	Port             string `json:"port"`
	User             string `json:"user"`
	Password         string `json:"password"`
	Status           string `json:"status"`
	CPU              string `json:"cpu"`
	Memory           string `json:"memory"`
	Type             string `json:"type"`
	DefaultProcess   string `json:"defaultProcess"`
	DefaultWorkspace string `json:"defaultWorkspace"`
}

func registerNodeRoutes(mux *http.ServeMux, database *db.DB) {
	mux.HandleFunc("/api/nodes", func(w http.ResponseWriter, r *http.Request) {
		switch r.Method {
		case http.MethodGet:
			handleListNodes(w, database)
		case http.MethodPost:
			handleCreateNode(w, r, database)
		default:
			http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
		}
	})

	mux.HandleFunc("/api/nodes/", func(w http.ResponseWriter, r *http.Request) {
		parts := strings.Split(strings.Trim(strings.TrimPrefix(r.URL.Path, "/api/nodes/"), "/"), "/")
		if len(parts) == 0 || parts[0] == "" {
			http.Error(w, "invalid node path", http.StatusBadRequest)
			return
		}

		id, err := strconv.ParseInt(parts[0], 10, 64)
		if err != nil {
			http.Error(w, "invalid node id", http.StatusBadRequest)
			return
		}

		if len(parts) == 2 && parts[1] == "sessions" {
			if r.Method != http.MethodGet {
				http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
				return
			}
			handleListSessions(w, database, id)
			return
		}

		if len(parts) == 2 && parts[1] == "update" && r.Method == http.MethodPost {
			handleUpdateNode(w, r, database, id)
			return
		}

		if len(parts) == 2 && parts[1] == "delete" && r.Method == http.MethodPost {
			handleDeleteNode(w, database, id)
			return
		}

		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
	})

	mux.HandleFunc("/api/sessions/", func(w http.ResponseWriter, r *http.Request) {
		parts := strings.Split(strings.Trim(strings.TrimPrefix(r.URL.Path, "/api/sessions/"), "/"), "/")
		if len(parts) != 2 {
			http.Error(w, "invalid session path", http.StatusBadRequest)
			return
		}

		id, err := strconv.ParseInt(parts[0], 10, 64)
		if err != nil {
			http.Error(w, "invalid session id", http.StatusBadRequest)
			return
		}

		switch {
		case parts[1] == "update" && r.Method == http.MethodPost:
			handleUpdateSession(w, r, database, id)
		case parts[1] == "delete" && r.Method == http.MethodPost:
			handleDeleteSession(w, database, id)
		default:
			http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
		}
	})
}

func handleListNodes(w http.ResponseWriter, database *db.DB) {
	nodes, err := database.ListNodes()
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	writeJSON(w, http.StatusOK, mapNodes(nodes))
}

func handleCreateNode(w http.ResponseWriter, r *http.Request, database *db.DB) {
	input, err := decodeNodeInput(r)
	if err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	node, err := database.CreateNode(input)
	if err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	writeJSON(w, http.StatusCreated, mapNode(node))
}

func handleUpdateNode(w http.ResponseWriter, r *http.Request, database *db.DB, id int64) {
	input, err := decodeNodeInput(r)
	if err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	node, err := database.UpdateNode(id, input)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			http.Error(w, "node not found", http.StatusNotFound)
			return
		}
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	writeJSON(w, http.StatusOK, mapNode(node))
}

func handleDeleteNode(w http.ResponseWriter, database *db.DB, id int64) {
	if err := database.DeleteNode(id); err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			http.Error(w, "node not found", http.StatusNotFound)
			return
		}
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	w.WriteHeader(http.StatusNoContent)
}

func handleListSessions(w http.ResponseWriter, database *db.DB, nodeID int64) {
	sessions, err := database.ListSessions(nodeID)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	writeJSON(w, http.StatusOK, mapSessions(sessions))
}

func handleUpdateSession(w http.ResponseWriter, r *http.Request, database *db.DB, id int64) {
	var payload sessionPayload
	if err := json.NewDecoder(r.Body).Decode(&payload); err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	session, err := database.UpdateSession(id, strings.TrimSpace(payload.Name), strings.TrimSpace(payload.Workspace))
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			http.Error(w, "session not found", http.StatusNotFound)
			return
		}
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	writeJSON(w, http.StatusOK, mapSession(session))
}

func handleDeleteSession(w http.ResponseWriter, database *db.DB, id int64) {
	if err := database.DeleteSession(id); err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			http.Error(w, "session not found", http.StatusNotFound)
			return
		}
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	w.WriteHeader(http.StatusNoContent)
}

func decodeNodeInput(r *http.Request) (db.NodeInput, error) {
	var payload nodePayload
	if err := json.NewDecoder(r.Body).Decode(&payload); err != nil {
		return db.NodeInput{}, err
	}

	port, err := db.ParsePort(payload.Port)
	if err != nil {
		return db.NodeInput{}, err
	}

	if strings.TrimSpace(payload.Name) == "" ||
		strings.TrimSpace(payload.IP) == "" ||
		strings.TrimSpace(payload.User) == "" ||
		strings.TrimSpace(payload.Password) == "" {
		return db.NodeInput{}, errors.New("name/ip/user/password are required")
	}

	return db.NodeInput{
		Name:     strings.TrimSpace(payload.Name),
		IP:       strings.TrimSpace(payload.IP),
		Port:     port,
		User:     strings.TrimSpace(payload.User),
		Password: payload.Password,
	}, nil
}

type sessionPayload struct {
	ID        string `json:"id,omitempty"`
	NodeID    string `json:"nodeId,omitempty"`
	Name      string `json:"name"`
	Workspace string `json:"workspace"`
	Status    string `json:"status"`
	CreatedAt string `json:"createdAt"`
}

func mapNodes(nodes []db.Node) []nodePayload {
	result := make([]nodePayload, 0, len(nodes))
	for _, node := range nodes {
		result = append(result, mapNode(node))
	}
	return result
}

func mapNode(node db.Node) nodePayload {
	return nodePayload{
		ID:               strconv.FormatInt(node.ID, 10),
		Name:             node.Name,
		IP:               node.IP,
		Port:             strconv.Itoa(node.Port),
		User:             node.User,
		Password:         node.Password,
		Status:           mapStatus(node.Status),
		CPU:              node.CPU,
		Memory:           node.Memory,
		Type:             mapNodeType(node.NodeType),
		DefaultProcess:   node.DefaultProcess,
		DefaultWorkspace: node.DefaultWorkspace,
	}
}

func mapSessions(sessions []db.Session) []sessionPayload {
	result := make([]sessionPayload, 0, len(sessions))
	for _, session := range sessions {
		result = append(result, mapSession(session))
	}
	return result
}

func mapSession(session db.Session) sessionPayload {
	return sessionPayload{
		ID:        strconv.FormatInt(session.ID, 10),
		NodeID:    strconv.FormatInt(session.NodeID, 10),
		Name:      session.Name,
		Workspace: session.Workspace,
		Status:    mapSessionStatus(session.Status),
		CreatedAt: session.CreatedAt.Format("15:04:05"),
	}
}

func mapStatus(status int) string {
	switch status {
	case db.NodeStatusWarning:
		return "warning"
	case db.NodeStatusOffline:
		return "offline"
	default:
		return "online"
	}
}

func mapNodeType(nodeType int) string {
	switch nodeType {
	case db.NodeTypeCore:
		return "Core"
	case db.NodeTypeStorage:
		return "Storage"
	default:
		return "Worker"
	}
}

func mapSessionStatus(status int) string {
	switch status {
	case db.SessionStatusConnecting:
		return "connecting"
	case db.SessionStatusClosed:
		return "closed"
	default:
		return "live"
	}
}

func writeJSON(w http.ResponseWriter, status int, payload any) {
	w.Header().Set("Content-Type", "application/json; charset=utf-8")
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(payload)
}
