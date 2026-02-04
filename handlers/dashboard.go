package handlers

import (
	"html/template"
	"log"
	"net/http"
	"sort"
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

// Fonctions de template pour le dashboard
var dashboardFuncs = template.FuncMap{
	"lower": strings.ToLower,
}

// Dashboard gÃ¨re la page d'accueil avec la liste des machines
func Dashboard(cfg *config.Config, pool *ssh.Pool, cache *cache.MetricsCache, am *auth.AuthManager) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		// Charger les templates avec les fonctions personnalisÃ©es
		tmpl, err := template.New("base.html").Funcs(dashboardFuncs).ParseFiles(
			"templates/layout/base.html",
			"templates/dashboard.html",
		)
		if err != nil {
			http.Error(w, "Erreur chargement templates: "+err.Error(), http.StatusInternalServerError)
			return
		}

		// Collecter les infos des machines en parallÃ¨le avec timeout et cache
		machines := CollectAllMachines(cfg, pool, cache, 1*time.Second, false)

		log.Printf("Dashboard: %d machines chargÃ©es", len(machines))

		// Grouper les machines
		groupsMap := make(map[string][]models.Machine)
		var groupNames []string

		for _, m := range machines {
			g := m.Group
			if g == "" {
				g = "Autres"
			}
			if _, exists := groupsMap[g]; !exists {
				groupsMap[g] = []models.Machine{}
				groupNames = append(groupNames, g)
			}
			groupsMap[g] = append(groupsMap[g], m)
		}

		sort.Strings(groupNames)

		var groups []models.MachineGroup
		for _, name := range groupNames {
			groups = append(groups, models.MachineGroup{
				Name:     name,
				Machines: groupsMap[name],
			})
		}

		role := ""
		if am != nil {
			role = am.GetUserRole(r)
		}

		// PrÃ©parer les donnÃ©es
		data := models.DashboardData{
			Title:           "Monitoring Infrastructure",
			Status:          getGlobalStatus(machines),
			Time:            time.Now().Format("15:04:05"),
			Groups:          groups,
			TotalMachines:   len(machines),
			RefreshInterval: cfg.Settings.RefreshInterval,
			Role:            role,
			CSRFToken:       middleware.GetCSRFToken(r),
		}

		// Rendre le template
		if err := tmpl.ExecuteTemplate(w, "base", data); err != nil {
			http.Error(w, "Erreur rendu template: "+err.Error(), http.StatusInternalServerError)
		}
	}
}

// ConfigManager Wrapper
func DashboardWithCM(cm *ConfigManager, am *auth.AuthManager) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		cfg, pool, cache := cm.GetConfigPoolAndCache()
		Dashboard(cfg, pool, cache, am)(w, r)
	}
}

// CollectAllMachines collecte les infos de toutes les machines en parallÃ¨le
func CollectAllMachines(cfg *config.Config, pool *ssh.Pool, cache *cache.MetricsCache, timeout time.Duration, forceRefresh bool) []models.Machine {
	machines := make([]models.Machine, len(cfg.Machines))
	var wg sync.WaitGroup

	for i, machineConfig := range cfg.Machines {
		wg.Add(1)
		go func(index int, mc config.MachineConfig) {
			defer wg.Done()

			// CrÃ©er la machine de base
			machine := models.Machine{
				ID:        mc.ID,
				Name:      mc.Name,
				Host:      mc.Host,
				Port:      mc.Port,
				User:      mc.User,
				KeyPath:   mc.KeyPath, // Copier le chemin de la clÃ©
				Group:     mc.Group,
				Status:    "checking",
				LastCheck: time.Now(),
			}

			// Canal pour recevoir le rÃ©sultat
			resultChan := make(chan models.Machine, 1)

			go func() {
				// 1. VÃ©rifier le cache si pas de forceRefresh
				if !forceRefresh {
					if cached, found := cache.Get(mc.ID); found {
						// VÃ©rifier la conformitÃ© mÃªme pour les donnÃ©es en cache
						// collectors.CheckCompliance(&cached, cfg.Settings.Thresholds)
						resultChan <- cached
						return
					}
				}

				// RÃ©cupÃ©rer l'Ã©tat prÃ©cÃ©dent pour le calcul des dÃ©bits
				prev, hasPrev := cache.GetLastKnown(mc.ID)

				// VÃ©rifier si c'est une machine locale
				if collectors.IsLocalHost(mc.Host) {
					// log.Printf("Dashboard: Machine locale dÃ©tectÃ©e: %s", mc.ID)
					res := collectLocalMachineInfo(machine)
					if hasPrev {
						CalculateRates(&res, &prev)
					}
					// VÃ©rifier la conformitÃ©
					// collectors.CheckCompliance(&res, cfg.Settings.Thresholds)
					cache.Set(res) // Mettre en cache
					resultChan <- res
					return
				}

				// Machine distante via SSH
				client, err := pool.GetClient(mc.ID)
				if err != nil {
					log.Printf("Dashboard: Erreur client SSH pour %s: %v", mc.ID, err)
					machine.Status = "error"
					// Maintien des infos prÃ©cÃ©dentes si disponibles
					if hasPrev {
						machine.OSType = prev.OSType
						machine.System = prev.System
						machine.Disks = prev.Disks
					}
					cache.Set(machine) // Mettre en cache l'erreur
					resultChan <- machine
					return
				}

				// Tester la connexion et collecter les infos de base
				// Utiliser l'OS configurÃ© ou auto-dÃ©tecter
				sysInfo, detectedOS, err := collectors.CollectSystemInfo(client, mc.OS)
				if err != nil {
					log.Printf("Dashboard: Machine %s offline: %v", mc.ID, err)
					machine.Status = "offline"
					// Maintien des infos prÃ©cÃ©dentes si disponibles
					if hasPrev {
						machine.OSType = prev.OSType
						machine.System = prev.System
					}
					cache.Set(machine) // Mettre en cache le statut offline
					resultChan <- machine
					return
				}

				machine.Status = "online"
				machine.System = sysInfo

				// Normalisation de l'OS pour l'affichage (linux/windows)
				machine.OSType = detectedOS
				osLower := strings.ToLower(machine.System.OS)
				if machine.OSType != "windows" {
					if strings.Contains(osLower, "linux") ||
						strings.Contains(osLower, "ubuntu") ||
						strings.Contains(osLower, "debian") ||
						strings.Contains(osLower, "centos") ||
						strings.Contains(osLower, "red hat") ||
						strings.Contains(osLower, "fedora") ||
						strings.Contains(osLower, "alpine") {
						machine.OSType = "linux"
					}
				}
				log.Printf("DEBUG OS MAPPING: Host=%s OS='%s' Detected='%s' Final='%s'", machine.Name, machine.System.OS, detectedOS, machine.OSType)

				// Collecter les mÃ©triques en parallÃ¨le pour de meilleures performances
				var collectWg sync.WaitGroup
				var mu sync.Mutex

				collectWg.Add(5)

				// CPU
				go func() {
					defer collectWg.Done()
					if cpuInfo, err := collectors.CollectCPUInfo(client, detectedOS); err == nil {
						mu.Lock()
						machine.CPU = cpuInfo
						mu.Unlock()
					}
				}()

				// Memory
				go func() {
					defer collectWg.Done()
					if memInfo, err := collectors.CollectMemoryInfo(client, detectedOS); err == nil {
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
					if disks, err := collectors.CollectDiskInfo(client, detectedOS); err == nil {
						mu.Lock()
						machine.Disks = disks
						mu.Unlock()
					}
				}()

				collectWg.Wait()

				// Calcul des taux
				if hasPrev {
					CalculateRates(&machine, &prev)
				}

				// VÃ©rifier la conformitÃ©
				// collectors.CheckCompliance(&machine, cfg.Settings.Thresholds)

				cache.Set(machine) // Mettre en cache
				resultChan <- machine
			}()

			// Attendre le rÃ©sultat avec timeout
			select {
			case result := <-resultChan:
				machines[index] = result
			case <-time.After(timeout):
				log.Printf("Dashboard: Timeout pour la machine %s", mc.ID)

				// Tentative de rÃ©cupÃ©ration des infos prÃ©cÃ©dentes
				if cached, found := cache.Get(mc.ID); found {
					machine = cached
				}
				machine.Status = "timeout"
				cache.Set(machine) // Mettre en cache le timeout
				machines[index] = machine
			}
		}(i, machineConfig)
	}

	wg.Wait()
	return machines
}

// CalculateRates calcule les dÃ©bits basÃ©s sur l'Ã©tat prÃ©cÃ©dent
func CalculateRates(current, previous *models.Machine) {
	duration := current.LastCheck.Sub(previous.LastCheck).Seconds()
	if duration <= 0 {
		return
	}

	// Network
	if current.Network.RxBytes >= previous.Network.RxBytes {
		current.Network.RxRate = float64(current.Network.RxBytes-previous.Network.RxBytes) / duration
	}
	if current.Network.TxBytes >= previous.Network.TxBytes {
		current.Network.TxRate = float64(current.Network.TxBytes-previous.Network.TxBytes) / duration
	}

	// Disk IO
	if current.DiskIO.ReadBytes >= previous.DiskIO.ReadBytes {
		current.DiskIO.ReadRate = float64(current.DiskIO.ReadBytes-previous.DiskIO.ReadBytes) / duration
	}
	if current.DiskIO.WriteBytes >= previous.DiskIO.WriteBytes {
		current.DiskIO.WriteRate = float64(current.DiskIO.WriteBytes-previous.DiskIO.WriteBytes) / duration
	}
}

// collectLocalMachineInfo collecte les infos d'une machine locale
func collectLocalMachineInfo(machine models.Machine) models.Machine {
	sysInfo, err := collectors.CollectLocalSystemInfo()
	if err != nil {
		log.Printf("Dashboard: Erreur systÃ¨me local: %v", err)
		machine.Status = "error"
		return machine
	}

	machine.Status = "online"
	machine.System = sysInfo

	cpuInfo, err := collectors.CollectLocalCPUInfo()
	if err != nil {
		log.Printf("Dashboard: Erreur CPU local: %v", err)
	} else {
		machine.CPU = cpuInfo
	}

	memInfo, err := collectors.CollectLocalMemoryInfo()
	if err != nil {
		log.Printf("Dashboard: Erreur mÃ©moire locale: %v", err)
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
	if err == nil {
		machine.Disks = disks
	}

	return machine
}

// getGlobalStatus retourne le status global de l'infrastructure
func getGlobalStatus(machines []models.Machine) string {
	if len(machines) == 0 {
		return "OK"
	}

	onlineCount := 0
	for _, m := range machines {
		if m.Status == "online" {
			onlineCount++
		}
	}

	if onlineCount == len(machines) {
		return "OK"
	} else if onlineCount == 0 {
		return "CRITIQUE"
	}
	return "ATTENTION"
}
