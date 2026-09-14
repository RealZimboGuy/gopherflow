package repository

import (
	"database/sql"
	"strings"
	"testing"
	"time"

	"github.com/RealZimboGuy/gopherflow/internal/config"
	"github.com/RealZimboGuy/gopherflow/pkg/gopherflow/models"
)

func TestParsePostgresInterval(t *testing.T) {
	tests := []struct {
		in   string
		want time.Duration
	}{
		{"", 0},
		{"   ", 0},
		{"1 hour", time.Hour},
		{"2 hours", 2 * time.Hour},
		{"5 minutes", 5 * time.Minute},
		{"1 minute", time.Minute},
		{"30 seconds", 30 * time.Second},
		{"1 second", time.Second},
		{"250 ms", 250 * time.Millisecond},
		{"250 milliseconds", 250 * time.Millisecond},
		{"1 millisecond", time.Millisecond},
		{"1 hour 30 minutes", 90 * time.Minute},
		{"1 hour 30 minutes 15 seconds", 90*time.Minute + 15*time.Second},
		{"1HOUR", time.Hour},               // case insensitive
		{"1.5 hours", 90 * time.Minute},    // fractional
		{"-30 minutes", -30 * time.Minute}, // negative
		{"2hours", 2 * time.Hour},          // no space
	}
	for _, tt := range tests {
		t.Run(tt.in, func(t *testing.T) {
			got, err := ParsePostgresInterval(tt.in)
			if err != nil {
				t.Fatalf("ParsePostgresInterval(%q) returned error: %v", tt.in, err)
			}
			if got != tt.want {
				t.Errorf("ParsePostgresInterval(%q) = %v, want %v", tt.in, got, tt.want)
			}
		})
	}
}

func TestParsePostgresIntervalInvalid(t *testing.T) {
	for _, in := range []string{"tomorrow", "5", "abc", "5 fortnights"} {
		t.Run(in, func(t *testing.T) {
			if _, err := ParsePostgresInterval(in); err == nil {
				t.Errorf("ParsePostgresInterval(%q) = nil error, want an error", in)
			}
		})
	}
}

func TestBuildLimitsAndOffset(t *testing.T) {
	tests := []struct {
		name string
		req  models.SearchWorkflowRequest
		want string
	}{
		{"no limit yields no clause", models.SearchWorkflowRequest{}, ""},
		{"limit only", models.SearchWorkflowRequest{Limit: 25}, " LIMIT 25 OFFSET 0"},
		{"limit and offset", models.SearchWorkflowRequest{Limit: 10, Offset: 30}, " LIMIT 10 OFFSET 30"},
		{"offset without limit is ignored", models.SearchWorkflowRequest{Offset: 30}, ""},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := buildLimitsAndOffset(tt.req); got != tt.want {
				t.Errorf("got %q, want %q", got, tt.want)
			}
		})
	}
}

func TestBuildWhereClauseEmptyRequest(t *testing.T) {
	withDBType(t, config.DATABASE_TYPE_SQLLITE)

	clause, args := buildWhereClause(models.SearchWorkflowRequest{})
	if clause != "" {
		t.Errorf("clause = %q, want empty", clause)
	}
	if len(args) != 0 {
		t.Errorf("args = %v, want none", args)
	}
}

func TestBuildWhereClauseAndFilters(t *testing.T) {
	withDBType(t, config.DATABASE_TYPE_SQLLITE)

	clause, args := buildWhereClause(models.SearchWorkflowRequest{
		ExecutorGroup: "default",
		WorkflowType:  "DemoWorkflow",
		State:         "Review",
		Status:        "IN_PROGRESS",
	})

	for _, want := range []string{"executor_group = ?", "workflow_type = ?", "state = ?", "status = ?"} {
		if !strings.Contains(clause, want) {
			t.Errorf("clause %q is missing %q", clause, want)
		}
	}
	if strings.Contains(clause, " OR ") {
		t.Errorf("clause %q should not contain OR when only AND filters are set", clause)
	}
	if len(args) != 4 {
		t.Errorf("args = %v, want 4 values", args)
	}
}

func TestBuildWhereClauseIdentityFiltersAreOred(t *testing.T) {
	withDBType(t, config.DATABASE_TYPE_SQLLITE)

	clause, args := buildWhereClause(models.SearchWorkflowRequest{
		ID:          7,
		ExternalID:  "ext-1",
		BusinessKey: "bk-1",
	})

	if !strings.Contains(clause, " OR ") {
		t.Errorf("clause %q should OR the identity filters together", clause)
	}
	if !strings.Contains(clause, "(") || !strings.Contains(clause, ")") {
		t.Errorf("clause %q should group the OR filters in parentheses", clause)
	}
	if len(args) != 3 {
		t.Errorf("args = %v, want 3 values", args)
	}
}

func TestBuildWhereClauseUsesPostgresPlaceholders(t *testing.T) {
	withDBType(t, config.DATABASE_TYPE_POSTGRES)

	clause, args := buildWhereClause(models.SearchWorkflowRequest{
		ExecutorGroup: "default",
		WorkflowType:  "DemoWorkflow",
	})

	if !strings.Contains(clause, "$1") || !strings.Contains(clause, "$2") {
		t.Errorf("clause %q should use numbered postgres placeholders", clause)
	}
	if len(args) != 2 {
		t.Errorf("args = %v, want 2 values", args)
	}
}

func TestFormatDateInDatabase(t *testing.T) {
	ts := time.Date(2026, 3, 4, 5, 6, 7, 123456000, time.UTC)

	tests := []struct {
		dbType string
		want   string
	}{
		{config.DATABASE_TYPE_SQLLITE, "2026-03-04 05:06:07.123"},
		{config.DATABASE_TYPE_MYSQL, "2026-03-04 05:06:07.123456"},
	}
	for _, tt := range tests {
		t.Run(tt.dbType, func(t *testing.T) {
			withDBType(t, tt.dbType)
			if got := formatDateInDatabase(ts); got != tt.want {
				t.Errorf("got %q, want %q", got, tt.want)
			}
		})
	}

	t.Run("postgres uses RFC3339", func(t *testing.T) {
		withDBType(t, config.DATABASE_TYPE_POSTGRES)
		got := formatDateInDatabase(ts)
		if !strings.Contains(got, "T") || !strings.HasSuffix(got, "Z") {
			t.Errorf("got %q, want an RFC3339 timestamp", got)
		}
	})
}

func TestFormatDateInDatabaseNull(t *testing.T) {
	ts := time.Date(2026, 3, 4, 5, 6, 7, 123456000, time.UTC)

	t.Run("invalid time yields nil", func(t *testing.T) {
		withDBType(t, config.DATABASE_TYPE_SQLLITE)
		if got := formatDateInDatabaseNull(sql.NullTime{Valid: false}); got != nil {
			t.Errorf("got %v, want nil", got)
		}
	})

	t.Run("sqlite yields a string", func(t *testing.T) {
		withDBType(t, config.DATABASE_TYPE_SQLLITE)
		got := formatDateInDatabaseNull(sql.NullTime{Time: ts, Valid: true})
		if got != "2026-03-04 05:06:07.123" {
			t.Errorf("got %v, want the sqlite string form", got)
		}
	})

	t.Run("mysql yields a string", func(t *testing.T) {
		withDBType(t, config.DATABASE_TYPE_MYSQL)
		got := formatDateInDatabaseNull(sql.NullTime{Time: ts, Valid: true})
		if got != "2026-03-04 05:06:07.123456" {
			t.Errorf("got %v, want the mysql string form", got)
		}
	})

	t.Run("postgres yields a time.Time", func(t *testing.T) {
		withDBType(t, config.DATABASE_TYPE_POSTGRES)
		got := formatDateInDatabaseNull(sql.NullTime{Time: ts, Valid: true})
		if _, ok := got.(time.Time); !ok {
			t.Errorf("got %T, want time.Time", got)
		}
	})
}
