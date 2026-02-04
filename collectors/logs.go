package collectors

import (
	"fmt"
	"strings"

	"go-monitoring/pkg/security"
	"go-monitoring/ssh"
)

// LogSource dÃ©finit une source de logs disponible
type LogSource struct {
	ID   string `json:"id"`
	Name string `json:"name"`
	Type string `json:"type"`           // file, journal, docker
	Path string `json:"path,omitempty"` // Pour les fichiers
	Cmd  string `json:"cmd,omitempty"`  // Pour journalctl ou docker logs
}

// GetAvailableLogSources retourne la liste des sources de logs pour un OS donnÃ©
func GetAvailableLogSources(client *ssh.Client, osType string) ([]LogSource, error) {
	var sources []LogSource

	if osType == "windows" {
		// Windows Event Logs
		sources = append(sources, LogSource{
			ID:   "win-system",
			Name: "Windows System Events",
			Type: "powershell",
			Cmd:  "Get-EventLog -LogName System -Newest 100 | Format-Table -AutoSize | Out-String -Width 120",
		})
		sources = append(sources, LogSource{
			ID:   "win-app",
			Name: "Windows Application Events",
			Type: "powershell",
			Cmd:  "Get-EventLog -LogName Application -Newest 100 | Format-Table -AutoSize | Out-String -Width 120",
		})
	} else {
		// Linux Standard Logs
		files := []string{
			"/var/log/syslog",
			"/var/log/messages",
			"/var/log/auth.log",
			"/var/log/kern.log",
			"/var/log/nginx/error.log",
			"/var/log/nginx/access.log",
			"/var/log/apache2/error.log",
			"/var/log/mysql/error.log",
		}

		// VÃ©rifier l'existence des fichiers
		for _, f := range files {
			// Test simple avec 'test -f'
			_, err := client.Execute("test -f " + f)
			if err == nil {
				sources = append(sources, LogSource{
					ID:   "file-" + strings.ReplaceAll(f, "/", "-"),
					Name: f,
					Type: "file",
					Path: f,
				})
			}
		}

		// Ajout Systemd Journal
		sources = append(sources, LogSource{
			ID:   "journal-sys",
			Name: "Journalctl (System)",
			Type: "journal",
			Cmd:  "journalctl -n 100 --no-pager",
		})

		// Ajout Docker Logs (si docker existe)
		_, err := client.Execute("which docker")
		if err == nil {
			// Lister les containers pour offrir leurs logs?
			// Pour l'instant on met juste une commande gÃ©nÃ©rique ou on laisse l'utilisateur choisir?
			// On va faire simple: juste les logs du daemon docker s'ils sont dispo via journalctl
			sources = append(sources, LogSource{
				ID:   "journal-docker",
				Name: "Journalctl (Docker Service)",
				Type: "journal",
				Cmd:  "journalctl -u docker -n 100 --no-pager",
			})
		}
	}

	return sources, nil
}

// FetchLogContent rÃ©cupÃ¨re le contenu des logs
func FetchLogContent(client *ssh.Client, source LogSource, lines int) (string, error) {
	var cmd string

	if lines <= 0 {
		lines = 100
	}

	if source.Type == "file" {
		// SÃ‰CURITÃ‰: Valider le chemin du fichier log
		if err := security.ValidateLogSource(source.Path); err != nil {
			return "", fmt.Errorf("source de log invalide: %w", err)
		}
		cmd = fmt.Sprintf("tail -n %d %q", lines, source.Path)
	} else if source.Type == "journal" {
		// Override -n in command if present, or reconstruct
		if strings.HasPrefix(source.Cmd, "journalctl") {
			// Reconstruction propre pour changer le nombre de lignes
			base := strings.Split(source.Cmd, " -n")[0] // hack simple
			if !strings.Contains(base, "journalctl") {
				base = source.Cmd // fallback
			} else {
				cmd = fmt.Sprintf("%s -n %d --no-pager", base, lines)
			}
		} else {
			cmd = source.Cmd
		}

		if cmd == "" {
			cmd = source.Cmd
		}
	} else if source.Type == "powershell" {
		// Pour windows, changer le -Newest X
		// C'est un peu complexe de parser la cmd string, on va juste exÃ©cuter tel quel pour l'instant
		// ou reconstruire si on est sÃ»r du format.
		cmd = source.Cmd
	} else {
		return "", fmt.Errorf("type de log inconnu: %s", source.Type)
	}

	return client.Execute(cmd)
}
