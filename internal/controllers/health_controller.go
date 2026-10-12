package controllers

import (
	"context"
	"database/sql"
	"encoding/json"
	"log/slog"
	"net/http"
	"time"
)

// HealthController exposes an unauthenticated readiness endpoint that can be
// probed by Kubernetes, load balancers or docker healthchecks.
type HealthController struct {
	DB *sql.DB
}

func NewHealthController(db *sql.DB) *HealthController {
	return &HealthController{DB: db}
}

// RegisterRoutes wires the HTTP routes for this controller.
func (c *HealthController) RegisterRoutes() {
	http.HandleFunc("GET /healthz", c.handleHealthz)
}

func (c *HealthController) handleHealthz(w http.ResponseWriter, r *http.Request) {
	ctx, cancel := context.WithTimeout(r.Context(), 2*time.Second)
	defer cancel()

	dbStatus := "ok"
	if c.DB == nil {
		dbStatus = "unavailable"
	} else if err := c.DB.PingContext(ctx); err != nil {
		slog.Warn("Health check database ping failed", "error", err)
		dbStatus = "error"
	}

	w.Header().Set("Content-Type", "application/json")
	if dbStatus != "ok" {
		w.WriteHeader(http.StatusServiceUnavailable)
		json.NewEncoder(w).Encode(map[string]string{
			"status":   "unavailable",
			"database": dbStatus,
		})
		return
	}

	w.WriteHeader(http.StatusOK)
	json.NewEncoder(w).Encode(map[string]string{
		"status":   "ok",
		"database": "ok",
	})
}
