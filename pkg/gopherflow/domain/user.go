package domain

import (
	"database/sql"
)

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
