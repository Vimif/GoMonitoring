package interfaces

import (
	"time"

	"go-monitoring/internal/domain"
)

// MachineRepository dÃ©finit l'interface pour la persistance des machines
type MachineRepository interface {
	// GetAll retourne toutes les machines configurÃ©es
	GetAll() ([]domain.Machine, error)

	// GetByID retourne une machine par son ID
	GetByID(id string) (*domain.Machine, error)

	// Create crÃ©e une nouvelle machine
	Create(machine *domain.Machine) error

	// Update met Ã  jour une machine existante
	Update(machine *domain.Machine) error

	// Delete supprime une machine
	Delete(id string) error

	// Exists vÃ©rifie si une machine existe
	Exists(id string) bool
}

// MetricRepository dÃ©finit l'interface pour la persistance des mÃ©triques
type MetricRepository interface {
	// Save sauvegarde un point de mÃ©trique
	Save(machineID string, metric *domain.MetricPoint) error

	// GetHistory retourne l'historique des mÃ©triques pour une pÃ©riode
	GetHistory(machineID string, duration time.Duration) ([]domain.MetricPoint, error)

	// GetLatest retourne la derniÃ¨re mÃ©trique enregistrÃ©e
	GetLatest(machineID string) (*domain.MetricPoint, error)

	// DeleteOlderThan supprime les mÃ©triques plus anciennes que la durÃ©e spÃ©cifiÃ©e
	DeleteOlderThan(duration time.Duration) error

	// GetStorageSize retourne la taille de stockage utilisÃ©e
	GetStorageSize() (int64, error)
}

// UserRepository dÃ©finit l'interface pour la persistance des utilisateurs
type UserRepository interface {
	// GetByUsername retourne un utilisateur par son nom
	GetByUsername(username string) (*domain.User, error)

	// Create crÃ©e un nouvel utilisateur
	Create(user *domain.User) error

	// Update met Ã  jour un utilisateur existant
	Update(user *domain.User) error

	// Delete supprime un utilisateur
	Delete(username string) error

	// GetAll retourne tous les utilisateurs
	GetAll() ([]domain.User, error)

	// LockAccount verrouille un compte jusqu'Ã  une date
	LockAccount(username string, until time.Time) error

	// UnlockAccount dÃ©verrouille un compte
	UnlockAccount(username string) error

	// UpdatePassword met Ã  jour le mot de passe d'un utilisateur
	UpdatePassword(username, passwordHash string) error
}

// AuditRepository dÃ©finit l'interface pour la persistance des logs d'audit
type AuditRepository interface {
	// Log enregistre une action dans le journal d'audit
	Log(log *domain.AuditLog) error

	// GetLogs retourne les logs d'audit pour une pÃ©riode
	GetLogs(since time.Time, limit int) ([]domain.AuditLog, error)

	// GetLogsByUser retourne les logs pour un utilisateur
	GetLogsByUser(username string, limit int) ([]domain.AuditLog, error)

	// DeleteOlderThan supprime les logs plus anciens que la durÃ©e spÃ©cifiÃ©e
	DeleteOlderThan(duration time.Duration) error
}

// ConfigRepository dÃ©finit l'interface pour la persistance de la configuration
type ConfigRepository interface {
	// Load charge la configuration depuis le stockage
	Load() error

	// Save sauvegarde la configuration
	Save() error

	// GetMachines retourne toutes les machines configurÃ©es
	GetMachines() ([]domain.Machine, error)

	// AddMachine ajoute une machine Ã  la configuration
	AddMachine(machine *domain.Machine) error

	// UpdateMachine met Ã  jour une machine dans la configuration
	UpdateMachine(machine *domain.Machine) error

	// DeleteMachine supprime une machine de la configuration
	DeleteMachine(id string) error
}
