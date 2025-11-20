package gopherflow

import (
	"context"
	"database/sql"
	"io/fs"
	"log"
	"log/slog"
	"net"
	"net/http"
	"os"
	"strings"
	"time"

	"github.com/RealZimboGuy/gopherflow/internal/config"
	"github.com/RealZimboGuy/gopherflow/internal/controllers"
	"github.com/RealZimboGuy/gopherflow/internal/engine"
	"github.com/RealZimboGuy/gopherflow/internal/migrations"
	"github.com/RealZimboGuy/gopherflow/internal/repository"
	"github.com/RealZimboGuy/gopherflow/internal/web"
	"github.com/RealZimboGuy/gopherflow/pkg/gopherflow/core"
	"github.com/lmittmann/tint"

	_ "github.com/go-sql-driver/mysql"
	migrate "github.com/golang-migrate/migrate/v4"
	_ "github.com/golang-migrate/migrate/v4/database/mysql"
	_ "github.com/golang-migrate/migrate/v4/database/postgres"
	_ "github.com/golang-migrate/migrate/v4/database/sqlite3"
	"github.com/golang-migrate/migrate/v4/source/iofs"
	_ "github.com/golang-migrate/migrate/v4/source/iofs"
	_ "github.com/mattn/go-sqlite3"
)

// App wires together the workflow engine, repositories, and HTTP server.
type App struct {
	DB               *sql.DB
	Manager          *engine.WorkflowManager
	WorkflowRegistry map[string]func() core.Workflow
	Repos            struct {
		Workflows   *repository.WorkflowRepository
		Actions     *repository.WorkflowActionRepository
		Executors   *repository.ExecutorRepository
		Definitions *repository.WorkflowDefinitionRepository
		Users       *repository.UserRepository
	}
}
type logHandler struct {
	slog.Handler
	Clock core.Clock
}

func Setup(registry map[string]func() core.Workflow) *App {
	return SetupWithClock(registry, core.NewRealClock())
}

// SetupWithClock sets up the database, repositories, workflow manager, and HTTP mux.
func SetupWithClock(registry map[string]func() core.Workflow, clock core.Clock) *App {
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
	app.Repos.Workflows = repository.NewWorkflowRepository(db, clock)
	app.Repos.Actions = repository.NewWorkflowActionRepository(db, clock)
	app.Repos.Executors = repository.NewExecutorRepository(db, clock)
	app.Repos.Definitions = repository.NewWorkflowDefinitionRepository(db, clock)
	app.Repos.Users = repository.NewUserRepository(db, clock)

	// Workflows manager
	app.Manager = engine.NewWorkflowManager(
		app.Repos.Workflows,
		app.Repos.Actions,
		app.Repos.Executors,
		app.Repos.Definitions,
		&registry,
		clock,
	)

	controllers.NewWorkflowsController(app.Repos.Workflows, app.Repos.Actions, app.Manager, app.Repos.Users).RegisterRoutes()
	controllers.NewActionsController(app.Repos.Workflows, app.Repos.Actions, app.Repos.Users).RegisterRoutes()
	controllers.NewExecutorsController(app.Repos.Executors, app.Repos.Users).RegisterRoutes()
	web.NewWebController(app.Manager, app.Repos.Users).RegisterRoutes()

	return app
}

// Run starts the workflow engine and HTTP server.
func (a *App) Run(ctx context.Context) error {
	// start engine in background
	dur, _ := time.ParseDuration(config.GetSystemSettingString(config.ENGINE_CHECK_DB_INTERVAL))
	go a.Manager.StartEngine(ctx, dur)

	addr := ":" + config.GetSystemSettingString(config.ENGINE_SERVER_WEB_PORT)
	if v := os.Getenv("HTTP_ADDR"); v != "" {
		addr = v
	}
	slog.Info("Starting HTTP server", "addr", addr)

	server := &http.Server{
		Addr:        addr,
		BaseContext: func(net.Listener) context.Context { return ctx },
	}

	// Graceful shutdown
	go func() {
		<-ctx.Done()
		shutdownCtx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		defer cancel()
		if err := server.Shutdown(shutdownCtx); err != nil {
			slog.Error("Server shutdown error", "error", err)
		}
		a.Shutdown()
	}()

	err := server.ListenAndServe()
	if err != nil && err != http.ErrServerClosed {
		return err
	}
	return nil
}

func (a *App) Shutdown() {

	//remove any global setups to clean up resources
	//WorkflowRegistry = make(map[string]func() core.Workflow)
	//a.DB.Close()
	slog.Info("DB connection closed")
	//remove any global registered routes
	http.DefaultServeMux = new(http.ServeMux)
	slog.Info("Shutdown complete")
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
	if fileName == "memory" {
		// Use shared in-memory database, primarily for testing
		fileName = "file::memory:?cache=shared"
		slog.Warn("Using in-memory SQLite database")
	}
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

func SetupLoggerWithClock(logLevel slog.Leveler, clock core.Clock) {
	w := os.Stderr
	baseHandler := tint.NewHandler(w, &tint.Options{
		Level:      logLevel,
		TimeFormat: "",
	})
	// set default logger to the tint handler first
	//slog.SetDefaultault(slog.New(baseHandler))
	// wrap the base handler with our custom logHandler for extra fields
	logger := slog.New(&logHandler{Handler: baseHandler, Clock: clock})
	slog.SetDefault(logger)
}
func SetupLogger(logLevel slog.Leveler) {
	SetupLoggerWithClock(logLevel, core.NewRealClock())
}

func (h *logHandler) Handle(ctx context.Context, r slog.Record) error {
	// Map slog level to Cloud severity, useful for google cloud run
	// Clone so we don't mutate the original record (handlers may share it).
	r2 := r.Clone()

	// Replace the internal timestamp with your accelerated clock.
	r2.Time = h.Clock.Now()

	var sev string
	switch {
	case r2.Level >= slog.LevelError:
		sev = "ERROR"
	case r2.Level >= slog.LevelWarn:
		sev = "WARNING"
	case r2.Level >= slog.LevelInfo:
		sev = "INFO"
	case r2.Level >= slog.LevelDebug:
		sev = "DEBUG"
	default:
		sev = "DEFAULT"
	}

	// Add Cloud Logging–compatible field
	r2.AddAttrs(slog.String("severity", sev))

	// Try to extract request ID from context
	if reqID := ctx.Value(core.CtxKeyExecutorId); reqID != nil {
		if s, ok := reqID.(string); ok && s != "" {
			r2.AddAttrs(slog.String(string(core.CtxKeyExecutorId), s))
		}
	}
	if reqID := ctx.Value(core.CtxKeyUsername); reqID != nil {
		if s, ok := reqID.(string); ok && s != "" {
			r2.AddAttrs(slog.String(string(core.CtxKeyUsername), s))
		}
	}

	// Call the wrapped handler
	return h.Handler.Handle(ctx, r2)
}
