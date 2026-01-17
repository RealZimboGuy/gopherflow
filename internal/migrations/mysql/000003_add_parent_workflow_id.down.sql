-- Remove foreign key constraint
ALTER TABLE workflow DROP FOREIGN KEY fk_parent_workflow;

-- Remove parent_workflow_id column
ALTER TABLE workflow DROP COLUMN parent_workflow_id;
