package web

import (
	"crypto/aes"
	"crypto/cipher"
	"crypto/rand"
	"crypto/sha256"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"os"
	"sync"

	"my-app/db"
)

var (
	embeddingKeyOnce sync.Once
	embeddingKey     []byte
	embeddingKeyErr  error
)

func serverEmbeddingKey() ([]byte, error) {
	embeddingKeyOnce.Do(func() {
		raw := os.Getenv("NOTEIKA_EMBEDDING_KEY")
		if raw == "" {
			log.Printf("[embedding_crypto] NOTEIKA_EMBEDDING_KEY not set — using dev-only default (set in production)")
			sum := sha256.Sum256([]byte("noteika-dev-embedding-key"))
			embeddingKey = sum[:]
			return
		}
		decoded, err := base64.StdEncoding.DecodeString(raw)
		if err != nil {
			embeddingKeyErr = fmt.Errorf("NOTEIKA_EMBEDDING_KEY must be base64: %w", err)
			return
		}
		if len(decoded) != 32 {
			embeddingKeyErr = fmt.Errorf("NOTEIKA_EMBEDDING_KEY must decode to 32 bytes, got %d", len(decoded))
			return
		}
		embeddingKey = decoded
	})
	return embeddingKey, embeddingKeyErr
}

func encryptEmbeddingVector(vec []float32) ([]byte, error) {
	if len(vec) == 0 {
		return nil, nil
	}
	key, err := serverEmbeddingKey()
	if err != nil {
		return nil, err
	}
	plain, err := json.Marshal(vec)
	if err != nil {
		return nil, err
	}
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
	return gcm.Seal(nonce, nonce, plain, nil), nil
}

func decryptEmbeddingVector(blob []byte) ([]float32, error) {
	if len(blob) == 0 {
		return nil, nil
	}
	key, err := serverEmbeddingKey()
	if err != nil {
		return nil, err
	}
	block, err := aes.NewCipher(key)
	if err != nil {
		return nil, err
	}
	gcm, err := cipher.NewGCM(block)
	if err != nil {
		return nil, err
	}
	nonceSize := gcm.NonceSize()
	if len(blob) < nonceSize {
		return nil, fmt.Errorf("encrypted embedding too short")
	}
	nonce, ciphertext := blob[:nonceSize], blob[nonceSize:]
	plain, err := gcm.Open(nil, nonce, ciphertext, nil)
	if err != nil {
		return nil, err
	}
	var vec []float32
	if err := json.Unmarshal(plain, &vec); err != nil {
		return nil, err
	}
	return vec, nil
}

func captureSearchVector(cap db.Capture) []float32 {
	if len(cap.EncryptedEmbedding) > 0 {
		vec, err := decryptEmbeddingVector(cap.EncryptedEmbedding)
		if err != nil {
			log.Printf("[embedding_crypto] decrypt embedding for %s: %v", cap.ID, err)
			return nil
		}
		return vec
	}
	return cap.LegacyEmbedding
}

func prepareCaptureForStorage(cap *db.Capture, clientEmbedding []float32) error {
	if len(clientEmbedding) > 0 {
		enc, err := encryptEmbeddingVector(clientEmbedding)
		if err != nil {
			return err
		}
		cap.EncryptedEmbedding = enc
		cap.LegacyEmbedding = nil
	}
	return nil
}
