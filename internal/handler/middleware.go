package handler

import (
	"fmt"
	"net/http"
	"os"
	"time"

	"github.com/golang-jwt/jwt/v5"
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