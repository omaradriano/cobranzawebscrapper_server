package middlewares

import (
	"net"
	"net/http"
	"sync"
	"time"

	"golang.org/x/time/rate"
)

type ipLimiter struct {
	limiter  *rate.Limiter
	lastSeen time.Time
}

var (
	limiters = make(map[string]*ipLimiter)
	mu       sync.Mutex
)

func init() {
	go cleanupLimiters()
}

func cleanupLimiters() {
	for {
		time.Sleep(time.Minute)
		mu.Lock()
		for ip, l := range limiters {
			if time.Since(l.lastSeen) > 3*time.Minute {
				delete(limiters, ip)
			}
		}
		mu.Unlock()
	}
}

func getLimiter(ip string) *rate.Limiter {
	mu.Lock()
	defer mu.Unlock()
	if l, exists := limiters[ip]; exists {
		l.lastSeen = time.Now()
		return l.limiter
	}
	l := &ipLimiter{
		limiter:  rate.NewLimiter(20, 40),
		lastSeen: time.Now(),
	}
	limiters[ip] = l
	return l.limiter
}

func RateLimitMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		ip, _, err := net.SplitHostPort(r.RemoteAddr)
		if err != nil {
			ip = r.RemoteAddr
		}
		if !getLimiter(ip).Allow() {
			http.Error(w, "Too Many Requests", http.StatusTooManyRequests)
			return
		}
		next.ServeHTTP(w, r)
	})
}
