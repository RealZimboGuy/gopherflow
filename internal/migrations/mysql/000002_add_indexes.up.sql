-- Add performance indexes for common query patterns (MySQL)
-- Workflow indexes
CREATE INDEX idx_workflow_status ON workflow (status(191));
CREATE INDEX idx_workflow_next_activation ON workflow (next_activation);
CREATE INDEX idx_workflow_status_next_activation ON workflow (status(191), next_activation);
CREATE INDEX idx_workflow_executor_group ON workflow (executor_group(191));
CREATE INDEX idx_workflow_executor_id ON workflow (executor_id);
CREATE INDEX idx_workflow_workflow_type ON workflow (workflow_type(191));
CREATE INDEX idx_workflow_state ON workflow (state(191));
CREATE INDEX idx_workflow_external_id ON workflow (external_id(191));
CREATE INDEX idx_workflow_business_key ON workflow (business_key(191));
CREATE INDEX idx_workflow_modified ON workflow (modified);
-- For pending selection combining filters and ordering
CREATE INDEX idx_workflow_execgrp_status_next ON workflow (executor_group(191), status(191), next_activation);
-- For executing list ordered by modified
CREATE INDEX idx_workflow_status_modified ON workflow (status(191), modified);

-- Users indexes
CREATE INDEX idx_users_session_id ON users (session_id);
CREATE INDEX idx_users_session_id_sessionExpiry ON users (session_id, sessionExpiry);
CREATE INDEX idx_users_api_key ON users (api_key);

-- Workflow actions indexes
CREATE INDEX idx_workflow_actions_workflow_id ON workflow_actions (workflow_id);

-- Executors indexes
CREATE INDEX idx_executors_last_active ON executors (last_active);
