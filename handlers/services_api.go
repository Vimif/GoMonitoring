package handlers

import (
	"encoding/json"
	"net/http"
	"strings"

	"go-monitoring/auth"
	"go-monitoring/collectors"
	"go-monitoring/storage"
)

// HandleServiceAction gère les actions sur les services (start, stop, restart)
func HandleServiceAction(cm *ConfigManager, db *storage.DB, am *auth.AuthManager) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		// Vérifier méthode POST
		if r.Method != http.MethodPost {
			http.Error(w, "Méthode non autorisée", http.StatusMethodNotAllowed)
			return
		}

		// Vérifier Admin
		role := am.GetUserRole(r)
		user := am.GetUsername(r)
		if role != "admin" {
			http.Error(w, "Accès refusé", http.StatusForbidden)
			return
		}

		machineID := r.PathValue("id")
		serviceName := r.PathValue("service")
		action := r.PathValue("action")

		if machineID == "" || serviceName == "" || action == "" {
			http.Error(w, "Paramètres manquants", http.StatusBadRequest)
			return
		}

		cfg, pool, _ := cm.GetConfigPoolAndCache()
		machineConfig := cfg.GetMachine(machineID)
		if machineConfig == nil {
			http.Error(w, "Machine introuvable", http.StatusNotFound)
			return
		}

		// Vérifier si le service est monitoré sur cette machine
		serviceAllowed := false
		for _, s := range machineConfig.Services {
			if s == serviceName {
				serviceAllowed = true
				break
			}
		}

		if !serviceAllowed {
			http.Error(w, "Service non géré par le monitoring", http.StatusBadRequest)
			return
		}

		// Obtenir client SSH
		client, err := pool.GetClient(machineID)
		if err != nil {
			http.Error(w, "Erreur connexion SSH: "+err.Error(), http.StatusServiceUnavailable)
			return
		}

		// Déterminer l'OS (simple check, ou utiliser le cache si on avait accès, ici on check la config)
		osType := machineConfig.OS
		if osType == "" {
			osType = "linux" // Default
		}

		// Exécuter l'action
		err = collectors.ServiceAction(client, serviceName, action, osType)

		// Logger l'action dans l'audit, succès ou échec
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
			http.Error(w, "Erreur exécution: "+err.Error(), http.StatusInternalServerError)
			return
		}

		// Réponse JSON succès
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(map[string]string{
			"status":  "success",
			"message": "Action " + action + " effectuée sur " + serviceName,
		})

		// Optionnel: rafraîchir le cache immédiatement ou attendre le prochain cycle
		// Pour l'instant on laisse le cycle de monitoring mettre à jour le statut
	}
}
