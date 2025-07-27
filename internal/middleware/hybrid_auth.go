package middleware

import (
	"context"
	"main/internal/server/response"
	"main/pkg/auth"
	"net/http"
	"strings"
)

// HybridMigrationAuthConfig holds configuration for hybrid authentication during migration
type HybridMigrationAuthConfig struct {
	FirebaseService interface{} // Will be properly typed when Firebase service is integrated
	AzureADConfig   interface{} // Will be properly typed when Azure AD config is integrated
	PreferFirebase  bool        // If true, try Firebase first, then Azure AD
}

// HybridMigrationAuth middleware supports both Firebase and Azure AD authentication
// This allows for gradual migration from Azure AD to Firebase
func HybridMigrationAuth(config HybridMigrationAuthConfig) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			// Extract token from Authorization header
			tokenString, err := auth.ExtractTokenFromRequest(r)
			if err != nil {
				response.Unauthorized(w, "Unauthorized: "+err.Error())
				return
			}

			var authSuccess bool
			var userID, username, email string
			var roles []string
			var authType string

			if config.PreferFirebase {
				// Try Firebase first
				if userID, username, email, roles, authSuccess = tryFirebaseAuth(r, tokenString); authSuccess {
					authType = "firebase"
				} else if userID, username, email, roles, authSuccess = tryAzureADAuth(r, tokenString); authSuccess {
					authType = "azure_ad"
				}
			} else {
				// Try Azure AD first
				if userID, username, email, roles, authSuccess = tryAzureADAuth(r, tokenString); authSuccess {
					authType = "azure_ad"
				} else if userID, username, email, roles, authSuccess = tryFirebaseAuth(r, tokenString); authSuccess {
					authType = "firebase"
				}
			}

			if !authSuccess {
				response.Unauthorized(w, "Invalid token for both Firebase and Azure AD")
				return
			}

			// Set user information in context
			ctx := context.WithValue(r.Context(), UserIDKey, userID)
			ctx = context.WithValue(ctx, UsernameKey, username)
			ctx = context.WithValue(ctx, "email", email)
			ctx = context.WithValue(ctx, UserRolesKey, roles)
			ctx = context.WithValue(ctx, "auth_type", authType)

			// Call the next handler with the updated context
			next.ServeHTTP(w, r.WithContext(ctx))
		})
	}
}

// tryFirebaseAuth attempts to authenticate using Firebase
func tryFirebaseAuth(r *http.Request, token string) (userID, username, email string, roles []string, success bool) {
	// TODO: Implement Firebase token validation
	// This will be implemented when Firebase service is integrated
	// For now, return false to indicate Firebase auth is not yet available
	return "", "", "", nil, false
}

// tryAzureADAuth attempts to authenticate using Azure AD
func tryAzureADAuth(r *http.Request, token string) (userID, username, email string, roles []string, success bool) {
	// Try to validate Azure AD token using existing auth package
	azureClaims, err := auth.ValidateAzureADToken(token)
	if err != nil {
		return "", "", "", nil, false
	}

	// Convert Azure AD claims to internal format
	claims := auth.ConvertAzureADClaimsToJWTClaims(azureClaims)
	return claims.UserID, claims.Username, azureClaims.Email, claims.Roles, true
}

// HybridMigrationRequireRole middleware for role-based authorization in hybrid auth
func HybridMigrationRequireRole(requiredRole string) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			roles, exists := GetUserRolesFromContext(r)
			if !exists {
				response.Unauthorized(w, "No roles found in context")
				return
			}

			// Check if user has the required role
			for _, role := range roles {
				if strings.EqualFold(role, requiredRole) {
					next.ServeHTTP(w, r)
					return
				}
			}

			response.Forbidden(w, "Insufficient permissions")
		})
	}
}

// HybridMigrationRequireAnyRole middleware for role-based authorization (any of the specified roles)
func HybridMigrationRequireAnyRole(requiredRoles ...string) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			roles, exists := GetUserRolesFromContext(r)
			if !exists {
				response.Unauthorized(w, "No roles found in context")
				return
			}

			// Check if user has any of the required roles
			for _, userRole := range roles {
				for _, requiredRole := range requiredRoles {
					if strings.EqualFold(userRole, requiredRole) {
						next.ServeHTTP(w, r)
						return
					}
				}
			}

			response.Forbidden(w, "Insufficient permissions")
		})
	}
}

// GetHybridMigrationAuthType extracts the authentication type from hybrid auth context
func GetHybridMigrationAuthType(r *http.Request) (string, bool) {
	authType, ok := r.Context().Value("auth_type").(string)
	return authType, ok
}

// IsFirebaseAuth checks if the current request was authenticated via Firebase
func IsFirebaseAuth(r *http.Request) bool {
	authType, ok := GetHybridMigrationAuthType(r)
	return ok && authType == "firebase"
}

// IsAzureADAuth checks if the current request was authenticated via Azure AD
func IsAzureADAuth(r *http.Request) bool {
	authType, ok := GetHybridMigrationAuthType(r)
	return ok && authType == "azure_ad"
}