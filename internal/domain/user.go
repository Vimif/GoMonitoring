package domain

import "time"

// User reprÃ©sente un utilisateur du systÃ¨me
type User struct {
	Username    string
	Password    string // Hash bcrypt
	Role        string // "admin" ou "viewer"
	IsActive    bool
	LockedUntil time.Time
	CreatedAt   time.Time
	UpdatedAt   time.Time
}

// IsAdmin retourne true si l'utilisateur est administrateur
func (u *User) IsAdmin() bool {
	return u.Role == "admin"
}

// IsViewer retourne true si l'utilisateur est viewer
func (u *User) IsViewer() bool {
	return u.Role == "viewer"
}

// IsLocked retourne true si le compte est verrouillÃ©
func (u *User) IsLocked() bool {
	return u.LockedUntil.After(time.Now())
}

// CanModify retourne true si l'utilisateur peut modifier la configuration
func (u *User) CanModify() bool {
	return u.IsAdmin() && u.IsActive && !u.IsLocked()
}
