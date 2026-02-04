package ssh

import (
	"bufio"
	"bytes"
	"crypto/md5"
	"crypto/sha256"
	"encoding/base64"
	"fmt"
	"net"
	"os"
	"path/filepath"
	"strings"
	"sync"

	"golang.org/x/crypto/ssh"
)

// HostKeyManager gÃ¨re les clÃ©s SSH des hÃ´tes connus
type HostKeyManager struct {
	knownHostsPath string
	knownHosts     map[string]ssh.PublicKey
	mu             sync.RWMutex
	trustOnFirstUse bool // Mode "Trust On First Use"
}

// NewHostKeyManager crÃ©e un nouveau gestionnaire de host keys
func NewHostKeyManager(knownHostsPath string, trustOnFirstUse bool) (*HostKeyManager, error) {
	if knownHostsPath == "" {
		// Chemin par dÃ©faut
		home, err := os.UserHomeDir()
		if err != nil {
			return nil, fmt.Errorf("impossible de dÃ©terminer le home directory: %w", err)
		}
		knownHostsPath = filepath.Join(home, ".ssh", "known_hosts_monitoring")
	}

	manager := &HostKeyManager{
		knownHostsPath:  knownHostsPath,
		knownHosts:      make(map[string]ssh.PublicKey),
		trustOnFirstUse: trustOnFirstUse,
	}

	// Charger les host keys existants
	if err := manager.loadKnownHosts(); err != nil {
		// Si le fichier n'existe pas, ce n'est pas grave
		if !os.IsNotExist(err) {
			return nil, err
		}
	}

	return manager, nil
}

// loadKnownHosts charge le fichier known_hosts
func (hkm *HostKeyManager) loadKnownHosts() error {
	file, err := os.Open(hkm.knownHostsPath)
	if err != nil {
		return err
	}
	defer file.Close()

	scanner := bufio.NewScanner(file)
	lineNum := 0

	for scanner.Scan() {
		lineNum++
		line := strings.TrimSpace(scanner.Text())

		// Ignorer les lignes vides et les commentaires
		if line == "" || strings.HasPrefix(line, "#") {
			continue
		}

		// Format: hostname keytype base64key
		parts := strings.Fields(line)
		if len(parts) < 3 {
			continue // Ligne invalide, ignorer
		}

		hostname := parts[0]
		keyType := parts[1]
		keyData := parts[2]

		// DÃ©coder la clÃ©
		decoded, err := base64.StdEncoding.DecodeString(keyData)
		if err != nil {
			continue // ClÃ© invalide, ignorer
		}

		// Parser la clÃ© publique
		pubKey, err := ssh.ParsePublicKey(decoded)
		if err != nil {
			continue // ClÃ© invalide, ignorer
		}

		// VÃ©rifier que le type correspond
		if pubKey.Type() != keyType {
			continue
		}

		hkm.knownHosts[hostname] = pubKey
	}

	return scanner.Err()
}

// saveHostKey sauvegarde une nouvelle clÃ© d'hÃ´te dans le fichier
func (hkm *HostKeyManager) saveHostKey(hostname string, key ssh.PublicKey) error {
	// CrÃ©er le rÃ©pertoire .ssh s'il n'existe pas
	dir := filepath.Dir(hkm.knownHostsPath)
	if err := os.MkdirAll(dir, 0700); err != nil {
		return fmt.Errorf("erreur crÃ©ation rÃ©pertoire: %w", err)
	}

	// Ouvrir le fichier en mode append
	file, err := os.OpenFile(hkm.knownHostsPath, os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0600)
	if err != nil {
		return fmt.Errorf("erreur ouverture fichier: %w", err)
	}
	defer file.Close()

	// Format: hostname keytype base64key
	keyData := base64.StdEncoding.EncodeToString(key.Marshal())
	line := fmt.Sprintf("%s %s %s\n", hostname, key.Type(), keyData)

	if _, err := file.WriteString(line); err != nil {
		return fmt.Errorf("erreur Ã©criture fichier: %w", err)
	}

	return nil
}

// HostKeyCallback retourne un callback pour vÃ©rifier les host keys
func (hkm *HostKeyManager) HostKeyCallback() ssh.HostKeyCallback {
	return func(hostname string, remote net.Addr, key ssh.PublicKey) error {
		hkm.mu.Lock()
		defer hkm.mu.Unlock()

		// Normaliser le hostname (enlever le port si prÃ©sent)
		host, _, _ := net.SplitHostPort(hostname)
		if host == "" {
			host = hostname
		}

		// VÃ©rifier si on connaÃ®t cet hÃ´te
		knownKey, exists := hkm.knownHosts[host]

		if exists {
			// Comparer les clÃ©s
			if !bytes.Equal(knownKey.Marshal(), key.Marshal()) {
				return fmt.Errorf("âš ï¸  HOST KEY MISMATCH pour %s!\n"+
					"Fingerprint attendu: %s\n"+
					"Fingerprint reÃ§u:     %s\n"+
					"POSSIBLE MAN-IN-THE-MIDDLE ATTACK!",
					host,
					FormatFingerprint(knownKey),
					FormatFingerprint(key))
			}
			// ClÃ© correspond, OK
			return nil
		}

		// HÃ´te inconnu
		if hkm.trustOnFirstUse {
			// Mode TOFU: accepter et sauvegarder automatiquement
			fmt.Printf("â„¹ï¸  Nouvel hÃ´te %s (TOFU)\n", host)
			fmt.Printf("   Fingerprint: %s\n", FormatFingerprint(key))

			if err := hkm.saveHostKey(host, key); err != nil {
				return fmt.Errorf("erreur sauvegarde host key: %w", err)
			}

			hkm.knownHosts[host] = key
			return nil
		}

		// Mode strict: demander confirmation (mode interactif non disponible en service)
		// En production, on devrait prÃ©-remplir known_hosts ou utiliser TOFU
		return fmt.Errorf("hÃ´te inconnu %s. Fingerprint: %s\n"+
			"Ajoutez cette clÃ© dans %s pour l'accepter",
			host,
			FormatFingerprint(key),
			hkm.knownHostsPath)
	}
}

// FormatFingerprint formatte une clÃ© publique en fingerprint SHA256 (format OpenSSH moderne)
func FormatFingerprint(key ssh.PublicKey) string {
	hash := sha256.Sum256(key.Marshal())
	b64 := base64.RawStdEncoding.EncodeToString(hash[:])
	return "SHA256:" + b64
}

// FormatFingerprintMD5 formatte une clÃ© publique en fingerprint MD5 (format legacy)
func FormatFingerprintMD5(key ssh.PublicKey) string {
	hash := md5.Sum(key.Marshal())
	parts := make([]string, len(hash))
	for i, b := range hash {
		parts[i] = fmt.Sprintf("%02x", b)
	}
	return strings.Join(parts, ":")
}

// GetKnownHostsPath retourne le chemin du fichier known_hosts
func (hkm *HostKeyManager) GetKnownHostsPath() string {
	return hkm.knownHostsPath
}

// GetHostCount retourne le nombre d'hÃ´tes connus
func (hkm *HostKeyManager) GetHostCount() int {
	hkm.mu.RLock()
	defer hkm.mu.RUnlock()
	return len(hkm.knownHosts)
}
