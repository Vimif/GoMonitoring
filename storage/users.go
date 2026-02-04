package storage

import (
	"database/sql"
	"errors"
	"time"
)

// -- User Management Methods --

// GetUser rÃ©cupÃ¨re un utilisateur par son username
func (db *DB) GetUser(username string) (*UserDB, error) {
	query := `SELECT username, password_hash, role, is_active, created_at, failed_attempts, locked_until 
			  FROM users WHERE username = ?`

	row := db.QueryRow(query, username)

	var u UserDB
	var lockedUntil sql.NullTime

	err := row.Scan(&u.Username, &u.PasswordHash, &u.Role, &u.IsActive, &u.CreatedAt, &u.FailedAttempts, &lockedUntil)
	if err != nil {
		return nil, err
	}

	if lockedUntil.Valid {
		u.LockedUntil = lockedUntil.Time
	}

	return &u, nil
}

// GetAllUsers rÃ©cupÃ¨re tous les utilisateurs
func (db *DB) GetAllUsers() ([]UserDB, error) {
	query := `SELECT username, role, is_active, created_at, failed_attempts, locked_until FROM users ORDER BY username ASC`

	rows, err := db.Query(query)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var users []UserDB
	for rows.Next() {
		var u UserDB
		var lockedUntil sql.NullTime

		if err := rows.Scan(&u.Username, &u.Role, &u.IsActive, &u.CreatedAt, &u.FailedAttempts, &lockedUntil); err != nil {
			continue
		}

		if lockedUntil.Valid {
			u.LockedUntil = lockedUntil.Time
		}
		users = append(users, u)
	}
	return users, nil
}

// CreateUser crÃ©e un nouvel utilisateur
func (db *DB) CreateUser(username, passwordHash, role string) error {
	query := `INSERT INTO users (username, password_hash, role, is_active, created_at) VALUES (?, ?, ?, 1, ?)`
	_, err := db.Exec(query, username, passwordHash, role, time.Now())
	return err
}

// DeleteUser supprime un utilisateur
func (db *DB) DeleteUser(username string) error {
	if username == "admin" {
		return errors.New("impossible de supprimer l'admin")
	}
	res, err := db.Exec("DELETE FROM users WHERE username = ?", username)
	if err != nil {
		return err
	}
	rows, _ := res.RowsAffected()
	if rows == 0 {
		return errors.New("utilisateur introuvable")
	}
	return nil
}

// UpdatePassword met Ã  jour le mot de passe
func (db *DB) UpdatePassword(username, passwordHash string) error {
	_, err := db.Exec("UPDATE users SET password_hash = ? WHERE username = ?", passwordHash, username)
	return err
}

// UpdateUserRole met Ã  jour le rÃ´le
func (db *DB) UpdateUserRole(username, role string) error {
	if username == "admin" {
		return errors.New("impossible de modifier le rÃ´le de l'admin principal")
	}
	_, err := db.Exec("UPDATE users SET role = ? WHERE username = ?", role, username)
	return err
}

// ToggleUserStatus active ou dÃ©sactive un utilisateur
func (db *DB) ToggleUserStatus(username string, active bool) error {
	if username == "admin" && !active {
		return errors.New("impossible de dÃ©sactiver l'admin")
	}
	_, err := db.Exec("UPDATE users SET is_active = ? WHERE username = ?", active, username)
	return err
}

// RecordLoginAttempt gÃ¨re les tentatives de connexion (lockout)
// Retourne (locked, err)
func (db *DB) RecordLoginAttempt(username string, success bool) (bool, error) {
	if success {
		// Reset counters
		_, err := db.Exec("UPDATE users SET failed_attempts = 0, locked_until = NULL WHERE username = ?", username)
		return false, err
	}

	// Increment failed attempts
	// Load current state first
	u, err := db.GetUser(username)
	if err != nil {
		return false, err // User not found essentially
	}

	// Check if already locked
	if !u.LockedUntil.IsZero() && u.LockedUntil.After(time.Now()) {
		return true, nil
	}

	newAttempts := u.FailedAttempts + 1
	var lockedUntil time.Time

	if newAttempts >= 5 {
		lockedUntil = time.Now().Add(15 * time.Minute) // Lock for 15 mins
	}

	query := `UPDATE users SET failed_attempts = ?, locked_until = ? WHERE username = ?`

	// Handle NULL for locked_until
	var lockedUntilVal interface{}
	if lockedUntil.IsZero() {
		lockedUntilVal = nil
	} else {
		lockedUntilVal = lockedUntil
	}

	_, err = db.Exec(query, newAttempts, lockedUntilVal, username)

	return !lockedUntil.IsZero(), err
}

// UnlockUser dÃ©verrouille un utilisateur manuellement
func (db *DB) UnlockUser(username string) error {
	_, err := db.Exec("UPDATE users SET failed_attempts = 0, locked_until = NULL WHERE username = ?", username)
	return err
}
