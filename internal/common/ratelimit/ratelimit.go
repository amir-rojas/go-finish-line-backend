// Package ratelimit provides a per-client-IP rate limiting middleware.
package ratelimit

import (
	"sync"

	"github.com/gin-gonic/gin"
	"golang.org/x/time/rate"

	"finish-line/internal/common/httpx"
)

// store keeps one token-bucket limiter per client IP.
//
// Note: the map is unbounded and lives for the process lifetime — fine for a
// single instance, but a multi-instance deployment should move this to a
// shared store (e.g. Redis), and the map should eventually be evicted.
type store struct {
	mu       sync.Mutex
	limiters map[string]*rate.Limiter
	rps      rate.Limit
	burst    int
}

// PerIP limits each client IP to rps requests per second with the given burst.
func PerIP(rps rate.Limit, burst int) gin.HandlerFunc {
	s := &store{
		limiters: make(map[string]*rate.Limiter),
		rps:      rps,
		burst:    burst,
	}
	return func(c *gin.Context) {
		if !s.limiterFor(c.ClientIP()).Allow() {
			httpx.TooManyRequests(c, "too many requests, slow down")
			c.Abort()
			return
		}
		c.Next()
	}
}

func (s *store) limiterFor(ip string) *rate.Limiter {
	s.mu.Lock()
	defer s.mu.Unlock()

	l, ok := s.limiters[ip]
	if !ok {
		l = rate.NewLimiter(s.rps, s.burst)
		s.limiters[ip] = l
	}
	return l
}
