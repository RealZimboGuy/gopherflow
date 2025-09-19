-- Initial schema for MySQL
-- Ensure using InnoDB and utf8mb4

CREATE TABLE IF NOT EXISTS workflow (
    id BIGINT AUTO_INCREMENT PRIMARY KEY,
    status TEXT,
    execution_count INT,
    retry_count INT,
    created DATETIME(3),
    modified DATETIME(3),
    next_activation DATETIME(3),
    started DATETIME(3),
    executor_id BIGINT,
    executor_group TEXT,
    workflow_type TEXT,
    external_id TEXT,
    business_key TEXT,
    state TEXT,
    state_vars TEXT
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_unicode_ci;

CREATE TABLE IF NOT EXISTS executors (
    id BIGINT AUTO_INCREMENT PRIMARY KEY,
    name TEXT,
    started DATETIME(3),
    last_active DATETIME(3)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_unicode_ci;

CREATE TABLE IF NOT EXISTS workflow_definitions (
    name VARCHAR(255) PRIMARY KEY,
    description TEXT,
    created DATETIME(3),
    updated DATETIME(3),
    flow_chart LONGTEXT
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_unicode_ci;

CREATE TABLE IF NOT EXISTS users (
    id BIGINT AUTO_INCREMENT PRIMARY KEY,
    username VARCHAR(255) NOT NULL UNIQUE,
    password TEXT NOT NULL,
    retry_count INT,
    session_id VARCHAR(255),
    api_key VARCHAR(255),
    sessionExpiry DATETIME(3),
    created DATETIME(3),
    enabled BOOLEAN
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_unicode_ci;

CREATE TABLE IF NOT EXISTS workflow_actions (
    id BIGINT AUTO_INCREMENT PRIMARY KEY,
    workflow_id BIGINT NOT NULL,
    execution_count INT,
    executor_id INT,
    retry_count INT,
    type TEXT,
    name TEXT,
    text LONGTEXT,
    date_time DATETIME(3),
    CONSTRAINT fk_workflow_actions_workflow FOREIGN KEY (workflow_id) REFERENCES workflow (id)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_unicode_ci;

-- seed admin user if not exists (idempotent)
INSERT INTO users (id, username, password, retry_count, session_id, api_key, sessionExpiry, created, enabled)
SELECT 1, 'admin', '$2a$12$5MohDGVmHcYuwfPuzoMemu5UmFlJu27sPj8KUn3jLToLjyV49eYly', 0,
       'd492c443e670f7ef54780fc94508900b2cdfdbabe2307b3aacaacdacce7b85ae',
       'b5f0e8c4-daa6-465c-bded-50ca22b798b2',
       '2025-09-17 10:08:52.002906',
       '2025-09-16 13:57:32.520000',
       true
WHERE NOT EXISTS (SELECT 1 FROM users WHERE id = 1 OR username = 'admin');
