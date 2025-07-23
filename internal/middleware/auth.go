package middleware

import (
	"context"
	"main/internal/server/response"
	"main/pkg/auth"
	"net/http"
	"strings"
)

// ContextKey is a custom type for context keys to avoid collisions
type ContextKey string

// Context keys for user information
const (
	UserIDKey    ContextKey = "userID"
	UsernameKey  ContextKey = "username"
	UserRolesKey ContextKey = "userRoles"
)

// JWTAuth middleware validates JWT tokens and adds user information to the request context
func JWTAuth(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// Extract token from request
		tokenString, err := auth.ExtractTokenFromRequest(r)
		if err != nil {
			response.Unauthorized(w, "Unauthorized: "+err.Error())
			return
		}

		// Validate token
		claims, err := auth.ValidateToken(tokenString)
		if err != nil {
			response.Unauthorized(w, "Unauthorized: Invalid token")
			return
		}

		// Add user information to request context
		ctx := context.WithValue(r.Context(), UserIDKey, claims.UserID)
		ctx = context.WithValue(ctx, UsernameKey, claims.Username)
		ctx = context.WithValue(ctx, UserRolesKey, claims.Roles)
		ctx = context.WithValue(ctx, "auth_type", "jwt")

		// Call the next handler with the updated context
		next.ServeHTTP(w, r.WithContext(ctx))
	})
}

// RequireRole middleware checks if the user has the required role
func RequireRole(role string) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			// Get user roles from context
			userRoles, ok := r.Context().Value(UserRolesKey).([]string)
			if !ok {
				response.Unauthorized(w, "Unauthorized: User roles not found")
				return
			}

			// Check if user has the required role
			hasRole := false
			for _, userRole := range userRoles {
				if strings.EqualFold(userRole, role) {
					hasRole = true
					break
				}
			}

			if !hasRole {
				response.Forbidden(w, "Forbidden: Insufficient permissions")
				return
			}

			// Call the next handler
			next.ServeHTTP(w, r)
		})
	}
}

// GetUserIDFromContext extracts the user ID from the request context
func GetUserIDFromContext(r *http.Request) (string, bool) {
	userID, ok := r.Context().Value(UserIDKey).(string)
	return userID, ok
}

// GetUsernameFromContext extracts the username from the request context
func GetUsernameFromContext(r *http.Request) (string, bool) {
	username, ok := r.Context().Value(UsernameKey).(string)
	return username, ok
}

// GetUserRolesFromContext extracts the user roles from the request context
func GetUserRolesFromContext(r *http.Request) ([]string, bool) {
	userRoles, ok := r.Context().Value(UserRolesKey).([]string)
	return userRoles, ok
}