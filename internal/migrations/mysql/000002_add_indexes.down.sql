-- Drop performance indexes (MySQL)
DROP INDEX idx_workflow_status ON workflow;
DROP INDEX idx_workflow_next_activation ON workflow;
DROP INDEX idx_workflow_status_next_activation ON workflow;
DROP INDEX idx_workflow_executor_group ON workflow;
DROP INDEX idx_workflow_executor_id ON workflow;
DROP INDEX idx_workflow_workflow_type ON workflow;
DROP INDEX idx_workflow_state ON workflow;
DROP INDEX idx_workflow_external_id ON workflow;
DROP INDEX idx_workflow_business_key ON workflow;
DROP INDEX idx_workflow_modified ON workflow;
DROP INDEX idx_workflow_execgrp_status_next ON workflow;
DROP INDEX idx_workflow_status_modified ON workflow;

DROP INDEX idx_users_session_id ON users;
DROP INDEX idx_users_session_id_sessionExpiry ON users;
DROP INDEX idx_users_api_key ON users;

DROP INDEX idx_workflow_actions_workflow_id ON workflow_actions;

DROP INDEX idx_executors_last_active ON executors;
