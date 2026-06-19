package utils

import (
	"crypto/aes"
	"crypto/cipher"
	"crypto/rand"
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"io"
	"os"
)

// GetEncryptionKey loads the 32-byte encryption key from the environment.
func GetEncryptionKey() []byte {
	key := os.Getenv("ENCRYPTION_KEY")
	if len(key) < 32 {
		// Fallback/pad to 32 bytes for testing
		padded := make([]byte, 32)
		copy(padded, key)
		return padded
	}
	return []byte(key[:32])
}

// HashEmail returns the SHA-256 hash of the email.
func HashEmail(email string) []byte {
	hash := sha256.Sum256([]byte(email))
	return hash[:]
}

// EncryptEmail encrypts the email string using AES-GCM.
func EncryptEmail(email string) ([]byte, error) {
	key := GetEncryptionKey()
	block, err := aes.NewCipher(key)
	if err != nil {
		return nil, err
	}

	gcm, err := cipher.NewGCM(block)
	if err != nil {
		return nil, err
	}

	nonce := make([]byte, gcm.NonceSize())
	if _, err := io.ReadFull(rand.Reader, nonce); err != nil {
		return nil, err
	}

	ciphertext := gcm.Seal(nonce, nonce, []byte(email), nil)
	return ciphertext, nil
}

// DecryptEmail decrypts the email ciphertext using AES-GCM.
func DecryptEmail(ciphertext []byte) (string, error) {
	if len(ciphertext) == 0 {
		return "", fmt.Errorf("empty ciphertext")
	}

	key := GetEncryptionKey()
	block, err := aes.NewCipher(key)
	if err != nil {
		return "", err
	}

	gcm, err := cipher.NewGCM(block)
	if err != nil {
		return "", err
	}

	nonceSize := gcm.NonceSize()
	if len(ciphertext) < nonceSize {
		return "", fmt.Errorf("ciphertext too short")
	}

	nonce, actualCiphertext := ciphertext[:nonceSize], ciphertext[nonceSize:]
	plaintext, err := gcm.Open(nil, nonce, actualCiphertext, nil)
	if err != nil {
		return "", err
	}

	return string(plaintext), nil
}

// GenerateRandomHex returns a random hex string of size bytes.
func GenerateRandomHex(size int) string {
	buf := make([]byte, size)
	_, _ = rand.Read(buf)
	return hex.EncodeToString(buf)
}

// RandomBytes returns cryptographically random bytes.
func RandomBytes(n int) ([]byte, error) {
	buf := make([]byte, n)
	if _, err := io.ReadFull(rand.Reader, buf); err != nil {
		return nil, err
	}
	return buf, nil
}
