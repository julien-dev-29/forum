package ratelimit

import (
	"fmt"
	"net"
	"net/http"
	"strings"
	"sync"
	"time"
)

type bucket struct {
	tokens     float64
	lastRefill time.Time
}

type RateLimiter struct {
	mu      sync.Mutex
	buckets map[string]*bucket
	rate    float64
	burst   float64
	window  time.Duration
	stop    chan struct{}
}

func New(rate int, window time.Duration, burst int) *RateLimiter {
	rl := &RateLimiter{
		buckets: make(map[string]*bucket),
		rate:    float64(rate),
		burst:   float64(burst),
		window:  window,
		stop:    make(chan struct{}),
	}
	go rl.cleanup(5 * time.Minute)
	return rl
}

func (rl *RateLimiter) Allow(key string) (bool, time.Duration) {
	rl.mu.Lock()
	defer rl.mu.Unlock()

	b, ok := rl.buckets[key]
	if !ok {
		b = &bucket{tokens: rl.burst, lastRefill: time.Now()}
		rl.buckets[key] = b
	}

	now := time.Now()
	elapsed := now.Sub(b.lastRefill)
	refillRate := rl.rate / rl.window.Seconds()
	b.tokens = min(rl.burst, b.tokens+elapsed.Seconds()*refillRate)
	b.lastRefill = now

	if b.tokens >= 1 {
		b.tokens--
		return true, 0
	}

	retryAfter := time.Duration(float64(rl.window) / rl.rate)
	return false, max(retryAfter, time.Second)
}

func (rl *RateLimiter) Middleware() func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			allowed, retryAfter := rl.Allow(IPKey(r))
			if !allowed {
				w.Header().Set("Retry-After", fmt.Sprintf("%.0f", retryAfter.Seconds()))
				w.Header().Set("Content-Type", "text/plain; charset=utf-8")
				w.WriteHeader(http.StatusTooManyRequests)
				fmt.Fprintln(w, "429 Too Many Requests")
				return
			}
			next.ServeHTTP(w, r)
		})
	}
}

func (rl *RateLimiter) Stop() {
	close(rl.stop)
}

func (rl *RateLimiter) cleanup(interval time.Duration) {
	ticker := time.NewTicker(interval)
	defer ticker.Stop()
	for {
		select {
		case <-ticker.C:
			rl.mu.Lock()
			now := time.Now()
			for k, b := range rl.buckets {
				if now.Sub(b.lastRefill) > 10*time.Minute {
					delete(rl.buckets, k)
				}
			}
			rl.mu.Unlock()
		case <-rl.stop:
			return
		}
	}
}

func IPKey(r *http.Request) string {
	if fwd := r.Header.Get("X-Forwarded-For"); fwd != "" {
		parts := strings.Split(fwd, ",")
		return strings.TrimSpace(parts[0])
	}
	if fwd := r.Header.Get("X-Real-IP"); fwd != "" {
		return strings.TrimSpace(fwd)
	}
	host, _, err := net.SplitHostPort(r.RemoteAddr)
	if err != nil {
		return r.RemoteAddr
	}
	return host
}
