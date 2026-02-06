package middleware

import (
	"crypto/rand"
	"encoding/base64"
	"net/http"
	"sync"
	"time"
)

// CSRFToken représente un token CSRF avec son expiration
type CSRFToken struct {
	Value     string
	ExpiresAt time.Time
}

// CSRFStore stocke les tokens CSRF par session
type CSRFStore struct {
	tokens map[string]*CSRFToken
	mu     sync.RWMutex
}

var csrfStore = &CSRFStore{
	tokens: make(map[string]*CSRFToken),
}

// generateCSRFToken génère un token CSRF aléatoire sécurisé
func generateCSRFToken() (string, error) {
	b := make([]byte, 32)
	if _, err := rand.Read(b); err != nil {
		return "", err
	}
	return base64.URLEncoding.EncodeToString(b), nil
}

// getSessionID extrait l'ID de session du cookie
func getSessionID(r *http.Request) string {
	cookie, err := r.Cookie("session_token")
	if err != nil {
		return ""
	}
	return cookie.Value
}

// GetCSRFToken récupère ou crée un token CSRF pour la session
func GetCSRFToken(r *http.Request) string {
	sessionID := getSessionID(r)
	if sessionID == "" {
		return ""
	}

	csrfStore.mu.Lock()
	defer csrfStore.mu.Unlock()

	// Vérifier si un token existe et n'est pas expiré
	if token, exists := csrfStore.tokens[sessionID]; exists {
		if time.Now().Before(token.ExpiresAt) {
			return token.Value
		}
		// Token expiré, le supprimer
		delete(csrfStore.tokens, sessionID)
	}

	// Créer un nouveau token
	tokenValue, err := generateCSRFToken()
	if err != nil {
		return ""
	}

	csrfStore.tokens[sessionID] = &CSRFToken{
		Value:     tokenValue,
		ExpiresAt: time.Now().Add(24 * time.Hour),
	}

	return tokenValue
}

// validateCSRFToken vérifie si le token CSRF est valide
func validateCSRFToken(r *http.Request, tokenValue string) bool {
	sessionID := getSessionID(r)
	if sessionID == "" || tokenValue == "" {
		return false
	}

	csrfStore.mu.RLock()
	defer csrfStore.mu.RUnlock()

	token, exists := csrfStore.tokens[sessionID]
	if !exists {
		return false
	}

	// Vérifier expiration
	if time.Now().After(token.ExpiresAt) {
		return false
	}

	// Comparaison à temps constant pour éviter timing attacks
	return constantTimeCompare(token.Value, tokenValue)
}

// constantTimeCompare compare deux strings en temps constant
func constantTimeCompare(a, b string) bool {
	if len(a) != len(b) {
		return false
	}

	result := 0
	for i := 0; i < len(a); i++ {
		result |= int(a[i]) ^ int(b[i])
	}
	return result == 0
}

// CleanupExpiredTokens nettoie les tokens expirés (à appeler périodiquement)
func CleanupExpiredTokens() {
	csrfStore.mu.Lock()
	defer csrfStore.mu.Unlock()

	now := time.Now()
	for sessionID, token := range csrfStore.tokens {
		if now.After(token.ExpiresAt) {
			delete(csrfStore.tokens, sessionID)
		}
	}
}

// DeleteCSRFToken supprime le token CSRF pour une session (lors de la déconnexion)
func DeleteCSRFToken(sessionID string) {
	csrfStore.mu.Lock()
	defer csrfStore.mu.Unlock()
	delete(csrfStore.tokens, sessionID)
}

// CSRFMiddleware protège contre les attaques CSRF
func CSRFMiddleware(next http.Handler) http.Handler {
	// Routes exemptées de la vérification CSRF
	exemptedPaths := map[string]bool{
		"/login":  true,
		"/logout": true,
	}

	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// Vérifier si la route est exemptée
		if exemptedPaths[r.URL.Path] {
			next.ServeHTTP(w, r)
			return
		}

		// Méthodes safe - pas de vérification nécessaire
		if r.Method == "GET" || r.Method == "HEAD" || r.Method == "OPTIONS" {
			next.ServeHTTP(w, r)
			return
		}

		// Méthodes state-changing - vérifier le token CSRF
		requestToken := r.Header.Get("X-CSRF-Token")
		if requestToken == "" {
			// Fallback: chercher dans le formulaire
			requestToken = r.FormValue("csrf_token")
		}

		if !validateCSRFToken(r, requestToken) {
			http.Error(w, "CSRF token invalid or missing", http.StatusForbidden)
			return
		}

		next.ServeHTTP(w, r)
	})
}

// StartCleanupRoutine démarre une goroutine pour nettoyer les tokens expirés
func StartCleanupRoutine() {
	go func() {
		ticker := time.NewTicker(1 * time.Hour)
		defer ticker.Stop()

		for range ticker.C {
			CleanupExpiredTokens()
		}
	}()
}
