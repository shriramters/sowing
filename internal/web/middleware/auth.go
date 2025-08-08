package middleware

import (
	"net/http"

	"sowing/internal/auth"
)

// Auth returns a new auth middleware
func Auth(authService *auth.Service) func(http.Handler) http.Handler {
	return authService.RequireLogin
}

// WithUser returns a new with user middleware
func WithUser(authService *auth.Service) func(http.Handler) http.Handler {
	return authService.WithUser
}
