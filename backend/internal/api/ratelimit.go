package api

import (
	"net"
	"net/http"
	"strconv"
	"strings"
	"sync"
	"time"

	"github.com/gin-gonic/gin"
	"golang.org/x/time/rate"
)

type ipLimiter struct {
	limiter  *rate.Limiter
	lastSeen time.Time
}

type rateLimiter struct {
	mu       sync.Mutex
	limiters map[string]*ipLimiter
	r        rate.Limit
	burst    int
	stop     chan struct{}
}

func newRateLimiter(r rate.Limit, burst int) *rateLimiter {
	rl := &rateLimiter{
		limiters: make(map[string]*ipLimiter),
		r:        r,
		burst:    burst,
		stop:     make(chan struct{}),
	}
	t := time.NewTicker(5 * time.Minute)
	go func() {
		for {
			select {
			case <-t.C:
				rl.purge()
			case <-rl.stop:
				t.Stop()
				return
			}
		}
	}()
	return rl
}

// shutdown stops the background cleanup goroutine.
func (rl *rateLimiter) shutdown() {
	close(rl.stop)
}

func (rl *rateLimiter) get(ip string) *rate.Limiter {
	rl.mu.Lock()
	defer rl.mu.Unlock()
	entry, ok := rl.limiters[ip]
	if !ok {
		entry = &ipLimiter{limiter: rate.NewLimiter(rl.r, rl.burst)}
		rl.limiters[ip] = entry
	}
	entry.lastSeen = time.Now()
	return entry.limiter
}

func (rl *rateLimiter) purge() {
	rl.mu.Lock()
	defer rl.mu.Unlock()
	for ip, entry := range rl.limiters {
		if time.Since(entry.lastSeen) > 10*time.Minute {
			delete(rl.limiters, ip)
		}
	}
}

// realIP は Cloud Run (Google LB) 環境でも実クライアント IP を返す。
// Google LB は X-Forwarded-For の末尾に実 IP を追記するため、最後のエントリが信頼できる。
// ヘッダー未設定時（ローカル開発）は RemoteAddr を使用する。
func realIP(c *gin.Context) string {
	if xff := c.GetHeader("X-Forwarded-For"); xff != "" {
		parts := strings.Split(xff, ",")
		if ip := strings.TrimSpace(parts[len(parts)-1]); ip != "" {
			return ip
		}
	}
	ip, _, err := net.SplitHostPort(c.Request.RemoteAddr)
	if err != nil {
		return c.Request.RemoteAddr
	}
	return ip
}

func (rl *rateLimiter) middleware() gin.HandlerFunc {
	return func(c *gin.Context) {
		res := rl.get(realIP(c)).Reserve()
		if d := res.Delay(); d > 0 {
			res.Cancel()
			waitSec := int(d.Seconds()) + 1
			c.Header("Retry-After", strconv.Itoa(waitSec))
			c.AbortWithStatusJSON(http.StatusTooManyRequests, gin.H{"error": "リクエスト数が上限を超えました。しばらく待ってから再試行してください。"})
			return
		}
		c.Next()
	}
}
