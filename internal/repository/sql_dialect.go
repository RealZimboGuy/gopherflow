package repository

import (
	"fmt"

	"github.com/RealZimboGuy/gopherflow/internal/config"
	"github.com/RealZimboGuy/gopherflow/pkg/gopherflow/core"
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

func nowFunc(clock core.Clock) string {
	// Format the clock's current time in UTC with microsecond precision

	db := config.GetSystemSettingString(config.DATABASE_TYPE)
	switch db {
	case config.DATABASE_TYPE_POSTGRES, config.DATABASE_TYPE_MYSQL:
		// Quote the timestamp literal for SQL
		return fmt.Sprintf("'%s'", clock.Now().UTC().Format("2006-01-02 15:04:05.000000"))
	case config.DATABASE_TYPE_SQLLITE:
		return fmt.Sprintf("'%s'", clock.Now().UTC().Format("2006-01-02 15:04:05.000"))
	default:
		return fmt.Sprintf("'%s'", clock.Now().UTC().Format("2006-01-02 15:04:05.000000"))
	}
}

// dateBeforeNow returns a DB-specific SQL predicate that checks if the provided
// datetime column is strictly before the current time. This avoids string
// comparisons in SQLite by coercing via julianday().
func dateBeforeNow(column string, clock core.Clock) string {
	now := clock.Now().UTC().Format("2006-01-02 15:04:05.000")

	db := config.GetSystemSettingString(config.DATABASE_TYPE)
	switch db {
	case config.DATABASE_TYPE_POSTGRES, config.DATABASE_TYPE_MYSQL:
		// Can compare directly
		return fmt.Sprintf("%s < '%s'", column, now)
	case config.DATABASE_TYPE_SQLLITE:
		// Use julianday for SQLite so TEXT/REAL/INTEGER timestamps are comparable
		return fmt.Sprintf("julianday(%s) < julianday('%s')", column, now)
	default:
		// Fallback to SQLite-compatible
		return fmt.Sprintf("julianday(%s) < julianday('%s')", column, now)
	}
}

func supportsReturning() bool {
	return config.GetSystemSettingString(config.DATABASE_TYPE) == config.DATABASE_TYPE_POSTGRES
}
