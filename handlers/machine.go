package handlers

import (
	"html/template"
	"log"
	"net/http"
	"strings"
	"sync"
	"time"

	"go-monitoring/auth"
	"go-monitoring/cache"
	"go-monitoring/collectors"
	"go-monitoring/config"
	"go-monitoring/middleware"
	"go-monitoring/models"
	"go-monitoring/ssh"
)

// Fonctions de formatage pour les templates
var templateFuncs = template.FuncMap{
	"formatBytes":   formatBytes,
	"formatPercent": formatPercent,
	"formatRate":    formatRate,
	"lower":         strings.ToLower,
	"upper":         strings.ToUpper,
}

// MachineDetail gère la page de détail d'une machine
func MachineDetail(cfg *config.Config, pool *ssh.Pool, cache *cache.MetricsCache, am *auth.AuthManager) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		machineID := r.PathValue("id")
		if machineID == "" {
			http.Error(w, "ID machine manquant", http.StatusBadRequest)
			return
		}

		// Trouver la configuration de la machine
		machineConfig := cfg.GetMachine(machineID)
		if machineConfig == nil {
			http.Error(w, "Machine non trouvée", http.StatusNotFound)
			return
		}

		// Construire les infos de base de la machine
		machine := models.Machine{
			ID:        machineConfig.ID,
			Name:      machineConfig.Name,
			Host:      machineConfig.Host,
			Port:      machineConfig.Port,
			User:      machineConfig.User,
			Status:    "checking",
			LastCheck: time.Now(),
		}

		// Collecter les infos avec timeout et cache
		machine = collectMachineDetailWithTimeout(machine, machineConfig, cfg, pool, cache, 1*time.Second)

		// Charger les templates avec les fonctions personnalisées
		tmpl, err := template.New("base.html").Funcs(templateFuncs).ParseFiles(
			"templates/layout/base.html",
			"templates/machine.html",
			"templates/partials/disk_card.html",
		)
		if err != nil {
			http.Error(w, "Erreur chargement templates: "+err.Error(), http.StatusInternalServerError)
			return
		}

		role := ""
		username := ""
		if am != nil {
			role = am.GetUserRole(r)
			username = am.GetUsername(r)
		}

		// Préparer les données
		data := models.MachineDetailData{
			Machine:   machine,
			Time:      time.Now().Format("15:04:05"),
			Status:    "OK",
			Role:      role,
			Username:  username,
			CSRFToken: middleware.GetCSRFToken(r),
		}

		// Rendre le template
		if err := tmpl.ExecuteTemplate(w, "base", data); err != nil {
			http.Error(w, "Erreur rendu template: "+err.Error(), http.StatusInternalServerError)
		}
	}
}

// collectMachineDetailWithTimeout collecte les infos d'une machine avec timeout
// collectMachineDetailWithTimeout collecte les infos d'une machine avec timeout
func collectMachineDetailWithTimeout(machine models.Machine, machineConfig *config.MachineConfig, cfg *config.Config, pool *ssh.Pool, cache *cache.MetricsCache, timeout time.Duration) models.Machine {
	// 1. Vérifier le cache (on veut les disques pour cette vue)
	if cached, found := cache.Get(machine.ID); found && len(cached.Disks) > 0 {
		// Vérifier la conformité même pour les données en cache
		// collectors.CheckCompliance(&cached, cfg.Settings.Thresholds)
		return cached
	}

	resultChan := make(chan models.Machine, 1)

	go func() {
		// Vérifier si c'est une machine locale
		if collectors.IsLocalHost(machineConfig.Host) {
			log.Printf("MachineDetail: Machine locale détectée: %s", machine.ID)
			res := collectLocalMachineDetailInfo(machine)
			// Vérifier la conformité
			// collectors.CheckCompliance(&res, cfg.Settings.Thresholds)
			resultChan <- res
			return
		}

		// Machine distante via SSH
		client, err := pool.GetClient(machine.ID)
		if err != nil {
			log.Printf("Erreur obtention client SSH pour %s: %v", machine.ID, err)
			machine.Status = "error"
			cache.Set(machine) // Mettre en cache
			resultChan <- machine
			return
		}

		// Tenter de collecter les informations système
		// Utiliser l'OS configuré ou auto-détecter
		sysInfo, detectedOS, err := collectors.CollectSystemInfo(client, machineConfig.OS)
		if err != nil {
			log.Printf("Erreur collecte système pour %s: %v", machine.ID, err)
			machine.Status = "offline"
			cache.Set(machine) // Mettre en cache
			resultChan <- machine
			return
		}

		machine.Status = "online"
		machine.System = sysInfo
		machine.OSType = detectedOS

		// Collecter les métriques en parallèle pour de meilleures performances
		var collectWg sync.WaitGroup
		var mu sync.Mutex

		// Nombre de collecteurs: CPU, Memory, Network, DiskIO, Disks + Services si configurés
		numCollectors := 5
		if len(machineConfig.Services) > 0 {
			numCollectors = 6
		}
		collectWg.Add(numCollectors)

		// CPU
		go func() {
			defer collectWg.Done()
			if cpuInfo, err := collectors.CollectCPUInfo(client, detectedOS); err != nil {
				log.Printf("Erreur collecte CPU pour %s: %v", machine.ID, err)
			} else {
				mu.Lock()
				machine.CPU = cpuInfo
				mu.Unlock()
			}
		}()

		// Memory
		go func() {
			defer collectWg.Done()
			if memInfo, err := collectors.CollectMemoryInfo(client, detectedOS); err != nil {
				log.Printf("Erreur collecte mémoire pour %s: %v", machine.ID, err)
			} else {
				mu.Lock()
				machine.Memory = memInfo
				mu.Unlock()
			}
		}()

		// Network
		go func() {
			defer collectWg.Done()
			if netStats, err := collectors.CollectNetworkStats(client, detectedOS); err == nil {
				mu.Lock()
				machine.Network = netStats
				mu.Unlock()
			}
		}()

		// DiskIO
		go func() {
			defer collectWg.Done()
			if diskIO, err := collectors.CollectDiskIOStats(client, detectedOS); err == nil {
				mu.Lock()
				machine.DiskIO = diskIO
				mu.Unlock()
			}
		}()

		// Disks
		go func() {
			defer collectWg.Done()
			if disks, err := collectors.CollectDiskInfo(client, detectedOS); err != nil {
				log.Printf("Erreur collecte disques pour %s: %v", machine.ID, err)
			} else {
				mu.Lock()
				machine.Disks = disks
				mu.Unlock()
			}
		}()

		// Services
		if len(machineConfig.Services) > 0 {
			go func() {
				defer collectWg.Done()
				if services, err := collectors.CollectServices(client, machineConfig.Services, detectedOS); err != nil {
					log.Printf("Erreur collecte services pour %s: %v", machine.ID, err)
				} else {
					mu.Lock()
					machine.Services = services
					mu.Unlock()
				}
			}()
		}

		collectWg.Wait()

		// Vérifier la conformité
		// collectors.CheckCompliance(&machine, cfg.Settings.Thresholds)

		resultChan <- machine
	}()

	// Attendre le résultat avec timeout
	select {
	case result := <-resultChan:
		if result.Status == "online" {
			cache.Set(result) // Mettre à jour le cache avec les infos détaillées
		}
		return result
	case <-time.After(timeout):
		log.Printf("MachineDetail: Timeout pour la machine %s", machine.ID)
		machine.Status = "timeout"
		cache.Set(machine) // Mettre en cache
		return machine
	}
}

// MachineDetailWithCM gère la page de détail avec ConfigManager
func MachineDetailWithCM(cm *ConfigManager, am *auth.AuthManager) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		cfg, pool, cache := cm.GetConfigPoolAndCache()
		MachineDetail(cfg, pool, cache, am)(w, r)
	}
}

// collectLocalMachineDetailInfo collecte les infos détaillées d'une machine locale
func collectLocalMachineDetailInfo(machine models.Machine) models.Machine {
	sysInfo, err := collectors.CollectLocalSystemInfo()
	if err != nil {
		log.Printf("MachineDetail: Erreur système local: %v", err)
		machine.Status = "error"
		return machine
	}

	machine.Status = "online"
	machine.System = sysInfo

	cpuInfo, err := collectors.CollectLocalCPUInfo()
	if err != nil {
		log.Printf("MachineDetail: Erreur CPU local: %v", err)
	} else {
		machine.CPU = cpuInfo
	}

	memInfo, err := collectors.CollectLocalMemoryInfo()
	if err != nil {
		log.Printf("MachineDetail: Erreur mémoire locale: %v", err)
	} else {
		machine.Memory = memInfo
	}

	netStats, err := collectors.CollectLocalNetworkStats()
	if err == nil {
		machine.Network = netStats
	}

	diskIO, err := collectors.CollectLocalDiskIOStats()
	if err == nil {
		machine.DiskIO = diskIO
	}

	disks, err := collectors.CollectLocalDiskInfo()
	if err != nil {
		log.Printf("MachineDetail: Erreur disques locaux: %v", err)
	} else {
		machine.Disks = disks
	}

	return machine
}

// formatBytes formate une taille en bytes de manière lisible
func formatBytes(bytes uint64) string {
	const unit = 1024
	if bytes < unit {
		return formatInt(int64(bytes)) + " B"
	}
	div, exp := uint64(unit), 0
	for n := bytes / unit; n >= unit; n /= unit {
		div *= unit
		exp++
	}
	return formatFloat(float64(bytes)/float64(div)) + " " + string("KMGTPE"[exp]) + "iB"
}

func formatFloat(f float64) string {
	if f == float64(int64(f)) {
		return formatInt(int64(f))
	}
	return floatToString(f, 2)
}

func formatInt(i int64) string {
	s := ""
	if i < 0 {
		s = "-"
		i = -i
	}
	str := intToString(i)
	return s + str
}

func intToString(i int64) string {
	if i == 0 {
		return "0"
	}
	result := ""
	for i > 0 {
		result = string(rune('0'+i%10)) + result
		i /= 10
	}
	return result
}

func floatToString(f float64, decimals int) string {
	intPart := int64(f)
	fracPart := f - float64(intPart)
	if fracPart < 0 {
		fracPart = -fracPart
	}

	result := intToString(intPart)
	if decimals > 0 {
		result += "."
		for i := 0; i < decimals; i++ {
			fracPart *= 10
			digit := int(fracPart)
			result += string(rune('0' + digit))
			fracPart -= float64(digit)
		}
	}
	return result
}

func formatPercent(p float64) string {
	return floatToString(p, 1) + "%"
}

func formatRate(rate float64) string {
	return formatBytes(uint64(rate)) + "/s"
}
