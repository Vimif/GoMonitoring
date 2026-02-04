package collectors

import (
	"go-monitoring/models"
	"go-monitoring/ssh"
	"strings"
)

// CollectServices checks status of given services
// osType: "linux", "windows" ou vide (dÃ©faut Linux)
func CollectServices(client *ssh.Client, services []string, osType string) ([]models.ServiceStatus, error) {
	if len(services) == 0 {
		return []models.ServiceStatus{}, nil
	}

	if osType == "windows" {
		return collectServicesWindows(client, services)
	}
	return collectServicesLinux(client, services)
}

// collectServicesLinux vÃ©rifie le status des services sur Linux via systemctl
func collectServicesLinux(client *ssh.Client, services []string) ([]models.ServiceStatus, error) {
	var results []models.ServiceStatus

	// Use || true to prevent exit code 1 if a service is inactive
	cmd := "systemctl is-active " + strings.Join(services, " ") + " || true"
	output, err := client.Execute(cmd)
	if err != nil {
		return nil, err
	}

	lines := strings.Split(strings.TrimSpace(output), "\n")

	for i, service := range services {
		status := "unknown"
		if i < len(lines) {
			status = strings.TrimSpace(lines[i])
		}
		results = append(results, models.ServiceStatus{
			Name:   service,
			Status: status,
		})
	}

	return results, nil
}

// collectServicesWindows vÃ©rifie le status des services sur Windows via PowerShell
func collectServicesWindows(client *ssh.Client, services []string) ([]models.ServiceStatus, error) {
	var results []models.ServiceStatus

	// Construire la liste des services pour PowerShell
	serviceList := "'" + strings.Join(services, "','") + "'"
	cmd := `powershell -Command "$services = @(` + serviceList + `); foreach($s in $services) { $svc = Get-Service -Name $s -ErrorAction SilentlyContinue; if($svc) { Write-Output ('{0}|{1}' -f $s, $svc.Status) } else { Write-Output ('{0}|not_found' -f $s) } }"`

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
		if len(parts) >= 2 {
			status := strings.ToLower(parts[1])
			// Mapper les statuts Windows vers les statuts Linux
			switch status {
			case "running":
				status = "active"
			case "stopped":
				status = "inactive"
			case "not_found":
				status = "not-found"
			}
			results = append(results, models.ServiceStatus{
				Name:   parts[0],
				Status: status,
			})
		}
	}

	return results, nil
}
