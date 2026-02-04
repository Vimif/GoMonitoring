package service

import (
	"fmt"

	"go-monitoring/internal/domain"
	"go-monitoring/pkg/interfaces"
	"go-monitoring/ssh"
)

// MachineService implÃ©mente la logique mÃ©tier pour les machines
type MachineService struct {
	repo    interfaces.MachineRepository
	sshPool *ssh.Pool
}

// NewMachineService crÃ©e un nouveau service de gestion des machines
func NewMachineService(repo interfaces.MachineRepository, sshPool *ssh.Pool) *MachineService {
	return &MachineService{
		repo:    repo,
		sshPool: sshPool,
	}
}

// GetAll retourne toutes les machines
func (s *MachineService) GetAll() ([]domain.Machine, error) {
	return s.repo.GetAll()
}

// GetByID retourne une machine par son ID
func (s *MachineService) GetByID(id string) (*domain.Machine, error) {
	return s.repo.GetByID(id)
}

// Create crÃ©e une nouvelle machine
func (s *MachineService) Create(machine *domain.Machine) error {
	// Validation mÃ©tier
	if machine.ID == "" {
		return fmt.Errorf("machine ID is required")
	}

	if machine.Name == "" {
		return fmt.Errorf("machine name is required")
	}

	if machine.Host == "" {
		return fmt.Errorf("machine host is required")
	}

	if machine.Port <= 0 || machine.Port > 65535 {
		return fmt.Errorf("invalid port number: %d", machine.Port)
	}

	if machine.User == "" {
		return fmt.Errorf("machine user is required")
	}

	// VÃ©rifier si la machine existe dÃ©jÃ 
	if s.repo.Exists(machine.ID) {
		return fmt.Errorf("machine already exists: %s", machine.ID)
	}

	// DÃ©finir les valeurs par dÃ©faut
	if machine.Port == 0 {
		machine.Port = 22
	}

	if machine.Group == "" {
		machine.Group = "Default"
	}

	if machine.OSType == "" {
		machine.OSType = "linux"
	}

	// CrÃ©er la machine
	return s.repo.Create(machine)
}

// Update met Ã  jour une machine
func (s *MachineService) Update(machine *domain.Machine) error {
	// Validation mÃ©tier
	if machine.ID == "" {
		return fmt.Errorf("machine ID is required")
	}

	// VÃ©rifier que la machine existe
	if !s.repo.Exists(machine.ID) {
		return fmt.Errorf("machine not found: %s", machine.ID)
	}

	// Mettre Ã  jour
	return s.repo.Update(machine)
}

// Delete supprime une machine
func (s *MachineService) Delete(id string) error {
	if id == "" {
		return fmt.Errorf("machine ID is required")
	}

	// VÃ©rifier que la machine existe
	if !s.repo.Exists(id) {
		return fmt.Errorf("machine not found: %s", id)
	}

	// Supprimer
	return s.repo.Delete(id)
}

// TestConnection teste la connexion SSH Ã  une machine
func (s *MachineService) TestConnection(machineID string) error {
	// RÃ©cupÃ©rer le client SSH depuis le pool
	client, err := s.sshPool.GetClient(machineID)
	if err != nil {
		return fmt.Errorf("failed to get SSH client: %w", err)
	}

	// Tenter de se connecter
	if err := client.Connect(); err != nil {
		return fmt.Errorf("connection failed: %w", err)
	}

	// ExÃ©cuter une commande simple pour vÃ©rifier
	_, err = client.Execute("echo test")
	if err != nil {
		return fmt.Errorf("command execution failed: %w", err)
	}

	return nil
}

// GroupByGroup regroupe les machines par groupe
func (s *MachineService) GroupByGroup() ([]domain.MachineGroup, error) {
	machines, err := s.repo.GetAll()
	if err != nil {
		return nil, err
	}

	// Regrouper par groupe
	groups := make(map[string][]domain.Machine)
	for _, machine := range machines {
		group := machine.Group
		if group == "" {
			group = "Default"
		}
		groups[group] = append(groups[group], machine)
	}

	// Convertir en slice
	var result []domain.MachineGroup
	for name, machines := range groups {
		result = append(result, domain.MachineGroup{
			Name:     name,
			Machines: machines,
		})
	}

	return result, nil
}
