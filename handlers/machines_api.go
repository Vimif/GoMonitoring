package handlers

import (
	"encoding/json"
	"log"
	"net/http"
	"sync"

	"go-monitoring/auth"
	"go-monitoring/cache"
	"go-monitoring/config"
	"go-monitoring/ssh"
)

// ConfigManager gère la configuration avec synchronisation
type ConfigManager struct {
	cfg         *config.Config
	pool        *ssh.Pool
	cache       *cache.MetricsCache
	userManager *auth.UserManager
	path        string
	mu          sync.RWMutex
}

// NewConfigManager crée un nouveau gestionnaire de configuration
func NewConfigManager(cfg *config.Config, pool *ssh.Pool, cache *cache.MetricsCache, path string) *ConfigManager {
	return &ConfigManager{
		cfg:   cfg,
		pool:  pool,
		cache: cache,
		path:  path,
	}
}

// SetUserManager associe le gestionnaire d'utilisateurs
func (cm *ConfigManager) SetUserManager(um *auth.UserManager) {
	cm.mu.Lock()
	defer cm.mu.Unlock()
	cm.userManager = um
}

// SaveUsers synchronise et sauvegarde la configuration des utilisateurs
func (cm *ConfigManager) SaveUsers() error {
	// Les utilisateurs sont maintenant gérés en base de données SQLite.
	// On ne sauvegarde plus dans le fichier config pour éviter les conflits.
	return nil
}

// GetConfig retourne la configuration actuelle
func (cm *ConfigManager) GetConfig() *config.Config {
	cm.mu.RLock()
	defer cm.mu.RUnlock()
	return cm.cfg
}

// GetPool retourne le pool SSH
func (cm *ConfigManager) GetPool() *ssh.Pool {
	cm.mu.RLock()
	defer cm.mu.RUnlock()
	return cm.pool
}

// GetCache retourne le cache
func (cm *ConfigManager) GetCache() *cache.MetricsCache {
	cm.mu.RLock()
	defer cm.mu.RUnlock()
	return cm.cache
}

// GetConfigPoolAndCache retourne tout le nécessaire de manière atomique
func (cm *ConfigManager) GetConfigPoolAndCache() (*config.Config, *ssh.Pool, *cache.MetricsCache) {
	cm.mu.RLock()
	defer cm.mu.RUnlock()
	return cm.cfg, cm.pool, cm.cache
}

// AddMachine ajoute une machine via l'API
func AddMachine(cm *ConfigManager) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		log.Println("API: Received AddMachine request")
		var machine config.MachineConfig
		if err := json.NewDecoder(r.Body).Decode(&machine); err != nil {
			log.Printf("API Error: JSON Decode failed: %v", err)
			jsonError(w, "Données invalides: "+err.Error(), http.StatusBadRequest)
			return
		}

		// Validation
		if machine.ID == "" {
			jsonError(w, "L'ID est requis", http.StatusBadRequest)
			return
		}
		if machine.Name == "" {
			jsonError(w, "Le nom est requis", http.StatusBadRequest)
			return
		}
		if machine.Host == "" {
			jsonError(w, "L'hôte est requis", http.StatusBadRequest)
			return
		}
		if machine.User == "" {
			jsonError(w, "L'utilisateur est requis", http.StatusBadRequest)
			return
		}
		if machine.KeyPath == "" && machine.Password == "" {
			jsonError(w, "Une clé SSH ou un mot de passe est requis", http.StatusBadRequest)
			return
		}

		cm.mu.Lock()
		defer cm.mu.Unlock()

		// Ajouter la machine
		if err := cm.cfg.AddMachine(machine); err != nil {
			jsonError(w, err.Error(), http.StatusConflict)
			return
		}

		// Sauvegarder la configuration
		if err := config.SaveConfig(cm.path, cm.cfg); err != nil {
			// Rollback
			cm.cfg.RemoveMachine(machine.ID)
			jsonError(w, "Erreur sauvegarde: "+err.Error(), http.StatusInternalServerError)
			return
		}

		// Recréer le pool SSH pour inclure la nouvelle machine
		cm.pool = ssh.NewPool(cm.cfg.Machines, cm.cfg.Settings.SSHTimeout)

		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusCreated)
		json.NewEncoder(w).Encode(map[string]interface{}{
			"success": true,
			"message": "Machine ajoutée avec succès",
			"machine": machine,
		})
	}
}

// UpdateMachine met à jour une machine via l'API
func UpdateMachine(cm *ConfigManager) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		machineID := r.PathValue("id")
		if machineID == "" {
			jsonError(w, "ID machine manquant", http.StatusBadRequest)
			return
		}

		var machine config.MachineConfig
		if err := json.NewDecoder(r.Body).Decode(&machine); err != nil {
			log.Printf("API Error: JSON Decode failed: %v", err)
			jsonError(w, "Données invalides: "+err.Error(), http.StatusBadRequest)
			return
		}

		// L'ID dans l'URL doit correspondre (ou on l'écrase pour être sûr)
		machine.ID = machineID

		// Validation minimale
		if machine.Name == "" || machine.Host == "" || machine.User == "" {
			jsonError(w, "Nom, Hôte et Utilisateur requis", http.StatusBadRequest)
			return
		}

		cm.mu.Lock()
		defer cm.mu.Unlock()

		// Mise à jour
		if err := cm.cfg.UpdateMachine(machine); err != nil {
			jsonError(w, err.Error(), http.StatusNotFound)
			return
		}

		// Sauvegarde
		if err := config.SaveConfig(cm.path, cm.cfg); err != nil {
			jsonError(w, "Erreur sauvegarde: "+err.Error(), http.StatusInternalServerError)
			return
		}

		// Refresh Pool
		cm.pool = ssh.NewPool(cm.cfg.Machines, cm.cfg.Settings.SSHTimeout)

		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(map[string]interface{}{
			"success": true,
			"message": "Machine mise à jour avec succès",
			"machine": machine,
		})
	}
}

// RemoveMachine supprime une machine via l'API
func RemoveMachine(cm *ConfigManager) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		machineID := r.PathValue("id")
		if machineID == "" {
			jsonError(w, "ID machine manquant", http.StatusBadRequest)
			return
		}

		cm.mu.Lock()
		defer cm.mu.Unlock()

		// Supprimer la machine
		if err := cm.cfg.RemoveMachine(machineID); err != nil {
			jsonError(w, err.Error(), http.StatusNotFound)
			return
		}

		// Sauvegarder la configuration
		if err := config.SaveConfig(cm.path, cm.cfg); err != nil {
			jsonError(w, "Erreur sauvegarde: "+err.Error(), http.StatusInternalServerError)
			return
		}

		// Recréer le pool SSH sans la machine supprimée
		cm.pool = ssh.NewPool(cm.cfg.Machines, cm.cfg.Settings.SSHTimeout)

		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(map[string]interface{}{
			"success": true,
			"message": "Machine supprimée avec succès",
		})
	}
}

// ListMachines retourne la liste des machines configurées
func ListMachines(cm *ConfigManager) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		cm.mu.RLock()
		defer cm.mu.RUnlock()

		// Ne pas exposer les mots de passe
		machines := make([]map[string]interface{}, len(cm.cfg.Machines))
		for i, m := range cm.cfg.Machines {
			machines[i] = map[string]interface{}{
				"id":       m.ID,
				"name":     m.Name,
				"host":     m.Host,
				"port":     m.Port,
				"user":     m.User,
				"has_key":  m.KeyPath != "",
				"has_pass": m.Password != "",
			}
		}

		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(machines)
	}
}
