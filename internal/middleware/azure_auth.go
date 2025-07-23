package middleware

import (
	"context"
	"main/internal/server/response"
	"main/pkg/auth"
	"net/http"
	"strings"
)

// Note: Using the same ContextKey constants from auth.go to ensure compatibility

// AzureADAuth middleware validates Azure AD JWT tokens and adds user information to the request context
func AzureADAuth(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// Extract token from request
		tokenString, err := auth.ExtractTokenFromRequest(r)
		if err != nil {
			response.Unauthorized(w, "Unauthorized: "+err.Error())
			return
		}

		// Validate Azure AD token
		azureClaims, err := auth.ValidateAzureADToken(tokenString)
		if err != nil {
			response.Unauthorized(w, "Unauthorized: Invalid Azure AD token - "+err.Error())
			return
		}

		// Convert Azure AD claims to internal JWT claims format
		claims := auth.ConvertAzureADClaimsToJWTClaims(azureClaims)

		// Add user information to request context
		ctx := context.WithValue(r.Context(), UserIDKey, claims.UserID)
		ctx = context.WithValue(ctx, UsernameKey, claims.Username)
		ctx = context.WithValue(ctx, UserRolesKey, claims.Roles)

		// Add Azure AD specific information to context
		ctx = context.WithValue(ctx, "azure_tenant_id", azureClaims.TenantID)
		ctx = context.WithValue(ctx, "azure_app_id", azureClaims.AppID)
		ctx = context.WithValue(ctx, "azure_object_id", azureClaims.ObjectID)

		// Call the next handler with the updated context
		next.ServeHTTP(w, r.WithContext(ctx))
	})
}

// HybridAuth middleware that can handle both regular JWT and Azure AD tokens
func HybridAuth(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// Extract token from request
		tokenString, err := auth.ExtractTokenFromRequest(r)
		if err != nil {
			response.Unauthorized(w, "Unauthorized: "+err.Error())
			return
		}

		// Check if Azure AD is configured before trying Azure AD validation
		if azureConfig := auth.GetAzureADConfig(); azureConfig != nil {
			// Try Azure AD token validation first
			if azureClaims, err := auth.ValidateAzureADToken(tokenString); err == nil {
				// Azure AD token is valid
				claims := auth.ConvertAzureADClaimsToJWTClaims(azureClaims)

				// Add user information to request context
				ctx := context.WithValue(r.Context(), UserIDKey, claims.UserID)
				ctx = context.WithValue(ctx, UsernameKey, claims.Username)
				ctx = context.WithValue(ctx, UserRolesKey, claims.Roles)
				ctx = context.WithValue(ctx, "auth_type", "azure_ad")

				// Add Azure AD specific information
				ctx = context.WithValue(ctx, "azure_tenant_id", azureClaims.TenantID)
				ctx = context.WithValue(ctx, "azure_app_id", azureClaims.AppID)
				ctx = context.WithValue(ctx, "azure_object_id", azureClaims.ObjectID)

				next.ServeHTTP(w, r.WithContext(ctx))
				return
			}
		}

		// Try regular JWT token validation
		if claims, err := auth.ValidateToken(tokenString); err == nil {
			// Regular JWT token is valid
			ctx := context.WithValue(r.Context(), UserIDKey, claims.UserID)
			ctx = context.WithValue(ctx, UsernameKey, claims.Username)
			ctx = context.WithValue(ctx, UserRolesKey, claims.Roles)
			ctx = context.WithValue(ctx, "auth_type", "jwt")

			next.ServeHTTP(w, r.WithContext(ctx))
			return
		} else {
			// Debug: Log the validation error
			// fmt.Printf("JWT validation failed: %v\n", err)
		}

		// Both token validations failed
		response.Unauthorized(w, "Unauthorized: Invalid token")
	})
}

// RequireAzureADRole middleware checks if the user has the required Azure AD role
func RequireAzureADRole(role string) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			// Check if this is an Azure AD authenticated request
			authType, ok := r.Context().Value("auth_type").(string)
			if !ok || authType != "azure_ad" {
				response.Forbidden(w, "Forbidden: Azure AD authentication required")
				return
			}

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
				response.Forbidden(w, "Forbidden: Insufficient Azure AD permissions")
				return
			}

			// Call the next handler
			next.ServeHTTP(w, r)
		})
	}
}

// GetAzureTenantIDFromContext extracts the Azure AD tenant ID from the request context
func GetAzureTenantIDFromContext(r *http.Request) (string, bool) {
	tenantID, ok := r.Context().Value("azure_tenant_id").(string)
	return tenantID, ok
}

// GetAzureAppIDFromContext extracts the Azure AD app ID from the request context
func GetAzureAppIDFromContext(r *http.Request) (string, bool) {
	appID, ok := r.Context().Value("azure_app_id").(string)
	return appID, ok
}

// GetAzureObjectIDFromContext extracts the Azure AD object ID from the request context
func GetAzureObjectIDFromContext(r *http.Request) (string, bool) {
	objectID, ok := r.Context().Value("azure_object_id").(string)
	return objectID, ok
}

// GetAuthTypeFromContext extracts the authentication type from the request context
func GetAuthTypeFromContext(r *http.Request) (string, bool) {
	authType, ok := r.Context().Value("auth_type").(string)
	return authType, ok
}