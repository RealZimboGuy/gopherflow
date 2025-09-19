package controllers

import (
	"gopherflow/internal/repository"
	"net/http"
	"time"
)

type AuthController struct {
	UserRepo *repository.UserRepository
}

func NewBaseController(userRepo *repository.UserRepository) *AuthController {
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
