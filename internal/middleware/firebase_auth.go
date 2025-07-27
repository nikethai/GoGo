package middleware

import (
	"context"
	"main/internal/server/response"
	"main/pkg/auth"
	"net/http"
	"strings"
)

// FirebaseAuth middleware validates Firebase ID tokens and adds user information to the request context
func FirebaseAuth(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// Extract token from request
		tokenString, err := auth.ExtractTokenFromRequest(r)
		if err != nil {
			response.Unauthorized(w, "Unauthorized: "+err.Error())
			return
		}

		// Get Firebase service
		firebaseService := auth.GetFirebaseService()
		if firebaseService == nil {
			response.InternalServerError(w, "Firebase service not initialized")
			return
		}

		// Validate Firebase ID token
		fbClaims, err := firebaseService.ValidateIDToken(r.Context(), tokenString)
		if err != nil {
			response.Unauthorized(w, "Unauthorized: Invalid Firebase token - "+err.Error())
			return
		}

		// Convert Firebase claims to internal JWT claims format for compatibility
		claims := auth.ConvertFirebaseClaimsToJWTClaims(fbClaims)

		// Add user information to request context
		ctx := context.WithValue(r.Context(), UserIDKey, claims.UserID)
		ctx = context.WithValue(ctx, UsernameKey, claims.Username)
		ctx = context.WithValue(ctx, UserRolesKey, claims.Roles)
		ctx = context.WithValue(ctx, "auth_type", "firebase")

		// Add Firebase specific information to context
		ctx = context.WithValue(ctx, "firebase_email", fbClaims.Email)
		ctx = context.WithValue(ctx, "firebase_name", fbClaims.Name)
		ctx = context.WithValue(ctx, "firebase_custom_claims", fbClaims.Custom)

		// Call the next handler with the updated context
		next.ServeHTTP(w, r.WithContext(ctx))
	})
}

// FirebaseRequireRole middleware checks if the user has the required role using Firebase custom claims
func FirebaseRequireRole(role string) func(http.Handler) http.Handler {
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

// FirebaseRequirePermission middleware checks if the user has the required permission
func FirebaseRequirePermission(permission string) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			// Get user ID from context
			userID, ok := r.Context().Value(UserIDKey).(string)
			if !ok {
				response.Unauthorized(w, "Unauthorized: User ID not found")
				return
			}

			// Get claims manager
			claimsManager := auth.GetClaimsManager()
			if claimsManager == nil {
				response.InternalServerError(w, "Claims manager not initialized")
				return
			}

			// Check if user has the required permission
			hasPermission, err := claimsManager.HasPermission(r.Context(), userID, permission)
			if err != nil {
				response.InternalServerError(w, "Error checking permissions: "+err.Error())
				return
			}

			if !hasPermission {
				response.Forbidden(w, "Forbidden: Insufficient permissions")
				return
			}

			// Call the next handler
			next.ServeHTTP(w, r)
		})
	}
}

// FirebaseRequireAnyRole middleware checks if the user has any of the required roles
func FirebaseRequireAnyRole(roles ...string) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			// Get user roles from context
			userRoles, ok := r.Context().Value(UserRolesKey).([]string)
			if !ok {
				response.Unauthorized(w, "Unauthorized: User roles not found")
				return
			}

			// Check if user has any of the required roles
			hasRole := false
			for _, userRole := range userRoles {
				for _, requiredRole := range roles {
					if strings.EqualFold(userRole, requiredRole) {
						hasRole = true
						break
					}
				}
				if hasRole {
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

// FirebaseRequireAnyPermission middleware checks if the user has any of the required permissions
func FirebaseRequireAnyPermission(permissions ...string) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			// Get user ID from context
			userID, ok := r.Context().Value(UserIDKey).(string)
			if !ok {
				response.Unauthorized(w, "Unauthorized: User ID not found")
				return
			}

			// Get claims manager
			claimsManager := auth.GetClaimsManager()
			if claimsManager == nil {
				response.InternalServerError(w, "Claims manager not initialized")
				return
			}

			// Check if user has any of the required permissions
			hasPermission := false
			for _, permission := range permissions {
				has, err := claimsManager.HasPermission(r.Context(), userID, permission)
				if err != nil {
					response.InternalServerError(w, "Error checking permissions: "+err.Error())
					return
				}
				if has {
					hasPermission = true
					break
				}
			}

			if !hasPermission {
				response.Forbidden(w, "Forbidden: Insufficient permissions")
				return
			}

			// Call the next handler
			next.ServeHTTP(w, r)
		})
	}
}

// GetFirebaseEmailFromContext extracts the Firebase email from the request context
func GetFirebaseEmailFromContext(r *http.Request) (string, bool) {
	email, ok := r.Context().Value("firebase_email").(string)
	return email, ok
}

// GetFirebaseNameFromContext extracts the Firebase name from the request context
func GetFirebaseNameFromContext(r *http.Request) (string, bool) {
	name, ok := r.Context().Value("firebase_name").(string)
	return name, ok
}

// GetFirebaseCustomClaimsFromContext extracts the Firebase custom claims from the request context
func GetFirebaseCustomClaimsFromContext(r *http.Request) (map[string]interface{}, bool) {
	claims, ok := r.Context().Value("firebase_custom_claims").(map[string]interface{})
	return claims, ok
}