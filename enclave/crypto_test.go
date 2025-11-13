package main

import (
	"crypto/rand"
	"crypto/rsa"
	"encoding/json"
	"testing"

	"github.com/peterldowns/testy/assert"
)

func TestGenerateRSAKeyPair(t *testing.T) {
	privateKey, err := GenerateRSAKeyPair()
	assert.NoError(t, err)
	assert.NotNil(t, privateKey)
	assert.Equal(t, 2048, privateKey.N.BitLen())

	// Verify we can use the key
	testData := []byte("test data")
	_, err = rsa.EncryptPKCS1v15(rand.Reader, &privateKey.PublicKey, testData)
	assert.NoError(t, err)
}

func TestHybridEncryptionDecryption(t *testing.T) {
	privateKey, err := GenerateRSAKeyPair()
	assert.NoError(t, err)

	tests := []struct {
		name      string
		plaintext []byte
	}{
		{name: "simple text", plaintext: []byte("Hello, World!")},
		{name: "json data", plaintext: []byte(`{"price":2.50}`)},
		{name: "empty", plaintext: []byte("")},
		{name: "large data", plaintext: make([]byte, 10000)},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result, err := EncryptHybrid(tt.plaintext, &privateKey.PublicKey)
			assert.NoError(t, err)
			assert.NotEqual(t, result.EncryptedAESKey, "")
			assert.NotEqual(t, result.EncryptedPayload, "")
			assert.NotEqual(t, result.Nonce, "")

			decrypted, err := DecryptHybrid(result.EncryptedAESKey, result.EncryptedPayload, result.Nonce, privateKey)
			assert.NoError(t, err)
			assert.Equal(t, string(tt.plaintext), string(decrypted))
		})
	}
}

func TestDecryptHybrid_InvalidInputs(t *testing.T) {
	privateKey, _ := GenerateRSAKeyPair()

	tests := []struct {
		name             string
		encryptedAESKey  string
		encryptedPayload string
	}{
		{
			name:             "invalid base64 in AES key",
			encryptedAESKey:  "invalid-base64!@#",
			encryptedPayload: "dGVzdA==",
		},
		{
			name:             "invalid base64 in payload",
			encryptedAESKey:  "dGVzdA==",
			encryptedPayload: "invalid-base64!@#",
		},
		{
			name:             "wrong key for decryption",
			encryptedAESKey:  "dGVzdGRhdGF0ZXN0ZGF0YXRlc3RkYXRh",
			encryptedPayload: "dGVzdA==",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			_, err := DecryptHybrid(tt.encryptedAESKey, tt.encryptedPayload, "dGVzdA==", privateKey)
			assert.NotNil(t, err) // Should have error
		})
	}
}

func TestDecryptHybrid_WrongPrivateKey(t *testing.T) {
	privateKey1, _ := GenerateRSAKeyPair()
	privateKey2, _ := GenerateRSAKeyPair()

	plaintext := []byte("secret message")

	result, err := EncryptHybrid(plaintext, &privateKey1.PublicKey)
	assert.NoError(t, err)

	// Try to decrypt with wrong key
	_, err = DecryptHybrid(result.EncryptedAESKey, result.EncryptedPayload, result.Nonce, privateKey2)
	assert.NotNil(t, err) // Should fail
}

func TestHybridEncryption_BidPayload(t *testing.T) {
	privateKey, _ := GenerateRSAKeyPair()

	bidPayload := map[string]any{
		"price": 2.50,
	}

	plaintext, _ := json.Marshal(bidPayload)

	result, err := EncryptHybrid(plaintext, &privateKey.PublicKey)
	assert.NoError(t, err)

	decrypted, err := DecryptHybrid(result.EncryptedAESKey, result.EncryptedPayload, result.Nonce, privateKey)
	assert.NoError(t, err)

	var decryptedPayload map[string]any
	err = json.Unmarshal(decrypted, &decryptedPayload)
	assert.NoError(t, err)

	assert.Equal(t, 2.50, decryptedPayload["price"])
}
