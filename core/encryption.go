package core

// EncryptedBidPrice represents encrypted price data using RSA-OAEP-SHA256/AES-256-GCM.
// Bidders may encrypt their bid prices using a public key provided in the initial bid request,
// ensuring that prices are only ever decrypted inside the TEE where the auction runs.
type EncryptedBidPrice struct {
	AESKeyEncrypted  string `json:"aes_key_encrypted"` // base64-encoded RSA-OAEP encrypted AES key
	EncryptedPayload string `json:"encrypted_payload"` // base64-encoded AES-GCM encrypted {"price": X}
	Nonce            string `json:"nonce"`             // base64-encoded GCM nonce (12 bytes)
}

// DecryptedBidPayload represents the decrypted bid payload structure.
// This is what's inside the EncryptedPayload after decryption inside the TEE.
type DecryptedBidPayload struct {
	Price        float64 `json:"price"`                   // Bid price in USD
	AuctionToken string  `json:"auction_token,omitempty"` // Optional single-use token for replay protection
}
