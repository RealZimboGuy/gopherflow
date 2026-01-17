-- Remove foreign key constraint
ALTER TABLE workflow DROP CONSTRAINT IF EXISTS fk_parent_workflow;

-- Remove parent_workflow_id column
ALTER TABLE workflow DROP COLUMN IF EXISTS parent_workflow_id;
