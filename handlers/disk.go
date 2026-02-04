package handlers

import (
	"encoding/json"
	"net/http"
	"strings"

	"go-monitoring/collectors"
	"go-monitoring/config"
	"go-monitoring/models"
	"go-monitoring/ssh"
)

// DiskList retourne la liste des disques d'une machine en JSON
func DiskList(cfg *config.Config, pool *ssh.Pool) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		machineID := r.PathValue("id")
		if machineID == "" {
			jsonError(w, "ID machine manquant", http.StatusBadRequest)
			return
		}

		// Trouver la configuration de la machine
		machineConfig := cfg.GetMachine(machineID)
		if machineConfig == nil {
			jsonError(w, "Machine non trouvÃ©e", http.StatusNotFound)
			return
		}

		var disks []models.DiskInfo
		var err error

		// VÃ©rifier si c'est une machine locale
		if collectors.IsLocalHost(machineConfig.Host) {
			disks, err = collectors.CollectLocalDiskInfo()
		} else {
			// Machine distante via SSH
			client, clientErr := pool.GetClient(machineID)
			if clientErr != nil {
				jsonError(w, "Erreur connexion SSH", http.StatusInternalServerError)
				return
			}
			// Utiliser l'OS configurÃ© ou laisser la dÃ©tection automatique
			disks, err = collectors.CollectDiskInfo(client, machineConfig.OS)
		}

		if err != nil {
			jsonError(w, "Erreur collecte disques: "+err.Error(), http.StatusInternalServerError)
			return
		}

		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(disks)
	}
}

// DiskDetails retourne les dÃ©tails d'un disque spÃ©cifique
func DiskDetails(cfg *config.Config, pool *ssh.Pool) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		machineID := r.PathValue("id")
		mountPoint := r.URL.Query().Get("mount")

		if machineID == "" {
			jsonError(w, "ID machine manquant", http.StatusBadRequest)
			return
		}
		if mountPoint == "" {
			jsonError(w, "Point de montage manquant", http.StatusBadRequest)
			return
		}

		// Trouver la configuration de la machine
		machineConfig := cfg.GetMachine(machineID)
		if machineConfig == nil {
			jsonError(w, "Machine non trouvÃ©e", http.StatusNotFound)
			return
		}

		var disk *models.DiskInfo
		var partitions []models.Partition
		var err error

		// VÃ©rifier si c'est une machine locale
		if collectors.IsLocalHost(machineConfig.Host) {
			disk, partitions, err = collectors.GetLocalDiskDetails(mountPoint)
		} else {
			// Machine distante via SSH
			client, clientErr := pool.GetClient(machineID)
			if clientErr != nil {
				jsonError(w, "Erreur connexion SSH", http.StatusInternalServerError)
				return
			}
			// Utiliser l'OS configurÃ© ou laisser la dÃ©tection automatique
			disk, partitions, err = collectors.GetDiskDetails(client, mountPoint, machineConfig.OS)
		}

		if err != nil {
			jsonError(w, err.Error(), http.StatusNotFound)
			return
		}

		response := struct {
			Disk       *models.DiskInfo   `json:"disk"`
			Partitions []models.Partition `json:"partitions"`
		}{
			Disk:       disk,
			Partitions: partitions,
		}

		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(response)
	}
}

// BrowseDirectory permet de naviguer dans l'arborescence des fichiers
func BrowseDirectory(cfg *config.Config, pool *ssh.Pool) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		machineID := r.PathValue("id")
		path := r.URL.Query().Get("path")

		if machineID == "" {
			jsonError(w, "ID machine manquant", http.StatusBadRequest)
			return
		}
		if path == "" {
			path = "/"
		}

		// Validation de sÃ©curitÃ©
		if !isAllowedPath(path) {
			jsonError(w, "Chemin non autorisÃ©", http.StatusForbidden)
			return
		}

		// Trouver la configuration de la machine
		machineConfig := cfg.GetMachine(machineID)
		if machineConfig == nil {
			jsonError(w, "Machine non trouvÃ©e", http.StatusNotFound)
			return
		}

		var listing *models.DirectoryListing
		var err error

		// VÃ©rifier si c'est une machine locale
		if collectors.IsLocalHost(machineConfig.Host) {
			listing, err = collectors.BrowseLocalDirectory(path)
		} else {
			// Machine distante via SSH
			client, clientErr := pool.GetClient(machineID)
			if clientErr != nil {
				jsonError(w, "Erreur connexion SSH", http.StatusInternalServerError)
				return
			}
			listing, err = collectors.BrowseDirectory(client, path)
		}

		if err != nil {
			jsonError(w, "Erreur lecture rÃ©pertoire: "+err.Error(), http.StatusInternalServerError)
			return
		}

		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(listing)
	}
}

// isAllowedPath vÃ©rifie si le chemin est autorisÃ©
func isAllowedPath(path string) bool {
	// Interdire les chemins avec ..
	if strings.Contains(path, "..") {
		return false
	}

	// Liste des chemins sensibles interdits
	forbiddenPaths := []string{
		"/etc/shadow",
		"/etc/passwd-",
		"/root/.ssh",
		"/.ssh",
		"/proc",
		"/sys",
	}

	for _, forbidden := range forbiddenPaths {
		if strings.HasPrefix(path, forbidden) {
			return false
		}
	}

	return true
}

// jsonError envoie une erreur JSON
func jsonError(w http.ResponseWriter, message string, code int) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(code)
	json.NewEncoder(w).Encode(map[string]string{
		"error": message,
	})
}

// DiskListWithCM retourne la liste des disques avec ConfigManager
func DiskListWithCM(cm *ConfigManager) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		cfg, pool, _ := cm.GetConfigPoolAndCache()
		DiskList(cfg, pool)(w, r)
	}
}

// DiskDetailsWithCM retourne les dÃ©tails d'un disque avec ConfigManager
func DiskDetailsWithCM(cm *ConfigManager) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		cfg, pool, _ := cm.GetConfigPoolAndCache()
		DiskDetails(cfg, pool)(w, r)
	}
}

// BrowseDirectoryWithCM permet de naviguer avec ConfigManager
func BrowseDirectoryWithCM(cm *ConfigManager) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		cfg, pool, _ := cm.GetConfigPoolAndCache()
		BrowseDirectory(cfg, pool)(w, r)
	}
}
