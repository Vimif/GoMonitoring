package middleware

import (
	"crypto/rand"
	"encoding/base64"
	"net/http"
	"sync"
	"time"
)

// CSRFToken reprÃ©sente un token CSRF avec son expiration
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

// generateCSRFToken gÃ©nÃ¨re un token CSRF alÃ©atoire sÃ©curisÃ©
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

// GetCSRFToken rÃ©cupÃ¨re ou crÃ©e un token CSRF pour la session
func GetCSRFToken(r *http.Request) string {
	sessionID := getSessionID(r)
	if sessionID == "" {
		return ""
	}

	csrfStore.mu.Lock()
	defer csrfStore.mu.Unlock()

	// VÃ©rifier si un token existe et n'est pas expirÃ©
	if token, exists := csrfStore.tokens[sessionID]; exists {
		if time.Now().Before(token.ExpiresAt) {
			return token.Value
		}
		// Token expirÃ©, le supprimer
		delete(csrfStore.tokens, sessionID)
	}

	// CrÃ©er un nouveau token
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

// validateCSRFToken vÃ©rifie si le token CSRF est valide
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

	// VÃ©rifier expiration
	if time.Now().After(token.ExpiresAt) {
		return false
	}

	// Comparaison Ã  temps constant pour Ã©viter timing attacks
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

// CleanupExpiredTokens nettoie les tokens expirÃ©s (Ã  appeler pÃ©riodiquement)
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

// DeleteCSRFToken supprime le token CSRF pour une session (lors de la dÃ©connexion)
func DeleteCSRFToken(sessionID string) {
	csrfStore.mu.Lock()
	defer csrfStore.mu.Unlock()
	delete(csrfStore.tokens, sessionID)
}

// CSRFMiddleware protÃ¨ge contre les attaques CSRF
func CSRFMiddleware(next http.Handler) http.Handler {
	// Routes exemptÃ©es de la vÃ©rification CSRF
	exemptedPaths := map[string]bool{
		"/login":  true,
		"/logout": true,
	}

	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// VÃ©rifier si la route est exemptÃ©e
		if exemptedPaths[r.URL.Path] {
			next.ServeHTTP(w, r)
			return
		}

		// MÃ©thodes safe - pas de vÃ©rification nÃ©cessaire
		if r.Method == "GET" || r.Method == "HEAD" || r.Method == "OPTIONS" {
			next.ServeHTTP(w, r)
			return
		}

		// MÃ©thodes state-changing - vÃ©rifier le token CSRF
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

// StartCleanupRoutine dÃ©marre une goroutine pour nettoyer les tokens expirÃ©s
func StartCleanupRoutine() {
	go func() {
		ticker := time.NewTicker(1 * time.Hour)
		defer ticker.Stop()

		for range ticker.C {
			CleanupExpiredTokens()
		}
	}()
}
