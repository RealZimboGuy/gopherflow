package repository

import (
	"database/sql"
	"time"

	"github.com/RealZimboGuy/gopherflow/pkg/gopherflow/core"
	"github.com/RealZimboGuy/gopherflow/pkg/gopherflow/domain"
)

// UserRepository provides persistence methods for the users table.
type UserRepository struct {
	db    *sql.DB
	clock core.Clock
}

func NewUserRepository(db *sql.DB, clock core.Clock) *UserRepository {
	return &UserRepository{db: db, clock: clock}
}

// Save inserts a new user and returns its generated id.
// It will set Created to now if it's not provided (null or zero).
func (r *UserRepository) Save(u *domain.User) (int64, error) {
	// Ensure created timestamp is set if not valid
	if !u.Created.Valid {
		u.Created = sql.NullTime{Time: r.clock.Now().UTC(), Valid: true}
	}

	base := `
        INSERT INTO users (username, password, retry_count, session_id,api_key, sessionExpiry, created, enabled)
        VALUES (` + placeholder(1) + `,` + placeholder(2) + `,` + placeholder(3) + `,` + placeholder(4) + `,` + placeholder(5) + `,` + placeholder(6) + `,` + placeholder(7) + `,` + placeholder(8) + `)
    `

	var id int64
	var err error
	if supportsReturning() {
		err = r.db.QueryRow(
			base+" RETURNING id",
			u.Username,
			u.Password,
			u.RetryCount,
			u.SessionID,
			u.ApiKey,
			u.SessionExpiry,
			u.Created,
			u.Enabled,
		).Scan(&id)
	} else {
		res, e := r.db.Exec(base,
			u.Username,
			u.Password,
			u.RetryCount,
			u.SessionID,
			u.ApiKey,
			u.SessionExpiry,
			u.Created,
			u.Enabled,
		)
		if e != nil {
			err = e
		} else {
			newID, e2 := res.LastInsertId()
			if e2 != nil {
				err = e2
			} else {
				id = newID
			}
		}
	}
	if err != nil {
		return 0, err
	}
	u.ID = id
	return id, nil
}

// FindByUsername fetches a user by exact username. Returns (nil, nil) if not found.
func (r *UserRepository) FindByUsername(username string) (*domain.User, error) {
	query := `
        SELECT id, username, password, retry_count, session_id, api_key,sessionExpiry, created, enabled
        FROM users
        WHERE username =` + placeholder(1) + `
        LIMIT 1
    `

	var u domain.User
	err := r.db.QueryRow(query, username).Scan(
		&u.ID,
		&u.Username,
		&u.Password,
		&u.RetryCount,
		&u.SessionID,
		&u.ApiKey,
		&u.SessionExpiry,
		&u.Created,
		&u.Enabled,
	)
	if err == sql.ErrNoRows {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}
	return &u, nil
}

// FindBySessionID fetches a user by session_id and ensures sessionExpiry is in the future.
func (r *UserRepository) FindBySessionID(sessionID string, now time.Time) (*domain.User, error) {
	query := `
        SELECT id, username, password, retry_count, session_id,api_key, sessionExpiry, created, enabled
        FROM users
        WHERE session_id = ` + placeholder(1) + ` AND sessionExpiry > ` + placeholder(2) + `
        LIMIT 1
    `
	var u domain.User
	err := r.db.QueryRow(query, sessionID, now).Scan(
		&u.ID,
		&u.Username,
		&u.Password,
		&u.RetryCount,
		&u.SessionID,
		&u.ApiKey,
		&u.SessionExpiry,
		&u.Created,
		&u.Enabled,
	)
	if err == sql.ErrNoRows {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}
	return &u, nil
}

// UpdateSession sets session_id and sessionExpiry for a user by id.
func (r *UserRepository) UpdateSession(userID int64, sessionID string, expiry time.Time) error {
	query := `
        UPDATE users
        SET session_id = ` + placeholder(1) + `, sessionExpiry = ` + placeholder(2) + `
        WHERE id = ` + placeholder(3) + `
    `
	_, err := r.db.Exec(query, sessionID, formatDateInDatabase(expiry), userID)
	return err
}

// ClearSessionBySessionID nulls session_id and sessionExpiry for the user with the given current session_id.
func (r *UserRepository) ClearSessionBySessionID(sessionID string) error {
	query := `
        UPDATE users
        SET session_id = NULL, sessionExpiry = NULL
        WHERE session_id =` + placeholder(1) + `
    `
	_, err := r.db.Exec(query, sessionID)
	return err
}

// FindByApiKey fetches a user by api_key (exact match). Returns (nil, nil) if not found.
func (r *UserRepository) FindByApiKey(apiKey string) (*domain.User, error) {
	query := `
        SELECT id, username, password, retry_count, session_id, api_key,sessionExpiry, created, enabled
        FROM users
        WHERE api_key = ` + placeholder(1) + `
        LIMIT 1
    `
	var u domain.User
	err := r.db.QueryRow(query, apiKey).Scan(
		&u.ID,
		&u.Username,
		&u.Password,
		&u.RetryCount,
		&u.SessionID,
		&u.ApiKey,
		&u.SessionExpiry,
		&u.Created,
		&u.Enabled,
	)
	if err == sql.ErrNoRows {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}
	return &u, nil
}

// FindAll returns all users ordered by id ascending.
func (r *UserRepository) FindAll() (*[]domain.User, error) {
	query := `
        SELECT id, username, password, retry_count, session_id, api_key,sessionExpiry, created, enabled
        FROM users
        ORDER BY id ASC
    `

	rows, err := r.db.Query(query)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	users := make([]domain.User, 0)
	for rows.Next() {
		var u domain.User
		if err := rows.Scan(
			&u.ID,
			&u.Username,
			&u.Password,
			&u.RetryCount,
			&u.SessionID,
			&u.ApiKey,
			&u.SessionExpiry,
			&u.Created,
			&u.Enabled,
		); err != nil {
			return nil, err
		}
		users = append(users, u)
	}

	if err := rows.Err(); err != nil {
		return nil, err
	}

	return &users, nil
}
