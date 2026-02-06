package service

import (
	"fmt"
	"log"
	"sync"
	"time"

	"go-monitoring/cache"
	"go-monitoring/collectors"
	"go-monitoring/internal/domain"
	"go-monitoring/pkg/interfaces"
	"go-monitoring/ssh"
)

// MonitoringService implémente la logique de monitoring
type MonitoringService struct {
	machineRepo interfaces.MachineRepository
	metricRepo  interfaces.MetricRepository
	sshPool     *ssh.Pool
	cache       *cache.MetricsCache
	stopChan    chan bool
	wg          sync.WaitGroup
}

// NewMonitoringService crée un nouveau service de monitoring
func NewMonitoringService(
	machineRepo interfaces.MachineRepository,
	metricRepo interfaces.MetricRepository,
	sshPool *ssh.Pool,
	metricsCache *cache.MetricsCache,
) *MonitoringService {
	return &MonitoringService{
		machineRepo: machineRepo,
		metricRepo:  metricRepo,
		sshPool:     sshPool,
		cache:       metricsCache,
		stopChan:    make(chan bool),
	}
}

// CollectMetrics collecte toutes les métriques pour une machine
func (s *MonitoringService) CollectMetrics(machineID string) (*domain.Machine, error) {
	// Récupérer la machine
	machine, err := s.machineRepo.GetByID(machineID)
	if err != nil {
		return nil, fmt.Errorf("machine not found: %w", err)
	}

	// Récupérer le client SSH
	client, err := s.sshPool.GetClient(machineID)
	if err != nil {
		machine.Status = "error"
		return machine, fmt.Errorf("failed to get SSH client: %w", err)
	}

	// Tenter de se connecter
	if err := client.Connect(); err != nil {
		machine.Status = "offline"
		return machine, nil
	}

	machine.Status = "online"
	machine.LastCheck = time.Now()

	// Collecter les métriques système
	if system, err := collectors.CollectSystemInfo(client, machine.OSType); err == nil {
		machine.System = domain.SystemInfo{
			Hostname:     system.Hostname,
			OS:           system.OS,
			Kernel:       system.Kernel,
			Architecture: system.Architecture,
			Uptime:       system.Uptime,
			BootTime:     system.BootTime,
		}
	}

	// Collecter CPU
	if cpu, err := collectors.CollectCPUInfo(client, machine.OSType); err == nil {
		machine.CPU = domain.CPUInfo{
			Model:        cpu.Model,
			Cores:        cpu.Cores,
			Threads:      cpu.Threads,
			MHz:          cpu.MHz,
			UsagePercent: cpu.UsagePercent,
		}
	}

	// Collecter mémoire
	if memory, err := collectors.CollectMemoryInfo(client, machine.OSType); err == nil {
		machine.Memory = domain.MemoryInfo{
			Total:       memory.Total,
			Used:        memory.Used,
			Free:        memory.Free,
			Available:   memory.Available,
			UsedPercent: memory.UsedPercent,
		}
	}

	// Collecter disques
	if disks, err := collectors.CollectDiskInfo(client, machine.OSType); err == nil {
		machine.Disks = make([]domain.DiskInfo, len(disks))
		for i, disk := range disks {
			machine.Disks[i] = domain.DiskInfo{
				Device:      disk.Device,
				MountPoint:  disk.MountPoint,
				FSType:      disk.FSType,
				Total:       disk.Total,
				Used:        disk.Used,
				Free:        disk.Free,
				UsedPercent: disk.UsedPercent,
				DriveType:   disk.DriveType,
			}
		}
	}

	// Collecter statistiques réseau et disque I/O
	if diskIO, err := collectors.CollectDiskIOStats(client, machine.OSType); err == nil {
		machine.DiskIO = domain.DiskStats{
			ReadBytes:  diskIO.ReadBytes,
			WriteBytes: diskIO.WriteBytes,
			ReadRate:   diskIO.ReadRate,
			WriteRate:  diskIO.WriteRate,
		}
	}

	// Mettre en cache
	if s.cache != nil {
		s.cache.Set(machineID, machine)
	}

	return machine, nil
}

// GetMachineStatus retourne le statut actuel d'une machine (avec cache)
func (s *MonitoringService) GetMachineStatus(machineID string) (*domain.Machine, error) {
	// Vérifier le cache d'abord
	if s.cache != nil {
		if cached, found := s.cache.Get(machineID); found {
			return cached, nil
		}
	}

	// Pas en cache, collecter
	return s.CollectMetrics(machineID)
}

// GetAllMachinesStatus retourne le statut de toutes les machines
func (s *MonitoringService) GetAllMachinesStatus() ([]domain.Machine, error) {
	machines, err := s.machineRepo.GetAll()
	if err != nil {
		return nil, err
	}

	var result []domain.Machine
	for _, machine := range machines {
		status, err := s.GetMachineStatus(machine.ID)
		if err != nil {
			log.Printf("Failed to get status for machine %s: %v", machine.ID, err)
			machine.Status = "error"
			result = append(result, machine)
		} else {
			result = append(result, *status)
		}
	}

	return result, nil
}

// GetMetricHistory retourne l'historique des métriques
func (s *MonitoringService) GetMetricHistory(machineID string, duration time.Duration) ([]domain.MetricPoint, error) {
	return s.metricRepo.GetHistory(machineID, duration)
}

// StartMonitoring démarre le monitoring continu
func (s *MonitoringService) StartMonitoring(interval time.Duration) {
	s.wg.Add(1)
	go func() {
		defer s.wg.Done()

		ticker := time.NewTicker(interval)
		defer ticker.Stop()

		for {
			select {
			case <-ticker.C:
				s.collectAndSaveAll()
			case <-s.stopChan:
				log.Println("Stopping monitoring service")
				return
			}
		}
	}()

	log.Printf("Monitoring service started with interval: %v", interval)
}

// StopMonitoring arrête le monitoring
func (s *MonitoringService) StopMonitoring() {
	close(s.stopChan)
	s.wg.Wait()
	log.Println("Monitoring service stopped")
}

// collectAndSaveAll collecte et sauvegarde les métriques de toutes les machines
func (s *MonitoringService) collectAndSaveAll() {
	machines, err := s.machineRepo.GetAll()
	if err != nil {
		log.Printf("Failed to get machines: %v", err)
		return
	}

	for _, machine := range machines {
		// Collecter les métriques
		status, err := s.CollectMetrics(machine.ID)
		if err != nil {
			log.Printf("Failed to collect metrics for %s: %v", machine.ID, err)
			continue
		}

		// Créer un point de métrique
		metric := &domain.MetricPoint{
			Timestamp:   time.Now(),
			CPU:         status.CPU.UsagePercent,
			MemoryUsed:  status.Memory.Used,
			MemoryTotal: status.Memory.Total,
			Status:      status.Status,
			NetRxRate:   status.Network.RxRate,
			NetTxRate:   status.Network.TxRate,
			DiskRead:    status.DiskIO.ReadRate,
			DiskWrite:   status.DiskIO.WriteRate,
		}

		// Sauvegarder dans la base de données
		if err := s.metricRepo.Save(machine.ID, metric); err != nil {
			log.Printf("Failed to save metrics for %s: %v", machine.ID, err)
		}
	}
}
