package handlers

import (
	"html/template"
	"net/http"
	"strings"

	"go-monitoring/auth"
	"go-monitoring/config"
	"go-monitoring/middleware"
	"go-monitoring/storage"
)

// AuditPage gÃ¨re la page d'affichage des logs d'audit
func AuditPage(cfg *config.Config, db *storage.DB, am *auth.AuthManager) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		// VÃ©rification Admin
		role := am.GetUserRole(r)
		if role != "admin" {
			http.Redirect(w, r, "/", http.StatusSeeOther)
			return
		}

		// RÃ©cupÃ©rer les logs (limit 100 pour l'instant)
		logs, err := db.GetAuditLogs(100)
		if err != nil {
			http.Error(w, "Erreur rÃ©cupÃ©ration logs: "+err.Error(), http.StatusInternalServerError)
			return
		}

		// Extraire les utilisateurs uniques pour le filtre
		userMap := make(map[string]bool)
		for _, log := range logs {
			userMap[log.User] = true
		}
		users := make([]string, 0, len(userMap))
		for user := range userMap {
			users = append(users, user)
		}

		// Charger les templates
		tmpl, err := template.New("base.html").Funcs(template.FuncMap{
			"lower": strings.ToLower,
		}).ParseFiles(
			"templates/layout/base.html",
			"templates/audit.html",
		)
		if err != nil {
			http.Error(w, "Erreur chargement template: "+err.Error(), http.StatusInternalServerError)
			return
		}

		data := struct {
			Title     string
			Status    string
			Role      string
			CSRFToken string
			Logs      []storage.AuditLog
			Users     []string
		}{
			Title:     "Journal d'Audit",
			Status:    "OK",
			Role:      role,
			CSRFToken: middleware.GetCSRFToken(r),
			Logs:      logs,
			Users:     users,
		}

		if err := tmpl.ExecuteTemplate(w, "base", data); err != nil {
			http.Error(w, "Erreur rendu template: "+err.Error(), http.StatusInternalServerError)
		}
	}
}
