package auth

import (
	"context"
	"net/http"
	"strings"
)

type contextKey string

// UserContextKey is the context key for storing the authenticated user
const UserContextKey contextKey = "user"

// Middleware validates Authorization header and adds user to context
func Middleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		authHeader := r.Header.Get("Authorization")
		if authHeader == "" {
			http.Error(w, `{"error": "Missing authorization header"}`, http.StatusUnauthorized)
			return
		}

		parts := strings.Split(authHeader, " ")
		if len(parts) != 2 || parts[0] != "Bearer" {
			http.Error(w, `{"error": "Invalid authorization header format"}`, http.StatusUnauthorized)
			return
		}

		username := parts[1]
		if username == "" {
			http.Error(w, `{"error": "Invalid username"}`, http.StatusUnauthorized)
			return
		}

		ctx := context.WithValue(r.Context(), UserContextKey, username)
		next.ServeHTTP(w, r.WithContext(ctx))
	})
}

// GetUserFromContext extracts the authenticated user from the context
func GetUserFromContext(ctx context.Context) string {
	user, ok := ctx.Value(UserContextKey).(string)
	if !ok {
		panic("user not found in context")
	}

	return user
}
