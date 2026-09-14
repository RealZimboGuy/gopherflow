package gopherflow

import "strings"

// sqliteDSNParams are appended to every SQLite DSN the engine opens.
//
// _busy_timeout is mandatory, not tuning. github.com/mattn/go-sqlite3 defaulted
// busy_timeout to 5000ms; modernc.org/sqlite defaults it to 0, so without this
// the engine's concurrent executors fail the overwhelming majority of writes
// with SQLITE_BUSY ("database is locked").
//
// journal_mode(WAL) lets readers run alongside a writer. It does not on its own
// prevent SQLITE_BUSY — the busy timeout is what does that.
var sqliteDSNParams = []string{
	"_busy_timeout=10000",
	"_pragma=journal_mode(WAL)",
}

// sqliteDSN appends the engine's required SQLite parameters to fileName,
// joining with "?" or "&" depending on whether fileName already carries a
// query string. A parameter the caller already set is left alone.
func sqliteDSN(fileName string) string {
	dsn := fileName
	for _, param := range sqliteDSNParams {
		key := param[:strings.Index(param, "=")]
		if strings.Contains(dsn, key+"=") {
			continue
		}
		sep := "?"
		if strings.Contains(dsn, "?") {
			sep = "&"
		}
		dsn += sep + param
	}
	return dsn
}
