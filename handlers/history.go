package handlers

import (
	"encoding/json"
	"net/http"
	"time"

	"go-monitoring/storage"
)

// GetMachineHistory retourne l'historique des mÃ©triques d'une machine
func GetMachineHistory(db *storage.DB) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		id := r.PathValue("id")
		if id == "" {
			http.Error(w, "ID requis", http.StatusBadRequest)
			return
		}

		// DurÃ©e par dÃ©faut : 24h
		duration := 24 * time.Hour
		durationStr := r.URL.Query().Get("duration")
		if durationStr != "" {
			if d, err := time.ParseDuration(durationStr); err == nil {
				duration = d
			}
		}

		points, err := db.GetHistory(id, duration)
		if err != nil {
			http.Error(w, "Erreur rÃ©cupÃ©ration historique: "+err.Error(), http.StatusInternalServerError)
			return
		}

		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(points)
	}
}
