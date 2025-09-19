module github.com/RealZimboGuy/gopherflow

go 1.24.0

toolchain go1.24.3

require github.com/lib/pq v1.10.9 // or latest

require github.com/mattn/go-sqlite3 v1.14.32

require github.com/go-sql-driver/mysql v1.9.3

require (
	filippo.io/edwards25519 v1.1.0 // indirect
	github.com/hashicorp/errwrap v1.1.0 // indirect
	github.com/hashicorp/go-multierror v1.1.1 // indirect
)

require (
	github.com/golang-migrate/migrate/v4 v4.19.0
	github.com/lmittmann/tint v1.1.2
	golang.org/x/crypto v0.42.0
)
