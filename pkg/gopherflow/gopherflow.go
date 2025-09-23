package gopherflow

import (
	"database/sql"
	"io/fs"
	"log"
	"log/slog"
	"net/http"
	"os"
	"reflect"
	"strings"
	"time"

	"github.com/RealZimboGuy/gopherflow/internal/config"
	"github.com/RealZimboGuy/gopherflow/internal/controllers"
	"github.com/RealZimboGuy/gopherflow/internal/engine"
	"github.com/RealZimboGuy/gopherflow/internal/migrations"
	"github.com/RealZimboGuy/gopherflow/internal/repository"
	"github.com/RealZimboGuy/gopherflow/internal/web"

	"github.com/golang-migrate/migrate/v4/source/iofs"
	"github.com/lmittmann/tint"

	_ "github.com/go-sql-driver/mysql"
	migrate "github.com/golang-migrate/migrate/v4"
	_ "github.com/golang-migrate/migrate/v4/database/mysql"
	_ "github.com/golang-migrate/migrate/v4/database/postgres"
	_ "github.com/golang-migrate/migrate/v4/database/sqlite3"
	_ "github.com/golang-migrate/migrate/v4/source/iofs"
	_ "github.com/mattn/go-sqlite3"
)

var WorkflowRegistry map[string]reflect.Type

// Start boots the workflow engine and HTTP server.
// It expects engine.WorkflowRegistry to be populated by the caller before invocation.
// This call blocks until the HTTP server stops.
func Start(mux *http.ServeMux) error {

	databaseType := config.GetSystemSettingString(config.DATABASE_TYPE)
	if databaseType == "" || (databaseType != config.DATABASE_TYPE_POSTGRES && databaseType != config.DATABASE_TYPE_MYSQL && databaseType != config.DATABASE_TYPE_SQLLITE) {
		panic("GFLOW_DATABASE_TYPE must be set to one of the following values: POSTGRES, MYSQL, SQLLITE")
	}

	var db *sql.DB
	if databaseType == config.DATABASE_TYPE_POSTGRES {
		db = setupPostgresDatabase()
		defer db.Close()
	}
	if databaseType == config.DATABASE_TYPE_SQLLITE {
		db = setupSqlLiteDatabase()
		defer db.Close()
	}
	if databaseType == config.DATABASE_TYPE_MYSQL {
		db = setupMysqlDatabase()
		defer db.Close()
	}

	workflowRepo := repository.NewWorkflowRepository(db)
	workflowActionRepo := repository.NewWorkflowActionRepository(db)
	executorRepo := repository.NewExecutorRepository(db)
	definitionRepo := repository.NewWorkflowDefinitionRepository(db)
	userRepo := repository.NewUserRepository(db)

	wfManager := engine.NewWorkflowManager(workflowRepo, workflowActionRepo, executorRepo, definitionRepo, &WorkflowRegistry)

	dur, _ := time.ParseDuration(config.GetSystemSettingString(config.ENGINE_CHECK_DB_INTERVAL))
	go wfManager.StartEngine(dur)

	if mux == nil {
		mux = http.NewServeMux()
	}
	workflowsController := controllers.NewWorkflowsController(workflowRepo, workflowActionRepo, wfManager, userRepo)
	workflowsController.RegisterRoutes(mux)
	actionsController := controllers.NewActionsController(workflowRepo, workflowActionRepo, userRepo)
	actionsController.RegisterRoutes(mux)
	executorsController := controllers.NewExecutorsController(executorRepo, userRepo)
	executorsController.RegisterRoutes(mux)
	webController := web.NewWebController(wfManager, userRepo)
	webController.RegisterRoutes(mux)

	addr := ":" + config.GetSystemSettingString(config.ENGINE_SERVER_WEB_PORT)
	if v := os.Getenv("HTTP_ADDR"); v != "" {
		addr = v
	}
	slog.Info("Starting HTTP server", "addr", addr)
	if err := http.ListenAndServe(addr, mux); err != nil {
		slog.Error("HTTP server failed", "error", err)
		return err
	}
	return nil
}

func setupPostgresDatabase() *sql.DB {
	dbURL := config.GetSystemSettingString(config.DATABASE_URL)
	if dbURL == "" {
		panic("GFLOW_DATABASE_URL must be set when using the POSTGRES database type")
	}
	slog.Info("Using Postgres database", "url", dbURL)
	slog.Info("Running migrations")
	if err := runMigrationsFromEmbed("postgres", dbURL); err != nil {
		slog.Error("DB migration failed", "error", err)
		os.Exit(1)
	}
	slog.Info("Opening Postgres database")
	dbPostgres, err := sql.Open("postgres", dbURL)
	if err != nil {
		slog.Error("DB connection failed", "error", err)
		os.Exit(1)
	}
	return dbPostgres
}

func setupSqlLiteDatabase() *sql.DB {
	fileName := config.GetSystemSettingString(config.DATABASE_SQLLITE_FILE_NAME)
	if fileName == "" {
		panic("DATABASE_SQLLITE_FILE_NAME must be set")
	}
	dbURL := "sqlite3://" + fileName
	slog.Info("Using SQLite database", "file", fileName)
	slog.Info("Running migrations")
	if err := runMigrationsFromEmbed("sqllite3", dbURL); err != nil {
		slog.Error("DB migration failed", "error", err)
		os.Exit(1)
	}
	slog.Info("Opening SQLite database")
	dbSqlLite, err := sql.Open("sqlite3", fileName)
	if err != nil {
		log.Fatalf("Failed to open SQLite DB: %v", err)
	}
	if err := dbSqlLite.Ping(); err != nil {
		log.Fatalf("Failed to ping SQLite DB: %v", err)
	}
	return dbSqlLite
}

func setupMysqlDatabase() *sql.DB {
	dbURL := config.GetSystemSettingString(config.DATABASE_URL)
	if dbURL == "" {
		panic("GFLOW_DATABASE_URL must be set when using the MYSQL database type")
	}
	// panic if url does not contain ?parseTime=true
	if !strings.Contains(dbURL, "parseTime=true") {
		panic("GFLOW_DATABASE_URL must contain 'parseTime=true' for MySQL")
	}
	// panic if url does not  start with mysql://
	if !strings.HasPrefix(dbURL, "mysql://") {
		panic("GFLOW_DATABASE_URL must start with 'mysql://' for MySQL")
	}

	slog.Info("Using MySQL database", "url", dbURL)
	slog.Info("Running migrations")
	if err := runMigrationsFromEmbed("mysql", dbURL); err != nil {
		slog.Error("DB migration failed", "error", err)
		os.Exit(1)
	}
	slog.Info("Opening MySQL database")
	//remove mysql:// prefix from url
	dbMysql, err := sql.Open("mysql", strings.Replace(dbURL, "mysql://", "", 1))
	if err != nil {
		slog.Error("DB connection failed", "error", err)
		os.Exit(1)
	}
	return dbMysql
}

func runMigrationsFromEmbed(migrationsPath string, dbURL string) error {
	sub, err := fs.Sub(migrations.FS, migrationsPath)
	if err != nil {
		return err
	}
	source, err := iofs.New(sub, ".")
	if err != nil {
		return err
	}
	m, err := migrate.NewWithSourceInstance("iofs", source, dbURL)
	if err != nil {
		return err
	}
	if err := m.Up(); err != nil && err != migrate.ErrNoChange {
		return err
	}
	return nil
}

func SetupLogger() {
	w := os.Stderr
	_ = slog.New(tint.NewHandler(w, nil))
	slog.SetDefault(slog.New(
		tint.NewHandler(w, &tint.Options{
			Level:      slog.LevelInfo,
			TimeFormat: time.RFC3339Nano,
		}),
	))
}
