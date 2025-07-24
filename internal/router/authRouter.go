package router

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"net/url"
	"strings"
	"time"

	"github.com/go-chi/chi/v5"

	"main/db"
	"main/internal/config"
	customError "main/internal/error"
	"main/internal/model"
	mongorepo "main/internal/repository/mongo"
	"main/internal/server/response"
	"main/internal/service"
	"main/pkg/auth"
)

// AuthRouter handles both traditional and Azure AD authentication routes
type AuthRouter struct {
	// Traditional auth services
	authService *service.AuthService
	userService *service.UserService
	
	// Azure AD services (optional)
	oauth2Service   auth.AzureADService
	sessionManager  auth.SessionManager
	tokenCache      auth.TokenCache
	config          *auth.OAuth2Config
	
	// Feature flags
	azureEnabled bool
}

// NewAuthRouter creates a new authentication router with traditional auth
func NewAuthRouter() *AuthRouter {
	// Initialize repositories
	userRepo := mongorepo.NewMongoRepository[*model.User](db.MongoDatabase, config.UserCollection)
	accountRepo := mongorepo.NewMongoRepository[*model.Account](db.MongoDatabase, config.AccountCollection)

	return &AuthRouter{
		authService: service.NewAuthService(),
		userService: service.NewUserService(userRepo, accountRepo),
		azureEnabled: false,
	}
}

// NewAuthRouterWithAzure creates a new authentication router with Azure AD support
func NewAuthRouterWithAzure(
	oauth2Service auth.AzureADService,
	sessionManager auth.SessionManager,
	tokenCache auth.TokenCache,
	oauth2Config *auth.OAuth2Config,
) *AuthRouter {
	// Initialize repositories
	userRepo := mongorepo.NewMongoRepository[*model.User](db.MongoDatabase, config.UserCollection)
	accountRepo := mongorepo.NewMongoRepository[*model.Account](db.MongoDatabase, config.AccountCollection)

	return &AuthRouter{
		authService:    service.NewAuthService(),
		userService:    service.NewUserService(userRepo, accountRepo),
		oauth2Service:  oauth2Service,
		sessionManager: sessionManager,
		tokenCache:     tokenCache,
		config:         oauth2Config,
		azureEnabled:   true,
	}
}

// SetupRoutes sets up authentication routes for both traditional and Azure AD auth
func (ar *AuthRouter) SetupRoutes() chi.Router {
	r := chi.NewRouter()
	
	// Traditional authentication routes
	r.Post("/login", ar.login)
	r.Post("/register", ar.register)
	
	// Azure AD authentication routes (if enabled)
	if ar.azureEnabled {
		r.Get("/azure/login", ar.handleAzureLogin)
		r.Get("/azure/callback", ar.handleAzureCallback)
		r.Post("/azure/logout", ar.handleAzureLogout)
		r.Get("/azure/profile", ar.handleAzureProfile)
		r.Post("/azure/refresh", ar.handleTokenRefresh)
	}
	
	return r
}

// Azure AD Authentication Handlers

// handleAzureLogin initiates Azure AD OAuth 2.0 login flow
func (ar *AuthRouter) handleAzureLogin(w http.ResponseWriter, r *http.Request) {
	if !ar.azureEnabled {
		http.Error(w, "Azure AD authentication not enabled", http.StatusNotImplemented)
		return
	}

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
func (ar *AuthRouter) handleAzureCallback(w http.ResponseWriter, r *http.Request) {
	if !ar.azureEnabled {
		http.Error(w, "Azure AD authentication not enabled", http.StatusNotImplemented)
		return
	}

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
func (ar *AuthRouter) handleAzureLogout(w http.ResponseWriter, r *http.Request) {
	if !ar.azureEnabled {
		http.Error(w, "Azure AD authentication not enabled", http.StatusNotImplemented)
		return
	}

	// Get session ID from request
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
	postLogoutRedirectURL := ar.config.RedirectURI
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

// handleAzureProfile returns user profile information
func (ar *AuthRouter) handleAzureProfile(w http.ResponseWriter, r *http.Request) {
	if !ar.azureEnabled {
		http.Error(w, "Azure AD authentication not enabled", http.StatusNotImplemented)
		return
	}

	// Get session ID from request
	sessionID := ar.extractSessionID(r)
	if sessionID == "" {
		http.Error(w, "Session ID required", http.StatusBadRequest)
		return
	}

	// Get session
	session, err := ar.sessionManager.GetSession(sessionID)
	if err != nil {
		http.Error(w, "Invalid session", http.StatusUnauthorized)
		return
	}

	// Get user sessions for additional context
	sessions, _ := ar.sessionManager.GetUserSessions(session.UserID)

	response := map[string]interface{}{
		"user_id":       session.UserID,
		"auth_type":     "azure_ad",
		"session_count": len(sessions),
		"expires_at":    session.ExpiresAt,
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(response)
}

// handleTokenRefresh handles token refresh requests
func (ar *AuthRouter) handleTokenRefresh(w http.ResponseWriter, r *http.Request) {
	if !ar.azureEnabled {
		http.Error(w, "Azure AD authentication not enabled", http.StatusNotImplemented)
		return
	}

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

	response := map[string]interface{}{
		"access_token": tokenResponse.AccessToken,
		"token_type":   tokenResponse.TokenType,
		"expires_in":   tokenResponse.ExpiresIn,
		"scope":        tokenResponse.Scope,
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(response)
}

// Helper Methods

// extractSessionID extracts session ID from request
func (ar *AuthRouter) extractSessionID(r *http.Request) string {
	// Try header first
	if sessionID := r.Header.Get("X-Session-ID"); sessionID != "" {
		return sessionID
	}

	// Try cookie
	if cookie, err := r.Cookie("session_id"); err == nil {
		return cookie.Value
	}

	return ""
}

// storeTemporarySession stores a temporary OAuth session
func (ar *AuthRouter) storeTemporarySession(sessionID string, data map[string]interface{}, ttl time.Duration) error {
	// This would typically use a temporary storage like Redis
	// For now, we'll use a simple in-memory approach (not production-ready)
	return nil
}

// getTemporarySession retrieves a temporary OAuth session
func (ar *AuthRouter) getTemporarySession(sessionID string) (map[string]interface{}, error) {
	// This would typically retrieve from temporary storage
	// For now, return mock data for demonstration
	return map[string]interface{}{
		"state":         strings.TrimPrefix(sessionID, "oauth_session_"),
		"pkce_verifier": "mock_verifier",
		"created_at":    time.Now(),
	}, nil
}

// deleteTemporarySession deletes a temporary OAuth session
func (ar *AuthRouter) deleteTemporarySession(sessionID string) {
	// This would typically delete from temporary storage
	// For now, this is a no-op
}

// login godoc
// @Summary User login
// @Description Authenticate user with username and password
// @Tags Authentication
// @Accept json
// @Produce json
// @Param request body model.AccountRequest true "Login credentials"
// @Success 200 {object} model.UserResponse "User successfully authenticated"
// @Success 200 {object} model.AccountResponse "Account found but no user profile"
// @Failure 400 {object} map[string]string "Bad request"
// @Failure 401 {object} map[string]string "Invalid credentials"
// @Failure 500 {object} map[string]string "Internal server error"
// @Router /auth/login [post]
func (ar *AuthRouter) login(w http.ResponseWriter, r *http.Request) {
	var authReq model.AccountRequest
	err := json.NewDecoder(r.Body).Decode(&authReq)

	if err != nil {
		response.BadRequest(w, customError.ErrInvalidRequestFormat.Error()+": "+err.Error())
		return
	}

	// Log login attempt
	fmt.Println("===== Login attempt for username:", authReq.Username, "=====")

	account, err := ar.authService.Login(authReq.Username, authReq.Password)
	if err != nil {
		fmt.Println("===== Login failed for username:", authReq.Username, "Error:", err.Error(), "=====")
		response.Unauthorized(w, customError.ErrInvalidCredentials.Error())
		return
	}

	// Get user by account ID using the new service method
	userWithAccount, usrErr := ar.userService.GetUserByAccountID(context.Background(), account.ID)

	if usrErr != nil {
		// If user not found, return just the account (maintaining old behavior)
		if usrErr == customError.ErrUserNotFound {
			fmt.Println("===== User not found for account ID:", account.ID.Hex(), "=====")
			response.Success(w, http.StatusOK, account, "Login successful")
			return
		}
		fmt.Println("===== Error retrieving user profile:", usrErr.Error(), "=====")
		response.InternalServerError(w, customError.ErrProfileRetrieval.Error()+": "+usrErr.Error())
		return
	}

	fmt.Println("===== Login successful for username:", authReq.Username, "=====")
	response.Success(w, http.StatusOK, userWithAccount, "Login successful")
}

// register godoc
// @Summary User registration
// @Description Register a new user account with roles
// @Tags Authentication
// @Accept json
// @Produce json
// @Param request body model.AccountRegister true "Registration details"
// @Success 200 {object} model.AccountResponse "Account successfully created"
// @Failure 400 {object} map[string]string "Bad request"
// @Failure 409 {object} map[string]string "Username already exists"
// @Failure 500 {object} map[string]string "Internal server error"
// @Router /auth/register [post]
func (ar *AuthRouter) register(w http.ResponseWriter, r *http.Request) {
	var authRegis model.AccountRegister
	err := json.NewDecoder(r.Body).Decode(&authRegis)

	if err != nil {
		response.BadRequest(w, customError.ErrInvalidRequestFormat.Error()+": "+err.Error())
		return
	}

	// Validate required fields
	if authRegis.Username == "" || authRegis.Password == "" || authRegis.Email == "" {
		response.BadRequest(w, customError.ErrMissingRegistrationDetails.Error())
		return
	}

	// Register the account
	accountResponse, err := ar.authService.Register(authRegis.Username, authRegis.Password, authRegis.Email, authRegis.Roles)
	if err != nil {
		if err == customError.ErrDuplicateUsername || err == customError.ErrDuplicateEmail {
			response.Conflict(w, customError.ErrDuplicateUsernameOrEmail.Error())
			return
		}
		response.InternalServerError(w, customError.ErrRegistrationFailed.Error()+": "+err.Error())
		return
	}

	response.Created(w, accountResponse, "Account registered successfully")
}
