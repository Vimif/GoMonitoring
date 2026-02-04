package handlers

import (
	"encoding/json"
	"net/http"
	"time"

	"go-monitoring/cache"
	"go-monitoring/config"
	"go-monitoring/ssh"
)

// GetStatus retourne l'Ã©tat actuel de toutes les machines au format JSON
func GetStatus(cfg *config.Config, pool *ssh.Pool, cache *cache.MetricsCache) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		// Collecte rapide (timeout court car c'est pour l'UI temps rÃ©el)
		// On utilise le cache si disponible
		machines := CollectAllMachines(cfg, pool, cache, 2*time.Second, false)

		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(machines)
	}
}
