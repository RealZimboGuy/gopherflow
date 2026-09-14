package repository

import (
	"os"
	"strings"
	"testing"
	"time"

	"github.com/RealZimboGuy/gopherflow/internal/config"
)

// withDBType sets the database type for the duration of one test. The dialect
// helpers read it from the environment on every call.
func withDBType(t *testing.T, dbType string) {
	t.Helper()
	prev, had := os.LookupEnv(config.DATABASE_TYPE)
	if err := os.Setenv(config.DATABASE_TYPE, dbType); err != nil {
		t.Fatalf("setenv: %v", err)
	}
	t.Cleanup(func() {
		if had {
			_ = os.Setenv(config.DATABASE_TYPE, prev)
		} else {
			_ = os.Unsetenv(config.DATABASE_TYPE)
		}
	})
}

// fixedClock returns a constant time so formatted SQL literals are comparable.
type fixedClock struct{ t time.Time }

func (c fixedClock) Now() time.Time                         { return c.t }
func (c fixedClock) After(d time.Duration) <-chan time.Time { return time.After(d) }
func (c fixedClock) Sleep(d time.Duration)                  { time.Sleep(d) }

func TestPlaceholder(t *testing.T) {
	tests := []struct {
		dbType string
		index  int
		want   string
	}{
		{config.DATABASE_TYPE_POSTGRES, 1, "$1"},
		{config.DATABASE_TYPE_POSTGRES, 7, "$7"},
		{config.DATABASE_TYPE_MYSQL, 1, "?"},
		{config.DATABASE_TYPE_MYSQL, 7, "?"},
		{config.DATABASE_TYPE_SQLLITE, 1, "?"},
		{"", 1, "?"},
	}
	for _, tt := range tests {
		t.Run(tt.dbType+"/"+tt.want, func(t *testing.T) {
			withDBType(t, tt.dbType)
			if got := placeholder(tt.index); got != tt.want {
				t.Errorf("placeholder(%d) with %q = %q, want %q", tt.index, tt.dbType, got, tt.want)
			}
		})
	}
}

func TestSupportsReturning(t *testing.T) {
	tests := []struct {
		dbType string
		want   bool
	}{
		{config.DATABASE_TYPE_POSTGRES, true},
		{config.DATABASE_TYPE_MYSQL, false},
		{config.DATABASE_TYPE_SQLLITE, false},
		{"", false},
	}
	for _, tt := range tests {
		t.Run(tt.dbType, func(t *testing.T) {
			withDBType(t, tt.dbType)
			if got := supportsReturning(); got != tt.want {
				t.Errorf("supportsReturning() with %q = %v, want %v", tt.dbType, got, tt.want)
			}
		})
	}
}

func TestNowFunc(t *testing.T) {
	clock := fixedClock{t: time.Date(2026, 3, 4, 5, 6, 7, 123456000, time.UTC)}

	tests := []struct {
		dbType string
		want   string
	}{
		{config.DATABASE_TYPE_POSTGRES, "'2026-03-04 05:06:07.123456'"},
		{config.DATABASE_TYPE_MYSQL, "'2026-03-04 05:06:07.123456'"},
		{config.DATABASE_TYPE_SQLLITE, "'2026-03-04 05:06:07.123'"},
		{"", "'2026-03-04 05:06:07.123456'"},
	}
	for _, tt := range tests {
		t.Run(tt.dbType, func(t *testing.T) {
			withDBType(t, tt.dbType)
			if got := nowFunc(clock); got != tt.want {
				t.Errorf("nowFunc() with %q = %s, want %s", tt.dbType, got, tt.want)
			}
		})
	}
}

func TestDateBeforeNow(t *testing.T) {
	clock := fixedClock{t: time.Date(2026, 3, 4, 5, 6, 7, 0, time.UTC)}

	t.Run("postgres compares directly", func(t *testing.T) {
		withDBType(t, config.DATABASE_TYPE_POSTGRES)
		got := dateBeforeNow("modified", clock)
		if !strings.HasPrefix(got, "modified < '") {
			t.Errorf("got %q, want a direct comparison", got)
		}
		if strings.Contains(got, "julianday") {
			t.Errorf("got %q, should not use julianday for postgres", got)
		}
	})

	t.Run("sqlite coerces with julianday", func(t *testing.T) {
		withDBType(t, config.DATABASE_TYPE_SQLLITE)
		got := dateBeforeNow("modified", clock)
		if !strings.Contains(got, "julianday(modified)") {
			t.Errorf("got %q, want julianday coercion", got)
		}
	})

	t.Run("unknown type falls back to the sqlite form", func(t *testing.T) {
		withDBType(t, "")
		if got := dateBeforeNow("modified", clock); !strings.Contains(got, "julianday") {
			t.Errorf("got %q, want the julianday fallback", got)
		}
	})
}
