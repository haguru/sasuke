package middleware

import (
	"encoding/json"
	"net/http"
	
	"github.com/haguru/sasuke/pkg/helper"
	"github.com/haguru/sasuke/internal/interfaces"
	"github.com/haguru/sasuke/internal/models/dto"
	"golang.org/x/time/rate"
)


func RateLimitMiddleware(limiter *rate.Limiter, logger interfaces.Logger) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			logger.Debug("Checking rate limit", "function", helper.GetFuncName(), "path", r.URL.Path, "remote_addr", r.RemoteAddr)
			if !limiter.Allow() {
				logger.Warn("Rate limit exceeded", "function", helper.GetFuncName(), "path", r.URL.Path, "remote_addr", r.RemoteAddr)
				w.Header().Set("Content-Type", "application/json")
				w.WriteHeader(http.StatusTooManyRequests)
				resp := dto.RateLimitResponse{Message: "Too many requests. Please try again later."}
				_ = json.NewEncoder(w).Encode(resp)
				return
			}
			logger.Debug("Rate limit allowed", "function", helper.GetFuncName(), "path", r.URL.Path, "remote_addr", r.RemoteAddr)
			next.ServeHTTP(w, r)
		})
	}
}
