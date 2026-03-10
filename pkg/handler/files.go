package handler

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"path"
	"sort"
	"strconv"
	"strings"
	"time"

	"github.com/benlocal/cli-manager/pkg/db"
	chttp "github.com/benlocal/cli-manager/pkg/http"
	"github.com/cloudwego/hertz/pkg/app"
	"github.com/cloudwego/hertz/pkg/protocol/consts"
	"github.com/cloudwego/hertz/pkg/route"
	"golang.org/x/crypto/ssh"
)

type fileNodePayload struct {
	Key      string `json:"key"`
	Label    string `json:"label"`
	Path     string `json:"path"`
	Leaf     bool   `json:"leaf"`
	Size     string `json:"size,omitempty"`
	Icon     string `json:"icon"`
	Children any    `json:"children,omitempty"`
}

func files(h *chttp.RegistryContext, router *route.Engine) {
	database := h.Database()
	router.GET("/api/nodes/:id/files", func(ctx context.Context, c *app.RequestContext) {
		handleListFiles(c, database)
	})
}

func handleListFiles(c *app.RequestContext, database *db.DB) {
	nodeID, err := parsePathID(c)
	if err != nil {
		writeError(c, consts.StatusBadRequest, "invalid node id")
		return
	}

	dirPath := strings.TrimSpace(string(c.Query("path")))
	if dirPath == "" {
		writeError(c, consts.StatusBadRequest, "path is required")
		return
	}

	node, err := database.GetNode(nodeID)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			writeError(c, consts.StatusNotFound, "node not found")
			return
		}
		writeError(c, consts.StatusInternalServerError, err.Error())
		return
	}

	items, err := listRemoteFiles(node, dirPath)
	if err != nil {
		writeError(c, consts.StatusBadRequest, err.Error())
		return
	}

	writeSuccess(c, items)
}

func listRemoteFiles(node db.Node, dirPath string) ([]fileNodePayload, error) {
	client, err := ssh.Dial("tcp", fmt.Sprintf("%s:%d", node.IP, node.Port), &ssh.ClientConfig{
		User:            node.User,
		Auth:            []ssh.AuthMethod{ssh.Password(node.Password)},
		HostKeyCallback: ssh.InsecureIgnoreHostKey(),
		Timeout:         8 * time.Second,
	})
	if err != nil {
		return nil, fmt.Errorf("ssh dial failed: %w", err)
	}
	defer func() { _ = client.Close() }()

	session, err := client.NewSession()
	if err != nil {
		return nil, fmt.Errorf("ssh session failed: %w", err)
	}
	defer func() { _ = session.Close() }()

	command := fmt.Sprintf(
		`find %s -mindepth 1 -maxdepth 1 -printf '%%f\t%%y\t%%s\n' 2>/dev/null | sort`,
		shellQuote(dirPath),
	)
	output, err := session.Output(command)
	if err != nil {
		return nil, fmt.Errorf("list remote files failed: %w", err)
	}

	lines := strings.Split(strings.TrimSpace(string(output)), "\n")
	items := make([]fileNodePayload, 0, len(lines))
	for _, line := range lines {
		if strings.TrimSpace(line) == "" {
			continue
		}

		parts := strings.Split(line, "\t")
		if len(parts) < 3 {
			continue
		}

		name := parts[0]
		fileType := parts[1]
		size := parts[2]
		fullPath := path.Join(dirPath, name)
		isDir := fileType == "d"

		items = append(items, fileNodePayload{
			Key:      fullPath,
			Label:    name,
			Path:     fullPath,
			Leaf:     !isDir,
			Size:     formatFileSize(size, isDir),
			Icon:     mapFileIcon(isDir),
			Children: []fileNodePayload{},
		})
	}

	sort.SliceStable(items, func(i, j int) bool {
		if items[i].Leaf == items[j].Leaf {
			return items[i].Label < items[j].Label
		}
		return !items[i].Leaf && items[j].Leaf
	})

	return items, nil
}

func mapFileIcon(isDir bool) string {
	if isDir {
		return "pi pi-folder"
	}
	return "pi pi-file"
}

func formatFileSize(raw string, isDir bool) string {
	if isDir {
		return ""
	}
	size, err := strconv.ParseInt(raw, 10, 64)
	if err != nil {
		return raw
	}
	switch {
	case size >= 1024*1024*1024:
		return fmt.Sprintf("%.1fGB", float64(size)/(1024*1024*1024))
	case size >= 1024*1024:
		return fmt.Sprintf("%.1fMB", float64(size)/(1024*1024))
	case size >= 1024:
		return fmt.Sprintf("%.1fKB", float64(size)/1024)
	default:
		return fmt.Sprintf("%dB", size)
	}
}
