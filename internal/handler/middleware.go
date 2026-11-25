package handler

import (
	"fmt"
	"net"
	"net/http"
	"os"
	"sync"
	"time"

	"github.com/golang-jwt/jwt/v5"
	"golang.org/x/time/rate"
)

func Logger(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		start := time.Now()

		next.ServeHTTP(w, r)

		fmt.Printf("%s %s → %v\n", r.Method, r.URL.Path, time.Since(start))
	})
}

func Recoverer(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		defer func() {
			if err := recover(); err != nil {
				fmt.Println("Panic হয়েছে:", err)
				http.Error(w, "Something went wrong", http.StatusInternalServerError)
			}
		}()

		next.ServeHTTP(w, r)
	})
}

func (app *App) AuthMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// ধাপ ১: Header থেকে token নাও
		authHeader := r.Header.Get("Authorization")
		if authHeader == "" {
			app.writeError(w, r, http.StatusUnauthorized, "unauthorized")
			return
		}

		// ধাপ ২: "Bearer eyJhbG..." থেকে শুধু token অংশ নাও
		if len(authHeader) < 7 || authHeader[:7] != "Bearer " {
			app.writeError(w, r, http.StatusUnauthorized, "unauthorized")
			return
		}
		tokenString := authHeader[7:]

		// ধাপ ৩: Token verify করো
		token, err := jwt.Parse(tokenString, func(token *jwt.Token) (any, error) {
			return []byte(os.Getenv("JWT_SECRET")), nil
		})

		if err != nil || !token.Valid {
			app.writeError(w, r, http.StatusUnauthorized, "unauthorized")
			return
		}

		// ধাপ ৪: Token valid, পরের handler এ যাও
		next.ServeHTTP(w, r)
	})
}

// প্রতিটা IP এর জন্য আলাদা limiter রাখব
type ipLimiter struct {
	limiter  *rate.Limiter
	lastSeen time.Time
}

var (
	limiters = make(map[string]*ipLimiter)
	mu       sync.Mutex
)

func init() {
	// ৩ মিনিট idle IP গুলো মুছে ফেলো, নাহলে map বাড়তেই থাকবে
	go func() {
		for range time.Tick(time.Minute) {
			mu.Lock()
			for ip, l := range limiters {
				if time.Since(l.lastSeen) > 3*time.Minute {
					delete(limiters, ip)
				}
			}
			mu.Unlock()
		}
	}()
}

// IP এর limiter নাও, না থাকলে বানাও
func getLimiter(ip string) *rate.Limiter {
	mu.Lock()
	defer mu.Unlock()

	l, ok := limiters[ip]
	if !ok {
		// প্রতি সেকেন্ডে ১০টা request, সর্বোচ্চ ৩০টা burst
		l = &ipLimiter{limiter: rate.NewLimiter(10, 30)}
		limiters[ip] = l
	}
	l.lastSeen = time.Now()
	return l.limiter
}

// ponytail: RemoteAddr ধরে limit, proxy এর পিছনে গেলে X-Forwarded-For লাগবে
func (app *App) RateLimit(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		ip, _, err := net.SplitHostPort(r.RemoteAddr)
		if err != nil {
			ip = r.RemoteAddr
		}
		if !getLimiter(ip).Allow() {
			app.writeError(w, r, http.StatusTooManyRequests, "too_many_requests")
			return
		}
		next.ServeHTTP(w, r)
	})
}
