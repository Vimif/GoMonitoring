package crypto

import (
	"crypto/aes"
	"crypto/cipher"
	"crypto/rand"
	"crypto/sha256"
	"encoding/base64"
	"errors"
	"fmt"
	"io"
	"os"
)

var (
	// ErrInvalidCiphertext indique que le texte chiffré est invalide
	ErrInvalidCiphertext = errors.New("texte chiffré invalide")

	// ErrMasterKeyNotSet indique que la master key n'est pas configurée
	ErrMasterKeyNotSet = errors.New("master key non configurée (variable d'environnement GO_MONITORING_MASTER_KEY)")
)

const (
	// EnvMasterKey est le nom de la variable d'environnement contenant la master key
	EnvMasterKey = "GO_MONITORING_MASTER_KEY"

	// NonceSize est la taille du nonce pour AES-GCM (12 bytes)
	NonceSize = 12
)

// GetMasterKey récupère la master key depuis la variable d'environnement
// et génère une clé AES-256 (32 bytes) via SHA-256
func GetMasterKey() ([]byte, error) {
	masterKeyStr := os.Getenv(EnvMasterKey)
	if masterKeyStr == "" {
		return nil, ErrMasterKeyNotSet
	}

	// Utiliser SHA-256 pour obtenir exactement 32 bytes (256 bits)
	hash := sha256.Sum256([]byte(masterKeyStr))
	return hash[:], nil
}

// Encrypt chiffre un texte en clair avec AES-256-GCM
// Format du résultat: nonce (12 bytes) + ciphertext + tag (16 bytes)
// Le tout est encodé en base64 pour le stockage
func Encrypt(plaintext string) (string, error) {
	if plaintext == "" {
		return "", nil
	}

	// Récupérer la master key
	key, err := GetMasterKey()
	if err != nil {
		return "", err
	}

	// Créer le cipher AES
	block, err := aes.NewCipher(key)
	if err != nil {
		return "", fmt.Errorf("erreur création cipher: %w", err)
	}

	// Créer GCM (Galois/Counter Mode)
	gcm, err := cipher.NewGCM(block)
	if err != nil {
		return "", fmt.Errorf("erreur création GCM: %w", err)
	}

	// Générer un nonce aléatoire
	nonce := make([]byte, gcm.NonceSize())
	if _, err := io.ReadFull(rand.Reader, nonce); err != nil {
		return "", fmt.Errorf("erreur génération nonce: %w", err)
	}

	// Chiffrer (GCM ajoute automatiquement le tag d'authentification)
	ciphertext := gcm.Seal(nonce, nonce, []byte(plaintext), nil)

	// Encoder en base64 pour le stockage
	return base64.StdEncoding.EncodeToString(ciphertext), nil
}

// Decrypt déchiffre un texte chiffré avec AES-256-GCM
func Decrypt(ciphertext string) (string, error) {
	if ciphertext == "" {
		return "", nil
	}

	// Récupérer la master key
	key, err := GetMasterKey()
	if err != nil {
		return "", err
	}

	// Décoder le base64
	data, err := base64.StdEncoding.DecodeString(ciphertext)
	if err != nil {
		return "", fmt.Errorf("erreur décodage base64: %w", err)
	}

	// Créer le cipher AES
	block, err := aes.NewCipher(key)
	if err != nil {
		return "", fmt.Errorf("erreur création cipher: %w", err)
	}

	// Créer GCM
	gcm, err := cipher.NewGCM(block)
	if err != nil {
		return "", fmt.Errorf("erreur création GCM: %w", err)
	}

	// Vérifier la taille minimale (nonce + tag)
	nonceSize := gcm.NonceSize()
	if len(data) < nonceSize {
		return "", ErrInvalidCiphertext
	}

	// Extraire le nonce et le ciphertext
	nonce, ciphertextBytes := data[:nonceSize], data[nonceSize:]

	// Déchiffrer et vérifier l'authenticité
	plaintext, err := gcm.Open(nil, nonce, ciphertextBytes, nil)
	if err != nil {
		return "", fmt.Errorf("erreur déchiffrement: %w", err)
	}

	return string(plaintext), nil
}

// IsEncrypted vérifie si une chaîne semble être chiffrée (base64 valide avec taille appropriée)
func IsEncrypted(text string) bool {
	if text == "" {
		return false
	}

	// Tenter de décoder le base64
	data, err := base64.StdEncoding.DecodeString(text)
	if err != nil {
		return false
	}

	// Vérifier que la taille est cohérente (au minimum: nonce + quelques bytes + tag)
	// Nonce: 12 bytes, Tag GCM: 16 bytes, minimum 1 byte de données
	minSize := NonceSize + 1 + 16
	return len(data) >= minSize
}

// GenerateMasterKey génère une master key aléatoire sécurisée (pour l'initialisation)
// Cette fonction ne doit être utilisée qu'une fois lors du premier déploiement
func GenerateMasterKey() (string, error) {
	key := make([]byte, 32) // 256 bits
	if _, err := io.ReadFull(rand.Reader, key); err != nil {
		return "", fmt.Errorf("erreur génération master key: %w", err)
	}

	// Encoder en base64 pour faciliter le stockage dans une variable d'environnement
	return base64.StdEncoding.EncodeToString(key), nil
}

// MigratePassword chiffre un password s'il n'est pas déjà chiffré
func MigratePassword(password string) (string, bool, error) {
	if password == "" {
		return "", false, nil
	}

	// Si déjà chiffré, ne rien faire
	if IsEncrypted(password) {
		return password, false, nil
	}

	// Chiffrer
	encrypted, err := Encrypt(password)
	if err != nil {
		return "", false, err
	}

	return encrypted, true, nil
}
