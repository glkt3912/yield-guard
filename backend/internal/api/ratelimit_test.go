package api

import (
	"net/http"
	"net/http/httptest"
	"strconv"
	"testing"
	"time"

	"github.com/gin-gonic/gin"
	"golang.org/x/time/rate"
)

func TestRateLimiter_AllowsUnderLimit(t *testing.T) {
	gin.SetMode(gin.TestMode)
	rl := newRateLimiter(rate.Every(time.Second), 5)
	t.Cleanup(rl.shutdown)

	r := gin.New()
	r.GET("/test", rl.middleware(), func(c *gin.Context) {
		c.Status(http.StatusOK)
	})

	for i := range 5 {
		req := httptest.NewRequest(http.MethodGet, "/test", nil)
		req.RemoteAddr = "1.2.3.4:1234"
		w := httptest.NewRecorder()
		r.ServeHTTP(w, req)
		if w.Code != http.StatusOK {
			t.Fatalf("request %d: want 200, got %d", i+1, w.Code)
		}
	}
}

func TestRateLimiter_BlocksOverLimit(t *testing.T) {
	gin.SetMode(gin.TestMode)
	// burst=2: 3rd request should trigger 429
	rl := newRateLimiter(rate.Every(time.Hour), 2)
	t.Cleanup(rl.shutdown)

	r := gin.New()
	r.GET("/test", rl.middleware(), func(c *gin.Context) {
		c.Status(http.StatusOK)
	})

	send := func() int {
		req := httptest.NewRequest(http.MethodGet, "/test", nil)
		req.RemoteAddr = "1.2.3.4:1234"
		w := httptest.NewRecorder()
		r.ServeHTTP(w, req)
		return w.Code
	}

	if got := send(); got != http.StatusOK {
		t.Fatalf("request 1: want 200, got %d", got)
	}
	if got := send(); got != http.StatusOK {
		t.Fatalf("request 2: want 200, got %d", got)
	}
	if got := send(); got != http.StatusTooManyRequests {
		t.Fatalf("request 3: want 429, got %d", got)
	}
}

func TestRateLimiter_RetryAfterHeader(t *testing.T) {
	gin.SetMode(gin.TestMode)
	// 1 token per hour, burst=1: 2nd request must wait ~3600s
	rl := newRateLimiter(rate.Every(time.Hour), 1)
	t.Cleanup(rl.shutdown)

	r := gin.New()
	r.GET("/test", rl.middleware(), func(c *gin.Context) {
		c.Status(http.StatusOK)
	})

	send := func() *httptest.ResponseRecorder {
		req := httptest.NewRequest(http.MethodGet, "/test", nil)
		req.RemoteAddr = "5.6.7.8:1234"
		w := httptest.NewRecorder()
		r.ServeHTTP(w, req)
		return w
	}

	send() // consume burst
	w := send()
	if w.Code != http.StatusTooManyRequests {
		t.Fatalf("want 429, got %d", w.Code)
	}
	retryAfter := w.Header().Get("Retry-After")
	secs, err := strconv.Atoi(retryAfter)
	if err != nil || secs <= 0 {
		t.Fatalf("Retry-After must be a positive integer, got %q", retryAfter)
	}
}

func TestRateLimiter_IsolatesPerIP(t *testing.T) {
	gin.SetMode(gin.TestMode)
	rl := newRateLimiter(rate.Every(time.Hour), 1)
	t.Cleanup(rl.shutdown)

	r := gin.New()
	r.GET("/test", rl.middleware(), func(c *gin.Context) {
		c.Status(http.StatusOK)
	})

	sendFrom := func(ip string) int {
		req := httptest.NewRequest(http.MethodGet, "/test", nil)
		req.RemoteAddr = ip + ":1234"
		w := httptest.NewRecorder()
		r.ServeHTTP(w, req)
		return w.Code
	}

	// exhaust IP A
	sendFrom("10.0.0.1")
	if got := sendFrom("10.0.0.1"); got != http.StatusTooManyRequests {
		t.Fatalf("IP A second request: want 429, got %d", got)
	}
	// IP B should still be allowed
	if got := sendFrom("10.0.0.2"); got != http.StatusOK {
		t.Fatalf("IP B first request: want 200, got %d", got)
	}
}

func TestRealIP(t *testing.T) {
	cases := []struct {
		name       string
		remoteAddr string
		xff        string
		want       string
	}{
		{
			name:       "no XFF — use RemoteAddr",
			remoteAddr: "1.2.3.4:5678",
			xff:        "",
			want:       "1.2.3.4",
		},
		{
			name:       "single XFF entry",
			remoteAddr: "10.0.0.1:1234",
			xff:        "203.0.113.1",
			want:       "203.0.113.1",
		},
		{
			name:       "Cloud Run: client spoofs XFF, LB appends real IP at end",
			remoteAddr: "10.0.0.1:1234",
			xff:        "spoofed-ip, 203.0.113.99",
			want:       "203.0.113.99",
		},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			gin.SetMode(gin.TestMode)
			var got string
			r := gin.New()
			r.GET("/", func(c *gin.Context) {
				got = realIP(c)
				c.Status(http.StatusOK)
			})
			req := httptest.NewRequest(http.MethodGet, "/", nil)
			req.RemoteAddr = tc.remoteAddr
			if tc.xff != "" {
				req.Header.Set("X-Forwarded-For", tc.xff)
			}
			r.ServeHTTP(httptest.NewRecorder(), req)
			if got != tc.want {
				t.Errorf("realIP = %q, want %q", got, tc.want)
			}
		})
	}
}
