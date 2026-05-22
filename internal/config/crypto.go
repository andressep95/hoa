package config

import (
	"crypto/aes"
	"crypto/cipher"
	"crypto/rand"
	"encoding/base64"
	"errors"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"
)

const (
	encPrefix  = "enc:v1:aes256gcm:"
	keySize    = 32 // AES-256
	keyringFile = "keyring"
)

// LoadOrCreateKey reads the master key from ~/.hoa/keyring, creating it if missing.
func LoadOrCreateKey(hoaDir string) ([]byte, error) {
	path := filepath.Join(hoaDir, keyringFile)
	data, err := os.ReadFile(path)
	if err == nil && len(data) == keySize {
		return data, nil
	}
	key := make([]byte, keySize)
	if _, err := io.ReadFull(rand.Reader, key); err != nil {
		return nil, fmt.Errorf("generating key: %w", err)
	}
	if err := os.MkdirAll(hoaDir, 0700); err != nil {
		return nil, err
	}
	if err := os.WriteFile(path, key, 0600); err != nil {
		return nil, fmt.Errorf("writing keyring: %w", err)
	}
	return key, nil
}

// Encrypt seals plaintext with AES-256-GCM and returns a prefixed base64 string.
func Encrypt(key []byte, plaintext string) (string, error) {
	block, err := aes.NewCipher(key)
	if err != nil {
		return "", err
	}
	gcm, err := cipher.NewGCM(block)
	if err != nil {
		return "", err
	}
	nonce := make([]byte, gcm.NonceSize())
	if _, err := io.ReadFull(rand.Reader, nonce); err != nil {
		return "", err
	}
	sealed := gcm.Seal(nonce, nonce, []byte(plaintext), nil)
	return encPrefix + base64.StdEncoding.EncodeToString(sealed), nil
}

// Decrypt opens a prefixed encrypted string. Returns plaintext.
func Decrypt(key []byte, encoded string) (string, error) {
	if !IsEncrypted(encoded) {
		return encoded, nil
	}
	data, err := base64.StdEncoding.DecodeString(strings.TrimPrefix(encoded, encPrefix))
	if err != nil {
		return "", fmt.Errorf("base64 decode: %w", err)
	}
	block, err := aes.NewCipher(key)
	if err != nil {
		return "", err
	}
	gcm, err := cipher.NewGCM(block)
	if err != nil {
		return "", err
	}
	nonceSize := gcm.NonceSize()
	if len(data) < nonceSize {
		return "", errors.New("ciphertext too short")
	}
	plaintext, err := gcm.Open(nil, data[:nonceSize], data[nonceSize:], nil)
	if err != nil {
		return "", fmt.Errorf("decrypt: %w", err)
	}
	return string(plaintext), nil
}

// IsEncrypted reports whether a string has the encryption prefix.
func IsEncrypted(s string) bool {
	return strings.HasPrefix(s, encPrefix)
}
