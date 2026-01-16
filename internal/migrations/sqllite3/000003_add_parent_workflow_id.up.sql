-- Add parent_workflow_id column to workflow table
ALTER TABLE workflow ADD COLUMN parent_workflow_id INTEGER NULL REFERENCES workflow(id);
