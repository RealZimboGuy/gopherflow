package sqllite

import (
	"database/sql"
	"os"
	"sync"
	"testing"

	_ "modernc.org/sqlite"
)

// TestSqliteConcurrentWrites guards the busy_timeout setting.
//
// modernc.org/sqlite defaults busy_timeout to 0, unlike mattn/go-sqlite3 which
// defaulted to 5000ms. Without an explicit busy timeout, 15 concurrent writers
// (the GFLOW_ENGINE_EXECUTOR_SIZE default) fail the great majority of writes
// with SQLITE_BUSY. If this test starts failing, check that the DSN still
// carries _busy_timeout.
func TestSqliteConcurrentWrites(t *testing.T) {
	RunTestWithSetup(t, func(t *testing.T, port int) {
		fileName := os.Getenv("GFLOW_DATABASE_SQLLITE_FILE_NAME")
		dsn := fileName + "?_busy_timeout=10000&_pragma=journal_mode(WAL)"

		db, err := sql.Open("sqlite", dsn)
		if err != nil {
			t.Fatalf("open: %v", err)
		}
		defer db.Close()

		if _, err := db.Exec(`CREATE TABLE concurrency_probe (id INTEGER PRIMARY KEY AUTOINCREMENT, v TEXT)`); err != nil {
			t.Fatalf("create table: %v", err)
		}

		const writers, perWriter = 15, 40

		var wg sync.WaitGroup
		errCh := make(chan error, writers*perWriter)
		for i := 0; i < writers; i++ {
			wg.Add(1)
			go func() {
				defer wg.Done()
				for j := 0; j < perWriter; j++ {
					tx, err := db.Begin()
					if err != nil {
						errCh <- err
						continue
					}
					if _, err := tx.Exec(`INSERT INTO concurrency_probe (v) VALUES (?)`, "x"); err != nil {
						errCh <- err
						_ = tx.Rollback()
						continue
					}
					if err := tx.Commit(); err != nil {
						errCh <- err
					}
				}
			}()
		}
		wg.Wait()
		close(errCh)

		for err := range errCh {
			t.Fatalf("concurrent write failed (is _busy_timeout set?): %v", err)
		}

		var count int
		if err := db.QueryRow(`SELECT count(*) FROM concurrency_probe`).Scan(&count); err != nil {
			t.Fatalf("count: %v", err)
		}
		if want := writers * perWriter; count != want {
			t.Fatalf("wrote %d rows, want %d", count, want)
		}
	})
}
