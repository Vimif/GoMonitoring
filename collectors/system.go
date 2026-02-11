package collectors

import (
	"strings"
	"time"

	"go-monitoring/models"
	"go-monitoring/ssh"
)

// DetectOS détecte le type d'OS de la machine (linux ou windows)
func DetectOS(client ssh.SSHExecutor) string {
	// Essayer une commande Linux d'abord
	output, err := client.Execute("uname -s 2>/dev/null || echo NOTLINUX")
	if err == nil {
		osName := strings.TrimSpace(output)
		if osName != "" && osName != "NOTLINUX" && !strings.Contains(strings.ToLower(osName), "windows") {
			return "linux"
		}
	}

	// Essayer une commande Windows
	output, err = client.Execute("echo %OS%")
	if err == nil && strings.Contains(strings.ToLower(output), "windows") {
		return "windows"
	}

	// Essayer avec PowerShell
	output, err = client.Execute("powershell -Command \"$env:OS\"")
	if err == nil && strings.Contains(strings.ToLower(output), "windows") {
		return "windows"
	}

	return "unknown"
}

// CollectSystemInfo collecte les informations système via SSH
// osType peut être "linux", "windows" ou vide (auto-détection)
func CollectSystemInfo(client ssh.SSHExecutor, osType string) (models.SystemInfo, string, error) {
	// Auto-détection si osType n'est pas spécifié
	if osType == "" || osType == "unknown" {
		osType = DetectOS(client)
	}

	if osType == "windows" {
		info, err := collectSystemInfoWindows(client)
		return info, osType, err
	}
	info, err := collectSystemInfoLinux(client)
	return info, osType, err
}

// collectSystemInfoLinux collecte les infos système sur Linux
func collectSystemInfoLinux(client ssh.SSHExecutor) (models.SystemInfo, error) {
	var info models.SystemInfo

	// Hostname
	hostname, err := client.Execute("hostname")
	if err == nil {
		info.Hostname = strings.TrimSpace(hostname)
	}

	// OS et distribution
	// OS et distribution
	// Tentative 1: Standard os-release
	osName, err := client.Execute("source /etc/os-release && echo $PRETTY_NAME")
	if err == nil && strings.TrimSpace(osName) != "" {
		info.OS = strings.TrimSpace(osName)
	}

	// Tentative 2: RedHat/Amazon system-release
	if info.OS == "" {
		osName, err = client.Execute("cat /etc/system-release 2>/dev/null")
		if err == nil && strings.TrimSpace(osName) != "" {
			info.OS = strings.TrimSpace(osName)
		}
	}

	// Tentative 3: Fallback uname
	if info.OS == "" {
		osName, err = client.Execute("uname -sr")
		if err == nil {
			info.OS = strings.TrimSpace(osName)
		}
	}

	// Version du kernel
	kernel, err := client.Execute("uname -r")
	if err == nil {
		info.Kernel = strings.TrimSpace(kernel)
	}

	// Architecture
	arch, err := client.Execute("uname -m")
	if err == nil {
		info.Architecture = strings.TrimSpace(arch)
	}

	// Uptime
	uptime, err := client.Execute("uptime -p")
	if err == nil {
		info.Uptime = strings.TrimSpace(uptime)
	} else {
		// Fallback pour les systèmes sans uptime -p
		uptime, err = client.Execute("cat /proc/uptime | awk '{print $1}'")
		if err == nil {
			info.Uptime = strings.TrimSpace(uptime) + " secondes"
		}
	}

	// Date de boot
	bootTime, err := client.Execute("who -b | awk '{print $3, $4}'")
	if err == nil {
		t, err := time.Parse("2006-01-02 15:04", strings.TrimSpace(bootTime))
		if err == nil {
			info.BootTime = t
		}
	}

	return info, nil
}

// collectSystemInfoWindows collecte les infos système sur Windows via PowerShell
func collectSystemInfoWindows(client ssh.SSHExecutor) (models.SystemInfo, error) {
	var info models.SystemInfo

	// Hostname
	hostname, err := client.Execute("powershell -Command \"$env:COMPUTERNAME\"")
	if err == nil {
		info.Hostname = strings.TrimSpace(hostname)
	}

	// OS Name
	osName, err := client.Execute("powershell -Command \"(Get-CimInstance Win32_OperatingSystem).Caption\"")
	if err == nil {
		info.OS = strings.TrimSpace(osName)
	}

	// OS Version (utilisé comme "Kernel" pour Windows)
	version, err := client.Execute("powershell -Command \"(Get-CimInstance Win32_OperatingSystem).Version\"")
	if err == nil {
		info.Kernel = strings.TrimSpace(version)
	}

	// Architecture
	arch, err := client.Execute("powershell -Command \"(Get-CimInstance Win32_OperatingSystem).OSArchitecture\"")
	if err == nil {
		info.Architecture = strings.TrimSpace(arch)
	}

	// Uptime - calculé à partir de LastBootUpTime
	uptimeCmd := `powershell -Command "$boot = (Get-CimInstance Win32_OperatingSystem).LastBootUpTime; $uptime = (Get-Date) - $boot; '{0}j {1}h {2}m' -f $uptime.Days, $uptime.Hours, $uptime.Minutes"`
	uptime, err := client.Execute(uptimeCmd)
	if err == nil {
		info.Uptime = strings.TrimSpace(uptime)
	}

	// Date de boot
	bootTimeCmd := `powershell -Command "(Get-CimInstance Win32_OperatingSystem).LastBootUpTime.ToString('yyyy-MM-dd HH:mm')"`
	bootTimeStr, err := client.Execute(bootTimeCmd)
	if err == nil {
		t, err := time.Parse("2006-01-02 15:04", strings.TrimSpace(bootTimeStr))
		if err == nil {
			info.BootTime = t
		}
	}

	return info, nil
}
