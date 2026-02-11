package collectors

import (
	"context"
	"fmt"
	"log"
	"sync"
	"time"

	"go-monitoring/models"
	"go-monitoring/ssh"
)

// ConcurrentCollector permet de collecter les métriques de plusieurs machines en parallèle
type ConcurrentCollector struct {
	pool          ssh.ClientPool
	maxConcurrent int
	semaphore     chan struct{}
}

// NewConcurrentCollector crée un nouveau collecteur concurrent
func NewConcurrentCollector(pool ssh.ClientPool, maxConcurrent int) *ConcurrentCollector {
	if maxConcurrent <= 0 {
		maxConcurrent = 5 // Valeur par défaut
	}

	return &ConcurrentCollector{
		pool:          pool,
		maxConcurrent: maxConcurrent,
		semaphore:     make(chan struct{}, maxConcurrent),
	}
}

// CollectResult représente le résultat de collection pour une machine
type CollectResult struct {
	MachineID string
	Machine   *models.Machine
	Error     error
	Duration  time.Duration
}

// CollectAll collecte les métriques de toutes les machines en parallèle
func (c *ConcurrentCollector) CollectAll(ctx context.Context, machines []models.Machine) []CollectResult {
	results := make([]CollectResult, len(machines))
	var wg sync.WaitGroup

	for i, machine := range machines {
		wg.Add(1)

		// Capturer l'index et la machine pour la goroutine
		idx := i
		m := machine

		go func() {
			defer wg.Done()

			// Acquérir le semaphore (limiter la concurrence)
			select {
			case c.semaphore <- struct{}{}:
				defer func() { <-c.semaphore }()
			case <-ctx.Done():
				results[idx] = CollectResult{
					MachineID: m.ID,
					Error:     ctx.Err(),
				}
				return
			}

			// Collecter avec timeout
			result := c.collectOne(ctx, &m)
			results[idx] = result
		}()
	}

	// Attendre que toutes les collectes soient terminées
	wg.Wait()

	return results
}

// collectOne collecte les métriques d'une seule machine
func (c *ConcurrentCollector) collectOne(ctx context.Context, machine *models.Machine) CollectResult {
	startTime := time.Now()

	result := CollectResult{
		MachineID: machine.ID,
		Machine:   machine,
	}

	// Récupérer le client SSH
	client, err := c.pool.GetClient(machine.ID)
	if err != nil {
		result.Error = fmt.Errorf("failed to get SSH client: %w", err)
		result.Duration = time.Since(startTime)
		return result
	}

	// Créer un contexte avec timeout pour cette machine
	collectCtx, cancel := context.WithTimeout(ctx, 30*time.Second)
	defer cancel()

	// Channel pour le résultat de la collection
	done := make(chan error, 1)

	go func() {
		done <- c.collectMetrics(client, machine)
	}()

	// Attendre la fin ou le timeout
	select {
	case err := <-done:
		result.Error = err
		result.Duration = time.Since(startTime)
	case <-collectCtx.Done():
		result.Error = fmt.Errorf("collection timeout: %w", collectCtx.Err())
		result.Duration = time.Since(startTime)
	}

	return result
}

// collectMetrics effectue la collection des métriques
func (c *ConcurrentCollector) collectMetrics(client *ssh.Client, machine *models.Machine) error {
	// Tenter de se connecter
	if err := client.Connect(); err != nil {
		machine.Status = "offline"
		return nil // Pas une erreur fatale, juste offline
	}

	machine.Status = "online"
	machine.LastCheck = time.Now()

	// Collecter les métriques système
	if system, _, err := CollectSystemInfo(client, machine.OSType); err == nil {
		machine.System = system
	} else {
		log.Printf("Warning: Failed to collect system info for %s: %v", machine.ID, err)
	}

	// Collecter CPU
	if cpu, err := CollectCPUInfo(client, machine.OSType); err == nil {
		machine.CPU = cpu
	} else {
		log.Printf("Warning: Failed to collect CPU info for %s: %v", machine.ID, err)
	}

	// Collecter mémoire
	if memory, err := CollectMemoryInfo(client, machine.OSType); err == nil {
		machine.Memory = memory
	} else {
		log.Printf("Warning: Failed to collect memory info for %s: %v", machine.ID, err)
	}

	// Collecter disques
	if disks, err := CollectDiskInfo(client, machine.OSType); err == nil {
		machine.Disks = disks
	} else {
		log.Printf("Warning: Failed to collect disk info for %s: %v", machine.ID, err)
	}

	// Collecter statistiques disque I/O
	if diskIO, err := CollectDiskIOStats(client, machine.OSType); err == nil {
		machine.DiskIO = diskIO
	} else {
		log.Printf("Warning: Failed to collect disk I/O for %s: %v", machine.ID, err)
	}

	return nil
}

// CollectBatch collecte les métriques d'un batch de machines spécifiques
func (c *ConcurrentCollector) CollectBatch(ctx context.Context, machineIDs []string, allMachines []models.Machine) []CollectResult {
	// Filtrer les machines du batch
	var batchMachines []models.Machine
	for _, m := range allMachines {
		for _, id := range machineIDs {
			if m.ID == id {
				batchMachines = append(batchMachines, m)
				break
			}
		}
	}

	return c.CollectAll(ctx, batchMachines)
}

// GetStatistics retourne des statistiques sur les résultats de collection
func GetStatistics(results []CollectResult) CollectionStats {
	stats := CollectionStats{
		Total: len(results),
	}

	var totalDuration time.Duration

	for _, r := range results {
		totalDuration += r.Duration

		if r.Duration > stats.MaxDuration {
			stats.MaxDuration = r.Duration
		}

		if stats.MinDuration == 0 || r.Duration < stats.MinDuration {
			stats.MinDuration = r.Duration
		}

		if r.Error != nil {
			stats.Failed++
		} else if r.Machine != nil && r.Machine.Status == "online" {
			stats.Online++
		} else {
			stats.Offline++
		}
	}

	if stats.Total > 0 {
		stats.AvgDuration = totalDuration / time.Duration(stats.Total)
	}

	return stats
}

// CollectionStats contient des statistiques sur une collection
type CollectionStats struct {
	Total       int
	Online      int
	Offline     int
	Failed      int
	MinDuration time.Duration
	MaxDuration time.Duration
	AvgDuration time.Duration
}

// String retourne une représentation textuelle des statistiques
func (s CollectionStats) String() string {
	return fmt.Sprintf(
		"Total: %d, Online: %d, Offline: %d, Failed: %d, Avg: %v, Min: %v, Max: %v",
		s.Total, s.Online, s.Offline, s.Failed,
		s.AvgDuration, s.MinDuration, s.MaxDuration,
	)
}
