package main

import (
	"crypto/aes"
	"crypto/cipher"
	"crypto/rand"
	"crypto/rsa"
	"crypto/sha256"
	"encoding/base64"
	"fmt"
)

// GenerateRSAKeyPair generates a new RSA-2048 key pair using crypto/rand
// In a TEE environment, crypto/rand uses NSM-enhanced entropy
func GenerateRSAKeyPair() (*rsa.PrivateKey, error) {
	privateKey, err := rsa.GenerateKey(rand.Reader, 2048)
	if err != nil {
		return nil, fmt.Errorf("failed to generate RSA key pair: %w", err)
	}
	return privateKey, nil
}

// DecryptHybrid decrypts data encrypted with hybrid RSA-OAEP + AES-256-GCM encryption
// Parameters:
//   - encryptedAESKey: RSA-encrypted AES key (base64-encoded)
//   - encryptedPayload: AES-GCM encrypted data (base64-encoded)
//   - nonce: GCM nonce (base64-encoded)
//   - privateKey: RSA private key for decrypting the AES key
//
// Returns the decrypted plaintext bytes
func DecryptHybrid(encryptedAESKey, encryptedPayload, nonceB64 string, privateKey *rsa.PrivateKey) ([]byte, error) {
	// Decode base64 inputs
	encryptedAESKeyBytes, err := base64.StdEncoding.DecodeString(encryptedAESKey)
	if err != nil {
		return nil, fmt.Errorf("failed to decode encrypted AES key: %w", err)
	}

	encryptedPayloadBytes, err := base64.StdEncoding.DecodeString(encryptedPayload)
	if err != nil {
		return nil, fmt.Errorf("failed to decode encrypted payload: %w", err)
	}

	nonceBytes, err := base64.StdEncoding.DecodeString(nonceB64)
	if err != nil {
		return nil, fmt.Errorf("failed to decode nonce: %w", err)
	}

	// Step 1: Decrypt AES key using RSA-OAEP with SHA-256
	aesKey, err := rsa.DecryptOAEP(sha256.New(), rand.Reader, privateKey, encryptedAESKeyBytes, nil)
	if err != nil {
		return nil, fmt.Errorf("failed to decrypt AES key: %w", err)
	}

	// Validate AES key length (should be 32 bytes for AES-256)
	if len(aesKey) != 32 {
		return nil, fmt.Errorf("invalid AES key length: expected 32 bytes, got %d", len(aesKey))
	}

	// Step 2: Decrypt payload using AES-256-GCM
	block, err := aes.NewCipher(aesKey)
	if err != nil {
		return nil, fmt.Errorf("failed to create AES cipher: %w", err)
	}

	aesgcm, err := cipher.NewGCM(block)
	if err != nil {
		return nil, fmt.Errorf("failed to create GCM: %w", err)
	}

	// Validate nonce length
	if len(nonceBytes) != aesgcm.NonceSize() {
		return nil, fmt.Errorf("invalid nonce length: expected %d bytes, got %d", aesgcm.NonceSize(), len(nonceBytes))
	}

	// Decrypt and authenticate
	plaintext, err := aesgcm.Open(nil, nonceBytes, encryptedPayloadBytes, nil)
	if err != nil {
		return nil, fmt.Errorf("failed to decrypt payload: %w", err)
	}

	return plaintext, nil
}
