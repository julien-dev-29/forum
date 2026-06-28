package ratelimit

import (
	"net/http"
	"net/http/httptest"
	"sync"
	"testing"
	"time"
)

func TestRateLimiter_Allow(t *testing.T) {
	rl := New(10, time.Second, 10)

	for i := 0; i < 10; i++ {
		allowed, _ := rl.Allow("test")
		if !allowed {
			t.Fatalf("request %d: expected allowed, got denied", i+1)
		}
	}

	allowed, retryAfter := rl.Allow("test")
	if allowed {
		t.Fatal("expected denied after exceeding rate")
	}
	if retryAfter <= 0 {
		t.Fatal("expected positive retry-after duration")
	}
}

func TestRateLimiter_Refill(t *testing.T) {
	rl := New(100, time.Second, 100)

	for i := 0; i < 100; i++ {
		rl.Allow("refill-test")
	}

	time.Sleep(50 * time.Millisecond)

	allowed, _ := rl.Allow("refill-test")
	if !allowed {
		t.Fatal("expected token refill after waiting")
	}
}

func TestRateLimiter_ConcurrentAccess(t *testing.T) {
	rl := New(1000, time.Second, 1000)

	var wg sync.WaitGroup
	for i := 0; i < 10; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			for j := 0; j < 100; j++ {
				rl.Allow("concurrent")
			}
		}()
	}
	wg.Wait()
}

func TestRateLimiter_Middleware(t *testing.T) {
	rl := New(2, time.Minute, 2)
	handler := rl.Middleware()(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	}))

	req := httptest.NewRequest("GET", "/", nil)

	rr := httptest.NewRecorder()
	handler.ServeHTTP(rr, req)
	if rr.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d", rr.Code)
	}

	rr = httptest.NewRecorder()
	handler.ServeHTTP(rr, req)
	if rr.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d", rr.Code)
	}

	rr = httptest.NewRecorder()
	handler.ServeHTTP(rr, req)
	if rr.Code != http.StatusTooManyRequests {
		t.Fatalf("expected 429, got %d", rr.Code)
	}
}

func TestRateLimiter_IsolatedKeys(t *testing.T) {
	rl := New(1, time.Minute, 1)

	allowed, _ := rl.Allow("key-a")
	if !allowed {
		t.Fatal("key-a should be allowed")
	}

	allowed, _ = rl.Allow("key-b")
	if !allowed {
		t.Fatal("key-b should be allowed (different key)")
	}

	allowed, _ = rl.Allow("key-a")
	if allowed {
		t.Fatal("key-a should be denied (exhausted)")
	}
}

func TestRateLimiter_Burst(t *testing.T) {
	rl := New(5, time.Minute, 10)

	for i := 0; i < 10; i++ {
		allowed, _ := rl.Allow("burst")
		if !allowed {
			t.Fatalf("burst request %d: expected allowed", i+1)
		}
	}

	allowed, _ := rl.Allow("burst")
	if allowed {
		t.Fatal("expected denied after burst consumed")
	}
}

func TestIPKey(t *testing.T) {
	tests := []struct {
		name     string
		remote   string
		headers  map[string]string
		expected string
	}{
		{"direct IP", "192.168.1.1:12345", nil, "192.168.1.1"},
		{"X-Forwarded-For", "10.0.0.1:9999", map[string]string{"X-Forwarded-For": "203.0.113.1"}, "203.0.113.1"},
		{"X-Forwarded-For chain", "10.0.0.1:9999", map[string]string{"X-Forwarded-For": "203.0.113.1, 10.0.0.1"}, "203.0.113.1"},
		{"X-Real-IP", "10.0.0.1:9999", map[string]string{"X-Real-IP": "198.51.100.1"}, "198.51.100.1"},
		{"no port", "::1", nil, "::1"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			req := httptest.NewRequest("GET", "/", nil)
			req.RemoteAddr = tt.remote
			for k, v := range tt.headers {
				req.Header.Set(k, v)
			}
			ip := IPKey(req)
			if ip != tt.expected {
				t.Fatalf("expected %q, got %q", tt.expected, ip)
			}
		})
	}
}
