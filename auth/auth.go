package auth

import (
	"crypto/rand"
	"encoding/base64"
	"html/template"
	"net/http"
	"time"

	"go-monitoring/middleware"
)

// Session structure pour stocker les sessions actives
type Session struct {
	Username string
	Expiry   time.Time
}

// AuthManager gère les sessions et middlewares
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

// Middleware protège les routes
func (am *AuthManager) Middleware(next http.HandlerFunc) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		// Ignorer les fichiers statiques (images, css, js) si nécessaire
		// Mais ici les statiques sont souvent publics.
		// Si on veut protéger, on applique middleware.

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

// LoginHandler gère la page de connexion
func (am *AuthManager) LoginHandler(w http.ResponseWriter, r *http.Request) {
	if r.Method == "GET" {
		// Vérifier ou créer un cookie de session pour le CSRF
		cookie, err := r.Cookie("session_token")
		var sessionID string
		if err != nil || cookie.Value == "" {
			sessionID = generateToken()
			http.SetCookie(w, &http.Cookie{
				Name:     "session_token",
				Value:    sessionID,
				Expires:  time.Now().Add(24 * time.Hour),
				Path:     "/",
				HttpOnly: true,
			})
		} else {
			sessionID = cookie.Value
		}

		csrfToken := middleware.GetCSRFTokenForSession(sessionID)

		tmpl, err := template.ParseFiles("templates/login.html")
		if err != nil {
			http.Error(w, "Erreur template login", http.StatusInternalServerError)
			return
		}
		tmpl.Execute(w, map[string]interface{}{
			"CSRFToken": csrfToken,
		})
		return
	}

	// POST Login
	username := r.FormValue("username")
	password := r.FormValue("password")

	user, err := am.UserManager.Authenticate(username, password)
	if err != nil {
		// Récupérer le token CSRF existant pour le réinjecter dans le formulaire
		csrfToken := ""
		if cookie, err := r.Cookie("session_token"); err == nil {
			csrfToken = middleware.GetCSRFTokenForSession(cookie.Value)
		}

		tmpl, _ := template.ParseFiles("templates/login.html")
		tmpl.Execute(w, map[string]interface{}{
			"Error":     "Identifiants incorrects",
			"CSRFToken": csrfToken,
		})
		return
	}

	// Créer session
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

// LogoutHandler gère la déconnexion
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

// IsAdminSession vérifie si la session appartient à un admin
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

// GetUserRole retourne le rôle de l'utilisateur connecté
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

	// Récupérer le rôle via UserManager pour être à jour
	// (Si on stockait le rôle dans la session, ce serait plus rapide mais moins "live")
	// Mais UserManager.users est en mémoire, c'est rapide.
	// On n'a pas accès direct à UserManager.users ici sans méthode publique,
	// mais on peut ajouter une méthode GetUser(username) ou juste déduire du IsAdmin.

	// Simplification: on récupère l'user complet ou on modifie Session pour inclure le rôle?
	// La session a juste Username.

	// On va utiliser GetAllUsers() pour trouver l'user? Non c'est lourd.
	// On va ajouter GetUserRole à UserManager.
	return am.UserManager.GetUserRole(session.Username)
}

// GetUsername retourne le nom de l'utilisateur connecté
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
