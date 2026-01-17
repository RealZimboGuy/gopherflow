-- SQLite doesn't support DROP COLUMN in older versions
-- Creating a new table without the column, copying data, and renaming
CREATE TABLE workflow_new (
    id INTEGER PRIMARY KEY AUTOINCREMENT,
    status TEXT,
    execution_count INT,
    retry_count INT,
    created TIMESTAMP,
    modified TIMESTAMP,
    next_activation TIMESTAMP,
    started TIMESTAMP,
    executor_id INT,
    executor_group TEXT,
    workflow_type TEXT,
    external_id TEXT,
    business_key TEXT,
    state TEXT,
    state_vars TEXT
);

-- Copy data from old table to new table
INSERT INTO workflow_new
SELECT id, status, execution_count, retry_count, created, modified, next_activation, 
       started, executor_id, executor_group, workflow_type, external_id, business_key, 
       state, state_vars
FROM workflow;

-- Drop old table
DROP TABLE workflow;

-- Rename new table to old table name
ALTER TABLE workflow_new RENAME TO workflow;
