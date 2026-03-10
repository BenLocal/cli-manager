CREATE TABLE IF NOT EXISTS node (
    id INTEGER PRIMARY KEY,
    name TEXT NOT NULL,
    ip TEXT NOT NULL,
    port INTEGER NOT NULL,
    user TEXT NOT NULL,
    password TEXT NOT NULL,
    status INTEGER NOT NULL DEFAULT 0,
    cpu TEXT NOT NULL DEFAULT '0%',
    memory TEXT NOT NULL DEFAULT '0GB',
    node_type INTEGER NOT NULL DEFAULT 2,
    default_process TEXT NOT NULL DEFAULT 'bash',
    default_workspace TEXT NOT NULL DEFAULT '/root',
    created_at DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP,
    updated_at DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP
);

CREATE UNIQUE INDEX IF NOT EXISTS uk_node_ip_port ON node(ip, port);
CREATE INDEX IF NOT EXISTS idx_node_status ON node(status);

CREATE TABLE IF NOT EXISTS session_record (
    id INTEGER PRIMARY KEY,
    node_id INTEGER NOT NULL,
    name TEXT NOT NULL,
    workspace TEXT NOT NULL,
    status INTEGER NOT NULL DEFAULT 0,
    created_at DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP,
    updated_at DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP,
    FOREIGN KEY (node_id) REFERENCES node(id) ON DELETE CASCADE
);

CREATE INDEX IF NOT EXISTS idx_session_node_updated ON session_record(node_id, updated_at DESC);

UPDATE session_record SET status = 2 WHERE status != 2;

INSERT INTO node (
    id, name, ip, port, user, password, status, cpu, memory, node_type,
    default_process, default_workspace, created_at, updated_at
)

