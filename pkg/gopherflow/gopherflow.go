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

// App wires together the workflow engine, repositories, and HTTP server.
type App struct {
	DB      *sql.DB
	Manager *engine.WorkflowManager
	Repos   struct {
		Workflows   *repository.WorkflowRepository
		Actions     *repository.WorkflowActionRepository
		Executors   *repository.ExecutorRepository
		Definitions *repository.WorkflowDefinitionRepository
		Users       *repository.UserRepository
	}
}

// Setup sets up the database, repositories, workflow manager, and HTTP mux.
func Setup() *App {
	databaseType := config.GetSystemSettingString(config.DATABASE_TYPE)
	if databaseType == "" || (databaseType != config.DATABASE_TYPE_POSTGRES &&
		databaseType != config.DATABASE_TYPE_MYSQL &&
		databaseType != config.DATABASE_TYPE_SQLLITE) {
		panic("GFLOW_DATABASE_TYPE must be set to one of: POSTGRES, MYSQL, SQLLITE")
	}

	var db *sql.DB
	switch databaseType {
	case config.DATABASE_TYPE_POSTGRES:
		db = setupPostgresDatabase()
	case config.DATABASE_TYPE_SQLLITE:
		db = setupSqlLiteDatabase()
	case config.DATABASE_TYPE_MYSQL:
		db = setupMysqlDatabase()
	}

	app := &App{DB: db}

	// Repositories
	app.Repos.Workflows = repository.NewWorkflowRepository(db)
	app.Repos.Actions = repository.NewWorkflowActionRepository(db)
	app.Repos.Executors = repository.NewExecutorRepository(db)
	app.Repos.Definitions = repository.NewWorkflowDefinitionRepository(db)
	app.Repos.Users = repository.NewUserRepository(db)

	// Workflows manager
	app.Manager = engine.NewWorkflowManager(
		app.Repos.Workflows,
		app.Repos.Actions,
		app.Repos.Executors,
		app.Repos.Definitions,
		&WorkflowRegistry,
	)

	controllers.NewWorkflowsController(app.Repos.Workflows, app.Repos.Actions, app.Manager, app.Repos.Users).RegisterRoutes()
	controllers.NewActionsController(app.Repos.Workflows, app.Repos.Actions, app.Repos.Users).RegisterRoutes()
	controllers.NewExecutorsController(app.Repos.Executors, app.Repos.Users).RegisterRoutes()
	web.NewWebController(app.Manager, app.Repos.Users).RegisterRoutes()

	return app
}

// Run starts the workflow engine and HTTP server.
func (a *App) Run() error {
	// start engine in background
	dur, _ := time.ParseDuration(config.GetSystemSettingString(config.ENGINE_CHECK_DB_INTERVAL))
	go a.Manager.StartEngine(dur)

	addr := ":" + config.GetSystemSettingString(config.ENGINE_SERVER_WEB_PORT)
	if v := os.Getenv("HTTP_ADDR"); v != "" {
		addr = v
	}
	slog.Info("Starting HTTP server", "addr", addr)
	return http.ListenAndServe(addr, nil)
}

func setupPostgresDatabase() *sql.DB {
	dbURL := config.GetSystemSettingString(config.DATABASE_URL)
	if dbURL == "" {
		panic("GFLOW_DATABASE_URL must be set when using POSTGRES")
	}
	slog.Info("Using Postgres database", "url", dbURL)
	slog.Info("Running migrations")
	if err := runMigrationsFromEmbed("postgres", dbURL); err != nil {
		slog.Error("DB migration failed", "error", err)
		os.Exit(1)
	}
	slog.Info("Opening Postgres database")
	db, err := sql.Open("postgres", dbURL)
	if err != nil {
		slog.Error("DB connection failed", "error", err)
		os.Exit(1)
	}
	return db
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
	db, err := sql.Open("sqlite3", fileName)
	if err != nil {
		log.Fatalf("Failed to open SQLite DB: %v", err)
	}
	if err := db.Ping(); err != nil {
		log.Fatalf("Failed to ping SQLite DB: %v", err)
	}
	return db
}

func setupMysqlDatabase() *sql.DB {
	dbURL := config.GetSystemSettingString(config.DATABASE_URL)
	if dbURL == "" {
		panic("GFLOW_DATABASE_URL must be set when using MYSQL")
	}
	if !strings.Contains(dbURL, "parseTime=true") {
		panic("GFLOW_DATABASE_URL must contain 'parseTime=true' for MySQL")
	}
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
	db, err := sql.Open("mysql", strings.TrimPrefix(dbURL, "mysql://"))
	if err != nil {
		slog.Error("DB connection failed", "error", err)
		os.Exit(1)
	}
	return db
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
