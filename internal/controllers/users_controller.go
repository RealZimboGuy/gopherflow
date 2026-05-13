package controllers

import (
	"database/sql"
	"encoding/json"
	"log/slog"
	"net/http"
	"strconv"
	"strings"

	"github.com/RealZimboGuy/gopherflow/internal/engine"
	"github.com/RealZimboGuy/gopherflow/pkg/gopherflow/domain"
	"golang.org/x/crypto/bcrypt"
)

type UsersController struct {
	AuthController
	UserRepo engine.UserRepo
}

func NewUsersController(userRepo engine.UserRepo) *UsersController {
	return &UsersController{
		UserRepo: userRepo,
		AuthController: AuthController{
			UserRepo: userRepo,
		},
	}
}

// RegisterRoutes wires up the HTTP routes for this controller
func (c *UsersController) RegisterRoutes() {
	http.HandleFunc("GET /api/users", c.RequireAuth(c.handleGetUsers))
	http.HandleFunc("POST /api/users", c.RequireAuth(c.handleCreateUser))
	http.HandleFunc("GET /api/users/{id}", c.RequireAuth(c.handleGetUserById))
	http.HandleFunc("DELETE /api/users/{id}", c.RequireAuth(c.handleDeleteUser))
}

// handleGetUsers returns all users
func (c *UsersController) handleGetUsers(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		w.WriteHeader(http.StatusMethodNotAllowed)
		return
	}

	users, err := c.UserRepo.FindAll()
	if err != nil {
		slog.Error("Failed to get users", "error", err)
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusInternalServerError)
		json.NewEncoder(w).Encode(map[string]string{"error": "Failed to get users"})
		return
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	json.NewEncoder(w).Encode(users)
}

// createUserRequest is the JSON body accepted by POST /api/users.
// We intentionally do not accept SessionID/SessionExpiry/ApiKey from the caller -
// those are server-managed.
type createUserRequest struct {
	Username string `json:"username"`
	Password string `json:"password"`
	Enabled  *bool  `json:"enabled,omitempty"`
}

// handleCreateUser creates a new user with a bcrypt-hashed password.
func (c *UsersController) handleCreateUser(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		w.WriteHeader(http.StatusMethodNotAllowed)
		return
	}

	var req createUserRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		slog.Error("Failed to decode user", "error", err)
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusBadRequest)
		json.NewEncoder(w).Encode(map[string]string{"error": "Invalid user data"})
		return
	}

	req.Username = strings.TrimSpace(req.Username)
	if req.Username == "" || req.Password == "" {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusBadRequest)
		json.NewEncoder(w).Encode(map[string]string{"error": "username and password are required"})
		return
	}

	hashed, err := bcrypt.GenerateFromPassword([]byte(req.Password), bcrypt.DefaultCost)
	if err != nil {
		slog.Error("Failed to hash password", "error", err)
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusInternalServerError)
		json.NewEncoder(w).Encode(map[string]string{"error": "Failed to create user"})
		return
	}

	enabled := true
	if req.Enabled != nil {
		enabled = *req.Enabled
	}

	user := domain.User{
		Username: req.Username,
		Password: string(hashed),
		Enabled:  sql.NullBool{Bool: enabled, Valid: true},
	}

	id, err := c.UserRepo.Save(&user)
	if err != nil {
		slog.Error("Failed to create user", "error", err)
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusInternalServerError)
		json.NewEncoder(w).Encode(map[string]string{"error": "Failed to create user"})
		return
	}

	user.ID = id
	user.Password = ""
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusCreated)
	json.NewEncoder(w).Encode(user)
}

// handleGetUserById gets a user by their ID
func (c *UsersController) handleGetUserById(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		w.WriteHeader(http.StatusMethodNotAllowed)
		return
	}

	// Extract ID from URL
	idStr := r.PathValue("id")
	id, err := strconv.ParseInt(idStr, 10, 64)
	if err != nil {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusBadRequest)
		json.NewEncoder(w).Encode(map[string]string{"error": "Invalid user ID"})
		return
	}

	// Find user by ID
	user, err := c.UserRepo.FindById(id)
	if err != nil {
		slog.Error("Failed to get user", "error", err)
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusInternalServerError)
		json.NewEncoder(w).Encode(map[string]string{"error": "Failed to get user"})
		return
	}

	if user == nil {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusNotFound)
		json.NewEncoder(w).Encode(map[string]string{"error": "User not found"})
		return
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	json.NewEncoder(w).Encode(user)
}

// handleDeleteUser deletes a user by ID
func (c *UsersController) handleDeleteUser(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodDelete {
		w.WriteHeader(http.StatusMethodNotAllowed)
		return
	}

	// Extract ID from URL
	idStr := r.PathValue("id")
	id, err := strconv.ParseInt(idStr, 10, 64)
	if err != nil {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusBadRequest)
		json.NewEncoder(w).Encode(map[string]string{"error": "Invalid user ID"})
		return
	}

	// Delete user by ID
	err = c.UserRepo.DeleteById(id)
	if err != nil {
		slog.Error("Failed to delete user", "error", err)
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusInternalServerError)
		json.NewEncoder(w).Encode(map[string]string{"error": "Failed to delete user"})
		return
	}

	w.WriteHeader(http.StatusNoContent)
}
