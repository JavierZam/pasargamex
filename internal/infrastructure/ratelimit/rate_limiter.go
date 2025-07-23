package ratelimit

import (
	"sync"
	"time"
)

// TokenBucket represents a token bucket for rate limiting
type TokenBucket struct {
	tokens      int           // Current tokens
	maxTokens   int           // Maximum tokens in bucket
	refillRate  int           // Tokens to add per refill interval
	refillTime  time.Duration // Refill interval
	lastRefill  time.Time     // Last refill time
	mutex       sync.Mutex    // Thread safety
}

// RateLimiter manages rate limiting for different users and actions
type RateLimiter struct {
	buckets map[string]*TokenBucket
	mutex   sync.RWMutex
}

// NewRateLimiter creates a new rate limiter
func NewRateLimiter() *RateLimiter {
	return &RateLimiter{
		buckets: make(map[string]*TokenBucket),
	}
}

// NewTokenBucket creates a new token bucket
func NewTokenBucket(maxTokens, refillRate int, refillTime time.Duration) *TokenBucket {
	return &TokenBucket{
		tokens:     maxTokens,
		maxTokens:  maxTokens,
		refillRate: refillRate,
		refillTime: refillTime,
		lastRefill: time.Now(),
	}
}

// Allow checks if an action is allowed and consumes a token if so
func (tb *TokenBucket) Allow() (bool, time.Duration) {
	tb.mutex.Lock()
	defer tb.mutex.Unlock()

	now := time.Now()
	
	// Calculate tokens to add based on time elapsed
	elapsed := now.Sub(tb.lastRefill)
	tokensToAdd := int(elapsed / tb.refillTime) * tb.refillRate
	
	if tokensToAdd > 0 {
		tb.tokens += tokensToAdd
		if tb.tokens > tb.maxTokens {
			tb.tokens = tb.maxTokens
		}
		tb.lastRefill = now
	}

	// Check if we have tokens available
	if tb.tokens > 0 {
		tb.tokens--
		return true, 0
	}

	// Calculate wait time until next token is available
	nextRefill := tb.lastRefill.Add(tb.refillTime)
	waitTime := nextRefill.Sub(now)
	return false, waitTime
}

// GetTokens returns current token count
func (tb *TokenBucket) GetTokens() int {
	tb.mutex.Lock()
	defer tb.mutex.Unlock()
	return tb.tokens
}

// Allow checks if a user action is allowed
func (rl *RateLimiter) Allow(userID, action string) (bool, time.Duration) {
	key := userID + ":" + action
	
	rl.mutex.RLock()
	bucket, exists := rl.buckets[key]
	rl.mutex.RUnlock()

	if !exists {
		rl.mutex.Lock()
		// Double-check pattern
		if bucket, exists = rl.buckets[key]; !exists {
			// Create bucket based on action type
			switch action {
			case "send_message":
				// Allow 10 messages per minute (1 token per 6 seconds)
				bucket = NewTokenBucket(10, 1, 6*time.Second)
			case "create_chat":
				// Allow 5 chat creations per hour (1 token per 12 minutes)
				bucket = NewTokenBucket(5, 1, 12*time.Minute)
			case "typing":
				// Allow 30 typing events per minute (1 token per 2 seconds)
				bucket = NewTokenBucket(30, 1, 2*time.Second)
			default:
				// Default rate limit: 20 actions per minute
				bucket = NewTokenBucket(20, 1, 3*time.Second)
			}
			rl.buckets[key] = bucket
		}
		rl.mutex.Unlock()
	}

	return bucket.Allow()
}

// GetStatus returns current rate limit status for a user action
func (rl *RateLimiter) GetStatus(userID, action string) (tokens int, maxTokens int) {
	key := userID + ":" + action
	
	rl.mutex.RLock()
	bucket, exists := rl.buckets[key]
	rl.mutex.RUnlock()

	if !exists {
		return 0, 0
	}

	return bucket.GetTokens(), bucket.maxTokens
}

// Cleanup removes old buckets that haven't been used recently
func (rl *RateLimiter) Cleanup() {
	rl.mutex.Lock()
	defer rl.mutex.Unlock()

	now := time.Now()
	for key, bucket := range rl.buckets {
		// Remove buckets that haven't been accessed for 1 hour
		if now.Sub(bucket.lastRefill) > time.Hour {
			delete(rl.buckets, key)
		}
	}
}

// StartCleanupRoutine starts a cleanup routine that runs periodically
func (rl *RateLimiter) StartCleanupRoutine() {
	go func() {
		ticker := time.NewTicker(30 * time.Minute)
		defer ticker.Stop()
		
		for range ticker.C {
			rl.Cleanup()
		}
	}()
}