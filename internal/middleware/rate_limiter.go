package middleware

import (
	"sync"
	"time"
)

// RateLimiter implements a simple in-memory rate limiter
type RateLimiter struct {
	userLimits map[int64]*userLimit
	ipLimits   map[string]*ipLimit
	mu         sync.RWMutex

	userMaxRequests int
	ipMaxRequests   int
	window          time.Duration
}

type userLimit struct {
	requests  int
	resetTime time.Time
}

type ipLimit struct {
	requests  int
	resetTime time.Time
}

// NewRateLimiter creates a new rate limiter
func NewRateLimiter(userMaxRequests, ipMaxRequests int, window time.Duration) *RateLimiter {
	rl := &RateLimiter{
		userLimits:      make(map[int64]*userLimit),
		ipLimits:        make(map[string]*ipLimit),
		userMaxRequests: userMaxRequests,
		ipMaxRequests:   ipMaxRequests,
		window:          window,
	}

	// Start cleanup goroutine
	go rl.cleanup()

	return rl
}

// CheckUserLimit checks if user has exceeded rate limit
func (rl *RateLimiter) CheckUserLimit(userID int64) bool {
	rl.mu.Lock()
	defer rl.mu.Unlock()

	now := time.Now()

	// Get or create user limit
	limit, exists := rl.userLimits[userID]
	if !exists || now.After(limit.resetTime) {
		rl.userLimits[userID] = &userLimit{
			requests:  1,
			resetTime: now.Add(rl.window),
		}
		return true
	}

	// Check if limit exceeded
	if limit.requests >= rl.userMaxRequests {
		return false
	}

	// Increment counter
	limit.requests++
	return true
}

// CheckIPLimit checks if IP has exceeded rate limit
func (rl *RateLimiter) CheckIPLimit(ip string) bool {
	rl.mu.Lock()
	defer rl.mu.Unlock()

	now := time.Now()

	// Get or create IP limit
	limit, exists := rl.ipLimits[ip]
	if !exists || now.After(limit.resetTime) {
		rl.ipLimits[ip] = &ipLimit{
			requests:  1,
			resetTime: now.Add(rl.window),
		}
		return true
	}

	// Check if limit exceeded
	if limit.requests >= rl.ipMaxRequests {
		return false
	}

	// Increment counter
	limit.requests++
	return true
}

// GetUserRemaining returns remaining requests for user
func (rl *RateLimiter) GetUserRemaining(userID int64) int {
	rl.mu.RLock()
	defer rl.mu.RUnlock()

	limit, exists := rl.userLimits[userID]
	if !exists || time.Now().After(limit.resetTime) {
		return rl.userMaxRequests
	}

	remaining := rl.userMaxRequests - limit.requests
	if remaining < 0 {
		return 0
	}
	return remaining
}

// GetIPRemaining returns remaining requests for IP
func (rl *RateLimiter) GetIPRemaining(ip string) int {
	rl.mu.RLock()
	defer rl.mu.RUnlock()

	limit, exists := rl.ipLimits[ip]
	if !exists || time.Now().After(limit.resetTime) {
		return rl.ipMaxRequests
	}

	remaining := rl.ipMaxRequests - limit.requests
	if remaining < 0 {
		return 0
	}
	return remaining
}

// cleanup removes expired entries
func (rl *RateLimiter) cleanup() {
	ticker := time.NewTicker(5 * time.Minute)
	defer ticker.Stop()

	for range ticker.C {
		rl.mu.Lock()
		now := time.Now()

		// Clean user limits
		for userID, limit := range rl.userLimits {
			if now.After(limit.resetTime) {
				delete(rl.userLimits, userID)
			}
		}

		// Clean IP limits
		for ip, limit := range rl.ipLimits {
			if now.After(limit.resetTime) {
				delete(rl.ipLimits, ip)
			}
		}

		rl.mu.Unlock()
	}
}

// Reset clears all rate limits (useful for testing)
func (rl *RateLimiter) Reset() {
	rl.mu.Lock()
	defer rl.mu.Unlock()

	rl.userLimits = make(map[int64]*userLimit)
	rl.ipLimits = make(map[string]*ipLimit)
}
