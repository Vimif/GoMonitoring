package handlers

import (
	"encoding/json"
	"net/http"
	"strconv"

	"go-monitoring/auth"
	"go-monitoring/collectors"
	"go-monitoring/storage"
)

// ListLogSources retourne les sources de logs disponibles
func ListLogSources(cm *ConfigManager) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		machineID := r.PathValue("id")

		cfg, pool, _ := cm.GetConfigPoolAndCache()
		machineConfig := cfg.GetMachine(machineID)
		if machineConfig == nil {
			http.Error(w, "Machine introuvable", http.StatusNotFound)
			return
		}

		client, err := pool.GetClient(machineID)
		if err != nil {
			http.Error(w, "Erreur connexion SSH: "+err.Error(), http.StatusServiceUnavailable)
			return
		}

		osType := machineConfig.OS
		if osType == "" {
			osType = "linux"
		}

		sources, err := collectors.GetAvailableLogSources(client, osType)
		if err != nil {
			http.Error(w, "Erreur rÃ©cupÃ©ration sources: "+err.Error(), http.StatusInternalServerError)
			return
		}

		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(sources)
	}
}

// GetLogContent retourne le contenu d'un log
func GetLogContent(cm *ConfigManager, db *storage.DB, am *auth.AuthManager) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		machineID := r.PathValue("id")

		// On lit sourceID depuis query param ou body
		sourceID := r.URL.Query().Get("source")
		linesStr := r.URL.Query().Get("lines")
		lines := 100
		if linesStr != "" {
			if l, err := strconv.Atoi(linesStr); err == nil {
				lines = l
			}
		}

		cfg, pool, _ := cm.GetConfigPoolAndCache()
		machineConfig := cfg.GetMachine(machineID)
		if machineConfig == nil {
			http.Error(w, "Machine introuvable", http.StatusNotFound)
			return
		}

		client, err := pool.GetClient(machineID)
		if err != nil {
			http.Error(w, "Erreur connexion SSH: "+err.Error(), http.StatusServiceUnavailable)
			return
		}

		osType := machineConfig.OS
		if osType == "" {
			osType = "linux"
		}

		// RÃ©cupÃ©rer la dÃ©finition de la source
		// Note: C'est un peu inefficace de tout relister, mais Ã§a sÃ©curise l'input
		// car on ne construit la commande qu'Ã  partir de la liste "autorisÃ©e" gÃ©nÃ©rÃ©e cÃ´tÃ© serveur.
		sources, err := collectors.GetAvailableLogSources(client, osType)
		if err != nil {
			http.Error(w, "Erreur: "+err.Error(), http.StatusInternalServerError)
			return
		}

		var selectedSource collectors.LogSource
		found := false
		for _, s := range sources {
			if s.ID == sourceID {
				selectedSource = s
				found = true
				break
			}
		}

		if !found {
			http.Error(w, "Source de log introuvable ou inaccessible", http.StatusBadRequest)
			return
		}

		// fetch content
		content, err := collectors.FetchLogContent(client, selectedSource, lines)
		if err != nil {
			http.Error(w, "Erreur lecture log: "+err.Error(), http.StatusInternalServerError)
			return
		}

		// Audit (Lecture seule, mais bon Ã  savoir)
		// db.LogAction(am.GetUsername(r), "VIEW_LOG", machineID+":"+sourceID, "Lines: "+linesStr, r.RemoteAddr)

		w.Header().Set("Content-Type", "text/plain")
		w.Write([]byte(content))
	}
}
