package handlers

import (
	"encoding/json"
	"net/http"

	"go-monitoring/auth"
)

// UpdateSelfPassword permet à un utilisateur de changer son propre mot de passe
func UpdateSelfPassword(cm *ConfigManager, am *auth.AuthManager) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		username := am.GetUsername(r)
		if username == "" {
			http.Error(w, "Non authentifié", http.StatusUnauthorized)
			return
		}

		var req struct {
			CurrentPassword string `json:"current_password"`
			NewPassword     string `json:"new_password"`
		}
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			http.Error(w, "JSON invalide", http.StatusBadRequest)
			return
		}

		// Verify current password first (Security best practice)
		user, err := cm.userManager.Authenticate(username, req.CurrentPassword)
		if err != nil {
			http.Error(w, "Mot de passe actuel incorrect", http.StatusUnauthorized)
			return
		}
		_ = user // variable used for check

		if err := cm.userManager.UpdatePassword(username, req.NewPassword); err != nil {
			http.Error(w, err.Error(), http.StatusBadRequest)
			return
		}

		w.WriteHeader(http.StatusOK)
	}
}
