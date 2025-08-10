package middleware

import (
	"log"
	"net/http"
	"sync"
	"time"

	"github.com/labstack/echo/v4"
)

// RateLimiter implements token bucket algorithm for rate limiting
type RateLimiter struct {
	visitors map[string]*Visitor
	mu       sync.RWMutex
	rate     int           // requests per window
	window   time.Duration // time window
}

type Visitor struct {
	tokens    int
	lastSeen  time.Time
	blocked   bool
	blockUntil time.Time
}

// NewRateLimiter creates a new rate limiter
func NewRateLimiter(rate int, window time.Duration) *RateLimiter {
	rl := &RateLimiter{
		visitors: make(map[string]*Visitor),
		rate:     rate,
		window:   window,
	}

	// Cleanup goroutine
	go rl.cleanup()
	
	return rl
}

// RateLimitMiddleware returns Echo middleware for rate limiting
func (rl *RateLimiter) RateLimitMiddleware() echo.MiddlewareFunc {
	return func(next echo.HandlerFunc) echo.HandlerFunc {
		return func(c echo.Context) error {
			ip := c.RealIP()
			
			// Check if request should be blocked
			if blocked, resetTime := rl.isBlocked(ip); blocked {
				log.Printf("RATE LIMIT: Blocked request from IP %s (reset in %v)", ip, time.Until(resetTime))
				
				return c.JSON(http.StatusTooManyRequests, map[string]interface{}{
					"error": "Rate limit exceeded",
					"retry_after": int(time.Until(resetTime).Seconds()),
				})
			}

			// Allow request
			rl.consume(ip)
			
			return next(c)
		}
	}
}

// isBlocked checks if IP should be blocked and returns reset time
func (rl *RateLimiter) isBlocked(ip string) (bool, time.Time) {
	rl.mu.Lock()
	defer rl.mu.Unlock()

	visitor, exists := rl.visitors[ip]
	if !exists {
		// New visitor
		rl.visitors[ip] = &Visitor{
			tokens:   rl.rate - 1, // Consume 1 token
			lastSeen: time.Now(),
			blocked:  false,
		}
		return false, time.Time{}
	}

	now := time.Now()

	// Check if still in block period
	if visitor.blocked && now.Before(visitor.blockUntil) {
		return true, visitor.blockUntil
	}

	// Reset if block period is over
	if visitor.blocked && now.After(visitor.blockUntil) {
		visitor.blocked = false
		visitor.tokens = rl.rate
		visitor.lastSeen = now
	}

	// Refill tokens based on time passed
	timePassed := now.Sub(visitor.lastSeen)
	tokensToAdd := int(timePassed / rl.window * time.Duration(rl.rate))
	visitor.tokens += tokensToAdd
	
	if visitor.tokens > rl.rate {
		visitor.tokens = rl.rate
	}

	visitor.lastSeen = now

	// Check if should be blocked
	if visitor.tokens <= 0 {
		visitor.blocked = true
		visitor.blockUntil = now.Add(rl.window)
		log.Printf("SECURITY: Rate limiting activated for IP %s", ip)
		return true, visitor.blockUntil
	}

	return false, time.Time{}
}

// consume reduces available tokens for IP
func (rl *RateLimiter) consume(ip string) {
	rl.mu.Lock()
	defer rl.mu.Unlock()
	
	if visitor, exists := rl.visitors[ip]; exists {
		visitor.tokens--
		visitor.lastSeen = time.Now()
	}
}

// cleanup removes old visitors to prevent memory leaks
func (rl *RateLimiter) cleanup() {
	for {
		time.Sleep(time.Hour) // Cleanup every hour
		
		rl.mu.Lock()
		now := time.Now()
		for ip, visitor := range rl.visitors {
			// Remove visitors not seen for 2 hours
			if now.Sub(visitor.lastSeen) > 2*time.Hour {
				delete(rl.visitors, ip)
			}
		}
		rl.mu.Unlock()
	}
}

// GetVisitorStats returns current visitor statistics (for monitoring)
func (rl *RateLimiter) GetVisitorStats() map[string]interface{} {
	rl.mu.RLock()
	defer rl.mu.RUnlock()
	
	stats := map[string]interface{}{
		"total_visitors": len(rl.visitors),
		"blocked_count":  0,
		"active_count":   0,
	}
	
	blockedCount := 0
	activeCount := 0
	now := time.Now()
	
	for _, visitor := range rl.visitors {
		if visitor.blocked && now.Before(visitor.blockUntil) {
			blockedCount++
		} else if now.Sub(visitor.lastSeen) < time.Hour {
			activeCount++
		}
	}
	
	stats["blocked_count"] = blockedCount
	stats["active_count"] = activeCount
	
	return stats
}

// Global rate limiters for different endpoints
var (
	// General API rate limiter: 60 requests per minute
	GeneralLimiter = NewRateLimiter(60, time.Minute)
	
	// Payment API rate limiter: 10 requests per minute (more restrictive)
	PaymentLimiter = NewRateLimiter(10, time.Minute)
	
	// Auth rate limiter: 5 attempts per minute
	AuthLimiter = NewRateLimiter(5, time.Minute)
	
	// Webhook rate limiter: 100 requests per minute (for legitimate webhooks)
	WebhookLimiter = NewRateLimiter(100, time.Minute)
)

// Specific middleware functions
func GeneralRateLimit() echo.MiddlewareFunc {
	return GeneralLimiter.RateLimitMiddleware()
}

func PaymentRateLimit() echo.MiddlewareFunc {
	return PaymentLimiter.RateLimitMiddleware()
}

func AuthRateLimit() echo.MiddlewareFunc {
	return AuthLimiter.RateLimitMiddleware()
}

func WebhookRateLimit() echo.MiddlewareFunc {
	return WebhookLimiter.RateLimitMiddleware()
}