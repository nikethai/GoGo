package router

import (
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"net/url"
	"strings"
	"time"

	"github.com/go-chi/chi/v5"

	"main/internal/middleware"
	"main/pkg/auth"
	customError "main/internal/error"
)

// AzureAuthRouter handles Azure AD authentication routes
type AzureAuthRouter struct {
	oauth2Service   auth.AzureADService
	sessionManager  auth.SessionManager
	tokenCache      auth.TokenCache
	config          *auth.OAuth2Config
}

// NewAzureAuthRouter creates a new Azure authentication router
func NewAzureAuthRouter(
	oauth2Service auth.AzureADService,
	sessionManager auth.SessionManager,
	tokenCache auth.TokenCache,
	config *auth.OAuth2Config,
) *AzureAuthRouter {
	return &AzureAuthRouter{
		oauth2Service:  oauth2Service,
		sessionManager: sessionManager,
		tokenCache:     tokenCache,
		config:         config,
	}
}

// SetupAzureAuthRoutes sets up Azure AD authentication routes
func (ar *AzureAuthRouter) SetupAzureAuthRoutes() chi.Router {
	router := chi.NewRouter()

	// Public routes (no authentication required)
	router.Route("/api/v1/auth/azure", func(r chi.Router) {
		// OAuth 2.0 Authorization Code Flow
		r.Get("/login", ar.handleAzureLogin)
		r.Get("/callback", ar.handleAzureCallback)
		r.Post("/logout", ar.handleAzureLogout)
		
		// Token management
		r.Post("/refresh", ar.handleTokenRefresh)
		r.Get("/userinfo", ar.handleUserInfo)
		
		// Health check
		r.Get("/health", ar.handleHealthCheck)
	})

	// Protected routes (require Azure AD authentication)
	router.Route("/api/v1/auth/azure", func(r chi.Router) {
		r.Use(middleware.EnhancedAzureADAuth(ar.oauth2Service))
		
		// Session management
		r.Get("/session", ar.handleGetSession)
		r.Delete("/session", ar.handleDeleteSession)
		r.Put("/session/extend", ar.handleExtendSession)
		
		// Token operations
		r.Get("/tokens", ar.handleGetTokens)
		r.Delete("/tokens", ar.handleRevokeTokens)
		
		// User profile
		r.Get("/profile", ar.handleGetProfile)
		r.Put("/profile", ar.handleUpdateProfile)
	})

	// Admin routes (require specific roles)
	router.Route("/api/v1/auth/azure/admin", func(r chi.Router) {
		r.Use(middleware.EnhancedAzureADAuth(ar.oauth2Service))
		r.Use(middleware.RequireAzureADRoles("admin"))
		
		// Cache management
		r.Get("/cache/stats", ar.handleGetCacheStats)
		r.Delete("/cache", ar.handleClearCache)
		r.Post("/cache/cleanup", ar.handleCleanupCache)
		
		// Session management
		r.Get("/sessions", ar.handleGetAllSessions)
		r.Delete("/sessions/{userID}", ar.handleDeleteUserSession)
	})

	return router
}

// OAuth 2.0 Authorization Code Flow Handlers

// handleAzureLogin initiates Azure AD OAuth 2.0 login flow
func (ar *AzureAuthRouter) handleAzureLogin(w http.ResponseWriter, r *http.Request) {
	// Generate PKCE challenge
	pkceChallenge, err := auth.GeneratePKCEChallenge()
	if err != nil {
		http.Error(w, "Failed to generate PKCE challenge", http.StatusInternalServerError)
		return
	}

	// Generate state parameter
	state, err := auth.GenerateAuthState()
	if err != nil {
		http.Error(w, "Failed to generate state parameter", http.StatusInternalServerError)
		return
	}

	// Store PKCE and state in session for validation
	sessionID := fmt.Sprintf("oauth_session_%s", state)
	sessionData := map[string]interface{}{
		"pkce_verifier": pkceChallenge.CodeVerifier,
		"state":         state,
		"created_at":    time.Now(),
		"redirect_uri":  ar.config.RedirectURI,
	}

	// Store session temporarily (5 minutes)
	if err := ar.storeTemporarySession(sessionID, sessionData, 5*time.Minute); err != nil {
		log.Printf("Failed to store OAuth session: %v", err)
		http.Error(w, "Failed to store session data", http.StatusInternalServerError)
		return
	}

	// Build authorization URL
	authURL := auth.BuildAuthorizationURL(ar.config, state, pkceChallenge)

	// Return authorization URL for client-side redirect
	response := map[string]interface{}{
		"authorization_url": authURL,
		"state":             state,
		"expires_in":        300, // 5 minutes
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(response)
}

// handleAzureCallback handles OAuth 2.0 callback from Azure AD
func (ar *AzureAuthRouter) handleAzureCallback(w http.ResponseWriter, r *http.Request) {
	// Extract parameters from callback
	code := r.URL.Query().Get("code")
	state := r.URL.Query().Get("state")
	errorParam := r.URL.Query().Get("error")
	errorDescription := r.URL.Query().Get("error_description")

	// Check for OAuth errors
	if errorParam != "" {
		response := map[string]interface{}{
			"error":             errorParam,
			"error_description": errorDescription,
			"code":              "OAUTH_ERROR",
		}
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusBadRequest)
		json.NewEncoder(w).Encode(response)
		return
	}

	// Validate required parameters
	if code == "" || state == "" {
		http.Error(w, "Missing required parameters (code or state)", http.StatusBadRequest)
		return
	}

	// Retrieve and validate session
	sessionID := fmt.Sprintf("oauth_session_%s", state)
	sessionData, err := ar.getTemporarySession(sessionID)
	if err != nil {
		http.Error(w, "Invalid or expired state parameter", http.StatusBadRequest)
		return
	}

	// Validate state parameter
	if err := auth.ValidateState(state, sessionData["state"].(string)); err != nil {
		http.Error(w, "State validation failed", http.StatusBadRequest)
		return
	}

	// Exchange authorization code for tokens
	pkceVerifier := sessionData["pkce_verifier"].(string)
	tokenResponse, err := auth.ExchangeCodeForToken(ar.config, code, pkceVerifier)
	if err != nil {
		log.Printf("Token exchange failed: %v", err)
		http.Error(w, "Failed to exchange code for tokens", http.StatusInternalServerError)
		return
	}

	// Extract user information from ID token
	userInfo, err := auth.ExtractUserInfoFromIDToken(tokenResponse.IDToken)
	if err != nil {
		log.Printf("Failed to extract user info: %v", err)
		http.Error(w, "Failed to extract user information", http.StatusInternalServerError)
		return
	}

	// Create session for the user
	session, err := ar.sessionManager.CreateSession(userInfo.ObjectID, tokenResponse)
	if err != nil {
		log.Printf("Failed to create session: %v", err)
		http.Error(w, "Failed to create user session", http.StatusInternalServerError)
		return
	}

	// Cache tokens for future use
	cachedToken := &auth.CachedToken{
		UserID:           userInfo.ObjectID,
		AccessToken:      tokenResponse.AccessToken,
		RefreshToken:     tokenResponse.RefreshToken,
		IDToken:          tokenResponse.IDToken,
		TokenType:        tokenResponse.TokenType,
		Scope:            tokenResponse.Scope,
		ExpiresAt:        time.Now().Add(time.Duration(tokenResponse.ExpiresIn) * time.Second),
		RefreshExpiresAt: time.Now().Add(24 * time.Hour), // Assume 24h refresh token lifetime
		Metadata: map[string]interface{}{
			"tenant_id": userInfo.TenantID,
			"app_id":    userInfo.AppID,
		},
	}

	if err := ar.tokenCache.StoreToken(userInfo.ObjectID, cachedToken); err != nil {
		log.Printf("Failed to cache tokens: %v", err)
		// Don't fail the login, just log the error
	}

	// Clean up temporary session
	ar.deleteTemporarySession(sessionID)

	// Return success response with session information
	response := map[string]interface{}{
		"message":    "Login successful",
		"session_id": session.ID,
		"user": map[string]interface{}{
			"id":                userInfo.ObjectID,
			"email":             userInfo.Email,
			"name":              userInfo.Name,
			"preferred_username": userInfo.PreferredUsername,
			"roles":             userInfo.Roles,
		},
		"expires_at": session.ExpiresAt,
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(response)
}

// handleAzureLogout handles Azure AD logout
func (ar *AzureAuthRouter) handleAzureLogout(w http.ResponseWriter, r *http.Request) {
	// Get session ID from request (header, cookie, or body)
	sessionID := ar.extractSessionID(r)
	if sessionID == "" {
		http.Error(w, "Session ID required for logout", http.StatusBadRequest)
		return
	}

	// Get session to extract user ID
	session, err := ar.sessionManager.GetSession(sessionID)
	if err != nil {
		// Session might already be expired/deleted
		response := map[string]interface{}{
			"message": "Logout successful",
		}
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(response)
		return
	}

	// Delete session
	if err := ar.sessionManager.DeleteSession(sessionID); err != nil {
		log.Printf("Failed to delete session: %v", err)
	}

	// Delete cached tokens
	if err := ar.tokenCache.DeleteToken(session.UserID); err != nil {
		log.Printf("Failed to delete cached tokens: %v", err)
	}

	// Build Azure AD logout URL for complete logout
	postLogoutRedirectURL := ar.config.RedirectURI // Use redirect URI as post-logout URL
	logoutURL := fmt.Sprintf(
		"https://login.microsoftonline.com/%s/oauth2/v2.0/logout?post_logout_redirect_uri=%s",
		ar.config.TenantID,
		url.QueryEscape(postLogoutRedirectURL),
	)

	response := map[string]interface{}{
		"message":    "Logout successful",
		"logout_url": logoutURL,
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(response)
}

// Token Management Handlers

// handleTokenRefresh handles token refresh requests
func (ar *AzureAuthRouter) handleTokenRefresh(w http.ResponseWriter, r *http.Request) {
	var request struct {
		RefreshToken string `json:"refresh_token"`
		UserID       string `json:"user_id"`
	}

	if err := json.NewDecoder(r.Body).Decode(&request); err != nil {
		http.Error(w, "Invalid request format", http.StatusBadRequest)
		return
	}

	if request.RefreshToken == "" || request.UserID == "" {
		http.Error(w, "Missing required fields", http.StatusBadRequest)
		return
	}

	// Refresh tokens using OAuth2 service
	tokenResponse, err := auth.RefreshAccessToken(ar.config, request.RefreshToken)
	if err != nil {
		log.Printf("Token refresh failed: %v", err)
		http.Error(w, "Failed to refresh tokens", http.StatusUnauthorized)
		return
	}

	// Update cached tokens
	cachedToken := &auth.CachedToken{
		UserID:           request.UserID,
		AccessToken:      tokenResponse.AccessToken,
		RefreshToken:     tokenResponse.RefreshToken,
		IDToken:          tokenResponse.IDToken,
		TokenType:        tokenResponse.TokenType,
		Scope:            tokenResponse.Scope,
		ExpiresAt:        time.Now().Add(time.Duration(tokenResponse.ExpiresIn) * time.Second),
		RefreshExpiresAt: time.Now().Add(24 * time.Hour),
	}

	if err := ar.tokenCache.UpdateToken(request.UserID, cachedToken); err != nil {
		log.Printf("Failed to update cached tokens: %v", err)
	}

	response := map[string]interface{}{
		"access_token": tokenResponse.AccessToken,
		"token_type":   tokenResponse.TokenType,
		"expires_in":   tokenResponse.ExpiresIn,
		"scope":        tokenResponse.Scope,
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(response)
}

// handleUserInfo returns user information from ID token
func (ar *AzureAuthRouter) handleUserInfo(w http.ResponseWriter, r *http.Request) {
	// Get ID token from request
	idToken := ar.extractIDToken(r)
	if idToken == "" {
		http.Error(w, "ID token required", http.StatusBadRequest)
		return
	}

	// Extract user information
	userInfo, err := auth.ExtractUserInfoFromIDToken(idToken)
	if err != nil {
		http.Error(w, "Invalid ID token", http.StatusBadRequest)
		return
	}

	response := map[string]interface{}{
		"sub":                userInfo.ObjectID,
		"email":              userInfo.Email,
		"name":               userInfo.Name,
		"preferred_username": userInfo.PreferredUsername,
		"tenant_id":          userInfo.TenantID,
		"app_id":             userInfo.AppID,
		"roles":              userInfo.Roles,
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(response)
}

// Protected Route Handlers

// handleGetSession returns current session information
func (ar *AzureAuthRouter) handleGetSession(w http.ResponseWriter, r *http.Request) {
	// Extract user info from Azure AD context
	userInfo, ok := r.Context().Value(middleware.AzureUserContextKey).(*middleware.AzureUserInfo)
	if !ok {
		http.Error(w, "User not authenticated", http.StatusUnauthorized)
		return
	}

	// Get active sessions for user
	sessions, err := ar.sessionManager.GetUserSessions(userInfo.ObjectID)
	if err != nil {
		http.Error(w, "Failed to retrieve sessions", http.StatusInternalServerError)
		return
	}

	response := map[string]interface{}{
		"sessions": sessions,
		"count":    len(sessions),
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(response)
}

// handleDeleteSession deletes current session
func (ar *AzureAuthRouter) handleDeleteSession(w http.ResponseWriter, r *http.Request) {
	sessionID := ar.extractSessionID(r)
	if sessionID == "" {
		http.Error(w, "Session ID required", http.StatusBadRequest)
		return
	}

	if err := ar.sessionManager.DeleteSession(sessionID); err != nil {
		http.Error(w, "Failed to delete session", http.StatusInternalServerError)
		return
	}

	response := map[string]interface{}{
		"message": "Session deleted successfully",
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(response)
}

// handleExtendSession extends current session
func (ar *AzureAuthRouter) handleExtendSession(w http.ResponseWriter, r *http.Request) {
	sessionID := ar.extractSessionID(r)
	if sessionID == "" {
		http.Error(w, "Session ID required", http.StatusBadRequest)
		return
	}

	var request struct {
		Duration string `json:"duration"` // e.g., "1h", "30m"
	}

	if err := json.NewDecoder(r.Body).Decode(&request); err != nil {
		http.Error(w, "Invalid request format", http.StatusBadRequest)
		return
	}

	// Parse duration
	duration := time.Hour // Default 1 hour
	if request.Duration != "" {
		if d, err := time.ParseDuration(request.Duration); err == nil {
			duration = d
		}
	}

	if err := ar.sessionManager.ExtendSession(sessionID, duration); err != nil {
		http.Error(w, "Failed to extend session", http.StatusInternalServerError)
		return
	}

	response := map[string]interface{}{
		"message": "Session extended successfully",
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(response)
}

// handleGetTokens returns cached tokens for user
func (ar *AzureAuthRouter) handleGetTokens(w http.ResponseWriter, r *http.Request) {
	userInfo, ok := r.Context().Value(middleware.AzureUserContextKey).(*middleware.AzureUserInfo)
	if !ok {
		http.Error(w, "User not authenticated", http.StatusUnauthorized)
		return
	}

	token, err := ar.tokenCache.GetToken(userInfo.ObjectID)
	if err != nil {
		http.Error(w, "No cached tokens found", http.StatusNotFound)
		return
	}

	// Return token metadata (not the actual tokens for security)
	response := map[string]interface{}{
		"token_type":         token.TokenType,
		"scope":              token.Scope,
		"expires_at":         token.ExpiresAt,
		"refresh_expires_at": token.RefreshExpiresAt,
		"cached_at":          token.CachedAt,
		"last_accessed_at":   token.LastAccessedAt,
		"access_count":       token.AccessCount,
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(response)
}

// handleRevokeTokens revokes cached tokens for user
func (ar *AzureAuthRouter) handleRevokeTokens(w http.ResponseWriter, r *http.Request) {
	userInfo, ok := r.Context().Value(middleware.AzureUserContextKey).(*middleware.AzureUserInfo)
	if !ok {
		http.Error(w, "User not authenticated", http.StatusUnauthorized)
		return
	}

	if err := ar.tokenCache.DeleteToken(userInfo.ObjectID); err != nil {
		http.Error(w, "Failed to revoke tokens", http.StatusInternalServerError)
		return
	}

	response := map[string]interface{}{
		"message": "Tokens revoked successfully",
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(response)
}

// handleGetProfile returns user profile information
func (ar *AzureAuthRouter) handleGetProfile(w http.ResponseWriter, r *http.Request) {
	userInfo, ok := r.Context().Value(middleware.AzureUserContextKey).(*middleware.AzureUserInfo)
	if !ok {
		http.Error(w, "User not authenticated", http.StatusUnauthorized)
		return
	}

	// Get user sessions for additional context
	sessions, _ := ar.sessionManager.GetUserSessions(userInfo.ObjectID)

	response := map[string]interface{}{
		"user_id":       userInfo.ObjectID,
		"tenant_id":     userInfo.TenantID,
		"app_id":        userInfo.AppID,
		"auth_type":     "azure_ad",
		"session_count": len(sessions),
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(response)
}

// handleUpdateProfile updates user profile (placeholder)
func (ar *AzureAuthRouter) handleUpdateProfile(w http.ResponseWriter, r *http.Request) {
	// This would typically update user preferences or metadata
	// For now, it's a placeholder
	response := map[string]interface{}{
		"message": "Profile update not implemented",
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(response)
}

// Admin Route Handlers

// handleGetCacheStats returns cache statistics
func (ar *AzureAuthRouter) handleGetCacheStats(w http.ResponseWriter, r *http.Request) {
	stats := ar.tokenCache.GetCacheStats()
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(stats)
}

// handleClearCache clears the token cache
func (ar *AzureAuthRouter) handleClearCache(w http.ResponseWriter, r *http.Request) {
	if err := ar.tokenCache.ClearCache(); err != nil {
		http.Error(w, "Failed to clear cache", http.StatusInternalServerError)
		return
	}

	response := map[string]interface{}{
		"message": "Cache cleared successfully",
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(response)
}

// handleCleanupCache triggers cache cleanup
func (ar *AzureAuthRouter) handleCleanupCache(w http.ResponseWriter, r *http.Request) {
	cleanedCount := ar.tokenCache.CleanupExpiredTokens()
	response := map[string]interface{}{
		"message":       "Cache cleanup completed",
		"cleaned_tokens": cleanedCount,
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(response)
}

// handleGetAllSessions returns all active sessions (admin only)
func (ar *AzureAuthRouter) handleGetAllSessions(w http.ResponseWriter, r *http.Request) {
	// This would require additional methods in SessionManager
	// For now, return a placeholder
	response := map[string]interface{}{
		"message": "All sessions endpoint not implemented",
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(response)
}

// handleDeleteUserSession deletes a specific user's session (admin only)
func (ar *AzureAuthRouter) handleDeleteUserSession(w http.ResponseWriter, r *http.Request) {
	userID := chi.URLParam(r, "userID")
	if userID == "" {
		http.Error(w, "User ID required", http.StatusBadRequest)
		return
	}

	// Delete all sessions for the user
	if err := ar.sessionManager.DeleteUserSessions(userID); err != nil {
		http.Error(w, "Failed to delete user sessions", http.StatusInternalServerError)
		return
	}

	// Also delete cached tokens
	if err := ar.tokenCache.DeleteToken(userID); err != nil {
		log.Printf("Failed to delete cached tokens for user %s: %v", userID, err)
	}

	response := map[string]interface{}{
		"message": "User sessions deleted successfully",
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(response)
}

// handleHealthCheck returns health status
func (ar *AzureAuthRouter) handleHealthCheck(w http.ResponseWriter, r *http.Request) {
	// Check if services are healthy
	health := map[string]interface{}{
		"status":    "healthy",
		"timestamp": time.Now(),
		"services": map[string]interface{}{
			"oauth2_service":   "healthy",
			"session_manager":  "healthy",
			"token_cache":      "healthy",
		},
	}

	// Check cache stats for health indicators
	stats := ar.tokenCache.GetCacheStats()
	health["cache_stats"] = map[string]interface{}{
		"total_tokens":  stats.TotalTokens,
		"active_tokens": stats.ActiveTokens,
		"hit_rate":      stats.HitRate,
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(health)
}

// Helper Methods

// extractSessionID extracts session ID from request
func (ar *AzureAuthRouter) extractSessionID(r *http.Request) string {
	// Try header first
	if sessionID := r.Header.Get("X-Session-ID"); sessionID != "" {
		return sessionID
	}

	// Try cookie
	if cookie, err := r.Cookie("session_id"); err == nil {
		return cookie.Value
	}

	// Try request body
	var body struct {
		SessionID string `json:"session_id"`
	}
	if err := json.NewDecoder(r.Body).Decode(&body); err == nil {
		return body.SessionID
	}

	return ""
}

// extractIDToken extracts ID token from request
func (ar *AzureAuthRouter) extractIDToken(r *http.Request) string {
	// Try Authorization header
	if authHeader := r.Header.Get("Authorization"); authHeader != "" {
		parts := strings.Split(authHeader, " ")
		if len(parts) == 2 && strings.ToLower(parts[0]) == "bearer" {
			return parts[1]
		}
	}

	// Try request body
	var body struct {
		IDToken string `json:"id_token"`
	}
	if err := json.NewDecoder(r.Body).Decode(&body); err == nil {
		return body.IDToken
	}

	return ""
}

// storeTemporarySession stores a temporary OAuth session
func (ar *AzureAuthRouter) storeTemporarySession(sessionID string, data map[string]interface{}, ttl time.Duration) error {
	// This would typically use a temporary storage like Redis
	// For now, we'll use a simple in-memory approach (not production-ready)
	// In production, implement proper temporary storage
	return nil
}

// getTemporarySession retrieves a temporary OAuth session
func (ar *AzureAuthRouter) getTemporarySession(sessionID string) (map[string]interface{}, error) {
	// This would typically retrieve from temporary storage
	// For now, return an error to indicate not implemented
	return nil, customError.ErrTokenInvalid
}

// deleteTemporarySession deletes a temporary OAuth session
func (ar *AzureAuthRouter) deleteTemporarySession(sessionID string) {
	// This would typically delete from temporary storage
	// For now, this is a no-op
}