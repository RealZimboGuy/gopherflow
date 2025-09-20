-- Drop performance indexes
DROP INDEX IF EXISTS idx_workflow_status;
DROP INDEX IF EXISTS idx_workflow_next_activation;
DROP INDEX IF EXISTS idx_workflow_status_next_activation;
DROP INDEX IF EXISTS idx_workflow_executor_group;
DROP INDEX IF EXISTS idx_workflow_executor_id;
DROP INDEX IF EXISTS idx_workflow_workflow_type;
DROP INDEX IF EXISTS idx_workflow_state;
DROP INDEX IF EXISTS idx_workflow_external_id;
DROP INDEX IF EXISTS idx_workflow_business_key;
DROP INDEX IF EXISTS idx_workflow_modified;
DROP INDEX IF EXISTS idx_workflow_execgrp_status_next;
DROP INDEX IF EXISTS idx_workflow_status_modified;

DROP INDEX IF EXISTS idx_users_session_id;
DROP INDEX IF EXISTS idx_users_session_id_sessionExpiry;
DROP INDEX IF EXISTS idx_users_api_key;

DROP INDEX IF EXISTS idx_workflow_actions_workflow_id;

DROP INDEX IF EXISTS idx_executors_last_active;
