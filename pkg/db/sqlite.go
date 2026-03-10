package db

import (
	"database/sql"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"sync/atomic"
	"time"

	"github.com/benlocal/cli-manager/migrations"
	"github.com/jmoiron/sqlx"
	_ "github.com/mattn/go-sqlite3"
)

const (
	NodeStatusOnline = iota
	NodeStatusWarning
	NodeStatusOffline
)

const (
	NodeTypeCore = iota
	NodeTypeStorage
	NodeTypeWorker
)

const (
	SessionStatusConnecting = iota
	SessionStatusLive
	SessionStatusClosed
)

type DB struct {
	sql     *sqlx.DB
	counter atomic.Uint64
}

type Node struct {
	ID               int64     `db:"id"`
	Name             string    `db:"name"`
	IP               string    `db:"ip"`
	Port             int       `db:"port"`
	User             string    `db:"user"`
	Password         string    `db:"password"`
	Status           int       `db:"status"`
	CPU              string    `db:"cpu"`
	Memory           string    `db:"memory"`
	NodeType         int       `db:"node_type"`
	DefaultProcess   string    `db:"default_process"`
	DefaultWorkspace string    `db:"default_workspace"`
	CreatedAt        time.Time `db:"created_at"`
	UpdatedAt        time.Time `db:"updated_at"`
}

type Session struct {
	ID        int64     `db:"id"`
	NodeID    int64     `db:"node_id"`
	Name      string    `db:"name"`
	Process   string    `db:"process"`
	Workspace string    `db:"workspace"`
	Status    int       `db:"status"`
	CreatedAt time.Time `db:"created_at"`
	UpdatedAt time.Time `db:"updated_at"`
}

type NodeInput struct {
	Name     string
	IP       string
	Port     int
	User     string
	Password string
}

type SessionInput struct {
	NodeID    int64
	Name      string
	Process   string
	Workspace string
	Status    int
}

func Open(dsn string) (*DB, error) {
	if dsn == "" {
		dsn = filepath.Join("data", "cli-manager.db")
	}
	if err := os.MkdirAll(filepath.Dir(dsn), 0o755); err != nil {
		return nil, err
	}

	sqlDB, err := sqlx.Open("sqlite3", dsn)
	if err != nil {
		return nil, err
	}
	sqlDB.SetMaxOpenConns(1)
	sqlDB.SetMaxIdleConns(1)

	db := &DB{sql: sqlDB}
	if err := db.configure(); err != nil {
		_ = sqlDB.Close()
		return nil, err
	}
	if err := migrations.Run(sqlDB.DB); err != nil {
		_ = sqlDB.Close()
		return nil, err
	}
	return db, nil
}

func (db *DB) Close() error {
	if db == nil || db.sql == nil {
		return nil
	}
	return db.sql.Close()
}

func (db *DB) configure() error {
	statements := []string{
		`PRAGMA foreign_keys = ON;`,
		`PRAGMA journal_mode = WAL;`,
	}

	for _, stmt := range statements {
		if _, err := db.sql.Exec(stmt); err != nil {
			return err
		}
	}

	return nil
}

func (db *DB) ListNodes() ([]Node, error) {
	nodes := make([]Node, 0)
	err := db.sql.Select(&nodes, `
		SELECT id, name, ip, port, user, password, status, cpu, memory, node_type,
		       default_process, default_workspace, created_at, updated_at
		FROM node
		ORDER BY updated_at DESC, id DESC
	`)
	return nodes, err
}

func (db *DB) GetNode(id int64) (Node, error) {
	var node Node
	err := db.sql.Get(&node, `
		SELECT id, name, ip, port, user, password, status, cpu, memory, node_type,
		       default_process, default_workspace, created_at, updated_at
		FROM node
		WHERE id = ?
	`, id)
	return node, err
}

func (db *DB) CreateNode(input NodeInput) (Node, error) {
	now := time.Now()
	node := Node{
		ID:               db.nextID(),
		Name:             input.Name,
		IP:               input.IP,
		Port:             input.Port,
		User:             input.User,
		Password:         input.Password,
		Status:           NodeStatusOnline,
		CPU:              "0%",
		Memory:           "0GB",
		NodeType:         NodeTypeWorker,
		DefaultProcess:   "bash",
		DefaultWorkspace: "/root",
		CreatedAt:        now,
		UpdatedAt:        now,
	}

	_, err := db.sql.Exec(
		`INSERT INTO node (
			id, name, ip, port, user, password, status, cpu, memory, node_type,
			default_process, default_workspace, created_at, updated_at
		) VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)`,
		node.ID, node.Name, node.IP, node.Port, node.User, node.Password, node.Status, node.CPU,
		node.Memory, node.NodeType, node.DefaultProcess, node.DefaultWorkspace, node.CreatedAt, node.UpdatedAt,
	)
	return node, err
}

func (db *DB) UpdateNode(id int64, input NodeInput) (Node, error) {
	now := time.Now()
	result, err := db.sql.Exec(
		`UPDATE node SET name = ?, ip = ?, port = ?, user = ?, password = ?, updated_at = ? WHERE id = ?`,
		input.Name, input.IP, input.Port, input.User, input.Password, now, id,
	)
	if err != nil {
		return Node{}, err
	}
	if err := ensureAffected(result); err != nil {
		return Node{}, err
	}
	return db.GetNode(id)
}

func (db *DB) DeleteNode(id int64) error {
	result, err := db.sql.Exec(`DELETE FROM node WHERE id = ?`, id)
	if err != nil {
		return err
	}
	return ensureAffected(result)
}

func (db *DB) ListSessions(nodeID int64) ([]Session, error) {
	sessions := make([]Session, 0)
	err := db.sql.Select(&sessions, `
		SELECT id, node_id, name, process, workspace, status, created_at, updated_at
		FROM session_record
		WHERE node_id = ?
		ORDER BY updated_at DESC, id DESC
	`, nodeID)
	return sessions, err
}

func (db *DB) CreateSession(input SessionInput) (Session, error) {
	now := time.Now()
	session := Session{
		ID:        db.nextID(),
		NodeID:    input.NodeID,
		Name:      input.Name,
		Process:   input.Process,
		Workspace: input.Workspace,
		Status:    input.Status,
		CreatedAt: now,
		UpdatedAt: now,
	}

	_, err := db.sql.Exec(
		`INSERT INTO session_record (id, node_id, name, process, workspace, status, created_at, updated_at)
		 VALUES (?, ?, ?, ?, ?, ?, ?, ?)`,
		session.ID, session.NodeID, session.Name, session.Process, session.Workspace, session.Status, session.CreatedAt, session.UpdatedAt,
	)
	return session, err
}

func (db *DB) UpdateSession(id int64, name, process, workspace string) (Session, error) {
	now := time.Now()
	result, err := db.sql.Exec(
		`UPDATE session_record SET name = ?, process = ?, workspace = ?, updated_at = ? WHERE id = ?`,
		name, process, workspace, now, id,
	)
	if err != nil {
		return Session{}, err
	}
	if err := ensureAffected(result); err != nil {
		return Session{}, err
	}
	return db.GetSession(id)
}

func (db *DB) SetSessionStatus(id int64, status int) error {
	result, err := db.sql.Exec(
		`UPDATE session_record SET status = ?, updated_at = ? WHERE id = ?`,
		status, time.Now(), id,
	)
	if err != nil {
		return err
	}
	return ensureAffected(result)
}

func (db *DB) DeleteSession(id int64) error {
	result, err := db.sql.Exec(`DELETE FROM session_record WHERE id = ?`, id)
	if err != nil {
		return err
	}
	return ensureAffected(result)
}

func (db *DB) GetSession(id int64) (Session, error) {
	var session Session
	err := db.sql.Get(&session, `
		SELECT id, node_id, name, process, workspace, status, created_at, updated_at
		FROM session_record
		WHERE id = ?
	`, id)
	return session, err
}

func (db *DB) nextID() int64 {
	sequence := db.counter.Add(1) & 0x0fff
	return (time.Now().UnixMilli() << 12) | int64(sequence)
}

func ParsePort(value string) (int, error) {
	var port int
	if _, err := fmt.Sscanf(value, "%d", &port); err != nil {
		return 0, errors.New("invalid port")
	}
	if port < 1 || port > 65535 {
		return 0, errors.New("port out of range")
	}
	return port, nil
}

func ensureAffected(result sql.Result) error {
	rows, err := result.RowsAffected()
	if err != nil {
		return err
	}
	if rows == 0 {
		return sql.ErrNoRows
	}
	return nil
}
