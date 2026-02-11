package collectors

import (
	"fmt"
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"time"

	"github.com/shirou/gopsutil/v3/cpu"
	"github.com/shirou/gopsutil/v3/disk"
	"github.com/shirou/gopsutil/v3/host"
	"github.com/shirou/gopsutil/v3/mem"
	"github.com/shirou/gopsutil/v3/net"

	"go-monitoring/models"
)

// IsLocalHost vérifie si l'hôte est local
func IsLocalHost(host string) bool {
	return host == "localhost" || host == "127.0.0.1" || host == "::1"
}

// CollectLocalSystemInfo collecte les informations système locales
func CollectLocalSystemInfo() (models.SystemInfo, error) {
	var info models.SystemInfo

	hostInfo, err := host.Info()
	if err != nil {
		return info, err
	}

	info.Hostname = hostInfo.Hostname
	info.OS = fmt.Sprintf("%s %s", hostInfo.Platform, hostInfo.PlatformVersion)
	info.Kernel = hostInfo.KernelVersion
	info.Architecture = hostInfo.KernelArch

	// Calculer l'uptime
	uptime := time.Duration(hostInfo.Uptime) * time.Second
	days := int(uptime.Hours()) / 24
	hours := int(uptime.Hours()) % 24
	minutes := int(uptime.Minutes()) % 60

	if days > 0 {
		info.Uptime = fmt.Sprintf("%d jours, %d heures, %d minutes", days, hours, minutes)
	} else if hours > 0 {
		info.Uptime = fmt.Sprintf("%d heures, %d minutes", hours, minutes)
	} else {
		info.Uptime = fmt.Sprintf("%d minutes", minutes)
	}

	info.BootTime = time.Unix(int64(hostInfo.BootTime), 0)

	return info, nil
}

// CollectLocalCPUInfo collecte les informations CPU locales
func CollectLocalCPUInfo() (models.CPUInfo, error) {
	var info models.CPUInfo

	// Infos CPU
	cpuInfo, err := cpu.Info()
	if err == nil && len(cpuInfo) > 0 {
		info.Model = cpuInfo[0].ModelName
		info.MHz = cpuInfo[0].Mhz
	}

	// Nombre de cores physiques
	cores, err := cpu.Counts(false)
	if err == nil {
		info.Cores = cores
	}

	// Nombre de threads (cores logiques)
	threads, err := cpu.Counts(true)
	if err == nil {
		info.Threads = threads
	}

	// Usage CPU (moyenne sur 100ms pour être rapide)
	percent, err := cpu.Percent(100*time.Millisecond, false)
	if err == nil && len(percent) > 0 {
		info.UsagePercent = percent[0]
	}

	return info, nil
}

// CollectLocalMemoryInfo collecte les informations mémoire locales
func CollectLocalMemoryInfo() (models.MemoryInfo, error) {
	var info models.MemoryInfo

	memInfo, err := mem.VirtualMemory()
	if err != nil {
		return info, err
	}

	info.Total = memInfo.Total
	info.Used = memInfo.Used
	info.Free = memInfo.Free
	info.Available = memInfo.Available
	info.UsedPercent = memInfo.UsedPercent

	return info, nil
}

// CollectLocalDiskInfo collecte les informations des disques locaux
func CollectLocalDiskInfo() ([]models.DiskInfo, error) {
	var disks []models.DiskInfo

	partitions, err := disk.Partitions(false)
	if err != nil {
		return nil, err
	}

	for _, partition := range partitions {
		// Ignorer certains types de systèmes de fichiers virtuels
		if isVirtualFS(partition.Fstype) {
			continue
		}

		usage, err := disk.Usage(partition.Mountpoint)
		if err != nil {
			continue
		}

		diskInfo := models.DiskInfo{
			Device:      partition.Device,
			MountPoint:  partition.Mountpoint,
			FSType:      partition.Fstype,
			Total:       usage.Total,
			Used:        usage.Used,
			Free:        usage.Free,
			UsedPercent: usage.UsedPercent,
			DriveType:   detectLocalDriveType(partition.Device),
		}

		disks = append(disks, diskInfo)
	}

	return disks, nil
}

// detectLocalDriveType détermine si le disque est SSD ou HDD (simplifié pour local)
func detectLocalDriveType(device string) string {
	// Sur Windows, on ne peut pas facilement déterminer le type
	if runtime.GOOS == "windows" {
		return "Unknown"
	}

	// Sur Linux, vérifier via /sys
	baseName := filepath.Base(device)
	diskName := strings.TrimRight(baseName, "0123456789")

	rotationalPath := fmt.Sprintf("/sys/block/%s/queue/rotational", diskName)
	data, err := os.ReadFile(rotationalPath)
	if err == nil {
		val := strings.TrimSpace(string(data))
		if val == "0" {
			return "SSD"
		} else if val == "1" {
			return "HDD"
		}
	}

	return "Unknown"
}

// BrowseLocalDirectory liste le contenu d'un répertoire local
func BrowseLocalDirectory(path string) (*models.DirectoryListing, error) {
	// Valider le chemin pour éviter les attaques path traversal
	cleanPath := filepath.Clean(path)
	if strings.Contains(cleanPath, "..") {
		return nil, fmt.Errorf("chemin invalide")
	}

	// Fix Windows Drive Letter issue (C: -> C:\)
	if runtime.GOOS == "windows" && len(cleanPath) == 2 && strings.HasSuffix(cleanPath, ":") {
		cleanPath += string(os.PathSeparator)
	}

	listing := &models.DirectoryListing{
		Path:   cleanPath,
		Parent: filepath.Dir(cleanPath),
	}

	entries, err := os.ReadDir(cleanPath)
	if err != nil {
		return nil, fmt.Errorf("erreur lecture répertoire: %w", err)
	}

	for _, entry := range entries {
		info, err := entry.Info()
		if err != nil {
			continue
		}

		dirEntry := models.DirectoryEntry{
			Name:        entry.Name(),
			Path:        filepath.Join(cleanPath, entry.Name()),
			IsDir:       entry.IsDir(),
			Size:        info.Size(),
			ModTime:     info.ModTime(),
			Permissions: info.Mode().String(),
			Owner:       "-",
			Group:       "-",
		}

		listing.Entries = append(listing.Entries, dirEntry)
	}

	return listing, nil
}

// GetLocalDiskDetails retourne les détails d'un disque local spécifique
func GetLocalDiskDetails(mountPoint string) (*models.DiskInfo, []models.Partition, error) {
	// Obtenir les infos de tous les disques
	disks, err := CollectLocalDiskInfo()
	if err != nil {
		return nil, nil, err
	}

	var targetDisk *models.DiskInfo
	for i, d := range disks {
		if d.MountPoint == mountPoint {
			targetDisk = &disks[i]
			break
		}
	}

	if targetDisk == nil {
		return nil, nil, fmt.Errorf("disque non trouvé: %s", mountPoint)
	}

	// Pour Windows et les systèmes locaux, on ne peut pas facilement obtenir les partitions
	// comme sur Linux, donc on retourne une partition unique basée sur le disque
	partition := models.Partition{
		Name:       targetDisk.Device,
		Size:       targetDisk.Total,
		Type:       targetDisk.FSType,
		MountPoint: targetDisk.MountPoint,
	}

	return targetDisk, []models.Partition{partition}, nil
}

// CollectLocalNetworkStats collecte les statistiques réseau locales
func CollectLocalNetworkStats() (models.NetworkStats, error) {
	var stats models.NetworkStats
	counters, err := net.IOCounters(false) // false = total
	if err != nil {
		return stats, err
	}
	if len(counters) > 0 {
		stats.RxBytes = counters[0].BytesRecv
		stats.TxBytes = counters[0].BytesSent
	}
	return stats, nil
}

// CollectLocalDiskIOStats collecte les statistiques d'E/S disque locales
func CollectLocalDiskIOStats() (models.DiskStats, error) {
	var stats models.DiskStats
	counters, err := disk.IOCounters()
	if err != nil {
		return stats, err
	}

	for _, c := range counters {
		stats.ReadBytes += c.ReadBytes
		stats.WriteBytes += c.WriteBytes
	}
	return stats, nil
}
