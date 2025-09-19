-- Initial schema for SQLite3
PRAGMA foreign_keys = ON;

CREATE TABLE IF NOT EXISTS workflow (
    id INTEGER PRIMARY KEY AUTOINCREMENT,
    status TEXT,
    execution_count INTEGER,
    retry_count INTEGER,
    created DATETIME,
    modified DATETIME,
    next_activation DATETIME,
    started DATETIME,
    executor_id INTEGER,
    executor_group TEXT,
    workflow_type TEXT,
    external_id TEXT,
    business_key TEXT,
    state TEXT,
    state_vars TEXT
);

CREATE TABLE IF NOT EXISTS executors (
    id INTEGER PRIMARY KEY AUTOINCREMENT,
    name TEXT,
    started DATETIME,
    last_active DATETIME
);

CREATE TABLE IF NOT EXISTS workflow_definitions (
    name TEXT PRIMARY KEY,
    description TEXT,
    created DATETIME,
    updated DATETIME,
    flow_chart TEXT
);

CREATE TABLE IF NOT EXISTS users (
    id INTEGER PRIMARY KEY AUTOINCREMENT,
    username TEXT NOT NULL UNIQUE,
    password TEXT NOT NULL,
    retry_count INTEGER,
    session_id TEXT,
    api_key TEXT,
    sessionExpiry DATETIME,
    created DATETIME,
    enabled INTEGER
);

CREATE TABLE IF NOT EXISTS workflow_actions (
    id INTEGER PRIMARY KEY AUTOINCREMENT,
    workflow_id INTEGER NOT NULL,
    execution_count INTEGER,
    executor_id INTEGER,
    retry_count INTEGER,
    type TEXT,
    name TEXT,
    text TEXT,
    date_time DATETIME,
    FOREIGN KEY (workflow_id) REFERENCES workflow (id)
);

-- seed admin user if not exists (idempotent)
INSERT INTO users (id, username, password, retry_count, session_id, api_key, sessionExpiry, created, enabled)
SELECT 1, 'admin', '$2a$12$5MohDGVmHcYuwfPuzoMemu5UmFlJu27sPj8KUn3jLToLjyV49eYly', 0,
       'd492c443e670f7ef54780fc94508900b2cdfdbabe2307b3aacaacdacce7b85ae',
       'b5f0e8c4-daa6-465c-bded-50ca22b798b2',
       '2025-09-17 10:08:52.002906',
       '2025-09-16 13:57:32.520000',
       1
WHERE NOT EXISTS (SELECT 1 FROM users WHERE id = 1 OR username = 'admin');
