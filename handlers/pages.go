package handlers

import (
	"go-monitoring/auth"
	"go-monitoring/config"
	"go-monitoring/middleware"
	"html/template"
	"net/http"
)

// RenderPage renders a static page within the base layout
func RenderPage(pageName string, config *config.Config, role string) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		tmpl, err := template.New("base.html").Funcs(dashboardFuncs).ParseFiles(
			"templates/layout/base.html",
			"templates/"+pageName+".html",
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
		}{
			Title:     pageName,
			Status:    "OK",
			Role:      role,
			CSRFToken: middleware.GetCSRFToken(r),
		}

		if err := tmpl.ExecuteTemplate(w, "base", data); err != nil {
			http.Error(w, "Erreur rendu template: "+err.Error(), http.StatusInternalServerError)
		}
	}
}

// ConfigManager Wrapper
func RenderPageWithCM(cm *ConfigManager, am *auth.AuthManager, pageName string) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		cfg, _, _ := cm.GetConfigPoolAndCache()
		role := ""
		if am != nil {
			role = am.GetUserRole(r)
		}
		RenderPage(pageName, cfg, role)(w, r)
	}
}
