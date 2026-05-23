package auth

import (
	"net/http"
	"net/http/httptest"
	"testing"
	"time"
)

func TestRateLimiterAllowsUnderLimit(t *testing.T) {
	limiter := NewRateLimiter(5, time.Minute)
	for i := 0; i < 5; i++ {
		if !limiter.Allow("192.0.2.1") {
			t.Fatalf("attempt %d should be allowed", i+1)
		}
	}
}

func TestRateLimiterBlocksOverLimit(t *testing.T) {
	limiter := NewRateLimiter(3, time.Minute)
	for i := 0; i < 3; i++ {
		limiter.Allow("192.0.2.2")
	}
	if limiter.Allow("192.0.2.2") {
		t.Error("4th attempt should be blocked")
	}
}

func TestRateLimiterIsolatesKeys(t *testing.T) {
	limiter := NewRateLimiter(2, time.Minute)
	limiter.Allow("10.0.0.1")
	limiter.Allow("10.0.0.1")

	if !limiter.Allow("10.0.0.2") {
		t.Error("different IP should not be affected")
	}
}

func TestRateLimiterExpiredAttemptsReset(t *testing.T) {
	limiter := NewRateLimiter(2, 10*time.Millisecond)
	limiter.Allow("192.0.2.3")
	limiter.Allow("192.0.2.3")

	time.Sleep(20 * time.Millisecond)

	if !limiter.Allow("192.0.2.3") {
		t.Error("should allow after window expires")
	}
}

func TestRateLimiterDefaultsOnZeroValues(t *testing.T) {
	limiter := NewRateLimiter(0, 0)
	if limiter.limit != 5 {
		t.Errorf("default limit = %d", limiter.limit)
	}
	if limiter.window != time.Minute {
		t.Errorf("default window = %v", limiter.window)
	}
}

func TestRateLimiterMiddlewareBlocks(t *testing.T) {
	limiter := NewRateLimiter(1, time.Minute)

	calls := 0
	handler := limiter.Middleware(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		calls++
		w.WriteHeader(http.StatusOK)
	}))

	send := func() int {
		req := httptest.NewRequest(http.MethodPost, "/admin/login", nil)
		req.RemoteAddr = "10.1.1.1:5000"
		w := httptest.NewRecorder()
		handler.ServeHTTP(w, req)
		return w.Code
	}

	if code := send(); code != http.StatusOK {
		t.Fatalf("first request status = %d", code)
	}
	if code := send(); code != http.StatusTooManyRequests {
		t.Fatalf("second request status = %d", code)
	}
	if calls != 1 {
		t.Fatalf("handler called %d times", calls)
	}
}

func TestRateLimiterMiddlewareReadsForwardedFor(t *testing.T) {
	limiter := NewRateLimiter(1, time.Minute)

	handler := limiter.Middleware(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	}))

	send := func(forwarded string) int {
		req := httptest.NewRequest(http.MethodPost, "/login", nil)
		req.Header.Set("X-Forwarded-For", forwarded)
		req.RemoteAddr = "10.0.0.1:9000"
		w := httptest.NewRecorder()
		handler.ServeHTTP(w, req)
		return w.Code
	}

	if code := send("203.0.113.5"); code != http.StatusOK {
		t.Fatalf("first = %d", code)
	}
	if code := send("203.0.113.5"); code != http.StatusTooManyRequests {
		t.Fatalf("second = %d", code)
	}
	if code := send("203.0.113.6"); code != http.StatusOK {
		t.Fatalf("different IP should be allowed = %d", code)
	}
}
