package handler

import (
	"fmt"
	"net/http"
	"time"
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