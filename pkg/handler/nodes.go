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

func nodes(h *chttp.RegistryContext, router *route.Engine) {
	database := h.Database()
	router.GET("/api/nodes", func(ctx context.Context, c *app.RequestContext) {
		handleListNodes(c, database)
	})
	router.POST("/api/nodes", func(ctx context.Context, c *app.RequestContext) {
		handleCreateNode(c, database)
	})
	router.POST("/api/nodes/:id/update", func(ctx context.Context, c *app.RequestContext) {
		handleUpdateNode(c, database)
	})
	router.POST("/api/nodes/:id/delete", func(ctx context.Context, c *app.RequestContext) {
		handleDeleteNode(c, database)
	})
}

func handleListNodes(c *app.RequestContext, database *db.DB) {
	nodes, err := database.ListNodes()
	if err != nil {
		writeError(c, consts.StatusInternalServerError, err.Error())
		return
	}

	writeSuccess(c, mapNodes(nodes))
}

func handleCreateNode(c *app.RequestContext, database *db.DB) {
	input, err := decodeNodeInput(c)
	if err != nil {
		writeError(c, consts.StatusBadRequest, err.Error())
		return
	}

	node, err := database.CreateNode(input)
	if err != nil {
		writeError(c, consts.StatusBadRequest, err.Error())
		return
	}

	writeSuccess(c, mapNode(node))
}

func handleUpdateNode(c *app.RequestContext, database *db.DB) {
	id, err := parsePathID(c)
	if err != nil {
		writeError(c, consts.StatusBadRequest, "invalid node id")
		return
	}

	input, err := decodeNodeInput(c)
	if err != nil {
		writeError(c, consts.StatusBadRequest, err.Error())
		return
	}

	node, err := database.UpdateNode(id, input)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			writeError(c, consts.StatusNotFound, "node not found")
			return
		}
		writeError(c, consts.StatusBadRequest, err.Error())
		return
	}

	writeSuccess(c, mapNode(node))
}

func handleDeleteNode(c *app.RequestContext, database *db.DB) {
	id, err := parsePathID(c)
	if err != nil {
		writeError(c, consts.StatusBadRequest, "invalid node id")
		return
	}

	if err := database.DeleteNode(id); err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			writeError(c, consts.StatusNotFound, "node not found")
			return
		}
		writeError(c, consts.StatusInternalServerError, err.Error())
		return
	}

	writeSuccess(c, map[string]any{})
}

func decodeNodeInput(c *app.RequestContext) (db.NodeInput, error) {
	var payload nodePayload
	if err := json.Unmarshal(c.Request.Body(), &payload); err != nil {
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

func parsePathID(c *app.RequestContext) (int64, error) {
	return strconv.ParseInt(c.Param("id"), 10, 64)
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
