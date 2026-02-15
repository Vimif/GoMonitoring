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
func CollectCPUInfo(client ssh.SSHExecutor, osType string) (models.CPUInfo, error) {
	if osType == "windows" {
		return collectCPUInfoWindows(client)
	}
	return collectCPUInfoLinux(client)
}

// collectCPUInfoLinux collecte les infos CPU sur Linux
func collectCPUInfoLinux(client ssh.SSHExecutor) (models.CPUInfo, error) {
	var info models.CPUInfo

	// Combined command to reduce SSH round-trips from 5 to 1
	// We use "::::::" as a separator between command outputs
	cmd := `cat /proc/cpuinfo | grep 'model name' | head -1 | cut -d':' -f2 || echo "UNKNOWN"; echo "::::::"; grep -c ^processor /proc/cpuinfo || echo "0"; echo "::::::"; lscpu 2>/dev/null | grep '^CPU(s):' | awk '{print $2}' || echo "0"; echo "::::::"; cat /proc/cpuinfo | grep 'cpu MHz' | head -1 | cut -d':' -f2 || echo "0"; echo "::::::"; top -bn1 2>/dev/null | grep 'Cpu(s)' | head -1 || echo ""`

	output, err := client.Execute(cmd)
	if err != nil {
		return info, err
	}

	parts := strings.Split(output, "::::::")
	for i := range parts {
		parts[i] = strings.TrimSpace(parts[i])
	}

	// 1. Modèle du CPU
	if len(parts) > 0 {
		info.Model = parts[0]
	}

	// 2. Nombre de cores physiques
	if len(parts) > 1 {
		n, _ := strconv.Atoi(parts[1])
		info.Cores = n
	}

	// 3. Nombre de threads
	if len(parts) > 2 {
		n, _ := strconv.Atoi(parts[2])
		info.Threads = n
	}
	// Fallback to cores if threads not found or 0
	if info.Threads == 0 {
		info.Threads = info.Cores
	}

	// 4. Fréquence MHz
	if len(parts) > 3 {
		f, _ := strconv.ParseFloat(parts[3], 64)
		info.MHz = f
	}

	// 5. Usage CPU
	if len(parts) > 4 {
		info.UsagePercent = parseCPUUsage(parts[4])
	}

	return info, nil
}

// collectCPUInfoWindows collecte les infos CPU sur Windows via PowerShell
func collectCPUInfoWindows(client ssh.SSHExecutor) (models.CPUInfo, error) {
	var info models.CPUInfo

	// Modèle du CPU
	model, err := client.Execute(`powershell -Command "(Get-CimInstance Win32_Processor).Name"`)
	if err != nil {
		return info, err
	}
	info.Model = strings.TrimSpace(model)

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
	// Supporte point ou virgule, et % optionnel
	re := regexp.MustCompile(`(\d+[\.,]?\d*)\s*%?id`)
	matches := re.FindStringSubmatch(topOutput)
	if len(matches) >= 2 {
		valStr := strings.Replace(matches[1], ",", ".", 1)
		idle, err := strconv.ParseFloat(valStr, 64)
		if err == nil {
			usage := 100.0 - idle
			if usage < 0 {
				return 0.0
			}
			if usage > 100 {
				return 100.0
			}
			return usage
		}
	}

	// Alternative: additionner us + sy + ni
	reUs := regexp.MustCompile(`(\d+[\.,]?\d*)\s*%?us`)
	reSy := regexp.MustCompile(`(\d+[\.,]?\d*)\s*%?sy`)
	reNi := regexp.MustCompile(`(\d+[\.,]?\d*)\s*%?ni`)

	var total float64
	parse := func(re *regexp.Regexp) float64 {
		if m := re.FindStringSubmatch(topOutput); len(m) >= 2 {
			valStr := strings.Replace(m[1], ",", ".", 1)
			v, _ := strconv.ParseFloat(valStr, 64)
			return v
		}
		return 0
	}

	total += parse(reUs)
	total += parse(reSy)
	total += parse(reNi)

	if total > 100 {
		return 100.0
	}
	return total
}
