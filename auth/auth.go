package auth

import (
	"crypto/rand"
	"encoding/base64"
	"html/template"
	"net/http"
	"time"
)

// Session structure pour stocker les sessions actives
type Session struct {
	Username string
	Expiry   time.Time
}

// AuthManager gÃ¨re les sessions et middlewares
type AuthManager struct {
	UserManager *UserManager
	sessions    map[string]Session
}

func NewAuthManager(um *UserManager) *AuthManager {
	return &AuthManager{
		UserManager: um,
		sessions:    make(map[string]Session),
	}
}

// Middleware protÃ¨ge les routes
func (am *AuthManager) Middleware(next http.HandlerFunc) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		// Ignorer les fichiers statiques (images, css, js) si nÃ©cessaire
		// Mais ici les statiques sont souvent publics.
		// Si on veut protÃ©ger, on applique middleware.

		cookie, err := r.Cookie("session_token")
		if err != nil {
			http.Redirect(w, r, "/login", http.StatusSeeOther)
			return
		}

		sessionToken := cookie.Value
		session, exists := am.sessions[sessionToken]
		if !exists {
			http.Redirect(w, r, "/login", http.StatusSeeOther)
			return
		}

		if session.Expiry.Before(time.Now()) {
			delete(am.sessions, sessionToken)
			http.Redirect(w, r, "/login", http.StatusSeeOther)
			return
		}

		// Tout est bon, on continue
		next(w, r)
	}
}

// LoginHandler gÃ¨re la page de connexion
func (am *AuthManager) LoginHandler(w http.ResponseWriter, r *http.Request) {
	if r.Method == "GET" {
		tmpl, err := template.ParseFiles("templates/login.html")
		if err != nil {
			http.Error(w, "Erreur template login", http.StatusInternalServerError)
			return
		}
		tmpl.Execute(w, nil)
		return
	}

	// POST Login
	username := r.FormValue("username")
	password := r.FormValue("password")

	user, err := am.UserManager.Authenticate(username, password)
	if err != nil {
		tmpl, _ := template.ParseFiles("templates/login.html")
		tmpl.Execute(w, map[string]string{"Error": "Identifiants incorrects"})
		return
	}

	// CrÃ©er session
	sessionToken := generateToken()
	expiresAt := time.Now().Add(24 * time.Hour)

	am.sessions[sessionToken] = Session{
		Username: user.Username,
		Expiry:   expiresAt,
	}

	http.SetCookie(w, &http.Cookie{
		Name:     "session_token",
		Value:    sessionToken,
		Expires:  expiresAt,
		Path:     "/",
		HttpOnly: true,
	})

	http.Redirect(w, r, "/", http.StatusSeeOther)
}

// LogoutHandler gÃ¨re la dÃ©connexion
func (am *AuthManager) LogoutHandler(w http.ResponseWriter, r *http.Request) {
	cookie, err := r.Cookie("session_token")
	if err == nil {
		delete(am.sessions, cookie.Value)
	}

	http.SetCookie(w, &http.Cookie{
		Name:    "session_token",
		Value:   "",
		Expires: time.Now(),
		Path:    "/",
	})

	http.Redirect(w, r, "/login", http.StatusSeeOther)
}

// IsAdminSession vÃ©rifie si la session appartient Ã  un admin
func (am *AuthManager) IsAdminSession(token string) bool {
	session, exists := am.sessions[token]
	if !exists {
		return false
	}
	if session.Expiry.Before(time.Now()) {
		return false
	}
	return am.UserManager.IsAdmin(session.Username)
}

// GetUserRole retourne le rÃ´le de l'utilisateur connectÃ©
func (am *AuthManager) GetUserRole(r *http.Request) string {
	cookie, err := r.Cookie("session_token")
	if err != nil {
		return ""
	}

	sessionToken := cookie.Value
	session, exists := am.sessions[sessionToken]
	if !exists || session.Expiry.Before(time.Now()) {
		return ""
	}

	// RÃ©cupÃ©rer le rÃ´le via UserManager pour Ãªtre Ã  jour
	// (Si on stockait le rÃ´le dans la session, ce serait plus rapide mais moins "live")
	// Mais UserManager.users est en mÃ©moire, c'est rapide.
	// On n'a pas accÃ¨s direct Ã  UserManager.users ici sans mÃ©thode publique,
	// mais on peut ajouter une mÃ©thode GetUser(username) ou juste dÃ©duire du IsAdmin.

	// Simplification: on rÃ©cupÃ¨re l'user complet ou on modifie Session pour inclure le rÃ´le?
	// La session a juste Username.

	// On va utiliser GetAllUsers() pour trouver l'user? Non c'est lourd.
	// On va ajouter GetUserRole Ã  UserManager.
	return am.UserManager.GetUserRole(session.Username)
}

// GetUsername retourne le nom de l'utilisateur connectÃ©
func (am *AuthManager) GetUsername(r *http.Request) string {
	cookie, err := r.Cookie("session_token")
	if err != nil {
		return ""
	}

	sessionToken := cookie.Value
	session, exists := am.sessions[sessionToken]
	if !exists || session.Expiry.Before(time.Now()) {
		return ""
	}

	return session.Username
}

func generateToken() string {
	b := make([]byte, 32)
	rand.Read(b)
	return base64.URLEncoding.EncodeToString(b)
}
