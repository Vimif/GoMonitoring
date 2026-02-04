package repository

import (
	"fmt"
	"sync"

	"go-monitoring/config"
	"go-monitoring/internal/domain"
	"go-monitoring/pkg/crypto"
)

// MachineRepository implÃ©mente l'interface MachineRepository avec config.yaml
type MachineRepository struct {
	configPath string
	cfg        *config.Config
	mu         sync.RWMutex
}

// NewMachineRepository crÃ©e un nouveau repository pour les machines
func NewMachineRepository(configPath string) (*MachineRepository, error) {
	cfg, err := config.LoadConfig(configPath)
	if err != nil {
		return nil, fmt.Errorf("failed to load config: %w", err)
	}

	return &MachineRepository{
		configPath: configPath,
		cfg:        cfg,
	}, nil
}

// GetAll retourne toutes les machines
func (r *MachineRepository) GetAll() ([]domain.Machine, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()

	machines := make([]domain.Machine, 0, len(r.cfg.Machines))
	for _, m := range r.cfg.Machines {
		machines = append(machines, configToMachine(m))
	}

	return machines, nil
}

// GetByID retourne une machine par son ID
func (r *MachineRepository) GetByID(id string) (*domain.Machine, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()

	for _, m := range r.cfg.Machines {
		if m.ID == id {
			machine := configToMachine(m)
			return &machine, nil
		}
	}

	return nil, fmt.Errorf("machine not found: %s", id)
}

// Create crÃ©e une nouvelle machine
func (r *MachineRepository) Create(machine *domain.Machine) error {
	r.mu.Lock()
	defer r.mu.Unlock()

	// VÃ©rifier si la machine existe dÃ©jÃ 
	for _, m := range r.cfg.Machines {
		if m.ID == machine.ID {
			return fmt.Errorf("machine already exists: %s", machine.ID)
		}
	}

	// Convertir domain -> config
	configMachine := machineToConfig(machine)

	// Chiffrer le mot de passe si nÃ©cessaire
	if configMachine.Password != "" && !crypto.IsEncrypted(configMachine.Password) {
		encrypted, err := crypto.Encrypt(configMachine.Password)
		if err != nil {
			return fmt.Errorf("failed to encrypt password: %w", err)
		}
		configMachine.Password = encrypted
	}

	// Ajouter Ã  la config
	r.cfg.Machines = append(r.cfg.Machines, configMachine)

	// Sauvegarder
	return config.SaveConfig(r.configPath, r.cfg)
}

// Update met Ã  jour une machine existante
func (r *MachineRepository) Update(machine *domain.Machine) error {
	r.mu.Lock()
	defer r.mu.Unlock()

	// Trouver l'index de la machine
	index := -1
	for i, m := range r.cfg.Machines {
		if m.ID == machine.ID {
			index = i
			break
		}
	}

	if index == -1 {
		return fmt.Errorf("machine not found: %s", machine.ID)
	}

	// Convertir domain -> config
	configMachine := machineToConfig(machine)

	// Chiffrer le mot de passe si nÃ©cessaire
	if configMachine.Password != "" && !crypto.IsEncrypted(configMachine.Password) {
		encrypted, err := crypto.Encrypt(configMachine.Password)
		if err != nil {
			return fmt.Errorf("failed to encrypt password: %w", err)
		}
		configMachine.Password = encrypted
	}

	// Mettre Ã  jour
	r.cfg.Machines[index] = configMachine

	// Sauvegarder
	return config.SaveConfig(r.configPath, r.cfg)
}

// Delete supprime une machine
func (r *MachineRepository) Delete(id string) error {
	r.mu.Lock()
	defer r.mu.Unlock()

	// Trouver l'index de la machine
	index := -1
	for i, m := range r.cfg.Machines {
		if m.ID == id {
			index = i
			break
		}
	}

	if index == -1 {
		return fmt.Errorf("machine not found: %s", id)
	}

	// Supprimer
	r.cfg.Machines = append(r.cfg.Machines[:index], r.cfg.Machines[index+1:]...)

	// Sauvegarder
	return config.SaveConfig(r.configPath, r.cfg)
}

// Exists vÃ©rifie si une machine existe
func (r *MachineRepository) Exists(id string) bool {
	r.mu.RLock()
	defer r.mu.RUnlock()

	for _, m := range r.cfg.Machines {
		if m.ID == id {
			return true
		}
	}

	return false
}

// Helper functions pour convertir entre domain et config

func configToMachine(cfg config.MachineConfig) domain.Machine {
	return domain.Machine{
		ID:      cfg.ID,
		Name:    cfg.Name,
		Host:    cfg.Host,
		Port:    cfg.Port,
		User:    cfg.User,
		KeyPath: cfg.KeyPath,
		Group:   cfg.Group,
		OSType:  cfg.OS,
	}
}

func machineToConfig(m *domain.Machine) config.MachineConfig {
	return config.MachineConfig{
		ID:      m.ID,
		Name:    m.Name,
		Host:    m.Host,
		Port:    m.Port,
		User:    m.User,
		KeyPath: m.KeyPath,
		Group:   m.Group,
		OS:      m.OSType,
	}
}
