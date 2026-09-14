package gopherflow

import "testing"

func TestSqliteDSN(t *testing.T) {
	tests := []struct {
		name     string
		fileName string
		want     string
	}{
		{
			name:     "absolute path gets params with ?",
			fileName: "/data/gflow.db",
			want:     "/data/gflow.db?_busy_timeout=10000&_pragma=journal_mode(WAL)",
		},
		{
			name:     "relative path gets params with ?",
			fileName: "./gflow.db",
			want:     "./gflow.db?_busy_timeout=10000&_pragma=journal_mode(WAL)",
		},
		{
			name:     "dsn that already has a query joins with &",
			fileName: "file::memory:?cache=shared",
			want:     "file::memory:?cache=shared&_busy_timeout=10000&_pragma=journal_mode(WAL)",
		},
		{
			name:     "caller supplied busy_timeout is not overridden",
			fileName: "/data/gflow.db?_busy_timeout=500",
			want:     "/data/gflow.db?_busy_timeout=500&_pragma=journal_mode(WAL)",
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := sqliteDSN(tt.fileName); got != tt.want {
				t.Errorf("sqliteDSN(%q)\n got: %q\nwant: %q", tt.fileName, got, tt.want)
			}
		})
	}
}
