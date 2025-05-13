// internal/middleware/auth.go
package middleware

import (
	"context"
	"net/http"
	"strings"

	"github.com/pizza-nz/restaurant-service/internal/models"
	"github.com/pizza-nz/restaurant-service/internal/service"
)

// contextKey is a type for context keys
type contextKey string

// Context keys
const (
	UserIDKey   contextKey = "userID"
	UserRoleKey contextKey = "userRole"
	UserKey     contextKey = "user"
)

// Auth middleware for authenticating requests
func Auth(authService *service.AuthService) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			// Get the Authorization header
			authHeader := r.Header.Get("Authorization")
			if authHeader == "" {
				http.Error(w, "Authorization header required", http.StatusUnauthorized)
				return
			}

			// Check if it's a Bearer token
			parts := strings.Split(authHeader, " ")
			if len(parts) != 2 || parts[0] != "Bearer" {
				http.Error(w, "Invalid Authorization header format", http.StatusUnauthorized)
				return
			}

			// Get the token
			tokenString := parts[1]

			// Validate the token
			claims, err := authService.ValidateToken(tokenString)
			if err != nil {
				http.Error(w, "Invalid or expired token", http.StatusUnauthorized)
				return
			}

			// Parse the user ID
			userID := claims.UserID
			userRole := claims.Role

			// Add user info to context
			ctx := context.WithValue(r.Context(), UserIDKey, userID)
			ctx = context.WithValue(ctx, UserRoleKey, userRole)

			// Call the next handler with the updated context
			next.ServeHTTP(w, r.WithContext(ctx))
		})
	}
}

// RequireRole middleware for checking user roles
func RequireRole(roles ...models.UserRole) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			// Get the role from context
			roleValue := r.Context().Value(UserRoleKey)
			if roleValue == nil {
				http.Error(w, "Unauthorized", http.StatusUnauthorized)
				return
			}

			role := models.UserRole(roleValue.(string))

			// Check if the role is allowed
			allowed := false
			for _, allowedRole := range roles {
				if role == allowedRole {
					allowed = true
					break
				}
			}

			if !allowed {
				http.Error(w, "Forbidden", http.StatusForbidden)
				return
			}

			// Call the next handler
			next.ServeHTTP(w, r)
		})
	}
}

// Helper functions for extracting values from context
func GetUserID(ctx context.Context) (string, bool) {
	id, ok := ctx.Value(UserIDKey).(string)
	return id, ok
}

func GetUserRole(ctx context.Context) (models.UserRole, bool) {
	role, ok := ctx.Value(UserRoleKey).(string)
	return models.UserRole(role), ok
}
