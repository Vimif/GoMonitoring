package collectors

import (
	"regexp"
	"strconv"
	"strings"

	"go-monitoring/models"
	"go-monitoring/ssh"
)

// CollectCPUInfo collecte les informations CPU via SSH
// osType: "linux", "windows" ou vide (défaut Linux)
func CollectCPUInfo(client *ssh.Client, osType string) (models.CPUInfo, error) {
	if osType == "windows" {
		return collectCPUInfoWindows(client)
	}
	return collectCPUInfoLinux(client)
}

// collectCPUInfoLinux collecte les infos CPU sur Linux
func collectCPUInfoLinux(client *ssh.Client) (models.CPUInfo, error) {
	var info models.CPUInfo

	// Modèle du CPU
	model, err := client.Execute("cat /proc/cpuinfo | grep 'model name' | head -1 | cut -d':' -f2")
	if err == nil {
		info.Model = strings.TrimSpace(model)
	}

	// Nombre de cores physiques
	cores, err := client.Execute("grep -c ^processor /proc/cpuinfo")
	if err == nil {
		n, _ := strconv.Atoi(strings.TrimSpace(cores))
		info.Cores = n
	}

	// Nombre de threads (avec lscpu si disponible)
	threads, err := client.Execute("lscpu | grep '^CPU(s):' | awk '{print $2}'")
	if err == nil {
		n, _ := strconv.Atoi(strings.TrimSpace(threads))
		info.Threads = n
	}
	if info.Threads == 0 {
		info.Threads = info.Cores
	}

	// Fréquence MHz
	mhz, err := client.Execute("cat /proc/cpuinfo | grep 'cpu MHz' | head -1 | cut -d':' -f2")
	if err == nil {
		f, _ := strconv.ParseFloat(strings.TrimSpace(mhz), 64)
		info.MHz = f
	}

	// Usage CPU (via top en mode batch)
	usage, err := client.Execute("top -bn1 | grep 'Cpu(s)' | head -1")
	if err == nil {
		info.UsagePercent = parseCPUUsage(usage)
	}

	return info, nil
}

// collectCPUInfoWindows collecte les infos CPU sur Windows via PowerShell
func collectCPUInfoWindows(client *ssh.Client) (models.CPUInfo, error) {
	var info models.CPUInfo

	// Modèle du CPU
	model, err := client.Execute(`powershell -Command "(Get-CimInstance Win32_Processor).Name"`)
	if err == nil {
		info.Model = strings.TrimSpace(model)
	}

	// Nombre de cores physiques
	cores, err := client.Execute(`powershell -Command "(Get-CimInstance Win32_Processor).NumberOfCores"`)
	if err == nil {
		n, _ := strconv.Atoi(strings.TrimSpace(cores))
		info.Cores = n
	}

	// Nombre de threads logiques
	threads, err := client.Execute(`powershell -Command "(Get-CimInstance Win32_Processor).NumberOfLogicalProcessors"`)
	if err == nil {
		n, _ := strconv.Atoi(strings.TrimSpace(threads))
		info.Threads = n
	}
	if info.Threads == 0 {
		info.Threads = info.Cores
	}

	// Fréquence MHz
	mhz, err := client.Execute(`powershell -Command "(Get-CimInstance Win32_Processor).MaxClockSpeed"`)
	if err == nil {
		f, _ := strconv.ParseFloat(strings.TrimSpace(mhz), 64)
		info.MHz = f
	}

	// Usage CPU
	usage, err := client.Execute(`powershell -Command "(Get-CimInstance Win32_Processor).LoadPercentage"`)
	if err == nil {
		f, _ := strconv.ParseFloat(strings.TrimSpace(usage), 64)
		info.UsagePercent = f
	}

	return info, nil
}

// parseCPUUsage extrait le pourcentage d'utilisation CPU de la sortie top
func parseCPUUsage(topOutput string) float64 {
	// Format: Cpu(s):  X.X us,  X.X sy,  X.X ni, XX.X id, ...
	// L'utilisation = 100 - idle (id)

	// Regex pour extraire le pourcentage idle
	re := regexp.MustCompile(`(\d+\.?\d*)\s*id`)
	matches := re.FindStringSubmatch(topOutput)
	if len(matches) >= 2 {
		idle, err := strconv.ParseFloat(matches[1], 64)
		if err == nil {
			return 100.0 - idle
		}
	}

	// Alternative: additionner us + sy + ni
	reUs := regexp.MustCompile(`(\d+\.?\d*)\s*us`)
	reSy := regexp.MustCompile(`(\d+\.?\d*)\s*sy`)
	reNi := regexp.MustCompile(`(\d+\.?\d*)\s*ni`)

	var total float64
	if m := reUs.FindStringSubmatch(topOutput); len(m) >= 2 {
		v, _ := strconv.ParseFloat(m[1], 64)
		total += v
	}
	if m := reSy.FindStringSubmatch(topOutput); len(m) >= 2 {
		v, _ := strconv.ParseFloat(m[1], 64)
		total += v
	}
	if m := reNi.FindStringSubmatch(topOutput); len(m) >= 2 {
		v, _ := strconv.ParseFloat(m[1], 64)
		total += v
	}

	return total
}
