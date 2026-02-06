package auth

import (
	"errors"
	"go-monitoring/config"
	"go-monitoring/storage"
	"log"
	"time"

	"golang.org/x/crypto/bcrypt"
)

// User représente un utilisateur authentifié (Business Object)
type User struct {
	Username    string
	Role        string
	IsActive    bool
	LockedUntil time.Time
}

// UserManager gère l'authentification des utilisateurs
type UserManager struct {
	db *storage.DB
}

// NewUserManager crée un gestionnaire d'utilisateur
// Il migre les utilisateurs de la config vers la DB si nécessaire
func NewUserManager(db *storage.DB, cfgUsers []config.UserConfig) *UserManager {
	um := &UserManager{
		db: db,
	}

	// Migration initiale : Si DB vide ou user manquant, on ajoute ceux de la config
	for _, u := range cfgUsers {
		_, err := db.GetUser(u.Username)
		if err != nil {
			// User n'existe pas, on le crée
			var hash string
			if len(u.Password) >= 60 && u.Password[:4] == "$2a$" {
				hash = u.Password
			} else {
				hashedBytes, _ := bcrypt.GenerateFromPassword([]byte(u.Password), bcrypt.DefaultCost)
				hash = string(hashedBytes)
			}

			log.Printf("Migration user config vers DB: %s", u.Username)
			db.CreateUser(u.Username, hash, u.Role)
		}
	}

	// Check if admin exists
	if _, err := db.GetUser("admin"); err != nil {
		log.Println("Création du compte admin par défaut")
		hash, _ := bcrypt.GenerateFromPassword([]byte("admin"), bcrypt.DefaultCost)
		db.CreateUser("admin", string(hash), "admin")
	}

	return um
}

// Authenticate vérifie le couple username/password avec gestion de lockout
func (um *UserManager) Authenticate(username, password string) (*User, error) {
	userDB, err := um.db.GetUser(username)
	if err != nil {
		return nil, errors.New("utilisateur ou mot de passe incorrect")
	}

	// 1. Check Active
	if !userDB.IsActive {
		return nil, errors.New("ce compte est désactivé")
	}

	// 2. Check Lockout
	if !userDB.LockedUntil.IsZero() && userDB.LockedUntil.After(time.Now()) {
		wait := time.Until(userDB.LockedUntil).Round(time.Minute)
		return nil, errors.New("compte verrouillé pour encore " + wait.String())
	}

	// 3. Verify Password
	err = bcrypt.CompareHashAndPassword([]byte(userDB.PasswordHash), []byte(password))

	// 4. Record Attempt
	locked, recordErr := um.db.RecordLoginAttempt(username, err == nil)
	if recordErr != nil {
		log.Printf("Erreur recording login attempt: %v", recordErr)
	}

	if locked {
		return nil, errors.New("compte verrouillé suite à trop d'échecs")
	}

	if err != nil {
		return nil, errors.New("utilisateur ou mot de passe incorrect")
	}

	return &User{
		Username:    userDB.Username,
		Role:        userDB.Role,
		IsActive:    userDB.IsActive,
		LockedUntil: userDB.LockedUntil,
	}, nil
}

// AddUser ajoute un nouvel utilisateur
func (um *UserManager) AddUser(username, password, role string) error {
	// Check exist
	if _, err := um.db.GetUser(username); err == nil {
		return errors.New("cet utilisateur existe déjà")
	}

	hash, err := bcrypt.GenerateFromPassword([]byte(password), bcrypt.DefaultCost)
	if err != nil {
		return err
	}

	return um.db.CreateUser(username, string(hash), role)
}

// UpdateUser met à jour le rôle (NON IMPLEMENTÉ en DB pour l'instant, on n'a pas methods UpdateUserRole)
// Pour simplifier on ne change pas le role pour l'instant via UI, on le fera plus tard si besoin.
// On garde la signature pour compatibilité si elle était utilisée, mais ici on va la supprimer ou l'adapter.
// D'après users_api.go, UpdateUser n'est pas utilisé, seulement Create, Delete, UpdatePassword.
// Mais on a ToggleUserStatus.

// ToggleUserStatus active/désactive
func (um *UserManager) ToggleUserStatus(username string, active bool) error {
	return um.db.ToggleUserStatus(username, active)
}

// UnlockUser débloque un compte
func (um *UserManager) UnlockUser(username string) error {
	return um.db.UnlockUser(username)
}

// UpdatePassword modifie le mot de passe
func (um *UserManager) UpdatePassword(username, newPassword string) error {
	_, err := um.db.GetUser(username)
	if err != nil {
		return errors.New("utilisateur introuvable")
	}

	hash, err := bcrypt.GenerateFromPassword([]byte(newPassword), bcrypt.DefaultCost)
	if err != nil {
		return err
	}

	return um.db.UpdatePassword(username, string(hash))
}

// UpdateUserRole modifie le rôle
func (um *UserManager) UpdateUserRole(username, role string) error {
	return um.db.UpdateUserRole(username, role)
}

// DeleteUser supprime un utilisateur
func (um *UserManager) DeleteUser(username string) error {
	return um.db.DeleteUser(username)
}

// GetAllUsers retourne la liste des utilisateurs
func (um *UserManager) GetAllUsers() []User {
	usersDB, err := um.db.GetAllUsers()
	if err != nil {
		log.Printf("Erreur GetAllUsers: %v", err)
		return []User{}
	}

	var users []User
	for _, u := range usersDB {
		users = append(users, User{
			Username:    u.Username,
			Role:        u.Role,
			IsActive:    u.IsActive,
			LockedUntil: u.LockedUntil,
		})
	}
	return users
}

// GetUserRole retourne le rôle d'un utilisateur
func (um *UserManager) GetUserRole(username string) string {
	u, err := um.db.GetUser(username)
	if err != nil {
		return ""
	}
	return u.Role
}

// IsAdmin vérifie si admin
func (um *UserManager) IsAdmin(username string) bool {
	role := um.GetUserRole(username)
	return role == "admin"
}
