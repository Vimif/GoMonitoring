package handlers

import (
	"encoding/json"
	"net/http"
	"strings"

	"go-monitoring/auth"
	"go-monitoring/collectors"
	"go-monitoring/storage"
)

// HandleServiceAction gÃ¨re les actions sur les services (start, stop, restart)
func HandleServiceAction(cm *ConfigManager, db *storage.DB, am *auth.AuthManager) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		// VÃ©rifier mÃ©thode POST
		if r.Method != http.MethodPost {
			http.Error(w, "MÃ©thode non autorisÃ©e", http.StatusMethodNotAllowed)
			return
		}

		// VÃ©rifier Admin
		role := am.GetUserRole(r)
		user := am.GetUsername(r)
		if role != "admin" {
			http.Error(w, "AccÃ¨s refusÃ©", http.StatusForbidden)
			return
		}

		machineID := r.PathValue("id")
		serviceName := r.PathValue("service")
		action := r.PathValue("action")

		if machineID == "" || serviceName == "" || action == "" {
			http.Error(w, "ParamÃ¨tres manquants", http.StatusBadRequest)
			return
		}

		cfg, pool, _ := cm.GetConfigPoolAndCache()
		machineConfig := cfg.GetMachine(machineID)
		if machineConfig == nil {
			http.Error(w, "Machine introuvable", http.StatusNotFound)
			return
		}

		// VÃ©rifier si le service est monitorÃ© sur cette machine
		serviceAllowed := false
		for _, s := range machineConfig.Services {
			if s == serviceName {
				serviceAllowed = true
				break
			}
		}

		if !serviceAllowed {
			http.Error(w, "Service non gÃ©rÃ© par le monitoring", http.StatusBadRequest)
			return
		}

		// Obtenir client SSH
		client, err := pool.GetClient(machineID)
		if err != nil {
			http.Error(w, "Erreur connexion SSH: "+err.Error(), http.StatusServiceUnavailable)
			return
		}

		// DÃ©terminer l'OS (simple check, ou utiliser le cache si on avait accÃ¨s, ici on check la config)
		osType := machineConfig.OS
		if osType == "" {
			osType = "linux" // Default
		}

		// ExÃ©cuter l'action
		err = collectors.ServiceAction(client, serviceName, action, osType)

		// Logger l'action dans l'audit, succÃ¨s ou Ã©chec
		status := "SUCCESS"
		details := ""
		if err != nil {
			status = "FAILED"
			details = err.Error()
		}

		// Enregistrer dans l'audit
		db.LogAction(
			user,
			strings.ToUpper(action)+"_SERVICE",
			machineID+":"+serviceName,
			status+" "+details,
			r.RemoteAddr,
		)

		if err != nil {
			http.Error(w, "Erreur exÃ©cution: "+err.Error(), http.StatusInternalServerError)
			return
		}

		// RÃ©ponse JSON succÃ¨s
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(map[string]string{
			"status":  "success",
			"message": "Action " + action + " effectuÃ©e sur " + serviceName,
		})

		// Optionnel: rafraÃ®chir le cache immÃ©diatement ou attendre le prochain cycle
		// Pour l'instant on laisse le cycle de monitoring mettre Ã  jour le statut
	}
}
