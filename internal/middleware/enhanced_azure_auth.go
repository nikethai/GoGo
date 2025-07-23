package middleware

import (
	"context"
	"fmt"
	"net/http"
	"strings"
	"time"

	"main/pkg/auth"
)

const (
	// Context keys for Azure AD authentication
	AzureUserContextKey     ContextKey = "azure_user"
	AzureTokenContextKey    ContextKey = "azure_token"
	AzureClaimsContextKey   ContextKey = "azure_claims"
	AzureSessionContextKey  ContextKey = "azure_session"
	AuthMethodContextKey    ContextKey = "auth_method"
)

// AuthMethod represents the authentication method used
type AuthMethod string

const (
	AuthMethodJWT     AuthMethod = "jwt"
	AuthMethodAzureAD AuthMethod = "azure_ad"
	AuthMethodHybrid  AuthMethod = "hybrid"
)

// AzureUserInfo represents authenticated Azure AD user information
type AzureUserInfo struct {
	UserID       string            `json:"user_id"`
	Email        string            `json:"email"`
	Name         string            `json:"name"`
	TenantID     string            `json:"tenant_id"`
	AppID        string            `json:"app_id"`
	ObjectID     string            `json:"object_id"`
	Roles        []string          `json:"roles"`
	Groups       []string          `json:"groups"`
	Claims       map[string]interface{} `json:"claims"`
	AuthenticatedAt time.Time      `json:"authenticated_at"`
}

// EnhancedAzureADAuth provides enhanced Azure AD authentication middleware
func EnhancedAzureADAuth(azureService auth.AzureADService) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			// Extract token from Authorization header
			authHeader := r.Header.Get("Authorization")
			if authHeader == "" {
				http.Error(w, "Missing authorization header", http.StatusUnauthorized)
				return
			}

			// Parse Bearer token
			tokenParts := strings.Split(authHeader, " ")
			if len(tokenParts) != 2 || tokenParts[0] != "Bearer" {
				http.Error(w, "Invalid authorization header format", http.StatusUnauthorized)
				return
			}

			token := tokenParts[1]

			// Validate Azure AD token
			claims, err := auth.ValidateAzureADToken(token)
			if err != nil {
				http.Error(w, fmt.Sprintf("Token validation failed: %v", err), http.StatusUnauthorized)
				return
			}

			// Extract user information from claims
			userInfo := extractUserInfoFromClaims(claims)

			// Add user info and claims to context
			ctx := context.WithValue(r.Context(), AzureUserContextKey, userInfo)
			ctx = context.WithValue(ctx, AzureTokenContextKey, token)
			ctx = context.WithValue(ctx, AzureClaimsContextKey, claims)
			ctx = context.WithValue(ctx, AuthMethodContextKey, AuthMethodAzureAD)

			// Continue with the request
			next.ServeHTTP(w, r.WithContext(ctx))
		})
	}
}

// EnhancedHybridAuth provides enhanced hybrid authentication (JWT + Azure AD)
func EnhancedHybridAuth(azureService auth.AzureADService) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			// Extract token from Authorization header
			authHeader := r.Header.Get("Authorization")
			if authHeader == "" {
				http.Error(w, "Missing authorization header", http.StatusUnauthorized)
				return
			}

			// Parse Bearer token
			tokenParts := strings.Split(authHeader, " ")
			if len(tokenParts) != 2 || tokenParts[0] != "Bearer" {
				http.Error(w, "Invalid authorization header format", http.StatusUnauthorized)
				return
			}

			token := tokenParts[1]

			// Try Azure AD validation first
			if azureClaims, err := auth.ValidateAzureADToken(token); err == nil {
				// Azure AD token is valid
				userInfo := extractUserInfoFromClaims(azureClaims)
				ctx := context.WithValue(r.Context(), AzureUserContextKey, userInfo)
				ctx = context.WithValue(ctx, AzureTokenContextKey, token)
				ctx = context.WithValue(ctx, AzureClaimsContextKey, azureClaims)
				ctx = context.WithValue(ctx, AuthMethodContextKey, AuthMethodAzureAD)
				next.ServeHTTP(w, r.WithContext(ctx))
				return
			}

			// Try regular JWT validation
			if jwtClaims, err := auth.ValidateToken(token); err == nil {
				// Regular JWT token is valid
				ctx := context.WithValue(r.Context(), "user_id", jwtClaims.UserID)
				ctx = context.WithValue(ctx, "username", jwtClaims.Username)
				ctx = context.WithValue(ctx, "user_roles", jwtClaims.Roles)
				ctx = context.WithValue(ctx, AuthMethodContextKey, AuthMethodJWT)
				next.ServeHTTP(w, r.WithContext(ctx))
				return
			}

			// Both validations failed
			http.Error(w, "Invalid token", http.StatusUnauthorized)
		})
	}
}

// RequireAzureADRoles middleware to check for specific Azure AD roles
func RequireAzureADRoles(requiredRoles ...string) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			// Get user info from context
			userInfo, ok := r.Context().Value(AzureUserContextKey).(*AzureUserInfo)
			if !ok {
				http.Error(w, "Azure AD authentication required", http.StatusUnauthorized)
				return
			}

			// Check if user has any of the required roles
			if !hasAnyRole(userInfo.Roles, requiredRoles) {
				http.Error(w, "Insufficient permissions", http.StatusForbidden)
				return
			}

			next.ServeHTTP(w, r)
		})
	}
}

// RequireAzureADGroups middleware to check for specific Azure AD groups
func RequireAzureADGroups(requiredGroups ...string) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			// Get user info from context
			userInfo, ok := r.Context().Value(AzureUserContextKey).(*AzureUserInfo)
			if !ok {
				http.Error(w, "Azure AD authentication required", http.StatusUnauthorized)
				return
			}

			// Check if user belongs to any of the required groups
			if !hasAnyRole(userInfo.Groups, requiredGroups) {
				http.Error(w, "Insufficient group membership", http.StatusForbidden)
				return
			}

			next.ServeHTTP(w, r)
		})
	}
}

// RequireTenant middleware to check for specific Azure AD tenant
func RequireTenant(allowedTenants ...string) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			// Get user info from context
			userInfo, ok := r.Context().Value(AzureUserContextKey).(*AzureUserInfo)
			if !ok {
				http.Error(w, "Azure AD authentication required", http.StatusUnauthorized)
				return
			}

			// Check if user belongs to allowed tenant
			if !contains(allowedTenants, userInfo.TenantID) {
				http.Error(w, "Tenant not allowed", http.StatusForbidden)
				return
			}

			next.ServeHTTP(w, r)
		})
	}
}

// TokenRefreshMiddleware automatically refreshes tokens when they're close to expiry
func TokenRefreshMiddleware(azureService auth.AzureADService, refreshThreshold time.Duration) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			// Get user info from context
			userInfo, ok := r.Context().Value(AzureUserContextKey).(*AzureUserInfo)
			if !ok {
				// No Azure AD user, continue without refresh
				next.ServeHTTP(w, r)
				return
			}

			// Check if token needs refresh
			if time.Until(userInfo.AuthenticatedAt.Add(time.Hour)) < refreshThreshold {
				// Try to get stored token for refresh
				storedToken, err := azureService.GetStoredToken(userInfo.UserID)
				if err == nil && storedToken.RefreshToken != "" {
					// Attempt token refresh
					newTokens, err := azureService.RefreshToken(storedToken.RefreshToken)
					if err == nil {
						// Store new tokens
						azureService.StoreTokenSecurely(userInfo.UserID, newTokens)
						
						// Add refreshed token to response headers
						w.Header().Set("X-Refreshed-Token", newTokens.AccessToken)
					}
				}
			}

			next.ServeHTTP(w, r)
		})
	}
}

// Helper functions

// extractUserInfoFromClaims extracts user information from Azure AD claims
func extractUserInfoFromClaims(claims *auth.AzureADClaims) *AzureUserInfo {
	return &AzureUserInfo{
		UserID:          claims.ObjectID,
		Email:           claims.Email,
		Name:            claims.Name,
		TenantID:        claims.TenantID,
		AppID:           claims.AppID,
		ObjectID:        claims.ObjectID,
		Roles:           claims.Roles,
		Groups:          []string{}, // Groups not available in standard Azure AD claims
		Claims:          map[string]interface{}{
			"iss": claims.Issuer,
			"aud": claims.Audience,
			"exp": claims.ExpiresAt,
			"iat": claims.IssuedAt,
			"sub": claims.Subject,
		},
		AuthenticatedAt: time.Now(),
	}
}

// hasAnyRole checks if user has any of the required roles
func hasAnyRole(userRoles, requiredRoles []string) bool {
	for _, required := range requiredRoles {
		for _, userRole := range userRoles {
			if userRole == required {
				return true
			}
		}
	}
	return false
}

// contains checks if a slice contains a specific string
func contains(slice []string, item string) bool {
	for _, s := range slice {
		if s == item {
			return true
		}
	}
	return false
}

// Context helper functions

// GetAzureUserFromContext retrieves Azure AD user info from context
func GetAzureUserFromContext(ctx context.Context) (*AzureUserInfo, bool) {
	userInfo, ok := ctx.Value(AzureUserContextKey).(*AzureUserInfo)
	return userInfo, ok
}

// GetAzureTokenFromContext retrieves Azure AD token from context
func GetAzureTokenFromContext(ctx context.Context) (string, bool) {
	token, ok := ctx.Value(AzureTokenContextKey).(string)
	return token, ok
}

// GetAzureClaimsFromContext retrieves Azure AD claims from context
func GetAzureClaimsFromContext(ctx context.Context) (*auth.AzureADClaims, bool) {
	claims, ok := ctx.Value(AzureClaimsContextKey).(*auth.AzureADClaims)
	return claims, ok
}

// GetAuthMethodFromContext retrieves authentication method from context
func GetAuthMethodFromContext(ctx context.Context) (AuthMethod, bool) {
	method, ok := ctx.Value(AuthMethodContextKey).(AuthMethod)
	return method, ok
}

// IsAzureADAuthenticated checks if request is authenticated via Azure AD
func IsAzureADAuthenticated(ctx context.Context) bool {
	method, ok := GetAuthMethodFromContext(ctx)
	return ok && (method == AuthMethodAzureAD || method == AuthMethodHybrid)
}