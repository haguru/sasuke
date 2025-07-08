package middleware

import (
	"encoding/json"
	"net/http"

	"github.com/haguru/sasuke/internal/models/dto"
	"golang.org/x/time/rate"
)

func RateLimitMiddleware(limiter *rate.Limiter) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			if !limiter.Allow() {
				w.Header().Set("Content-Type", "application/json")
				w.WriteHeader(http.StatusTooManyRequests)
				resp := dto.RateLimitResponse{Message: "Too many requests. Please try again later."}
				_ = json.NewEncoder(w).Encode(resp)
				return
			}
			next.ServeHTTP(w, r)
		})
	}
}
