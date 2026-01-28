package controllers

import (
	"context"
	"net/http"
	"time"

	"github.com/RealZimboGuy/gopherflow/internal/engine"
	"github.com/RealZimboGuy/gopherflow/pkg/gopherflow/core"
)

type AuthController struct {
	UserRepo engine.UserRepo
}

func NewBaseController(userRepo engine.UserRepo) *AuthController {
	return &AuthController{UserRepo: userRepo}
}

func (wc *AuthController) RequireAuth(next http.HandlerFunc) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path == "/login" {
			next(w, r)
			return
		}
		// 1) Try session cookie
		if c, err := r.Cookie("sessionId"); err == nil && c.Value != "" {
			u, err := wc.UserRepo.FindBySessionID(c.Value, time.Now().UTC())
			if err == nil && u != nil {
				ctx := context.WithValue(r.Context(), core.CtxKeyUsername, u.Username)
				r = r.WithContext(ctx)
				next(w, r)
				return
			}
		}
		// 2) Try API key from headers
		// Supported headers: X-API-Key: <key>
		apiKey := r.Header.Get("X-API-Key")
		if apiKey != "" {
			u, err := wc.UserRepo.FindByApiKey(apiKey)
			if err == nil && u != nil {

				// Add the username to the request context
				ctx := context.WithValue(r.Context(), core.CtxKeyUsername, u.Username)
				r = r.WithContext(ctx)
				// Proceed as authenticated
				next(w, r)
				return
			}
			http.Error(w, "Unauthorized", http.StatusUnauthorized)
			return
		}
		// Otherwise redirect to login for browser flows
		http.Redirect(w, r, "/login", http.StatusSeeOther)
	}
}
