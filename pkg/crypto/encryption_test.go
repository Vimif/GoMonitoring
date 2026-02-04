package crypto

import (
	"os"
	"strings"
	"testing"
)

func TestEncryptDecrypt(t *testing.T) {
	// Setup: dÃ©finir une master key pour les tests
	originalKey := os.Getenv(EnvMasterKey)
	defer os.Setenv(EnvMasterKey, originalKey) // Restore aprÃ¨s test

	testKey := "test-master-key-for-unit-tests-12345678"
	os.Setenv(EnvMasterKey, testKey)

	tests := []struct {
		name      string
		plaintext string
	}{
		{"simple password", "mypassword123"},
		{"complex password", "P@ssw0rd!#$%^&*()_+{}[]|:;<>?,./"},
		{"empty string", ""},
		{"long password", strings.Repeat("a", 1000)},
		{"unicode", "ãƒ‘ã‚¹ãƒ¯ãƒ¼ãƒ‰ðŸ”’"},
		{"spaces", "password with spaces"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Encrypt
			encrypted, err := Encrypt(tt.plaintext)
			if err != nil {
				t.Fatalf("Encrypt() error = %v", err)
			}

			// VÃ©rifier que le chiffrÃ© est diffÃ©rent du clair
			if tt.plaintext != "" && encrypted == tt.plaintext {
				t.Error("Encrypted text should be different from plaintext")
			}

			// Decrypt
			decrypted, err := Decrypt(encrypted)
			if err != nil {
				t.Fatalf("Decrypt() error = %v", err)
			}

			// VÃ©rifier que le dÃ©chiffrÃ© correspond au clair
			if decrypted != tt.plaintext {
				t.Errorf("Decrypted text = %q, want %q", decrypted, tt.plaintext)
			}
		})
	}
}

func TestEncryptWithoutMasterKey(t *testing.T) {
	// Setup: retirer la master key
	originalKey := os.Getenv(EnvMasterKey)
	os.Unsetenv(EnvMasterKey)
	defer os.Setenv(EnvMasterKey, originalKey)

	_, err := Encrypt("test")
	if err != ErrMasterKeyNotSet {
		t.Errorf("Encrypt() without master key should return ErrMasterKeyNotSet, got %v", err)
	}
}

func TestDecryptWithoutMasterKey(t *testing.T) {
	// Setup: retirer la master key
	originalKey := os.Getenv(EnvMasterKey)
	os.Unsetenv(EnvMasterKey)
	defer os.Setenv(EnvMasterKey, originalKey)

	_, err := Decrypt("someencryptedtext")
	if err != ErrMasterKeyNotSet {
		t.Errorf("Decrypt() without master key should return ErrMasterKeyNotSet, got %v", err)
	}
}

func TestDecryptInvalidCiphertext(t *testing.T) {
	// Setup master key
	originalKey := os.Getenv(EnvMasterKey)
	defer os.Setenv(EnvMasterKey, originalKey)
	os.Setenv(EnvMasterKey, "test-key")

	tests := []struct {
		name       string
		ciphertext string
	}{
		{"not base64", "not-valid-base64!@#"},
		{"too short", "YWJj"}, // "abc" en base64, trop court
		{"corrupted", "YWJjZGVmZ2hpamtsbW5vcHFyc3R1dnd4eXo="}, // base64 valide mais pas un chiffrÃ© valide
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			_, err := Decrypt(tt.ciphertext)
			if err == nil {
				t.Error("Decrypt() with invalid ciphertext should return error")
			}
		})
	}
}

func TestIsEncrypted(t *testing.T) {
	// Setup master key
	originalKey := os.Getenv(EnvMasterKey)
	defer os.Setenv(EnvMasterKey, originalKey)
	os.Setenv(EnvMasterKey, "test-key")

	// GÃ©nÃ©rer un texte chiffrÃ© valide
	encrypted, _ := Encrypt("test")

	tests := []struct {
		name string
		text string
		want bool
	}{
		{"encrypted text", encrypted, true},
		{"plain text", "password123", false},
		{"empty string", "", false},
		{"not base64", "not-base64!@#", false},
		{"short base64", "YWJj", false}, // "abc" trop court
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := IsEncrypted(tt.text)
			if got != tt.want {
				t.Errorf("IsEncrypted(%q) = %v, want %v", tt.text, got, tt.want)
			}
		})
	}
}

func TestMigratePassword(t *testing.T) {
	// Setup master key
	originalKey := os.Getenv(EnvMasterKey)
	defer os.Setenv(EnvMasterKey, originalKey)
	os.Setenv(EnvMasterKey, "test-key")

	// Test avec password en clair
	plainPassword := "myplainpassword"
	encrypted, migrated, err := MigratePassword(plainPassword)
	if err != nil {
		t.Fatalf("MigratePassword() error = %v", err)
	}
	if !migrated {
		t.Error("MigratePassword() should indicate password was migrated")
	}
	if encrypted == plainPassword {
		t.Error("Encrypted password should be different from plain password")
	}

	// Test avec password dÃ©jÃ  chiffrÃ©
	alreadyEncrypted, _ := Encrypt("another")
	result, migrated, err := MigratePassword(alreadyEncrypted)
	if err != nil {
		t.Fatalf("MigratePassword() error = %v", err)
	}
	if migrated {
		t.Error("MigratePassword() should not migrate already encrypted password")
	}
	if result != alreadyEncrypted {
		t.Error("Already encrypted password should remain unchanged")
	}

	// Test avec password vide
	result, migrated, err = MigratePassword("")
	if err != nil {
		t.Fatalf("MigratePassword() error = %v", err)
	}
	if migrated {
		t.Error("Empty password should not be migrated")
	}
	if result != "" {
		t.Error("Empty password should remain empty")
	}
}

func TestGenerateMasterKey(t *testing.T) {
	key, err := GenerateMasterKey()
	if err != nil {
		t.Fatalf("GenerateMasterKey() error = %v", err)
	}

	if key == "" {
		t.Error("Generated key should not be empty")
	}

	// VÃ©rifier que c'est du base64 valide
	if !strings.Contains("ABCDEFGHIJKLMNOPQRSTUVWXYZabcdefghijklmnopqrstuvwxyz0123456789+/=", string(key[0])) {
		t.Error("Generated key should be valid base64")
	}

	// GÃ©nÃ©rer une deuxiÃ¨me clÃ© pour vÃ©rifier qu'elle est diffÃ©rente
	key2, _ := GenerateMasterKey()
	if key == key2 {
		t.Error("Generated keys should be unique")
	}
}

func TestEncryptionDeterminism(t *testing.T) {
	// Setup master key
	originalKey := os.Getenv(EnvMasterKey)
	defer os.Setenv(EnvMasterKey, originalKey)
	os.Setenv(EnvMasterKey, "test-key")

	plaintext := "test-password"

	// Chiffrer deux fois le mÃªme texte
	encrypted1, _ := Encrypt(plaintext)
	encrypted2, _ := Encrypt(plaintext)

	// Les chiffrÃ©s doivent Ãªtre diffÃ©rents (car nonce alÃ©atoire)
	if encrypted1 == encrypted2 {
		t.Error("Encrypting same text twice should produce different ciphertexts (due to random nonce)")
	}

	// Mais les deux doivent se dÃ©chiffrer vers le mÃªme plaintext
	decrypted1, _ := Decrypt(encrypted1)
	decrypted2, _ := Decrypt(encrypted2)

	if decrypted1 != plaintext || decrypted2 != plaintext {
		t.Error("Both ciphertexts should decrypt to original plaintext")
	}
}

// Benchmark tests
func BenchmarkEncrypt(b *testing.B) {
	os.Setenv(EnvMasterKey, "test-key-for-benchmark")
	defer os.Unsetenv(EnvMasterKey)

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		Encrypt("mypassword123")
	}
}

func BenchmarkDecrypt(b *testing.B) {
	os.Setenv(EnvMasterKey, "test-key-for-benchmark")
	defer os.Unsetenv(EnvMasterKey)

	encrypted, _ := Encrypt("mypassword123")

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		Decrypt(encrypted)
	}
}

func BenchmarkIsEncrypted(b *testing.B) {
	os.Setenv(EnvMasterKey, "test-key-for-benchmark")
	defer os.Unsetenv(EnvMasterKey)

	encrypted, _ := Encrypt("test")

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		IsEncrypted(encrypted)
	}
}
