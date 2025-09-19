package repository

import (
	"fmt"
	"gopherflow/internal/config"
	"time"
)

// placeholder returns the correct bind variable for the given index based on DB type.
// Postgres uses $1, $2... while MySQL and SQLite use ?
func placeholder(i int) string {
	db := config.GetSystemSettingString(config.DATABASE_TYPE)
	if db == config.DATABASE_TYPE_POSTGRES {
		return fmt.Sprintf("$%d", i)
	}
	return "?"
}

// nowFunc returns the appropriate current timestamp SQL function per DB.
// Prefer passing timestamps as parameters when possible, but for queries that need inline time
// expressions (e.g., ORDER BY CURRENT_TIMESTAMP), this helps.
func nowFunc() string {
	db := config.GetSystemSettingString(config.DATABASE_TYPE)
	switch db {
	case config.DATABASE_TYPE_POSTGRES:
		return "NOW()"
	case config.DATABASE_TYPE_SQLLITE:
		return fmt.Sprintf("'%s'", time.Now().UTC().Format("2006-01-02 15:04:05.000"))
	case config.DATABASE_TYPE_MYSQL:
		return "CURRENT_TIMESTAMP(6)"
	default: // SQLite
		return "CURRENT_TIMESTAMP"
	}
}

// dateBeforeNow returns a DB-specific SQL predicate that checks if the provided
// datetime column is strictly before the current time. This avoids string
// comparisons in SQLite by coercing via julianday().
func dateBeforeNow(column string) string {
	db := config.GetSystemSettingString(config.DATABASE_TYPE)
	switch db {
	case config.DATABASE_TYPE_POSTGRES:
		return fmt.Sprintf("%s < NOW()", column)
	case config.DATABASE_TYPE_SQLLITE:
		return fmt.Sprintf(
			"substr(%s,1,19) < '%s'",
			column,
			time.Now().Format("2006-01-02 15:04:05"),
		)
	case config.DATABASE_TYPE_MYSQL:
		return fmt.Sprintf("%s < CURRENT_TIMESTAMP(6)", column)
	default: // SQLite
		// julianday handles TEXT, REAL, or INTEGER date storage properly
		return fmt.Sprintf("julianday(%s) < julianday('now')", column)
	}
}

func supportsReturning() bool {
	return config.GetSystemSettingString(config.DATABASE_TYPE) == config.DATABASE_TYPE_POSTGRES
}
