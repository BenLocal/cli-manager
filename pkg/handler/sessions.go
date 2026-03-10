package handler

import (
	"context"
	"database/sql"
	"encoding/json"
	"errors"
	"strconv"
	"strings"

	"github.com/benlocal/cli-manager/pkg/db"
	chttp "github.com/benlocal/cli-manager/pkg/http"
	"github.com/cloudwego/hertz/pkg/app"
	"github.com/cloudwego/hertz/pkg/protocol/consts"
	"github.com/cloudwego/hertz/pkg/route"
)

type sessionPayload struct {
	ID        string `json:"id,omitempty"`
	NodeID    string `json:"nodeId,omitempty"`
	Name      string `json:"name"`
	Process   string `json:"process"`
	Workspace string `json:"workspace"`
	Status    string `json:"status"`
	CreatedAt string `json:"createdAt"`
}

func sessions(h *chttp.RegistryContext, router *route.Engine) {
	database := h.Database()
	router.GET("/api/nodes/:id/sessions", func(ctx context.Context, c *app.RequestContext) {
		handleListSessions(c, database)
	})
	router.POST("/api/sessions/:id/update", func(ctx context.Context, c *app.RequestContext) {
		handleUpdateSession(c, database)
	})
	router.POST("/api/sessions/:id/delete", func(ctx context.Context, c *app.RequestContext) {
		handleDeleteSession(c, database)
	})
}

func handleListSessions(c *app.RequestContext, database *db.DB) {
	nodeID, err := parsePathID(c)
	if err != nil {
		writeError(c, consts.StatusBadRequest, "invalid node id")
		return
	}

	sessions, err := database.ListSessions(nodeID)
	if err != nil {
		writeError(c, consts.StatusInternalServerError, err.Error())
		return
	}

	writeSuccess(c, mapSessions(sessions))
}

func handleUpdateSession(c *app.RequestContext, database *db.DB) {
	id, err := parsePathID(c)
	if err != nil {
		writeError(c, consts.StatusBadRequest, "invalid session id")
		return
	}

	var payload sessionPayload
	if err := json.Unmarshal(c.Request.Body(), &payload); err != nil {
		writeError(c, consts.StatusBadRequest, err.Error())
		return
	}

	session, err := database.UpdateSession(
		id,
		strings.TrimSpace(payload.Name),
		strings.TrimSpace(payload.Process),
		strings.TrimSpace(payload.Workspace),
	)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			writeError(c, consts.StatusNotFound, "session not found")
			return
		}
		writeError(c, consts.StatusBadRequest, err.Error())
		return
	}

	writeSuccess(c, mapSession(session))
}

func handleDeleteSession(c *app.RequestContext, database *db.DB) {
	id, err := parsePathID(c)
	if err != nil {
		writeError(c, consts.StatusBadRequest, "invalid session id")
		return
	}

	if err := database.DeleteSession(id); err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			writeError(c, consts.StatusNotFound, "session not found")
			return
		}
		writeError(c, consts.StatusInternalServerError, err.Error())
		return
	}

	writeSuccess(c, map[string]any{})
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
		Process:   session.Process,
		Workspace: session.Workspace,
		Status:    mapSessionStatus(session.Status),
		CreatedAt: session.CreatedAt.Format("15:04:05"),
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
