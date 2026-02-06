package config

import (
	"fmt"
	"log"
	"os"

	"go-monitoring/pkg/crypto"
	"gopkg.in/yaml.v3"
)

// Config représente la configuration complète
type Config struct {
	Machines []MachineConfig `yaml:"machines"`
	Settings Settings        `yaml:"settings"`
	Users    []UserConfig    `yaml:"users"`
}

// UserConfig représente un utilisateur
type UserConfig struct {
	Username string `yaml:"username"`
	Password string `yaml:"password"`
	Role     string `yaml:"role"`
}

// MachineConfig représente la configuration d'une machine
type MachineConfig struct {
	ID       string   `yaml:"id" json:"id"`
	Name     string   `yaml:"name" json:"name"`
	Host     string   `yaml:"host" json:"host"`
	Port     int      `yaml:"port" json:"port"`
	User     string   `yaml:"user" json:"user"`
	KeyPath  string   `yaml:"key_path,omitempty" json:"key_path,omitempty"`
	Password string   `yaml:"password,omitempty" json:"password,omitempty"`
	Group    string   `yaml:"group,omitempty" json:"group,omitempty"`
	OS       string   `yaml:"os,omitempty" json:"os,omitempty"` // "linux", "windows"
	Services []string `yaml:"services,omitempty" json:"services,omitempty"`
}

// Thresholds contient les seuils d'alerte pour la conformité
type Thresholds struct {
	DiskMinPercent   float64 `yaml:"disk_min_percent"`   // Alerte si espace libre < X%
	MemoryMinPercent float64 `yaml:"memory_min_percent"` // Alerte si mémoire libre < X%
	CPUMaxPercent    float64 `yaml:"cpu_max_percent"`    // Alerte si CPU > X%
}

// Settings contient les paramètres généraux
type Settings struct {
	RefreshInterval int        `yaml:"refresh_interval"`
	SSHTimeout      int        `yaml:"ssh_timeout"`
	Thresholds      Thresholds `yaml:"thresholds,omitempty"`
}

// LoadConfig charge la configuration depuis un fichier YAML
func LoadConfig(path string) (*Config, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, fmt.Errorf("erreur lecture fichier config: %w", err)
	}

	var cfg Config
	if err := yaml.Unmarshal(data, &cfg); err != nil {
		return nil, fmt.Errorf("erreur parsing YAML: %w", err)
	}

	// Valeurs par défaut
	if cfg.Settings.RefreshInterval == 0 {
		cfg.Settings.RefreshInterval = 30
	}
	if cfg.Settings.SSHTimeout == 0 {
		cfg.Settings.SSHTimeout = 10
	}
	// Seuils par défaut pour la conformité
	if cfg.Settings.Thresholds.DiskMinPercent == 0 {
		cfg.Settings.Thresholds.DiskMinPercent = 10 // Alerte si < 10% libre
	}
	if cfg.Settings.Thresholds.MemoryMinPercent == 0 {
		cfg.Settings.Thresholds.MemoryMinPercent = 5 // Alerte si < 5% libre
	}
	if cfg.Settings.Thresholds.CPUMaxPercent == 0 {
		cfg.Settings.Thresholds.CPUMaxPercent = 90 // Alerte si > 90%
	}

	// Valeurs par défaut pour les machines et déchiffrement des passwords
	for i := range cfg.Machines {
		if cfg.Machines[i].Port == 0 {
			cfg.Machines[i].Port = 22
		}

		// Déchiffrer le password s'il est chiffré
		if cfg.Machines[i].Password != "" {
			if crypto.IsEncrypted(cfg.Machines[i].Password) {
				decrypted, err := crypto.Decrypt(cfg.Machines[i].Password)
				if err != nil {
					log.Printf("AVERTISSEMENT: Impossible de déchiffrer le password pour %s: %v", cfg.Machines[i].ID, err)
					// On continue avec le password chiffré (échec de connexion probable)
				} else {
					cfg.Machines[i].Password = decrypted
				}
			}
		}
	}

	return &cfg, nil
}

// GetMachine retourne la configuration d'une machine par son ID
func (c *Config) GetMachine(id string) *MachineConfig {
	for i := range c.Machines {
		if c.Machines[i].ID == id {
			return &c.Machines[i]
		}
	}
	return nil
}

// SaveConfig sauvegarde la configuration dans un fichier YAML
func SaveConfig(path string, cfg *Config) error {
	data, err := yaml.Marshal(cfg)
	if err != nil {
		return fmt.Errorf("erreur sérialisation YAML: %w", err)
	}

	if err := os.WriteFile(path, data, 0644); err != nil {
		return fmt.Errorf("erreur écriture fichier config: %w", err)
	}

	return nil
}

// AddMachine ajoute une nouvelle machine à la configuration
func (c *Config) AddMachine(machine MachineConfig) error {
	// Vérifier si l'ID existe déjà
	if c.GetMachine(machine.ID) != nil {
		return fmt.Errorf("une machine avec l'ID '%s' existe déjà", machine.ID)
	}

	// Valeurs par défaut
	if machine.Port == 0 {
		machine.Port = 22
	}

	// Chiffrer le password s'il est en clair
	if machine.Password != "" && !crypto.IsEncrypted(machine.Password) {
		encrypted, err := crypto.Encrypt(machine.Password)
		if err != nil {
			log.Printf("AVERTISSEMENT: Impossible de chiffrer le password pour %s: %v", machine.ID, err)
			// On continue avec le password en clair (pas idéal mais fonctionnel)
		} else {
			machine.Password = encrypted
		}
	}

	c.Machines = append(c.Machines, machine)
	return nil
}

// UpdateMachine met à jour une machine existante
func (c *Config) UpdateMachine(machine MachineConfig) error {
	for i := range c.Machines {
		if c.Machines[i].ID == machine.ID {
			// Garder le port par défaut si 0
			if machine.Port == 0 {
				machine.Port = 22
			}

			// Chiffrer le password s'il est en clair (nouveau password)
			if machine.Password != "" && !crypto.IsEncrypted(machine.Password) {
				encrypted, err := crypto.Encrypt(machine.Password)
				if err != nil {
					log.Printf("AVERTISSEMENT: Impossible de chiffrer le password pour %s: %v", machine.ID, err)
				} else {
					machine.Password = encrypted
				}
			}

			c.Machines[i] = machine
			return nil
		}
	}
	return fmt.Errorf("machine non trouvée: %s", machine.ID)
}

// RemoveMachine supprime une machine de la configuration
func (c *Config) RemoveMachine(id string) error {
	for i := range c.Machines {
		if c.Machines[i].ID == id {
			c.Machines = append(c.Machines[:i], c.Machines[i+1:]...)
			return nil
		}
	}
	return fmt.Errorf("machine non trouvée: %s", id)
}
