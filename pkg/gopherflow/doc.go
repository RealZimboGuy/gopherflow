// Package gopherflow is a durable workflow engine with a built-in web console.
//
// Workflows are ordinary Go types whose states are methods. Every state
// transition is persisted, so workflows survive restarts and resume where they
// stopped. The engine, the REST API and the console all run inside your own
// binary; the only infrastructure required is a Postgres, MySQL or SQLite
// database.
//
// A minimal application registers its workflow types and runs the app:
//
//	registry := map[string]func() core.Workflow{
//		"GetIpWorkflow": func() core.Workflow { return &workflows.GetIpWorkflow{} },
//	}
//	app := gopherflow.Setup(registry)
//	if err := app.Run(ctx); err != nil {
//		slog.Error("Engine exited with error", "error", err)
//	}
//
// The database is selected with the GFLOW_DATABASE_TYPE environment variable
// (POSTGRES, MYSQL or SQLLITE); schema migrations run automatically on start.
package gopherflow
