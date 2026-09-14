package domain

import (
	"database/sql"
)

// User is a console or API account. Password holds a bcrypt hash, ApiKey
// authenticates REST requests, and SessionID with SessionExpiry backs the
// console's cookie session.
type User struct {
	ID            int64          `json:"id"`
	Username      string         `json:"username"`
	Password      string         `json:"password"`
	RetryCount    sql.NullInt32  `json:"retryCount"`
	SessionID     sql.NullString `json:"sessionId"`
	ApiKey        sql.NullString `json:"apiKey"`
	SessionExpiry sql.NullTime   `json:"sessionExpiry"`
	Created       sql.NullTime   `json:"created"`
	Enabled       sql.NullBool   `json:"enabled"`
}
