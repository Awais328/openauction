package main

import (
	"context"
	"sync"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/peterldowns/testy/assert"
)

func TestNewTokenManager(t *testing.T) {
	tm := NewTokenManager()
	assert.NotNil(t, tm)
	assert.Equal(t, 0, tm.Count())
}

func TestTokenManager_GenerateToken(t *testing.T) {
	tm := NewTokenManager()

	token := tm.GenerateToken()
	assert.NotEqual(t, "", token)

	// Verify it's a valid UUID
	_, err := uuid.Parse(token)
	assert.NoError(t, err)

	// Verify token is stored
	assert.Equal(t, 1, tm.Count())
	assert.True(t, tm.ValidateToken(token))
}

func TestTokenManager_GenerateToken_UniqueTokens(t *testing.T) {
	tm := NewTokenManager()

	tokens := make([]string, 100)
	for i := 0; i < 100; i++ {
		token := tm.GenerateToken()
		tokens[i] = token
	}

	// Verify all tokens are unique
	seen := make(map[string]bool)
	for _, token := range tokens {
		if seen[token] {
			t.Fatalf("duplicate token found: %s", token)
		}
		seen[token] = true
	}

	assert.Equal(t, 100, tm.Count())
}

func TestTokenManager_GenerateToken_Concurrent(t *testing.T) {
	tm := NewTokenManager()
	numGoroutines := 50
	tokensPerGoroutine := 20

	var wg sync.WaitGroup
	tokensChan := make(chan string, numGoroutines*tokensPerGoroutine)

	for i := 0; i < numGoroutines; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			for j := 0; j < tokensPerGoroutine; j++ {
				token := tm.GenerateToken()
				tokensChan <- token
			}
		}()
	}

	wg.Wait()
	close(tokensChan)

	// Collect all tokens
	tokens := make([]string, 0, numGoroutines*tokensPerGoroutine)
	for token := range tokensChan {
		tokens = append(tokens, token)
	}

	// Verify all tokens are unique
	seen := make(map[string]bool)
	for _, token := range tokens {
		if seen[token] {
			t.Fatalf("duplicate token found: %s", token)
		}
		seen[token] = true
	}

	assert.Equal(t, numGoroutines*tokensPerGoroutine, tm.Count())
}

func TestTokenManager_ValidateToken_Valid(t *testing.T) {
	tm := NewTokenManager()

	token := tm.GenerateToken()

	assert.True(t, tm.ValidateToken(token))
}

func TestTokenManager_ValidateToken_Invalid(t *testing.T) {
	tm := NewTokenManager()

	assert.False(t, tm.ValidateToken("nonexistent-token"))
}

func TestTokenManager_ValidateToken_Empty(t *testing.T) {
	tm := NewTokenManager()

	assert.False(t, tm.ValidateToken(""))
}

func TestTokenManager_ValidateToken_AfterConsumed(t *testing.T) {
	tm := NewTokenManager()

	token := tm.GenerateToken()

	tm.ConsumeToken(token)

	assert.False(t, tm.ValidateToken(token))
}

func TestTokenManager_ConsumeToken(t *testing.T) {
	tm := NewTokenManager()

	token := tm.GenerateToken()
	assert.Equal(t, 1, tm.Count())

	tm.ConsumeToken(token)

	assert.Equal(t, 0, tm.Count())
	assert.False(t, tm.ValidateToken(token))
}

func TestTokenManager_ConsumeToken_Nonexistent(t *testing.T) {
	tm := NewTokenManager()

	// Should not panic or error
	tm.ConsumeToken("nonexistent-token")
	assert.Equal(t, 0, tm.Count())
}

func TestTokenManager_ConsumeTokens(t *testing.T) {
	tm := NewTokenManager()

	tokens := make([]string, 10)
	for i := 0; i < 10; i++ {
		token := tm.GenerateToken()
		tokens[i] = token
	}

	assert.Equal(t, 10, tm.Count())

	tm.ConsumeTokens(tokens)

	assert.Equal(t, 0, tm.Count())
	for _, token := range tokens {
		assert.False(t, tm.ValidateToken(token))
	}
}

func TestTokenManager_ConsumeTokens_Mixed(t *testing.T) {
	tm := NewTokenManager()

	// Generate 5 tokens
	tokens := make([]string, 5)
	for i := 0; i < 5; i++ {
		token := tm.GenerateToken()
		tokens[i] = token
	}

	// Add some nonexistent tokens to the list
	tokensToConsume := append(tokens, "nonexistent-1", "nonexistent-2")

	tm.ConsumeTokens(tokensToConsume)

	assert.Equal(t, 0, tm.Count())
}

func TestTokenManager_ConsumeTokens_Concurrent(t *testing.T) {
	tm := NewTokenManager()
	numTokens := 1000

	// Generate tokens
	tokens := make([]string, numTokens)
	for i := 0; i < numTokens; i++ {
		token := tm.GenerateToken()
		tokens[i] = token
	}

	assert.Equal(t, numTokens, tm.Count())

	// Consume tokens concurrently
	var wg sync.WaitGroup
	batchSize := 100
	for i := 0; i < numTokens; i += batchSize {
		wg.Add(1)
		end := i + batchSize
		if end > numTokens {
			end = numTokens
		}
		batch := tokens[i:end]
		go func(b []string) {
			defer wg.Done()
			tm.ConsumeTokens(b)
		}(batch)
	}

	wg.Wait()

	assert.Equal(t, 0, tm.Count())
}

func TestTokenManager_RemoveExpired(t *testing.T) {
	tm := NewTokenManager()

	// Generate some tokens with different ages
	oldToken1 := tm.GenerateToken()
	oldToken2 := tm.GenerateToken()

	// Manually backdate these tokens using sync.Map
	if val, ok := tm.tokens.Load(oldToken1); ok {
		token := val.(*AuctionToken)
		token.CreatedAt = time.Now().Add(-2 * time.Minute)
		tm.tokens.Store(oldToken1, token)
	}
	if val, ok := tm.tokens.Load(oldToken2); ok {
		token := val.(*AuctionToken)
		token.CreatedAt = time.Now().Add(-90 * time.Second)
		tm.tokens.Store(oldToken2, token)
	}

	// Generate recent tokens
	recentToken1 := tm.GenerateToken()
	recentToken2 := tm.GenerateToken()

	assert.Equal(t, 4, tm.Count())

	// Remove tokens older than 1 minute
	removed := tm.RemoveExpired(1 * time.Minute)

	assert.Equal(t, 2, removed)
	assert.Equal(t, 2, tm.Count())
	assert.False(t, tm.ValidateToken(oldToken1))
	assert.False(t, tm.ValidateToken(oldToken2))
	assert.True(t, tm.ValidateToken(recentToken1))
	assert.True(t, tm.ValidateToken(recentToken2))
}

func TestTokenManager_RemoveExpired_NoExpired(t *testing.T) {
	tm := NewTokenManager()

	// Generate recent tokens
	for i := 0; i < 5; i++ {
		_ = tm.GenerateToken()
	}

	assert.Equal(t, 5, tm.Count())

	removed := tm.RemoveExpired(1 * time.Minute)

	assert.Equal(t, 0, removed)
	assert.Equal(t, 5, tm.Count())
}

func TestTokenManager_RemoveExpired_AllExpired(t *testing.T) {
	tm := NewTokenManager()

	// Generate tokens and backdate them
	for i := 0; i < 5; i++ {
		token := tm.GenerateToken()

		if val, ok := tm.tokens.Load(token); ok {
			t := val.(*AuctionToken)
			t.CreatedAt = time.Now().Add(-2 * time.Minute)
			tm.tokens.Store(token, t)
		}
	}

	assert.Equal(t, 5, tm.Count())

	removed := tm.RemoveExpired(1 * time.Minute)

	assert.Equal(t, 5, removed)
	assert.Equal(t, 0, tm.Count())
}

func TestTokenManager_RemoveExpired_Concurrent(t *testing.T) {
	tm := NewTokenManager()

	// Generate a mix of old and new tokens concurrently
	var wg sync.WaitGroup
	for i := 0; i < 50; i++ {
		wg.Add(1)
		go func(idx int) {
			defer wg.Done()
			token := tm.GenerateToken()

			// Make half of them old
			if idx%2 == 0 {
				if val, ok := tm.tokens.Load(token); ok {
					t := val.(*AuctionToken)
					t.CreatedAt = time.Now().Add(-2 * time.Minute)
					tm.tokens.Store(token, t)
				}
			}
		}(i)
	}

	wg.Wait()

	assert.Equal(t, 50, tm.Count())

	removed := tm.RemoveExpired(1 * time.Minute)

	assert.Equal(t, 25, removed)
	assert.Equal(t, 25, tm.Count())
}

func TestTokenManager_StartExpirationCleanup(t *testing.T) {
	tm := NewTokenManager()

	// Generate some tokens and backdate them
	oldToken := tm.GenerateToken()
	if val, ok := tm.tokens.Load(oldToken); ok {
		token := val.(*AuctionToken)
		token.CreatedAt = time.Now().Add(-2 * time.Minute)
		tm.tokens.Store(oldToken, token)
	}

	// Generate a recent token
	recentToken := tm.GenerateToken()

	assert.Equal(t, 2, tm.Count())

	// Start cleanup with short interval for testing
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	tm.StartExpirationCleanup(ctx, 100*time.Millisecond, 1*time.Minute)

	// Wait for cleanup to run
	time.Sleep(200 * time.Millisecond)

	// Old token should be removed
	assert.Equal(t, 1, tm.Count())
	assert.False(t, tm.ValidateToken(oldToken))
	assert.True(t, tm.ValidateToken(recentToken))
}

func TestTokenManager_Count(t *testing.T) {
	tm := NewTokenManager()

	assert.Equal(t, 0, tm.Count())

	for i := 1; i <= 10; i++ {
		_ = tm.GenerateToken()
		assert.Equal(t, i, tm.Count())
	}
}

func TestTokenManager_Count_Concurrent(t *testing.T) {
	tm := NewTokenManager()

	var wg sync.WaitGroup
	numGoroutines := 10
	tokensPerGoroutine := 10

	for i := 0; i < numGoroutines; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			for j := 0; j < tokensPerGoroutine; j++ {
				_ = tm.GenerateToken()
			}
		}()
	}

	wg.Wait()

	assert.Equal(t, numGoroutines*tokensPerGoroutine, tm.Count())
}

// Atomic validation and consumption tests

func TestTokenManager_ValidateAndConsumeToken_Valid(t *testing.T) {
	tm := NewTokenManager()

	token := tm.GenerateToken()
	assert.Equal(t, 1, tm.Count())

	// Should succeed and consume
	result := tm.ValidateAndConsumeToken(token)
	assert.True(t, result)
	assert.Equal(t, 0, tm.Count())

	// Token should be gone
	assert.False(t, tm.ValidateToken(token))
}

func TestTokenManager_ValidateAndConsumeToken_Invalid(t *testing.T) {
	tm := NewTokenManager()

	result := tm.ValidateAndConsumeToken("nonexistent-token")
	assert.False(t, result)
}

func TestTokenManager_ValidateAndConsumeToken_Empty(t *testing.T) {
	tm := NewTokenManager()

	result := tm.ValidateAndConsumeToken("")
	assert.False(t, result)
}

func TestTokenManager_ValidateAndConsumeToken_AlreadyConsumed(t *testing.T) {
	tm := NewTokenManager()

	token := tm.GenerateToken()

	// First consumption succeeds
	result1 := tm.ValidateAndConsumeToken(token)
	assert.True(t, result1)

	// Second consumption fails
	result2 := tm.ValidateAndConsumeToken(token)
	assert.False(t, result2)
}

func TestTokenManager_ValidateAndConsumeToken_ConcurrentSameToken(t *testing.T) {
	tm := NewTokenManager()

	token := tm.GenerateToken()

	// Try to consume the same token from multiple goroutines
	var wg sync.WaitGroup
	numGoroutines := 100
	results := make(chan bool, numGoroutines)

	for i := 0; i < numGoroutines; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			result := tm.ValidateAndConsumeToken(token)
			results <- result
		}()
	}

	wg.Wait()
	close(results)

	// Count how many succeeded
	successCount := 0
	for result := range results {
		if result {
			successCount++
		}
	}

	// CRITICAL: Only ONE goroutine should have succeeded (atomic operation)
	assert.Equal(t, 1, successCount)

	// Token should be gone
	assert.Equal(t, 0, tm.Count())
	assert.False(t, tm.ValidateToken(token))
}

func TestTokenManager_ValidateAndConsumeToken_ConcurrentDifferentTokens(t *testing.T) {
	tm := NewTokenManager()

	// Generate many tokens
	numTokens := 100
	tokens := make([]string, numTokens)
	for i := 0; i < numTokens; i++ {
		token := tm.GenerateToken()
		tokens[i] = token
	}

	assert.Equal(t, numTokens, tm.Count())

	// Consume them all concurrently
	var wg sync.WaitGroup
	successCount := make(chan bool, numTokens)

	for _, token := range tokens {
		wg.Add(1)
		go func(t string) {
			defer wg.Done()
			result := tm.ValidateAndConsumeToken(t)
			successCount <- result
		}(token)
	}

	wg.Wait()
	close(successCount)

	// All should succeed (different tokens)
	count := 0
	for result := range successCount {
		if result {
			count++
		}
	}
	assert.Equal(t, numTokens, count)

	// All tokens should be gone
	assert.Equal(t, 0, tm.Count())
}
