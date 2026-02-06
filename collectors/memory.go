package collectors

import (
	"strconv"
	"strings"

	"go-monitoring/models"
	"go-monitoring/ssh"
)

// CollectMemoryInfo collecte les informations mémoire via SSH
// osType: "linux", "windows" ou vide (défaut Linux)
func CollectMemoryInfo(client *ssh.Client, osType string) (models.MemoryInfo, error) {
	if osType == "windows" {
		return collectMemoryInfoWindows(client)
	}
	return collectMemoryInfoLinux(client)
}

// collectMemoryInfoLinux collecte les infos mémoire sur Linux
func collectMemoryInfoLinux(client *ssh.Client) (models.MemoryInfo, error) {
	var info models.MemoryInfo

	// Utiliser free -b pour obtenir les valeurs en bytes
	output, err := client.Execute("free -b | grep Mem")
	if err != nil {
		return info, err
	}

	// Format: Mem:   total   used   free   shared   buff/cache   available
	fields := strings.Fields(output)
	if len(fields) >= 7 {
		info.Total, _ = strconv.ParseUint(fields[1], 10, 64)
		info.Used, _ = strconv.ParseUint(fields[2], 10, 64)
		info.Free, _ = strconv.ParseUint(fields[3], 10, 64)
		info.Available, _ = strconv.ParseUint(fields[6], 10, 64)

		// Recalculer Used pour exclure strictement le cache (Total - Available)
		if info.Total > 0 && info.Available > 0 {
			info.Used = info.Total - info.Available
		}
	}

	// Calculer le pourcentage d'utilisation
	if info.Total > 0 {
		info.UsedPercent = float64(info.Used) / float64(info.Total) * 100
	}

	return info, nil
}

// collectMemoryInfoWindows collecte les infos mémoire sur Windows via PowerShell
func collectMemoryInfoWindows(client *ssh.Client) (models.MemoryInfo, error) {
	var info models.MemoryInfo

	// Récupérer Total et Free en une seule commande
	cmd := `powershell -Command "$os = Get-CimInstance Win32_OperatingSystem; Write-Output ('{0}|{1}' -f ($os.TotalVisibleMemorySize * 1KB), ($os.FreePhysicalMemory * 1KB))"`
	output, err := client.Execute(cmd)
	if err != nil {
		return info, err
	}

	// Parse "total|free"
	parts := strings.Split(strings.TrimSpace(output), "|")
	if len(parts) >= 2 {
		info.Total, _ = strconv.ParseUint(parts[0], 10, 64)
		info.Free, _ = strconv.ParseUint(parts[1], 10, 64)
		info.Available = info.Free
		info.Used = info.Total - info.Free
	}

	// Calculer le pourcentage d'utilisation
	if info.Total > 0 {
		info.UsedPercent = float64(info.Used) / float64(info.Total) * 100
	}

	return info, nil
}
