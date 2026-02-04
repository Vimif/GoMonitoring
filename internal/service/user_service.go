package service

import (
	"fmt"
	"time"

	"go-monitoring/internal/domain"
	"go-monitoring/pkg/interfaces"

	"golang.org/x/crypto/bcrypt"
)

// UserService implÃ©mente la logique mÃ©tier pour les utilisateurs
type UserService struct {
	repo interfaces.UserRepository
}

// NewUserService crÃ©e un nouveau service de gestion des utilisateurs
func NewUserService(repo interfaces.UserRepository) *UserService {
	return &UserService{
		repo: repo,
	}
}

// Authenticate authentifie un utilisateur
func (s *UserService) Authenticate(username, password string) (*domain.User, error) {
	// RÃ©cupÃ©rer l'utilisateur
	user, err := s.repo.GetByUsername(username)
	if err != nil {
		return nil, fmt.Errorf("invalid credentials")
	}

	// VÃ©rifier si le compte est actif
	if !user.IsActive {
		return nil, fmt.Errorf("account is inactive")
	}

	// VÃ©rifier si le compte est verrouillÃ©
	if user.IsLocked() {
		return nil, fmt.Errorf("account is locked until %s", user.LockedUntil.Format("2006-01-02 15:04:05"))
	}

	// VÃ©rifier le mot de passe
	err = bcrypt.CompareHashAndPassword([]byte(user.Password), []byte(password))
	if err != nil {
		return nil, fmt.Errorf("invalid credentials")
	}

	return user, nil
}

// GetByUsername retourne un utilisateur
func (s *UserService) GetByUsername(username string) (*domain.User, error) {
	return s.repo.GetByUsername(username)
}

// Create crÃ©e un nouvel utilisateur
func (s *UserService) Create(username, password, role string) error {
	// Validation
	if username == "" {
		return fmt.Errorf("username is required")
	}

	if password == "" {
		return fmt.Errorf("password is required")
	}

	if len(password) < 8 {
		return fmt.Errorf("password must be at least 8 characters")
	}

	if role != "admin" && role != "viewer" {
		return fmt.Errorf("invalid role: must be 'admin' or 'viewer'")
	}

	// VÃ©rifier si l'utilisateur existe dÃ©jÃ 
	if _, err := s.repo.GetByUsername(username); err == nil {
		return fmt.Errorf("user already exists: %s", username)
	}

	// Hasher le mot de passe
	hashedPassword, err := bcrypt.GenerateFromPassword([]byte(password), bcrypt.DefaultCost)
	if err != nil {
		return fmt.Errorf("failed to hash password: %w", err)
	}

	// CrÃ©er l'utilisateur
	user := &domain.User{
		Username:    username,
		Password:    string(hashedPassword),
		Role:        role,
		IsActive:    true,
		LockedUntil: time.Unix(0, 0),
		CreatedAt:   time.Now(),
		UpdatedAt:   time.Now(),
	}

	return s.repo.Create(user)
}

// Update met Ã  jour un utilisateur
func (s *UserService) Update(user *domain.User) error {
	if user.Username == "" {
		return fmt.Errorf("username is required")
	}

	// VÃ©rifier que l'utilisateur existe
	if _, err := s.repo.GetByUsername(user.Username); err != nil {
		return fmt.Errorf("user not found: %s", user.Username)
	}

	user.UpdatedAt = time.Now()
	return s.repo.Update(user)
}

// Delete supprime un utilisateur
func (s *UserService) Delete(username string) error {
	if username == "" {
		return fmt.Errorf("username is required")
	}

	// Ne pas supprimer le dernier admin
	users, err := s.repo.GetAll()
	if err != nil {
		return fmt.Errorf("failed to check admin count: %w", err)
	}

	adminCount := 0
	var targetUser *domain.User
	for _, u := range users {
		if u.IsAdmin() {
			adminCount++
		}
		if u.Username == username {
			targetUser = &u
		}
	}

	if targetUser == nil {
		return fmt.Errorf("user not found: %s", username)
	}

	if targetUser.IsAdmin() && adminCount <= 1 {
		return fmt.Errorf("cannot delete the last admin user")
	}

	return s.repo.Delete(username)
}

// GetAll retourne tous les utilisateurs
func (s *UserService) GetAll() ([]domain.User, error) {
	return s.repo.GetAll()
}

// ChangePassword change le mot de passe d'un utilisateur
func (s *UserService) ChangePassword(username, oldPassword, newPassword string) error {
	// VÃ©rifier l'ancien mot de passe
	user, err := s.Authenticate(username, oldPassword)
	if err != nil {
		return fmt.Errorf("current password is incorrect")
	}

	// Valider le nouveau mot de passe
	if len(newPassword) < 8 {
		return fmt.Errorf("new password must be at least 8 characters")
	}

	// Hasher le nouveau mot de passe
	hashedPassword, err := bcrypt.GenerateFromPassword([]byte(newPassword), bcrypt.DefaultCost)
	if err != nil {
		return fmt.Errorf("failed to hash password: %w", err)
	}

	// Mettre Ã  jour
	user.Password = string(hashedPassword)
	user.UpdatedAt = time.Now()
	return s.repo.Update(user)
}

// LockAccount verrouille un compte
func (s *UserService) LockAccount(username string, duration time.Duration) error {
	if username == "" {
		return fmt.Errorf("username is required")
	}

	until := time.Now().Add(duration)
	return s.repo.LockAccount(username, until)
}

// UnlockAccount dÃ©verrouille un compte
func (s *UserService) UnlockAccount(username string) error {
	if username == "" {
		return fmt.Errorf("username is required")
	}

	return s.repo.UnlockAccount(username)
}
