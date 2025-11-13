package main

import (
	"encoding/json"
	"testing"

	"github.com/peterldowns/testy/assert"

	"github.com/cloudx-io/openauction/core"
	"github.com/cloudx-io/openauction/enclaveapi"
)

func TestDecryptBids_NoEncryptedData(t *testing.T) {
	km, _ := NewKeyManager()

	encBids := []enclaveapi.EncryptedCoreBid{
		{CoreBid: core.CoreBid{ID: "bid1", Bidder: "bidder1", Price: 2.50}},
		{CoreBid: core.CoreBid{ID: "bid2", Bidder: "bidder2", Price: 3.00}},
	}

	decryptedData, _, errors := decryptAllBids(encBids, km)
	assert.Equal(t, 0, len(errors))

	finalBids, _ := filterBidsByConsumedTokens(decryptedData, make(map[string]bool))
	assert.Equal(t, 2, len(finalBids))
	assert.Equal(t, 2.50, finalBids[0].Price)
}

func TestDecryptBids_WithEncryptedData(t *testing.T) {
	km, _ := NewKeyManager()

	payload := map[string]any{
		"price": 5.75,
	}
	plaintextBytes, _ := json.Marshal(payload)
	result, _ := EncryptHybrid(plaintextBytes, km.PublicKey)

	encBids := []enclaveapi.EncryptedCoreBid{
		{
			CoreBid: core.CoreBid{
				ID:     "bid1",
				Bidder: "bidder1",
			},
			EncryptedPrice: &core.EncryptedBidPrice{
				AESKeyEncrypted:  result.EncryptedAESKey,
				EncryptedPayload: result.EncryptedPayload,
				Nonce:            result.Nonce,
			},
		},
	}

	decryptedData, _, errors := decryptAllBids(encBids, km)
	assert.Equal(t, 0, len(errors))

	finalBids, _ := filterBidsByConsumedTokens(decryptedData, make(map[string]bool))
	assert.Equal(t, 1, len(finalBids))

	bid := finalBids[0]
	assert.Equal(t, 5.75, bid.Price)
}

func TestDecryptBids_MixedEncryptedUnencrypted(t *testing.T) {
	km, _ := NewKeyManager()

	payload := map[string]any{
		"price": 4.25,
	}
	plaintextBytes, _ := json.Marshal(payload)
	result, _ := EncryptHybrid(plaintextBytes, km.PublicKey)

	encBids := []enclaveapi.EncryptedCoreBid{
		{CoreBid: core.CoreBid{ID: "bid1", Bidder: "bidder1", Price: 2.50}},
		{
			CoreBid: core.CoreBid{
				ID:     "bid2",
				Bidder: "bidder2",
			},
			EncryptedPrice: &core.EncryptedBidPrice{
				AESKeyEncrypted:  result.EncryptedAESKey,
				EncryptedPayload: result.EncryptedPayload,
				Nonce:            result.Nonce,
			},
		},
		{CoreBid: core.CoreBid{ID: "bid3", Bidder: "bidder3", Price: 3.75}},
	}

	decryptedData, _, errors := decryptAllBids(encBids, km)
	assert.Equal(t, 0, len(errors))

	finalBids, _ := filterBidsByConsumedTokens(decryptedData, make(map[string]bool))
	assert.Equal(t, 3, len(finalBids))
	assert.Equal(t, 2.50, finalBids[0].Price)
	assert.Equal(t, 4.25, finalBids[1].Price)
	assert.Equal(t, 3.75, finalBids[2].Price)
}

func TestDecryptBids_InvalidEncryptedData(t *testing.T) {
	km, _ := NewKeyManager()

	encBids := []enclaveapi.EncryptedCoreBid{
		{
			CoreBid: core.CoreBid{
				ID:     "bid1",
				Bidder: "bidder1",
			},
			EncryptedPrice: &core.EncryptedBidPrice{
				AESKeyEncrypted:  "invalid-base64",
				EncryptedPayload: "invalid-base64",
				Nonce:            "invalid-base64",
			},
		},
	}

	decryptedData, _, errors := decryptAllBids(encBids, km)
	assert.Equal(t, 1, len(errors))

	finalBids, _ := filterBidsByConsumedTokens(decryptedData, make(map[string]bool))
	assert.Equal(t, 0, len(finalBids)) // Excluded
}

func TestDecryptBids_InvalidPrice(t *testing.T) {
	km, _ := NewKeyManager()

	payload := map[string]any{
		"price": -1.50,
	}
	plaintextBytes, _ := json.Marshal(payload)
	result, _ := EncryptHybrid(plaintextBytes, km.PublicKey)

	encBids := []enclaveapi.EncryptedCoreBid{
		{
			CoreBid: core.CoreBid{
				ID:     "bid1",
				Bidder: "bidder1",
			},
			EncryptedPrice: &core.EncryptedBidPrice{
				AESKeyEncrypted:  result.EncryptedAESKey,
				EncryptedPayload: result.EncryptedPayload,
				Nonce:            result.Nonce,
			},
		},
	}

	decryptedData, _, errors := decryptAllBids(encBids, km)
	assert.Equal(t, 0, len(errors)) // Decryption succeeds

	finalBids, _ := filterBidsByConsumedTokens(decryptedData, make(map[string]bool))
	assert.Equal(t, 0, len(finalBids)) // Excluded due to invalid price in filtering stage
}

func TestDecryptBids_NilKeyManager(t *testing.T) {
	encBids := []enclaveapi.EncryptedCoreBid{
		{CoreBid: core.CoreBid{ID: "bid1", Bidder: "bidder1", Price: 2.50}},
	}

	decryptedData, excludedBids, errors := decryptAllBids(encBids, nil)
	assert.Equal(t, 0, len(errors))
	assert.Equal(t, 0, len(excludedBids))

	finalBids, _ := filterBidsByConsumedTokens(decryptedData, make(map[string]bool))
	assert.Equal(t, 1, len(finalBids))
}

func TestDecryptBids_WrongKey(t *testing.T) {
	km1, _ := NewKeyManager()
	km2, _ := NewKeyManager()

	payload := map[string]any{
		"price": 2.50,
	}
	plaintextBytes, _ := json.Marshal(payload)
	result, _ := EncryptHybrid(plaintextBytes, km1.PublicKey)

	encBids := []enclaveapi.EncryptedCoreBid{
		{
			CoreBid: core.CoreBid{
				ID:     "bid1",
				Bidder: "bidder1",
			},
			EncryptedPrice: &core.EncryptedBidPrice{
				AESKeyEncrypted:  result.EncryptedAESKey,
				EncryptedPayload: result.EncryptedPayload,
				Nonce:            result.Nonce,
			},
		},
	}

	decryptedData, excludedBids, errors := decryptAllBids(encBids, km2)
	assert.Equal(t, 1, len(errors))
	assert.Equal(t, 1, len(excludedBids)) // Should be excluded
	assert.Equal(t, "bid1", excludedBids[0].BidID)
	assert.Equal(t, "bidder1", excludedBids[0].Bidder)

	finalBids, _ := filterBidsByConsumedTokens(decryptedData, make(map[string]bool))
	assert.Equal(t, 0, len(finalBids)) // Should fail
}

func TestDecryptBids_BothEncryptedAndUnencryptedPrice(t *testing.T) {
	km, _ := NewKeyManager()

	// Create encrypted price payload
	payload := map[string]any{
		"price": 7.25, // This should take precedence
	}
	plaintextBytes, _ := json.Marshal(payload)
	result, _ := EncryptHybrid(plaintextBytes, km.PublicKey)

	encBids := []enclaveapi.EncryptedCoreBid{
		{
			CoreBid: core.CoreBid{
				ID:     "bid1",
				Bidder: "bidder1",
				Price:  2.50, // This should be ignored in favor of encrypted price
			},
			EncryptedPrice: &core.EncryptedBidPrice{
				AESKeyEncrypted:  result.EncryptedAESKey,
				EncryptedPayload: result.EncryptedPayload,
				Nonce:            result.Nonce,
			},
		},
	}

	decryptedData, excludedBids, errors := decryptAllBids(encBids, km)
	// Should successfully decrypt
	assert.Equal(t, 0, len(errors))
	assert.Equal(t, 0, len(excludedBids))

	finalBids, _ := filterBidsByConsumedTokens(decryptedData, make(map[string]bool))
	assert.Equal(t, 1, len(finalBids))

	bid := finalBids[0]
	assert.Equal(t, "bid1", bid.ID)
	assert.Equal(t, "bidder1", bid.Bidder)
	assert.Equal(t, 7.25, bid.Price) // Should use encrypted price, not CoreBid.Price
}
