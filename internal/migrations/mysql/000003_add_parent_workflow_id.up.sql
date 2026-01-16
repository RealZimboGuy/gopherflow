-- Add parent_workflow_id column to workflow table
ALTER TABLE workflow ADD COLUMN parent_workflow_id BIGINT NULL;
-- Add foreign key constraint
ALTER TABLE workflow ADD CONSTRAINT fk_parent_workflow
    FOREIGN KEY (parent_workflow_id)
    REFERENCES workflow (id);
