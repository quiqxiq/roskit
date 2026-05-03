package middleware

import (
	"net/http"
	"sync"
	"time"

	"github.com/gin-gonic/gin"
)

type rateLimiter struct {
	mu       sync.Mutex
	visitors map[string]*visitorState
	rate     int
	window   time.Duration
}

type visitorState struct {
	count    int
	windowAt time.Time
}

func newRateLimiter(rate int, window time.Duration) *rateLimiter {
	rl := &rateLimiter{
		visitors: make(map[string]*visitorState),
		rate:     rate,
		window:   window,
	}
	go rl.cleanup()
	return rl
}

func (rl *rateLimiter) allow(ip string) bool {
	rl.mu.Lock()
	defer rl.mu.Unlock()

	now := time.Now()
	v, ok := rl.visitors[ip]
	if !ok || now.After(v.windowAt.Add(rl.window)) {
		rl.visitors[ip] = &visitorState{count: 1, windowAt: now}
		return true
	}
	if v.count >= rl.rate {
		return false
	}
	v.count++
	return true
}

func (rl *rateLimiter) cleanup() {
	ticker := time.NewTicker(5 * time.Minute)
	defer ticker.Stop()
	for range ticker.C {
		rl.mu.Lock()
		now := time.Now()
		for ip, v := range rl.visitors {
			if now.After(v.windowAt.Add(rl.window * 2)) {
				delete(rl.visitors, ip)
			}
		}
		rl.mu.Unlock()
	}
}

func RateLimit(rate int, window time.Duration) gin.HandlerFunc {
	rl := newRateLimiter(rate, window)
	return func(c *gin.Context) {
		if !rl.allow(c.ClientIP()) {
			c.AbortWithStatusJSON(http.StatusTooManyRequests, gin.H{
				"data":  nil,
				"error": "too many requests, please try again later",
			})
			return
		}
		c.Next()
	}
}
