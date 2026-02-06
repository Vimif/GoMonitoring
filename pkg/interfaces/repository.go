package interfaces

import (
	"time"

	"go-monitoring/internal/domain"
)

// MachineRepository définit l'interface pour la persistance des machines
type MachineRepository interface {
	// GetAll retourne toutes les machines configurées
	GetAll() ([]domain.Machine, error)

	// GetByID retourne une machine par son ID
	GetByID(id string) (*domain.Machine, error)

	// Create crée une nouvelle machine
	Create(machine *domain.Machine) error

	// Update met à jour une machine existante
	Update(machine *domain.Machine) error

	// Delete supprime une machine
	Delete(id string) error

	// Exists vérifie si une machine existe
	Exists(id string) bool
}

// MetricRepository définit l'interface pour la persistance des métriques
type MetricRepository interface {
	// Save sauvegarde un point de métrique
	Save(machineID string, metric *domain.MetricPoint) error

	// GetHistory retourne l'historique des métriques pour une période
	GetHistory(machineID string, duration time.Duration) ([]domain.MetricPoint, error)

	// GetLatest retourne la dernière métrique enregistrée
	GetLatest(machineID string) (*domain.MetricPoint, error)

	// DeleteOlderThan supprime les métriques plus anciennes que la durée spécifiée
	DeleteOlderThan(duration time.Duration) error

	// GetStorageSize retourne la taille de stockage utilisée
	GetStorageSize() (int64, error)
}

// UserRepository définit l'interface pour la persistance des utilisateurs
type UserRepository interface {
	// GetByUsername retourne un utilisateur par son nom
	GetByUsername(username string) (*domain.User, error)

	// Create crée un nouvel utilisateur
	Create(user *domain.User) error

	// Update met à jour un utilisateur existant
	Update(user *domain.User) error

	// Delete supprime un utilisateur
	Delete(username string) error

	// GetAll retourne tous les utilisateurs
	GetAll() ([]domain.User, error)

	// LockAccount verrouille un compte jusqu'à une date
	LockAccount(username string, until time.Time) error

	// UnlockAccount déverrouille un compte
	UnlockAccount(username string) error

	// UpdatePassword met à jour le mot de passe d'un utilisateur
	UpdatePassword(username, passwordHash string) error
}

// AuditRepository définit l'interface pour la persistance des logs d'audit
type AuditRepository interface {
	// Log enregistre une action dans le journal d'audit
	Log(log *domain.AuditLog) error

	// GetLogs retourne les logs d'audit pour une période
	GetLogs(since time.Time, limit int) ([]domain.AuditLog, error)

	// GetLogsByUser retourne les logs pour un utilisateur
	GetLogsByUser(username string, limit int) ([]domain.AuditLog, error)

	// DeleteOlderThan supprime les logs plus anciens que la durée spécifiée
	DeleteOlderThan(duration time.Duration) error
}

// ConfigRepository définit l'interface pour la persistance de la configuration
type ConfigRepository interface {
	// Load charge la configuration depuis le stockage
	Load() error

	// Save sauvegarde la configuration
	Save() error

	// GetMachines retourne toutes les machines configurées
	GetMachines() ([]domain.Machine, error)

	// AddMachine ajoute une machine à la configuration
	AddMachine(machine *domain.Machine) error

	// UpdateMachine met à jour une machine dans la configuration
	UpdateMachine(machine *domain.Machine) error

	// DeleteMachine supprime une machine de la configuration
	DeleteMachine(id string) error
}
