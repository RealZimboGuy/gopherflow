package repository

import (
	"database/sql"
	"github.com/RealZimboGuy/gopherflow/internal/domain"
	"strings"
	"time"
)

// ExecutorRepository provides persistence for executors table.
type ExecutorRepository struct {
	db *sql.DB
}

func NewExecutorRepository(db *sql.DB) *ExecutorRepository {
	return &ExecutorRepository{db: db}
}

// Save inserts a new executor row and returns its ID.

func (r *ExecutorRepository) Save(e *domain.Executor) (int64, error) {
	// Ensure timestamps are set if zero; started defaults to now if unset
	var started time.Time = e.Started
	if started.IsZero() {
		started = time.Now()
	}
	var lastActive time.Time = e.LastActive
	if lastActive.IsZero() {
		lastActive = started
	}
	vals := []interface{}{e.Name, formatDateInDatabase(started), formatDateInDatabase(lastActive)}
	pps := []string{placeholder(1), placeholder(2), placeholder(3)}
	base := `INSERT INTO executors (name, started, last_active) VALUES (` + strings.Join(pps, ", ") + `)`
	if supportsReturning() {
		query := base + " RETURNING id"
		if err := r.db.QueryRow(query, vals...).Scan(&e.ID); err != nil {
			return 0, err
		}
	} else {
		res, err := r.db.Exec(base, vals...)
		if err != nil {
			return 0, err
		}
		id, err := res.LastInsertId()
		if err != nil {
			return 0, err
		}
		e.ID = id
	}
	// Update struct with any defaults we applied
	e.Started = started
	e.LastActive = lastActive
	return e.ID, nil
}

// UpdateLastActive sets last_active for the executor id to the provided timestamp.
func (r *ExecutorRepository) UpdateLastActive(id int64, ts time.Time) error {
	query := `UPDATE executors SET last_active = ` + placeholder(1) + ` WHERE id = ` + placeholder(2) + ``
	_, err := r.db.Exec(query, formatDateInDatabase(ts), id)
	return err
}
func (r *ExecutorRepository) GetExecutorsByLastActive(limit int) ([]*domain.Executor, error) {
	query := `
		SELECT id, name, started, last_active
		FROM executors
		ORDER BY last_active DESC
		LIMIT ` + placeholder(1) + `
	`
	rows, err := r.db.Query(query, limit)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var executors []*domain.Executor
	for rows.Next() {
		var e domain.Executor
		//var lastActive time.Time
		if err := rows.Scan(&e.ID, &e.Name, &e.Started, &e.LastActive); err != nil {
			return nil, err
		}

		//e.LastActive = lastActive.UTC() // treat DB time as UTC

		executors = append(executors, &e)
	}
	if err := rows.Err(); err != nil {
		return nil, err
	}
	return executors, nil
}
