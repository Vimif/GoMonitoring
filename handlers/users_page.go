package handlers

import (
	"html/template"
	"net/http"

	"go-monitoring/auth"
	"go-monitoring/config"
	"go-monitoring/middleware"
)

// UsersPage gère la page de gestion des utilisateurs
func UsersPage(cfg *config.Config, am *auth.AuthManager) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		// Vérification explicite du rôle admin (bien que le middleware le fasse aussi si configuré)
		role := am.GetUserRole(r)
		if role != "admin" {
			http.Redirect(w, r, "/", http.StatusSeeOther)
			return
		}

		// Prevent Caching
		w.Header().Set("Cache-Control", "no-cache, no-store, must-revalidate")
		w.Header().Set("Pragma", "no-cache")
		w.Header().Set("Expires", "0")

		// Charger les templates
		tmpl, err := template.New("base.html").Funcs(templateFuncs).ParseFiles(
			"templates/layout/base.html",
			"templates/users.html",
		)
		if err != nil {
			http.Error(w, "Erreur chargement template: "+err.Error(), http.StatusInternalServerError)
			return
		}

		username := am.GetUsername(r)

		data := struct {
			Title     string
			Status    string
			Role      string
			Username  string
			CSRFToken string
		}{
			Title:     "Gestion des Utilisateurs",
			Status:    "OK",
			Role:      role,
			Username:  username,
			CSRFToken: middleware.GetCSRFToken(r),
		}

		if err := tmpl.ExecuteTemplate(w, "base", data); err != nil {
			http.Error(w, "Erreur rendu template: "+err.Error(), http.StatusInternalServerError)
		}
	}
}
