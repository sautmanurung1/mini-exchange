package http

import (
	"context"
	"net/http"
	"strings"
	"sync"

	"mini-exchange/internal/domain"
	"golang.org/x/time/rate"
)

type Middleware struct {
	authUC  domain.AuthUseCase
	limiters map[string]*rate.Limiter
	mu       sync.Mutex
}

func NewMiddleware(authUC domain.AuthUseCase) *Middleware {
	return &Middleware{
		authUC:  authUC,
		limiters: make(map[string]*rate.Limiter),
	}
}

func (m *Middleware) Auth(next http.HandlerFunc) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		authHeader := r.Header.Get("Authorization")
		if authHeader == "" {
			http.Error(w, "Authorization header required", http.StatusUnauthorized)
			return
		}

		parts := strings.Split(authHeader, " ")
		if len(parts) != 2 || parts[0] != "Bearer" {
			http.Error(w, "Invalid authorization format", http.StatusUnauthorized)
			return
		}

		userID, err := m.authUC.ValidateToken(parts[1])
		if err != nil {
			http.Error(w, err.Error(), http.StatusUnauthorized)
			return
		}

		// Set user ID to context
		ctx := context.WithValue(r.Context(), "user_id", userID)
		next.ServeHTTP(w, r.WithContext(ctx))
	}
}

func (m *Middleware) RateLimit(next http.HandlerFunc) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		ip := r.RemoteAddr
		m.mu.Lock()
		limiter, ok := m.limiters[ip]
		if !ok {
			limiter = rate.NewLimiter(1, 5) // 1 request per second, burst of 5
			m.limiters[ip] = limiter
		}
		m.mu.Unlock()

		if !limiter.Allow() {
			http.Error(w, "Too many requests", http.StatusTooManyRequests)
			return
		}

		next.ServeHTTP(w, r)
	}
}
