package main

import (
	"context"
	"log"
	"sync"
	"time"

	"github.com/google/uuid"
)

// AuctionToken represents a single-use token for bid replay protection
type AuctionToken struct {
	TokenID   string    // Unique token identifier (UUID)
	CreatedAt time.Time // When the token was generated
}

// TokenManager manages auction tokens for replay protection
// Uses sync.Map for fine-grained concurrency without global lock contention
// so that we are not contending on a single lock when processing multiple bids or multiple auctions at once.
type TokenManager struct {
	tokens sync.Map // map[string]*AuctionToken with lock-free reads
}

// NewTokenManager creates a new TokenManager
func NewTokenManager() *TokenManager {
	return &TokenManager{}
}

// GenerateToken creates a new auction token using cryptographically secure randomness.
//
// uuid.New() uses crypto/rand internally, which calls the getrandom syscall to obtain
// entropy from the Linux kernel. The kernel mixes multiple entropy sources including
// the nsm-hwrng (AWS Nitro hardware RNG). In production, verify that nsm-hwrng is the
// active RNG by checking /sys/devices/virtual/misc/hw_random/rng_current.
//
// See: https://blog.trailofbits.com/2024/09/24/notes-on-aws-nitro-enclaves-attack-surface/
func (tm *TokenManager) GenerateToken() string {
	tokenID := uuid.New().String()

	tm.tokens.Store(tokenID, &AuctionToken{
		TokenID:   tokenID,
		CreatedAt: time.Now(),
	})

	return tokenID
}

// ValidateToken checks if a token exists and hasn't been consumed
func (tm *TokenManager) ValidateToken(tokenID string) bool {
	if tokenID == "" {
		return false
	}

	_, exists := tm.tokens.Load(tokenID)
	return exists
}

// ValidateAndConsumeToken atomically validates and consumes a token in one operation
// Returns true if the token was valid and consumed, false if it was invalid or already consumed
//
// Performance: Uses sync.Map.LoadAndDelete() which provides per-token locking.
// This eliminates lock contention in high-throughput scenarios where different
// auctions consume different tokens concurrently.
func (tm *TokenManager) ValidateAndConsumeToken(tokenID string) bool {
	if tokenID == "" {
		return false
	}

	_, existed := tm.tokens.LoadAndDelete(tokenID)
	return existed
}

// ConsumeToken removes a token from the store (marks it as used)
func (tm *TokenManager) ConsumeToken(tokenID string) {
	tm.tokens.Delete(tokenID)
}

// ConsumeTokens removes multiple tokens (batch consumption)
func (tm *TokenManager) ConsumeTokens(tokenIDs []string) {
	for _, tokenID := range tokenIDs {
		tm.tokens.Delete(tokenID)
	}
}

// RemoveExpired removes tokens older than maxAge
func (tm *TokenManager) RemoveExpired(maxAge time.Duration) int {
	now := time.Now()
	expiredCount := 0

	tm.tokens.Range(func(key, value any) bool {
		tokenID := key.(string)
		token := value.(*AuctionToken)

		if now.Sub(token.CreatedAt) > maxAge {
			tm.tokens.Delete(tokenID)
			expiredCount++
		}
		return true // Continue iteration
	})

	return expiredCount
}

// StartExpirationCleanup starts background goroutine to clean up expired tokens
// Exits when context is canceled
func (tm *TokenManager) StartExpirationCleanup(ctx context.Context, interval time.Duration, maxAge time.Duration) {
	go func() {
		ticker := time.NewTicker(interval)
		defer ticker.Stop()

		for {
			select {
			case <-ctx.Done():
				log.Printf("Token cleanup: stopping due to context cancellation")
				return
			case <-ticker.C:
				removed := tm.RemoveExpired(maxAge)
				if removed > 0 {
					log.Printf("Token cleanup: removed %d expired tokens", removed)
				}
			}
		}
	}()
}

// Count returns the number of active tokens (for monitoring)
func (tm *TokenManager) Count() int {
	count := 0
	tm.tokens.Range(func(_, _ any) bool {
		count++
		return true
	})
	return count
}
