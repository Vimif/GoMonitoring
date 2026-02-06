package handlers

import (
	"encoding/json"
	"net/http"
	"time"

	"go-monitoring/auth"
)

// ListUsers retourne la liste des utilisateurs (admin seulement)
func ListUsers(cm *ConfigManager, am *auth.AuthManager) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		sessionCookie, _ := r.Cookie("session_token")
		if sessionCookie == nil || !am.IsAdminSession(sessionCookie.Value) {
			http.Error(w, "Accès refusé", http.StatusForbidden)
			return
		}

		users := cm.userManager.GetAllUsers()

		// Filtrer les données sensibles
		type UserResponse struct {
			Username    string `json:"username"`
			Role        string `json:"role"`
			IsActive    bool   `json:"is_active"`
			LockedUntil string `json:"locked_until,omitempty"`
			IsLocked    bool   `json:"is_locked"`
		}

		response := make([]UserResponse, len(users))
		for i, u := range users {
			isLocked := !u.LockedUntil.IsZero() && u.LockedUntil.After(time.Now())
			lockedStr := ""
			if isLocked {
				lockedStr = u.LockedUntil.String()
			}

			response[i] = UserResponse{
				Username:    u.Username,
				Role:        u.Role,
				IsActive:    u.IsActive,
				LockedUntil: lockedStr,
				IsLocked:    isLocked,
			}
		}

		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(response)
	}
}

// CreateUser crée un nouvel utilisateur
func CreateUser(cm *ConfigManager, am *auth.AuthManager) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		sessionCookie, _ := r.Cookie("session_token")
		if sessionCookie == nil || !am.IsAdminSession(sessionCookie.Value) {
			http.Error(w, "Accès refusé", http.StatusForbidden)
			return
		}

		var req struct {
			Username string `json:"username"`
			Password string `json:"password"`
			Role     string `json:"role"`
		}

		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			http.Error(w, "JSON invalide", http.StatusBadRequest)
			return
		}

		if req.Username == "" || req.Password == "" {
			http.Error(w, "Username et Password requis", http.StatusBadRequest)
			return
		}

		if req.Role != "admin" && req.Role != "user" {
			req.Role = "user" // Default
		}

		if err := cm.userManager.AddUser(req.Username, req.Password, req.Role); err != nil {
			http.Error(w, err.Error(), http.StatusConflict) // ex: user exists
			return
		}
		// No implicit save needed, DB is immediate

		w.WriteHeader(http.StatusCreated)
	}
}

// DeleteUser supprime un utilisateur
func DeleteUser(cm *ConfigManager, am *auth.AuthManager) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		sessionCookie, _ := r.Cookie("session_token")
		if sessionCookie == nil || !am.IsAdminSession(sessionCookie.Value) {
			http.Error(w, "Accès refusé", http.StatusForbidden)
			return
		}

		username := r.PathValue("username")
		if username == "" {
			http.Error(w, "Username requis", http.StatusBadRequest)
			return
		}

		if err := cm.userManager.DeleteUser(username); err != nil {
			http.Error(w, err.Error(), http.StatusBadRequest)
			return
		}

		w.WriteHeader(http.StatusOK)
	}
}

// UpdateUserPassword modifie le mot de passe
func UpdateUserPassword(cm *ConfigManager, am *auth.AuthManager) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		sessionCookie, _ := r.Cookie("session_token")
		if sessionCookie == nil || !am.IsAdminSession(sessionCookie.Value) {
			http.Error(w, "Accès refusé", http.StatusForbidden)
			return
		}

		username := r.PathValue("username")

		var req struct {
			Password string `json:"password"`
		}
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			http.Error(w, "JSON invalide", http.StatusBadRequest)
			return
		}

		if err := cm.userManager.UpdatePassword(username, req.Password); err != nil {
			http.Error(w, err.Error(), http.StatusBadRequest)
			return
		}

		w.WriteHeader(http.StatusOK)
	}
}

// UpdateUserRole modifie le rôle
func UpdateUserRole(cm *ConfigManager, am *auth.AuthManager) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		sessionCookie, _ := r.Cookie("session_token")
		if sessionCookie == nil || !am.IsAdminSession(sessionCookie.Value) {
			http.Error(w, "Accès refusé", http.StatusForbidden)
			return
		}

		username := r.PathValue("username")

		var req struct {
			Role string `json:"role"`
		}
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			http.Error(w, "JSON invalide", http.StatusBadRequest)
			return
		}

		if req.Role != "admin" && req.Role != "user" {
			http.Error(w, "Rôle invalide", http.StatusBadRequest)
			return
		}

		if err := cm.userManager.UpdateUserRole(username, req.Role); err != nil {
			http.Error(w, err.Error(), http.StatusBadRequest)
			return
		}

		w.WriteHeader(http.StatusOK)
	}
}

// ToggleUserStatus active/désactive un user
func ToggleUserStatus(cm *ConfigManager, am *auth.AuthManager) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		sessionCookie, _ := r.Cookie("session_token")
		if sessionCookie == nil || !am.IsAdminSession(sessionCookie.Value) {
			http.Error(w, "Accès refusé", http.StatusForbidden)
			return
		}

		username := r.PathValue("username")
		var req struct {
			Active bool `json:"active"`
		}
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			http.Error(w, "JSON invalide", http.StatusBadRequest)
			return
		}

		if err := cm.userManager.ToggleUserStatus(username, req.Active); err != nil {
			http.Error(w, err.Error(), http.StatusBadRequest)
			return
		}

		w.WriteHeader(http.StatusOK)
	}
}

// UnlockUser déverrouille un user
func UnlockUser(cm *ConfigManager, am *auth.AuthManager) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		sessionCookie, _ := r.Cookie("session_token")
		if sessionCookie == nil || !am.IsAdminSession(sessionCookie.Value) {
			http.Error(w, "Accès refusé", http.StatusForbidden)
			return
		}

		username := r.PathValue("username")

		if err := cm.userManager.UnlockUser(username); err != nil {
			http.Error(w, err.Error(), http.StatusBadRequest)
			return
		}

		w.WriteHeader(http.StatusOK)
	}
}
