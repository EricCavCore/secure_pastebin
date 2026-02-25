/*********************************
 *  File     : ratelimit.go
 *  Purpose  : Per-IP token bucket rate limiter
 *  Authors  : Eric Caverly
 */

package main

import (
	"net/http"
	"sync"
	"time"
)

type visitor struct {
	tokens    float64
	lastSeen  time.Time
	maxTokens float64
	rate      float64
}

type RateLimiter struct {
	mu       sync.Mutex
	visitors map[string]*visitor
	rate     float64
	burst    float64
}

func NewRateLimiter(rate float64, burst float64) *RateLimiter {
	rl := &RateLimiter{
		visitors: make(map[string]*visitor),
		rate:     rate,
		burst:    burst,
	}
	go func() {
		for {
			time.Sleep(time.Minute)
			rl.mu.Lock()
			for ip, v := range rl.visitors {
				if time.Since(v.lastSeen) > 3*time.Minute {
					delete(rl.visitors, ip)
				}
			}
			rl.mu.Unlock()
		}
	}()
	return rl
}

func (rl *RateLimiter) Allow(ip string) bool {
	rl.mu.Lock()
	defer rl.mu.Unlock()

	v, exists := rl.visitors[ip]
	if !exists {
		rl.visitors[ip] = &visitor{
			tokens:    rl.burst - 1,
			lastSeen:  time.Now(),
			maxTokens: rl.burst,
			rate:      rl.rate,
		}
		return true
	}

	elapsed := time.Since(v.lastSeen).Seconds()
	v.lastSeen = time.Now()
	v.tokens += elapsed * v.rate
	if v.tokens > v.maxTokens {
		v.tokens = v.maxTokens
	}

	if v.tokens < 1 {
		return false
	}

	v.tokens--
	return true
}

func rateLimitedHandler(rl *RateLimiter, handler http.HandlerFunc) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		ip := getClientIP(r)
		if !rl.Allow(ip) {
			write_error(w, "Rate limit exceeded. Please try again later.", http.StatusTooManyRequests)
			return
		}
		handler(w, r)
	}
}
