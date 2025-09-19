package repository

import (
	_ "github.com/lib/pq"
)

// NewDB opens a Postgres connection using lib/pq and forces simple protocol
// to avoid server-side unnamed prepared statements which can trigger errors like:
// "pq: unnamed prepared statement does not exist" after connection resets.
// We do this by ensuring binary_parameters=no in the DSN.

//func NewDB(connStr string) (*sql.DB, error) {
//	db, err := sql.Open("postgres", connStr)
//	if err != nil {
//		return nil, err
//	}
//	// Reasonable pool settings to reduce stale connections
//	db.SetMaxOpenConns(10)
//	db.SetMaxIdleConns(10)
//	db.SetConnMaxLifetime(30 * time.Minute)
//	db.SetConnMaxIdleTime(10 * time.Minute)
//
//	if err := db.Ping(); err != nil {
//		return nil, err
//	}
//	return db, nil
//}

//
//// ensureBinaryParametersNo appends binary_parameters=no if not present.
//func ensureBinaryParametersNo(dsn string) string {
//	// If DSN is empty or not a URL we still try a simple append
//	if dsn == "" {
//		return "binary_parameters=no"
//	}
//	// Check if already contains binary_parameters
//	lower := strings.ToLower(dsn)
//	if strings.Contains(lower, "binary_parameters=") {
//		return dsn
//	}
//	// If it's a URL, append correctly
//	if strings.HasPrefix(lower, "postgres://") || strings.HasPrefix(lower, "postgresql://") {
//		u, err := url.Parse(dsn)
//		if err == nil {
//			q := u.Query()
//			q.Set("binary_parameters", "no")
//			u.RawQuery = q.Encode()
//			return u.String()
//		}
//	}
//	// Else treat as DSN key=val or other; append with proper separator
//	sep := "?"
//	if strings.Contains(dsn, "?") {
//		sep = "&"
//	}
//	return dsn + sep + "binary_parameters=no"
//}
//
//// ensureBinaryParametersNo appends binary_parameters=no if not present.
//func ensureDisabledPreparedStatements(dsn string) string {
//	// If DSN is empty or not a URL we still try a simple append
//	if dsn == "" {
//		return "disable_prepared_statements=true"
//	}
//	// Check if already contains binary_parameters
//	lower := strings.ToLower(dsn)
//	if strings.Contains(lower, "disable_prepared_statements=") {
//		return dsn
//	}
//	// If it's a URL, append correctly
//	if strings.HasPrefix(lower, "postgres://") || strings.HasPrefix(lower, "postgresql://") {
//		u, err := url.Parse(dsn)
//		if err == nil {
//			q := u.Query()
//			q.Set("disable_prepared_statements", "true")
//			u.RawQuery = q.Encode()
//			return u.String()
//		}
//	}
//	// Else treat as DSN key=val or other; append with proper separator
//	sep := "?"
//	if strings.Contains(dsn, "?") {
//		sep = "&"
//	}
//	return dsn + sep + "disable_prepared_statements=true"
//}
