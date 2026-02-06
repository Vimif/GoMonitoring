package collectors

import (
	"strconv"
	"strings"

	"go-monitoring/models"
	"go-monitoring/ssh"
)

// CollectNetworkStats collecte les statistiques réseau via SSH
// osType: "linux", "windows" ou vide (défaut Linux)
func CollectNetworkStats(client *ssh.Client, osType string) (models.NetworkStats, error) {
	if osType == "windows" {
		return collectNetworkStatsWindows(client)
	}
	return collectNetworkStatsLinux(client)
}

// collectNetworkStatsLinux collecte les stats réseau sur Linux
func collectNetworkStatsLinux(client *ssh.Client) (models.NetworkStats, error) {
	var stats models.NetworkStats

	// Lire /proc/net/dev
	// Format: Interface | Receive (bytes packets errs drop...) | Transmit (bytes packets errs drop...)
	output, err := client.Execute("cat /proc/net/dev")
	if err != nil {
		return stats, err
	}

	lines := strings.Split(strings.TrimSpace(output), "\n")
	for _, line := range lines {
		// Ignorer les en-têtes
		if strings.Contains(line, "|") {
			continue
		}

		parts := strings.Split(line, ":")
		if len(parts) != 2 {
			continue
		}

		// interface := strings.TrimSpace(parts[0])
		values := strings.Fields(parts[1])

		if len(values) < 9 {
			continue
		}

		// RxBytes idx 0, TxBytes idx 8
		rx, _ := strconv.ParseUint(values[0], 10, 64)
		tx, _ := strconv.ParseUint(values[8], 10, 64)

		// Somme de toutes les interfaces
		stats.RxBytes += rx
		stats.TxBytes += tx
	}

	return stats, nil
}

// collectNetworkStatsWindows collecte les stats réseau sur Windows via PowerShell
func collectNetworkStatsWindows(client *ssh.Client) (models.NetworkStats, error) {
	var stats models.NetworkStats

	// Utiliser Get-NetAdapterStatistics
	cmd := `powershell -Command "Get-NetAdapterStatistics | ForEach-Object { Write-Output ('{0}|{1}' -f $_.ReceivedBytes, $_.SentBytes) }"`
	output, err := client.Execute(cmd)
	if err != nil {
		return stats, err
	}

	lines := strings.Split(strings.TrimSpace(output), "\n")
	for _, line := range lines {
		line = strings.TrimSpace(line)
		if line == "" {
			continue
		}

		parts := strings.Split(line, "|")
		if len(parts) >= 2 {
			rx, _ := strconv.ParseUint(parts[0], 10, 64)
			tx, _ := strconv.ParseUint(parts[1], 10, 64)
			stats.RxBytes += rx
			stats.TxBytes += tx
		}
	}

	return stats, nil
}
