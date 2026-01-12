package privacy

import (
	"crypto/aes"
	"crypto/cipher"
	"crypto/rand"
	"encoding/base64"
	"fmt"
	"io"
)

// DevModeEncryptor provides simple AES encryption for dev/test environments.
// WARNING: In production, use HSM-based encryption via a proper key management system.
type DevModeEncryptor struct {
	key []byte
}

// NewDevModeEncryptor creates a new dev-mode encryptor with the given key.
// Key must be 16, 24, or 32 bytes for AES-128, AES-192, or AES-256.
func NewDevModeEncryptor(key []byte) (*DevModeEncryptor, error) {
	if len(key) != 16 && len(key) != 24 && len(key) != 32 {
		return nil, fmt.Errorf("key must be 16, 24, or 32 bytes (got %d)", len(key))
	}
	return &DevModeEncryptor{key: key}, nil
}

// Encrypt encrypts plaintext using AES-GCM.
func (e *DevModeEncryptor) Encrypt(plaintext string) ([]byte, error) {
	block, err := aes.NewCipher(e.key)
	if err != nil {
		return nil, fmt.Errorf("failed to create cipher: %w", err)
	}

	aesGCM, err := cipher.NewGCM(block)
	if err != nil {
		return nil, fmt.Errorf("failed to create GCM: %w", err)
	}

	nonce := make([]byte, aesGCM.NonceSize())
	if _, err := io.ReadFull(rand.Reader, nonce); err != nil {
		return nil, fmt.Errorf("failed to generate nonce: %w", err)
	}

	ciphertext := aesGCM.Seal(nonce, nonce, []byte(plaintext), nil)
	return ciphertext, nil
}

// Decrypt decrypts ciphertext using AES-GCM.
func (e *DevModeEncryptor) Decrypt(ciphertext []byte) (string, error) {
	if len(ciphertext) == 0 {
		return "", fmt.Errorf("ciphertext is empty")
	}

	block, err := aes.NewCipher(e.key)
	if err != nil {
		return "", fmt.Errorf("failed to create cipher: %w", err)
	}

	aesGCM, err := cipher.NewGCM(block)
	if err != nil {
		return "", fmt.Errorf("failed to create GCM: %w", err)
	}

	nonceSize := aesGCM.NonceSize()
	if len(ciphertext) < nonceSize {
		return "", fmt.Errorf("ciphertext too short")
	}

	nonce, ciphertext := ciphertext[:nonceSize], ciphertext[nonceSize:]
	plaintext, err := aesGCM.Open(nil, nonce, ciphertext, nil)
	if err != nil {
		return "", fmt.Errorf("failed to decrypt: %w", err)
	}

	return string(plaintext), nil
}

// EncryptToBase64 encrypts and returns base64-encoded ciphertext.
func (e *DevModeEncryptor) EncryptToBase64(plaintext string) (string, error) {
	ciphertext, err := e.Encrypt(plaintext)
	if err != nil {
		return "", err
	}
	return base64.StdEncoding.EncodeToString(ciphertext), nil
}

// DecryptFromBase64 decrypts base64-encoded ciphertext.
func (e *DevModeEncryptor) DecryptFromBase64(encoded string) (string, error) {
	ciphertext, err := base64.StdEncoding.DecodeString(encoded)
	if err != nil {
		return "", fmt.Errorf("failed to decode base64: %w", err)
	}
	return e.Decrypt(ciphertext)
}

// GenerateDevKey generates a random 32-byte key for dev mode.
// The key should be stored securely and reused across restarts.
func GenerateDevKey() ([]byte, error) {
	key := make([]byte, 32)
	if _, err := rand.Read(key); err != nil {
		return nil, fmt.Errorf("failed to generate key: %w", err)
	}
	return key, nil
}
