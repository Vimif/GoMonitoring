package repository

import (
	"database/sql"
	"fmt"
	"time"

	"go-monitoring/internal/domain"
)

// UserRepository implÃ©mente l'interface UserRepository avec SQLite
type UserRepository struct {
	db *sql.DB
}

// NewUserRepository crÃ©e un nouveau repository pour les utilisateurs
func NewUserRepository(db *sql.DB) *UserRepository {
	return &UserRepository{
		db: db,
	}
}

// GetByUsername retourne un utilisateur par son nom
func (r *UserRepository) GetByUsername(username string) (*domain.User, error) {
	query := `
		SELECT username, password, role, is_active, locked_until
		FROM users
		WHERE username = ?
	`

	var user domain.User
	var lockedUntil int64

	err := r.db.QueryRow(query, username).Scan(
		&user.Username,
		&user.Password,
		&user.Role,
		&user.IsActive,
		&lockedUntil,
	)

	if err == sql.ErrNoRows {
		return nil, fmt.Errorf("user not found: %s", username)
	}

	if err != nil {
		return nil, fmt.Errorf("failed to get user: %w", err)
	}

	user.LockedUntil = time.Unix(lockedUntil, 0)
	return &user, nil
}

// Create crÃ©e un nouvel utilisateur
func (r *UserRepository) Create(user *domain.User) error {
	query := `
		INSERT INTO users (username, password, role, is_active, locked_until)
		VALUES (?, ?, ?, ?, ?)
	`

	_, err := r.db.Exec(query,
		user.Username,
		user.Password,
		user.Role,
		user.IsActive,
		user.LockedUntil.Unix(),
	)

	if err != nil {
		return fmt.Errorf("failed to create user: %w", err)
	}

	return nil
}

// Update met Ã  jour un utilisateur existant
func (r *UserRepository) Update(user *domain.User) error {
	query := `
		UPDATE users
		SET password = ?, role = ?, is_active = ?, locked_until = ?
		WHERE username = ?
	`

	result, err := r.db.Exec(query,
		user.Password,
		user.Role,
		user.IsActive,
		user.LockedUntil.Unix(),
		user.Username,
	)

	if err != nil {
		return fmt.Errorf("failed to update user: %w", err)
	}

	rowsAffected, _ := result.RowsAffected()
	if rowsAffected == 0 {
		return fmt.Errorf("user not found: %s", user.Username)
	}

	return nil
}

// Delete supprime un utilisateur
func (r *UserRepository) Delete(username string) error {
	query := `DELETE FROM users WHERE username = ?`

	result, err := r.db.Exec(query, username)
	if err != nil {
		return fmt.Errorf("failed to delete user: %w", err)
	}

	rowsAffected, _ := result.RowsAffected()
	if rowsAffected == 0 {
		return fmt.Errorf("user not found: %s", username)
	}

	return nil
}

// GetAll retourne tous les utilisateurs
func (r *UserRepository) GetAll() ([]domain.User, error) {
	query := `
		SELECT username, password, role, is_active, locked_until
		FROM users
		ORDER BY username
	`

	rows, err := r.db.Query(query)
	if err != nil {
		return nil, fmt.Errorf("failed to query users: %w", err)
	}
	defer rows.Close()

	var users []domain.User

	for rows.Next() {
		var user domain.User
		var lockedUntil int64

		err := rows.Scan(
			&user.Username,
			&user.Password,
			&user.Role,
			&user.IsActive,
			&lockedUntil,
		)

		if err != nil {
			return nil, fmt.Errorf("failed to scan user: %w", err)
		}

		user.LockedUntil = time.Unix(lockedUntil, 0)
		users = append(users, user)
	}

	if err = rows.Err(); err != nil {
		return nil, fmt.Errorf("row iteration error: %w", err)
	}

	return users, nil
}

// LockAccount verrouille un compte jusqu'Ã  une date
func (r *UserRepository) LockAccount(username string, until time.Time) error {
	query := `UPDATE users SET locked_until = ? WHERE username = ?`

	result, err := r.db.Exec(query, until.Unix(), username)
	if err != nil {
		return fmt.Errorf("failed to lock account: %w", err)
	}

	rowsAffected, _ := result.RowsAffected()
	if rowsAffected == 0 {
		return fmt.Errorf("user not found: %s", username)
	}

	return nil
}

// UnlockAccount dÃ©verrouille un compte
func (r *UserRepository) UnlockAccount(username string) error {
	query := `UPDATE users SET locked_until = 0 WHERE username = ?`

	result, err := r.db.Exec(query, username)
	if err != nil {
		return fmt.Errorf("failed to unlock account: %w", err)
	}

	rowsAffected, _ := result.RowsAffected()
	if rowsAffected == 0 {
		return fmt.Errorf("user not found: %s", username)
	}

	return nil
}

// UpdatePassword met Ã  jour le mot de passe d'un utilisateur
func (r *UserRepository) UpdatePassword(username, passwordHash string) error {
	query := `UPDATE users SET password = ? WHERE username = ?`

	result, err := r.db.Exec(query, passwordHash, username)
	if err != nil {
		return fmt.Errorf("failed to update password: %w", err)
	}

	rowsAffected, _ := result.RowsAffected()
	if rowsAffected == 0 {
		return fmt.Errorf("user not found: %s", username)
	}

	return nil
}
