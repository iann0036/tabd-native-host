package main

import (
	"crypto/aes"
	"crypto/cipher"
	"crypto/rand"
	"encoding/base64"
	"fmt"
	"os"
	"path/filepath"

	"github.com/zalando/go-keyring"
	"golang.org/x/crypto/argon2"
)

// SecureStorage interface for platform-specific secure storage
type SecureStorage interface {
	Store(key string, data []byte) error
	Retrieve(key string) ([]byte, error)
	Delete(key string) error
}

// KeyringStorage uses the system keyring for secure storage
type KeyringStorage struct {
	serviceName string
}

// EncryptedFileStorage uses encrypted files as fallback storage
type EncryptedFileStorage struct {
	storageDir string
	passphrase string
}

// NewSecureStorage creates the appropriate secure storage for the platform
func NewSecureStorage(tabdDir string) SecureStorage {
	// Try keyring first (works on macOS, Windows, and most Linux distros)
	// TODO: Fix, broken
	/*if supportsKeyring() {
		return &KeyringStorage{serviceName: "tabd-native-host"}
	}*/

	// Fallback to encrypted file storage
	passphrase := generateOrRetrievePassphrase(tabdDir)
	return &EncryptedFileStorage{
		storageDir: tabdDir,
		passphrase: passphrase,
	}
}

// supportsKeyring checks if the system supports keyring operations
func supportsKeyring() bool {
	// Test keyring availability by trying to set and get a test value
	testKey := "tabd-test-key"
	testValue := "test"

	err := keyring.Set("tabd-native-host", testKey, testValue)
	if err != nil {
		return false
	}

	retrieved, err := keyring.Get("tabd-native-host", testKey)
	if err != nil || retrieved != testValue {
		return false
	}

	// Clean up test key
	keyring.Delete("tabd-native-host", testKey)
	return true
}

// generateOrRetrievePassphrase creates or retrieves a passphrase for encrypted storage
func generateOrRetrievePassphrase(tabdDir string) string {
	passphrasePath := filepath.Join(tabdDir, ".passphrase")

	// Try to read existing passphrase
	if data, err := os.ReadFile(passphrasePath); err == nil {
		return string(data)
	}

	// Generate new passphrase
	passphraseBytes := make([]byte, 32)
	rand.Read(passphraseBytes)
	passphrase := base64.URLEncoding.EncodeToString(passphraseBytes)

	// Save passphrase (with restricted permissions)
	os.WriteFile(passphrasePath, []byte(passphrase), 0600)

	return passphrase
}

// KeyringStorage implementation
func (k *KeyringStorage) Store(key string, data []byte) error {
	encoded := base64.StdEncoding.EncodeToString(data)
	return keyring.Set(k.serviceName, key, encoded)
}

func (k *KeyringStorage) Retrieve(key string) ([]byte, error) {
	encoded, err := keyring.Get(k.serviceName, key)
	if err != nil {
		return nil, err
	}
	return base64.StdEncoding.DecodeString(encoded)
}

func (k *KeyringStorage) Delete(key string) error {
	return keyring.Delete(k.serviceName, key)
}

// EncryptedFileStorage implementation
func (e *EncryptedFileStorage) Store(key string, data []byte) error {
	encrypted, err := e.encrypt(data)
	if err != nil {
		return fmt.Errorf("failed to encrypt data: %v", err)
	}

	filePath := filepath.Join(e.storageDir, key+".enc")
	return os.WriteFile(filePath, encrypted, 0600)
}

func (e *EncryptedFileStorage) Retrieve(key string) ([]byte, error) {
	filePath := filepath.Join(e.storageDir, key+".enc")
	encrypted, err := os.ReadFile(filePath)
	if err != nil {
		return nil, err
	}

	return e.decrypt(encrypted)
}

func (e *EncryptedFileStorage) Delete(key string) error {
	filePath := filepath.Join(e.storageDir, key+".enc")
	return os.Remove(filePath)
}

func (e *EncryptedFileStorage) encrypt(data []byte) ([]byte, error) {
	// Derive key from passphrase using Argon2
	salt := make([]byte, 16)
	rand.Read(salt)
	key := argon2.IDKey([]byte(e.passphrase), salt, 1, 64*1024, 4, 32)

	// Create AES cipher
	block, err := aes.NewCipher(key)
	if err != nil {
		return nil, err
	}

	// Create GCM mode
	gcm, err := cipher.NewGCM(block)
	if err != nil {
		return nil, err
	}

	// Generate nonce
	nonce := make([]byte, gcm.NonceSize())
	rand.Read(nonce)

	// Encrypt data
	ciphertext := gcm.Seal(nil, nonce, data, nil)

	// Combine salt + nonce + ciphertext
	result := make([]byte, 16+len(nonce)+len(ciphertext))
	copy(result[:16], salt)
	copy(result[16:16+len(nonce)], nonce)
	copy(result[16+len(nonce):], ciphertext)

	return result, nil
}

func (e *EncryptedFileStorage) decrypt(data []byte) ([]byte, error) {
	if len(data) < 16+12 { // salt + nonce minimum
		return nil, fmt.Errorf("invalid encrypted data")
	}

	// Extract components
	salt := data[:16]
	nonceSize := 12 // GCM standard nonce size
	nonce := data[16 : 16+nonceSize]
	ciphertext := data[16+nonceSize:]

	// Derive key from passphrase
	key := argon2.IDKey([]byte(e.passphrase), salt, 1, 64*1024, 4, 32)

	// Create AES cipher
	block, err := aes.NewCipher(key)
	if err != nil {
		return nil, err
	}

	// Create GCM mode
	gcm, err := cipher.NewGCM(block)
	if err != nil {
		return nil, err
	}

	// Decrypt data
	return gcm.Open(nil, nonce, ciphertext, nil)
}
