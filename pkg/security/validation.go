package security

import (
	"errors"
	"path/filepath"
	"regexp"
	"strings"
)

var (
	// ErrInvalidServiceName indique que le nom du service est invalide
	ErrInvalidServiceName = errors.New("nom de service invalide")

	// ErrInvalidPath indique que le chemin est invalide
	ErrInvalidPath = errors.New("chemin invalide ou dangereux")

	// ErrInvalidLogSource indique que la source de log est invalide
	ErrInvalidLogSource = errors.New("source de log invalide")
)

// serviceNameRegex valide les noms de services (alphanumeric, tirets, underscores, points)
var serviceNameRegex = regexp.MustCompile(`^[a-zA-Z0-9._-]+$`)

// ValidateServiceName valide un nom de service pour systemctl
// Accepte uniquement les caractères alphanumériques, tirets, underscores et points
func ValidateServiceName(serviceName string) error {
	if serviceName == "" {
		return ErrInvalidServiceName
	}

	// Longueur maximale raisonnable pour un nom de service
	if len(serviceName) > 256 {
		return ErrInvalidServiceName
	}

	// Vérifier le pattern
	if !serviceNameRegex.MatchString(serviceName) {
		return ErrInvalidServiceName
	}

	// Bloquer les caractères dangereux explicitement
	dangerous := []string{";", "&", "|", "$", "`", "(", ")", "<", ">", "\n", "\r", "\\"}
	for _, char := range dangerous {
		if strings.Contains(serviceName, char) {
			return ErrInvalidServiceName
		}
	}

	return nil
}

// IsServiceInWhitelist vérifie si un service est dans la whitelist
// Cette fonction peut être utilisée pour une validation encore plus stricte
func IsServiceInWhitelist(serviceName string, whitelist []string) bool {
	for _, allowed := range whitelist {
		if serviceName == allowed {
			return true
		}
	}
	return false
}

// ValidatePath valide un chemin de fichier pour éviter les path traversal attacks
func ValidatePath(path string) error {
	if path == "" {
		return ErrInvalidPath
	}

	// Nettoyer le chemin
	cleanPath := filepath.Clean(path)

	// Le chemin doit être absolu (commencer par /)
	if !filepath.IsAbs(cleanPath) {
		return ErrInvalidPath
	}

	// Détecter les tentatives de path traversal
	if strings.Contains(path, "..") {
		return ErrInvalidPath
	}

	// Bloquer les caractères dangereux
	dangerous := []string{";", "&", "|", "$", "`", "(", ")", "<", ">", "\n", "\r"}
	for _, char := range dangerous {
		if strings.Contains(path, char) {
			return ErrInvalidPath
		}
	}

	// Bloquer les chemins système sensibles
	sensitivePaths := []string{
		"/etc/shadow",
		"/etc/passwd",
		"/etc/sudoers",
		"/root/.ssh",
		"/home/*/.ssh/id_rsa",
		"/home/*/.ssh/id_ed25519",
	}

	for _, sensitive := range sensitivePaths {
		// Utiliser filepath.Match pour supporter les wildcards
		matched, _ := filepath.Match(sensitive, cleanPath)
		if matched || strings.HasPrefix(cleanPath, strings.TrimSuffix(sensitive, "/*")) {
			return ErrInvalidPath
		}
	}

	return nil
}

// ValidateLogSource valide une source de log (chemin de fichier log)
func ValidateLogSource(source string) error {
	if source == "" {
		return ErrInvalidLogSource
	}

	// Les logs doivent être dans des répertoires standards
	allowedPrefixes := []string{
		"/var/log/",
		"/var/log/nginx/",
		"/var/log/apache2/",
		"/var/log/mysql/",
		"/var/log/postgresql/",
		"/var/log/redis/",
		"/var/log/mongodb/",
		"/var/log/syslog",
		"/var/log/auth.log",
		"/var/log/kern.log",
		"/var/log/dmesg",
		"/var/log/messages",
	}

	// Vérifier si le chemin commence par un préfixe autorisé
	hasValidPrefix := false
	for _, prefix := range allowedPrefixes {
		if strings.HasPrefix(source, prefix) {
			hasValidPrefix = true
			break
		}
	}

	if !hasValidPrefix {
		return ErrInvalidLogSource
	}

	// Utiliser ValidatePath pour les vérifications générales
	return ValidatePath(source)
}

// SanitizeInput retire les caractères potentiellement dangereux d'une entrée utilisateur
// À utiliser en dernier recours - la validation stricte est préférable
func SanitizeInput(input string) string {
	// Remplacer les caractères dangereux par des underscores
	dangerous := []string{";", "&", "|", "$", "`", "(", ")", "<", ">", "\\", "\n", "\r", "\t"}
	sanitized := input

	for _, char := range dangerous {
		sanitized = strings.ReplaceAll(sanitized, char, "_")
	}

	return sanitized
}

// ValidateAction valide une action systemctl (start, stop, restart, status)
func ValidateAction(action string) error {
	validActions := map[string]bool{
		"start":   true,
		"stop":    true,
		"restart": true,
		"status":  true,
		"reload":  true,
		"enable":  true,
		"disable": true,
	}

	if !validActions[action] {
		return errors.New("action systemctl invalide")
	}

	return nil
}
