package repository

import (
	"database/sql"
	"log/slog"

	"github.com/RealZimboGuy/gopherflow/pkg/gopherflow/core"
	"github.com/RealZimboGuy/gopherflow/pkg/gopherflow/domain"
)

// WorkflowActionRepository provides methods to persist and query workflow action records.
type WorkflowActionRepository struct {
	db    *sql.DB
	clock core.Clock
}

func NewWorkflowActionRepository(db *sql.DB, clock core.Clock) *WorkflowActionRepository {
	return &WorkflowActionRepository{db: db, clock: clock}
}

// Save inserts a new workflow action and returns its ID.
// It expects the following table schema (PostgreSQL):
//
//	workflow_actions(id BIGSERIAL PK, workflow_id BIGINT, executor_id BIGINT, execution_count INT,
//	                retry_count INT, type TEXT, name TEXT, text TEXT, date_time TIMESTAMP)
func (r *WorkflowActionRepository) Save(a *domain.WorkflowAction) (int64, error) {
	base := `
		INSERT INTO workflow_actions (
			workflow_id, executor_id, execution_count, retry_count, type, name, text, date_time
		) VALUES (
			` + placeholder(1) + `, ` + placeholder(2) + `, ` + placeholder(3) + `, ` + placeholder(4) + `, ` + placeholder(5) + `, ` + placeholder(6) + `, ` + placeholder(7) + `, ` + placeholder(8) + `
		)`
	var err error
	if supportsReturning() {
		query := base + " RETURNING id"
		err = r.db.QueryRow(
			query,
			a.WorkflowID,
			a.ExecutorID,
			a.ExecutionCount,
			a.RetryCount,
			a.Type,
			a.Name,
			a.Text,
			a.DateTime,
		).Scan(&a.ID)
	} else {
		res, e := r.db.Exec(base,
			a.WorkflowID,
			a.ExecutorID,
			a.ExecutionCount,
			a.RetryCount,
			a.Type,
			a.Name,
			a.Text,
			a.DateTime,
		)
		if e != nil {
			err = e
		} else {
			id, e2 := res.LastInsertId()
			if e2 != nil {
				err = e2
			} else {
				a.ID = id
			}
		}
	}

	if err != nil {
		slog.Error("Failed to save workflow action", "error", err)
	}

	return a.ID, err
}

// FindByID fetches a single workflow action by its ID.
func (r *WorkflowActionRepository) FindByID(id int64) (*domain.WorkflowAction, error) {
	query := `
		SELECT id, workflow_id, executor_id, execution_count, retry_count, type, name, text, date_time
		FROM workflow_actions
		WHERE id = ` + placeholder(1) + `
	`
	var a domain.WorkflowAction
	err := r.db.QueryRow(query, id).Scan(
		&a.ID,
		&a.WorkflowID,
		&a.ExecutorID,
		&a.ExecutionCount,
		&a.RetryCount,
		&a.Type,
		&a.Name,
		&a.Text,
		&a.DateTime,
	)
	if err != nil {
		return nil, err
	}
	return &a, nil
}

// FindAllByWorkflowID returns all actions for a specific workflow ordered by date_time ascending.
func (r *WorkflowActionRepository) FindAllByWorkflowID(workflowID int64) (*[]domain.WorkflowAction, error) {
	query := `
		SELECT id, workflow_id, executor_id, execution_count, retry_count, type, name, text, date_time
		FROM workflow_actions
		WHERE workflow_id = ` + placeholder(1) + `
		ORDER BY  id DESC
	`
	rows, err := r.db.Query(query, workflowID)
	defer rows.Close()

	if err != nil {
		return nil, err
	}

	var actions []domain.WorkflowAction
	for rows.Next() {
		var a domain.WorkflowAction
		if err := rows.Scan(
			&a.ID,
			&a.WorkflowID,
			&a.ExecutorID,
			&a.ExecutionCount,
			&a.RetryCount,
			&a.Type,
			&a.Name,
			&a.Text,
			&a.DateTime,
		); err != nil {
			return nil, err
		}
		actions = append(actions, a)
	}
	return &actions, nil
}
