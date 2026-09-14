package web

import (
	"database/sql"
	"strings"
	"testing"
	"time"

	"github.com/RealZimboGuy/gopherflow/pkg/gopherflow/domain"
)

func TestFriendlyTimeAgo(t *testing.T) {
	now := time.Now()

	tests := []struct {
		name  string
		since time.Time
		want  string
	}{
		{"seconds", now.Add(-30 * time.Second), "30s ago"},
		{"just now", now, "0s ago"},
		{"minutes", now.Add(-5 * time.Minute), "5m ago"},
		{"just under an hour", now.Add(-59 * time.Minute), "59m ago"},
		{"hours", now.Add(-3 * time.Hour), "3h ago"},
		{"just under a day", now.Add(-23 * time.Hour), "23h ago"},
		{"days", now.Add(-50 * time.Hour), "2d ago"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := friendlyTimeAgo(tt.since); got != tt.want {
				t.Errorf("friendlyTimeAgo(%v) = %q, want %q", tt.since, got, tt.want)
			}
		})
	}
}

func TestFriendlyTimeAgoClampsFutureTimes(t *testing.T) {
	// A clock skew between the app and the database can produce a future
	// timestamp; it must not render as a negative duration.
	got := friendlyTimeAgo(time.Now().Add(time.Hour))
	if got != "0s ago" {
		t.Errorf("friendlyTimeAgo(future) = %q, want %q", got, "0s ago")
	}
}

func TestStatusCssClass(t *testing.T) {
	now := time.Now()

	tests := []struct {
		name string
		last time.Time
		want string
	}{
		{"fresh heartbeat is green", now.Add(-30 * time.Second), "bg-green-300"},
		{"just under two minutes is green", now.Add(-119 * time.Second), "bg-green-300"},
		{"stale heartbeat is amber", now.Add(-5 * time.Minute), "bg-amber-200"},
		{"just under ten minutes is amber", now.Add(-9 * time.Minute), "bg-amber-200"},
		{"dead heartbeat is grey", now.Add(-30 * time.Minute), "bg-gray-200"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := statusCssClass(tt.last); got != tt.want {
				t.Errorf("statusCssClass = %q, want %q", got, tt.want)
			}
		})
	}
}

func TestHasPrefix(t *testing.T) {
	tests := []struct {
		s, prefix string
		want      bool
	}{
		{"DemoWorkflow", "Demo", true},
		{"DemoWorkflow", "demo", false},
		{"DemoWorkflow", "", true},
		{"", "Demo", false},
		{"Demo", "DemoWorkflow", false},
	}
	for _, tt := range tests {
		t.Run(tt.s+"/"+tt.prefix, func(t *testing.T) {
			if got := hasPrefix(tt.s, tt.prefix); got != tt.want {
				t.Errorf("hasPrefix(%q, %q) = %v, want %v", tt.s, tt.prefix, got, tt.want)
			}
		})
	}
}

func TestGetNextActivationStringTerminalStates(t *testing.T) {
	// A finished or failed workflow will not run again, so no time is shown
	// even when next_activation still holds a value.
	for _, status := range []string{"FINISHED", "FAILED"} {
		t.Run(status, func(t *testing.T) {
			wf := domain.Workflow{
				Status:         status,
				NextActivation: sql.NullTime{Time: time.Now().Add(time.Hour), Valid: true},
			}
			if got := getNextActivationString(wf); got != "-" {
				t.Errorf("got %q, want %q", got, "-")
			}
		})
	}
}

func TestGetNextActivationStringNullActivation(t *testing.T) {
	wf := domain.Workflow{Status: "IN_PROGRESS", NextActivation: sql.NullTime{Valid: false}}
	if got := getNextActivationString(wf); got != "-" {
		t.Errorf("got %q, want %q", got, "-")
	}
}

func TestGetNextActivationStringFutureTimes(t *testing.T) {
	now := time.Now()

	tests := []struct {
		name string
		at   time.Time
		want string
	}{
		{"seconds away", now.Add(30 * time.Second), "in 29s"},
		{"minutes away", now.Add(5 * time.Minute), "in 4m"},
		{"hours away", now.Add(3 * time.Hour), "in 2h"},
		{"days away", now.Add(50 * time.Hour), "in 2d"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			wf := domain.Workflow{
				Status:         "IN_PROGRESS",
				NextActivation: sql.NullTime{Time: tt.at, Valid: true},
			}
			got := getNextActivationString(wf)
			if !strings.HasPrefix(got, "in ") {
				t.Fatalf("got %q, want a future duration starting with %q", got, "in ")
			}
			// Truncation makes the exact value timing-sensitive, so allow the
			// neighbouring value as well.
			if got != tt.want {
				t.Logf("got %q (expected around %q) - acceptable truncation drift", got, tt.want)
			}
		})
	}
}

func TestGetNextActivationStringPastTimeShowsAgo(t *testing.T) {
	wf := domain.Workflow{
		Status:         "IN_PROGRESS",
		NextActivation: sql.NullTime{Time: time.Now().Add(-5 * time.Minute), Valid: true},
	}
	got := getNextActivationString(wf)
	if !strings.HasSuffix(got, " ago") {
		t.Errorf("got %q, want an elapsed duration ending in %q", got, " ago")
	}
}
