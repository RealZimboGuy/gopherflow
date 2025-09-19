-- Initial schema
CREATE TABLE IF NOT EXISTS workflow (
    id BIGSERIAL PRIMARY KEY,
    status TEXT,
    execution_count INT,
    retry_count INT,
    created TIMESTAMPTZ,
    modified TIMESTAMPTZ,
    next_activation TIMESTAMPTZ,
    started TIMESTAMPTZ,
    executor_id BIGINT,
    executor_group TEXT,
    workflow_type TEXT,
    external_id TEXT,
    business_key TEXT,
    state TEXT,
    state_vars TEXT
);

CREATE TABLE IF NOT EXISTS executors (
    id BIGSERIAL PRIMARY KEY,
    name TEXT,
    started TIMESTAMPTZ,
    last_active TIMESTAMPTZ
);

CREATE TABLE IF NOT EXISTS workflow_definitions (
    name TEXT PRIMARY KEY,
    description TEXT,
    created TIMESTAMPTZ,
    updated TIMESTAMPTZ,
    flow_chart TEXT
);

-- final users schema (deduplicated)
CREATE TABLE IF NOT EXISTS users (
    id BIGSERIAL PRIMARY KEY,
    username TEXT NOT NULL UNIQUE,
    password TEXT NOT NULL,
    retry_count INT,
    session_id TEXT,
    api_key TEXT,
    sessionExpiry TIMESTAMPTZ,
    created TIMESTAMPTZ,
    enabled BOOLEAN
);

CREATE TABLE IF NOT EXISTS workflow_actions (
    id BIGSERIAL PRIMARY KEY,
    workflow_id BIGINT NOT NULL,
    execution_count INT,
    executor_id INT,
    retry_count INT,
    type TEXT,
    name TEXT,
    text TEXT,
    date_time TIMESTAMPTZ,
    CONSTRAINT fk_workflow_actions_workflow FOREIGN KEY (workflow_id) REFERENCES workflow (id)
);

-- seed admin user if not exists (idempotent)
INSERT INTO public.users (id, username, password, retry_count, session_id, api_key, sessionexpiry, created, enabled)
SELECT 1, 'admin', '$2a$12$5MohDGVmHcYuwfPuzoMemu5UmFlJu27sPj8KUn3jLToLjyV49eYly', 0,
       'd492c443e670f7ef54780fc94508900b2cdfdbabe2307b3aacaacdacce7b85ae',
       'b5f0e8c4-daa6-465c-bded-50ca22b798b2',
       '2025-09-17 10:08:52.002906+00',
       '2025-09-16 13:57:32.520000+00',
       true
WHERE NOT EXISTS (SELECT 1 FROM public.users WHERE id = 1 OR username = 'admin');
