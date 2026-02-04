package interfaces

import (
	"time"

	"go-monitoring/internal/domain"
)

// MonitoringService dÃ©finit l'interface pour les services de monitoring
type MonitoringService interface {
	// CollectMetrics collecte toutes les mÃ©triques pour une machine
	CollectMetrics(machineID string) (*domain.Machine, error)

	// GetMachineStatus retourne le statut actuel d'une machine
	GetMachineStatus(machineID string) (*domain.Machine, error)

	// GetAllMachinesStatus retourne le statut de toutes les machines
	GetAllMachinesStatus() ([]domain.Machine, error)

	// GetMetricHistory retourne l'historique des mÃ©triques
	GetMetricHistory(machineID string, duration time.Duration) ([]domain.MetricPoint, error)

	// StartMonitoring dÃ©marre le monitoring continu
	StartMonitoring(interval time.Duration)

	// StopMonitoring arrÃªte le monitoring
	StopMonitoring()
}

// MachineService dÃ©finit l'interface pour la gestion des machines
type MachineService interface {
	// GetAll retourne toutes les machines
	GetAll() ([]domain.Machine, error)

	// GetByID retourne une machine par son ID
	GetByID(id string) (*domain.Machine, error)

	// Create crÃ©e une nouvelle machine
	Create(machine *domain.Machine) error

	// Update met Ã  jour une machine
	Update(machine *domain.Machine) error

	// Delete supprime une machine
	Delete(id string) error

	// TestConnection teste la connexion SSH Ã  une machine
	TestConnection(machineID string) error

	// GroupByGroup regroupe les machines par groupe
	GroupByGroup() ([]domain.MachineGroup, error)
}

// UserService dÃ©finit l'interface pour la gestion des utilisateurs
type UserService interface {
	// Authenticate authentifie un utilisateur
	Authenticate(username, password string) (*domain.User, error)

	// GetByUsername retourne un utilisateur
	GetByUsername(username string) (*domain.User, error)

	// Create crÃ©e un nouvel utilisateur
	Create(username, password, role string) error

	// Update met Ã  jour un utilisateur
	Update(user *domain.User) error

	// Delete supprime un utilisateur
	Delete(username string) error

	// GetAll retourne tous les utilisateurs
	GetAll() ([]domain.User, error)

	// ChangePassword change le mot de passe d'un utilisateur
	ChangePassword(username, oldPassword, newPassword string) error

	// LockAccount verrouille un compte
	LockAccount(username string, duration time.Duration) error

	// UnlockAccount dÃ©verrouille un compte
	UnlockAccount(username string) error
}

// AuditService dÃ©finit l'interface pour le logging d'audit
type AuditService interface {
	// Log enregistre une action
	Log(username, action, target, status, ipAddress string) error

	// GetRecentLogs retourne les logs rÃ©cents
	GetRecentLogs(limit int) ([]domain.AuditLog, error)

	// GetLogsByUser retourne les logs d'un utilisateur
	GetLogsByUser(username string, limit int) ([]domain.AuditLog, error)

	// CleanOldLogs supprime les logs anciens
	CleanOldLogs(olderThan time.Duration) error
}
