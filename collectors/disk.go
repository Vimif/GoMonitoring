package collectors

import (
	"fmt"
	"path/filepath"
	"strconv"
	"strings"
	"time"

	"go-monitoring/models"
	"go-monitoring/pkg/security"
	"go-monitoring/ssh"
)

// CollectDiskInfo collecte les informations des disques via SSH
// osType: "linux", "windows" ou vide (défaut Linux)
func CollectDiskInfo(client *ssh.Client, osType string) ([]models.DiskInfo, error) {
	if osType == "windows" {
		return collectDiskInfoWindows(client)
	}
	return collectDiskInfoLinux(client)
}

// collectDiskInfoLinux collecte les infos disque sur Linux
func collectDiskInfoLinux(client *ssh.Client) ([]models.DiskInfo, error) {
	var disks []models.DiskInfo

	// Utiliser df pour obtenir l'espace disque
	// Format: Filesystem, Type, Size, Used, Avail, Use%, Mounted
	output, err := client.Execute("df -B1 -T | tail -n +2")
	if err != nil {
		return nil, err
	}

	lines := strings.Split(strings.TrimSpace(output), "\n")
	for _, line := range lines {
		fields := strings.Fields(line)
		if len(fields) < 7 {
			continue
		}

		// Ignorer les systèmes de fichiers virtuels
		fsType := fields[1]
		if isVirtualFS(fsType) {
			continue
		}

		disk := models.DiskInfo{
			Device:     fields[0],
			FSType:     fsType,
			MountPoint: fields[6],
		}

		disk.Total, _ = strconv.ParseUint(fields[2], 10, 64)
		disk.Used, _ = strconv.ParseUint(fields[3], 10, 64)
		disk.Free, _ = strconv.ParseUint(fields[4], 10, 64)

		// Calculer le pourcentage
		if disk.Total > 0 {
			disk.UsedPercent = float64(disk.Used) / float64(disk.Total) * 100
		}

		// Détecter le type de disque (SSD/HDD)
		disk.DriveType = detectDriveType(client, disk.Device)

		disks = append(disks, disk)
	}

	return disks, nil
}

// collectDiskInfoWindows collecte les infos disque sur Windows via PowerShell
func collectDiskInfoWindows(client *ssh.Client) ([]models.DiskInfo, error) {
	var disks []models.DiskInfo

	// DriveType: 2=Removable, 3=Fixed, 4=Network, 5=CD-ROM
	cmd := `powershell -Command "Get-CimInstance Win32_LogicalDisk | Where-Object {$_.DriveType -eq 3} | ForEach-Object { Write-Output ('{0}|{1}|{2}|{3}' -f $_.DeviceID, $_.Size, $_.FreeSpace, $_.MediaType) }"`
	output, err := client.Execute(cmd)
	if err != nil {
		return nil, err
	}

	lines := strings.Split(strings.TrimSpace(output), "\n")
	for _, line := range lines {
		line = strings.TrimSpace(line)
		if line == "" {
			continue
		}

		parts := strings.Split(line, "|")
		if len(parts) < 3 {
			continue
		}

		disk := models.DiskInfo{
			Device:     parts[0],
			MountPoint: parts[0] + "\\",
			FSType:     "NTFS",
			DriveType:  "Unknown",
		}

		disk.Total, _ = strconv.ParseUint(parts[1], 10, 64)
		disk.Free, _ = strconv.ParseUint(parts[2], 10, 64)
		disk.Used = disk.Total - disk.Free

		if disk.Total > 0 {
			disk.UsedPercent = float64(disk.Used) / float64(disk.Total) * 100
		}

		// Détecter SSD/HDD via MediaType
		if len(parts) >= 4 && parts[3] != "" {
			mediaType := strings.ToLower(parts[3])
			if strings.Contains(mediaType, "ssd") || strings.Contains(mediaType, "solid") {
				disk.DriveType = "SSD"
			} else if strings.Contains(mediaType, "hdd") || strings.Contains(mediaType, "hard") {
				disk.DriveType = "HDD"
			}
		}

		disks = append(disks, disk)
	}

	return disks, nil
}

// detectDriveType détermine si le disque est SSD ou HDD
func detectDriveType(client *ssh.Client, device string) string {
	// Extraire le nom du disque sans partition (ex: /dev/sda1 -> sda)
	baseName := filepath.Base(device)
	// Enlever les chiffres à la fin pour obtenir le disque de base
	diskName := strings.TrimRight(baseName, "0123456789")

	// Vérifier via /sys/block/XXX/queue/rotational
	// 0 = SSD, 1 = HDD
	cmd := fmt.Sprintf("cat /sys/block/%s/queue/rotational 2>/dev/null", diskName)
	output, err := client.Execute(cmd)
	if err == nil {
		val := strings.TrimSpace(output)
		if val == "0" {
			return "SSD"
		} else if val == "1" {
			return "HDD"
		}
	}

	// Vérifier avec lsblk si disponible
	cmd = fmt.Sprintf("lsblk -d -o rota /dev/%s 2>/dev/null | tail -1", diskName)
	output, err = client.Execute(cmd)
	if err == nil {
		val := strings.TrimSpace(output)
		if val == "0" {
			return "SSD"
		} else if val == "1" {
			return "HDD"
		}
	}

	return "Unknown"
}

// BrowseDirectory liste le contenu d'un répertoire via SSH
func BrowseDirectory(client *ssh.Client, path string) (*models.DirectoryListing, error) {
	// SÉCURITÉ: Valider le chemin pour éviter path traversal et injection de commandes
	if err := security.ValidatePath(path); err != nil {
		return nil, fmt.Errorf("chemin invalide: %w", err)
	}

	cleanPath := filepath.Clean(path)

	listing := &models.DirectoryListing{
		Path:   cleanPath,
		Parent: filepath.Dir(cleanPath),
	}

	// Utiliser ls -la pour obtenir les détails
	cmd := fmt.Sprintf("ls -la %q 2>/dev/null", cleanPath)
	output, err := client.Execute(cmd)
	if err != nil {
		return nil, fmt.Errorf("erreur lecture répertoire: %w", err)
	}

	lines := strings.Split(strings.TrimSpace(output), "\n")
	for i, line := range lines {
		// Ignorer la première ligne (total)
		if i == 0 && strings.HasPrefix(line, "total") {
			continue
		}

		entry := parseLsLine(line, cleanPath)
		if entry != nil && entry.Name != "." && entry.Name != ".." {
			listing.Entries = append(listing.Entries, *entry)
		}
	}

	return listing, nil
}

// parseLsLine parse une ligne de sortie ls -la
func parseLsLine(line, basePath string) *models.DirectoryEntry {
	// Format: drwxr-xr-x 2 owner group size date time name
	fields := strings.Fields(line)
	if len(fields) < 9 {
		return nil
	}

	entry := &models.DirectoryEntry{
		Permissions: fields[0],
		Owner:       fields[2],
		Group:       fields[3],
		IsDir:       strings.HasPrefix(fields[0], "d"),
	}

	// Taille
	size, _ := strconv.ParseInt(fields[4], 10, 64)
	entry.Size = size

	// Nom (peut contenir des espaces)
	entry.Name = strings.Join(fields[8:], " ")
	entry.Path = filepath.Join(basePath, entry.Name)

	// Date de modification (format variable selon les locales)
	// Essayer de parser la date
	dateStr := strings.Join(fields[5:8], " ")
	t, err := time.Parse("Jan 2 15:04", dateStr)
	if err != nil {
		t, _ = time.Parse("Jan 2 2006", dateStr)
	}
	if !t.IsZero() {
		entry.ModTime = t
	}

	return entry
}

// GetDiskDetails retourne les détails d'un disque spécifique
// osType: "linux", "windows" ou vide (défaut Linux)
func GetDiskDetails(client *ssh.Client, mountPoint string, osType string) (*models.DiskInfo, []models.Partition, error) {
	// Obtenir les infos du disque
	disks, err := CollectDiskInfo(client, osType)
	if err != nil {
		return nil, nil, err
	}

	var targetDisk *models.DiskInfo
	for i, disk := range disks {
		if disk.MountPoint == mountPoint {
			targetDisk = &disks[i]
			break
		}
	}

	if targetDisk == nil {
		return nil, nil, fmt.Errorf("disque non trouvé: %s", mountPoint)
	}

	// Obtenir les partitions via lsblk
	partitions := getPartitions(client, targetDisk.Device)

	return targetDisk, partitions, nil
}

// getPartitions récupère les partitions d'un disque
func getPartitions(client *ssh.Client, device string) []models.Partition {
	var partitions []models.Partition

	// Utiliser lsblk pour obtenir les partitions
	baseName := filepath.Base(device)
	diskName := strings.TrimRight(baseName, "0123456789")

	cmd := fmt.Sprintf("lsblk -b -o NAME,SIZE,TYPE,MOUNTPOINT /dev/%s 2>/dev/null | tail -n +2", diskName)
	output, err := client.Execute(cmd)
	if err != nil {
		return partitions
	}

	lines := strings.Split(strings.TrimSpace(output), "\n")
	for _, line := range lines {
		fields := strings.Fields(line)
		if len(fields) < 3 {
			continue
		}

		// Nettoyer le nom (enlever les caractères de dessin)
		name := strings.Trim(fields[0], "├─└│")

		partition := models.Partition{
			Name: name,
			Type: fields[2],
		}

		size, _ := strconv.ParseUint(fields[1], 10, 64)
		partition.Size = size

		if len(fields) >= 4 {
			partition.MountPoint = fields[3]
		}

		partitions = append(partitions, partition)
	}

	// Obtenir les options de montage
	for i := range partitions {
		if partitions[i].MountPoint != "" {
			partitions[i].Options = getMountOptions(client, partitions[i].MountPoint)
		}
	}

	return partitions
}

// getMountOptions récupère les options de montage d'un point de montage
func getMountOptions(client *ssh.Client, mountPoint string) []string {
	cmd := fmt.Sprintf("findmnt -o OPTIONS %q 2>/dev/null | tail -1", mountPoint)
	output, err := client.Execute(cmd)
	if err != nil {
		return nil
	}

	opts := strings.TrimSpace(output)
	if opts == "" {
		return nil
	}

	return strings.Split(opts, ",")
}

// CollectDiskIOStats collecte les statistiques d'E/S disque via SSH
// osType: "linux", "windows" ou vide (défaut Linux)
func CollectDiskIOStats(client *ssh.Client, osType string) (models.DiskStats, error) {
	if osType == "windows" {
		return collectDiskIOStatsWindows(client)
	}
	return collectDiskIOStatsLinux(client)
}

// collectDiskIOStatsLinux collecte les stats I/O sur Linux
func collectDiskIOStatsLinux(client *ssh.Client) (models.DiskStats, error) {
	var stats models.DiskStats

	// Lire /proc/diskstats
	output, err := client.Execute("cat /proc/diskstats")
	if err != nil {
		return stats, err
	}

	lines := strings.Split(strings.TrimSpace(output), "\n")
	for _, line := range lines {
		fields := strings.Fields(line)
		if len(fields) < 14 {
			continue
		}

		name := fields[2]
		// Filtrer loop, ram, sr (rom)
		if strings.HasPrefix(name, "loop") || strings.HasPrefix(name, "ram") || strings.HasPrefix(name, "sr") {
			continue
		}

		sectorsRead, _ := strconv.ParseUint(fields[5], 10, 64)
		sectorsWritten, _ := strconv.ParseUint(fields[9], 10, 64)

		// Sector = 512 bytes (approximation standard)
		stats.ReadBytes += sectorsRead * 512
		stats.WriteBytes += sectorsWritten * 512
	}

	return stats, nil
}

// collectDiskIOStatsWindows collecte les stats I/O sur Windows via PowerShell
func collectDiskIOStatsWindows(client *ssh.Client) (models.DiskStats, error) {
	var stats models.DiskStats

	// Utiliser Get-Counter pour les performances disque
	cmd := `powershell -Command "$counters = Get-Counter '\PhysicalDisk(_Total)\Disk Read Bytes/sec','\PhysicalDisk(_Total)\Disk Write Bytes/sec' -ErrorAction SilentlyContinue; if($counters) { Write-Output ('{0}|{1}' -f [math]::Round($counters.CounterSamples[0].CookedValue), [math]::Round($counters.CounterSamples[1].CookedValue)) } else { Write-Output '0|0' }"`
	output, err := client.Execute(cmd)
	if err != nil {
		return stats, err
	}

	parts := strings.Split(strings.TrimSpace(output), "|")
	if len(parts) >= 2 {
		stats.ReadBytes, _ = strconv.ParseUint(parts[0], 10, 64)
		stats.WriteBytes, _ = strconv.ParseUint(parts[1], 10, 64)
	}

	return stats, nil
}
