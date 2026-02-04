package collectors

import (
	"fmt"
	"strings"

	"go-monitoring/pkg/security"
	"go-monitoring/ssh"
)

// ServiceAction exÃ©cute une action sur un service (start, stop, restart)
func ServiceAction(client *ssh.Client, serviceName, action, osType string) error {
	var cmd string

	// Validation de l'action pour Ã©viter l'injection de commandes
	if err := security.ValidateAction(action); err != nil {
		return fmt.Errorf("action invalide: %s", action)
	}

	// SÃ‰CURITÃ‰: Valider le nom du service pour Ã©viter l'injection de commandes
	if err := security.ValidateServiceName(serviceName); err != nil {
		return fmt.Errorf("nom de service invalide '%s': %w", serviceName, err)
	}

	if osType == "windows" {
		// PowerShell commands
		// Note: NÃ©cessite des droits admin. On suppose que l'utilisateur SSH a les droits.
		psAction := action
		if action == "restart" {
			psAction = "Restart"
		} else if action == "stop" {
			psAction = "Stop"
		} else if action == "start" {
			psAction = "Start"
		}

		cmd = fmt.Sprintf("powershell -Command \"%s-Service -Name '%s' -Force\"", psAction, serviceName)
	} else {
		// Linux (Systemd)
		// Note: NÃ©cessite souvent sudo.
		// On tente avec sudo -n (non-interactive) pour voir si Ã§a passe sans mot de passe
		// Sinon la commande Ã©chouera et l'erreur sera remontÃ©e
		cmd = fmt.Sprintf("sudo -n systemctl %s %s", action, serviceName)
	}

	output, err := client.Execute(cmd)
	if err != nil {
		return fmt.Errorf("erreur exÃ©cution '%s': %v (Output: %s)", cmd, err, strings.TrimSpace(string(output)))
	}

	return nil
}
