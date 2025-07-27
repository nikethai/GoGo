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

	firebaseAuth "firebase.google.com/go/v4/auth"
	"github.com/go-chi/chi/v5"

	"main/db"
	"main/internal/config"
	customError "main/internal/error"
	"main/internal/model"
	mongorepo "main/internal/repository/mongo"
	"main/internal/server/response"
	"main/internal/service"
	mainAuth "main/pkg/auth"
)

// AuthRouter handles traditional, Azure AD, and Firebase authentication routes
type AuthRouter struct {
	// Traditional auth services
	authService *service.AuthService
	userService *service.UserService
	
	// Azure AD services (optional)
	oauth2Service   mainAuth.AzureADService
	sessionManager  mainAuth.SessionManager
	tokenCache      mainAuth.TokenCache
	config          *mainAuth.OAuth2Config
	
	// Firebase services (optional)
	firebaseService *mainAuth.FirebaseService
	
	// Feature flags
	azureEnabled    bool
	firebaseEnabled bool
}

// NewAuthRouter creates a new authentication router with traditional auth
func NewAuthRouter() *AuthRouter {
	// Initialize repositories
	userRepo := mongorepo.NewMongoRepository[*model.User](db.MongoDatabase, config.UserCollection)
	accountRepo := mongorepo.NewMongoRepository[*model.Account](db.MongoDatabase, config.AccountCollection)

	return &AuthRouter{
		authService:     service.NewAuthService(),
		userService:     service.NewUserService(userRepo, accountRepo),
		azureEnabled:    false,
		firebaseEnabled: false,
	}
}

// NewAuthRouterWithAzure creates a new authentication router with Azure AD support
func NewAuthRouterWithAzure(
	oauth2Service mainAuth.AzureADService,
	sessionManager mainAuth.SessionManager,
	tokenCache mainAuth.TokenCache,
	oauth2Config *mainAuth.OAuth2Config,
) *AuthRouter {
	// Initialize repositories
	userRepo := mongorepo.NewMongoRepository[*model.User](db.MongoDatabase, config.UserCollection)
	accountRepo := mongorepo.NewMongoRepository[*model.Account](db.MongoDatabase, config.AccountCollection)

	return &AuthRouter{
		authService:     service.NewAuthService(),
		userService:     service.NewUserService(userRepo, accountRepo),
		oauth2Service:   oauth2Service,
		sessionManager:  sessionManager,
		tokenCache:      tokenCache,
		config:          oauth2Config,
		azureEnabled:    true,
		firebaseEnabled: false,
	}
}

// NewAuthRouterWithFirebase creates a new authentication router with Firebase support
func NewAuthRouterWithFirebase(firebaseService *mainAuth.FirebaseService) *AuthRouter {
	// Initialize repositories
	userRepo := mongorepo.NewMongoRepository[*model.User](db.MongoDatabase, config.UserCollection)
	accountRepo := mongorepo.NewMongoRepository[*model.Account](db.MongoDatabase, config.AccountCollection)

	return &AuthRouter{
		authService:     service.NewAuthService(),
		userService:     service.NewUserService(userRepo, accountRepo),
		firebaseService: firebaseService,
		azureEnabled:    false,
		firebaseEnabled: true,
	}
}

// NewAuthRouterWithAll creates a new authentication router with both Azure AD and Firebase support
func NewAuthRouterWithAll(
	oauth2Service mainAuth.AzureADService,
	sessionManager mainAuth.SessionManager,
	tokenCache mainAuth.TokenCache,
	oauth2Config *mainAuth.OAuth2Config,
	firebaseService *mainAuth.FirebaseService,
) *AuthRouter {
	// Initialize repositories
	userRepo := mongorepo.NewMongoRepository[*model.User](db.MongoDatabase, config.UserCollection)
	accountRepo := mongorepo.NewMongoRepository[*model.Account](db.MongoDatabase, config.AccountCollection)

	return &AuthRouter{
		authService:     service.NewAuthService(),
		userService:     service.NewUserService(userRepo, accountRepo),
		oauth2Service:   oauth2Service,
		sessionManager:  sessionManager,
		tokenCache:      tokenCache,
		config:          oauth2Config,
		firebaseService: firebaseService,
		azureEnabled:    true,
		firebaseEnabled: true,
	}
}

// SetupRoutes sets up authentication routes for traditional, Azure AD, and Firebase auth
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
	
	// Firebase authentication routes (if enabled)
	if ar.firebaseEnabled {
		r.Post("/firebase/verify", ar.handleFirebaseTokenVerification)
		r.Post("/firebase/register", ar.handleFirebaseUserRegistration)
		r.Get("/firebase/profile", ar.handleFirebaseProfile)
		r.Post("/firebase/claims", ar.handleFirebaseCustomClaims)
		r.Delete("/firebase/user/{uid}", ar.handleFirebaseUserDeletion)
		r.Put("/firebase/user/{uid}", ar.handleFirebaseUserUpdate)
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
	pkceChallenge, err := mainAuth.GeneratePKCEChallenge()
	if err != nil {
		http.Error(w, "Failed to generate PKCE challenge", http.StatusInternalServerError)
		return
	}

	// Generate state parameter
	state, err := mainAuth.GenerateAuthState()
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
	authURL := mainAuth.BuildAuthorizationURL(ar.config, state, pkceChallenge)

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
	if err := mainAuth.ValidateState(state, sessionData["state"].(string)); err != nil {
		http.Error(w, "State validation failed", http.StatusBadRequest)
		return
	}

	// Exchange authorization code for tokens
	pkceVerifier := sessionData["pkce_verifier"].(string)
	tokenResponse, err := mainAuth.ExchangeCodeForToken(ar.config, code, pkceVerifier)
	if err != nil {
		log.Printf("Token exchange failed: %v", err)
		http.Error(w, "Failed to exchange code for tokens", http.StatusInternalServerError)
		return
	}

	// Extract user information from ID token
	userInfo, err := mainAuth.ExtractUserInfoFromIDToken(tokenResponse.IDToken)
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
	tokenResponse, err := mainAuth.RefreshAccessToken(ar.config, request.RefreshToken)
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
			response.Success(w, http.StatusOK, map[string]interface{}{"account": account, "access_token": account.Token}, "Login successful")
			return
		}
		fmt.Println("===== Error retrieving user profile:", usrErr.Error(), "=====")
		response.InternalServerError(w, customError.ErrProfileRetrieval.Error()+": "+usrErr.Error())
		return
	}

	fmt.Println("===== Login successful for username:", authReq.Username, "=====")
	response.Success(w, http.StatusOK, map[string]interface{}{"user": userWithAccount, "access_token": account.Token}, "Login successful")
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

// Firebase Authentication Handlers

// handleFirebaseTokenVerification verifies Firebase ID tokens
func (ar *AuthRouter) handleFirebaseTokenVerification(w http.ResponseWriter, r *http.Request) {
	if !ar.firebaseEnabled {
		http.Error(w, "Firebase authentication not enabled", http.StatusNotImplemented)
		return
	}

	var request struct {
		IDToken string `json:"id_token"`
	}

	if err := json.NewDecoder(r.Body).Decode(&request); err != nil {
		http.Error(w, "Invalid request format", http.StatusBadRequest)
		return
	}

	if request.IDToken == "" {
		http.Error(w, "ID token is required", http.StatusBadRequest)
		return
	}

	// Verify the Firebase ID token
	claims, err := ar.firebaseService.ValidateIDToken(context.Background(), request.IDToken)
	if err != nil {
		http.Error(w, "Invalid ID token: "+err.Error(), http.StatusUnauthorized)
		return
	}

	response := map[string]interface{}{
		"valid":   true,
		"uid":     claims.UserID,
		"email":   claims.Email,
		"claims":  claims.Custom,
		"expires": claims.Expiry,
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(response)
}

// handleFirebaseUserRegistration creates a new Firebase user
func (ar *AuthRouter) handleFirebaseUserRegistration(w http.ResponseWriter, r *http.Request) {
	if !ar.firebaseEnabled {
		http.Error(w, "Firebase authentication not enabled", http.StatusNotImplemented)
		return
	}

	var request struct {
		Email    string            `json:"email"`
		Password string            `json:"password"`
		Claims   map[string]interface{} `json:"custom_claims,omitempty"`
	}

	if err := json.NewDecoder(r.Body).Decode(&request); err != nil {
		http.Error(w, "Invalid request format", http.StatusBadRequest)
		return
	}

	if request.Email == "" || request.Password == "" {
		http.Error(w, "Email and password are required", http.StatusBadRequest)
		return
	}

	// Create Firebase user
	user, err := ar.firebaseService.CreateUser(context.Background(), request.Email, request.Password, "")
	if err != nil {
		http.Error(w, "Failed to create user: "+err.Error(), http.StatusInternalServerError)
		return
	}

	// Set custom claims if provided
	if request.Claims != nil {
		if err := ar.firebaseService.SetCustomClaims(context.Background(), user.UID, request.Claims); err != nil {
			log.Printf("Warning: Failed to set custom claims for user %s: %v", user.UID, err)
		}
	}

	response := map[string]interface{}{
		"uid":   user.UID,
		"email": user.Email,
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(response)
}

// handleFirebaseProfile retrieves Firebase user profile
func (ar *AuthRouter) handleFirebaseProfile(w http.ResponseWriter, r *http.Request) {
	if !ar.firebaseEnabled {
		http.Error(w, "Firebase authentication not enabled", http.StatusNotImplemented)
		return
	}

	// Extract UID from query parameter or header
	uid := r.URL.Query().Get("uid")
	if uid == "" {
		uid = r.Header.Get("X-Firebase-UID")
	}

	if uid == "" {
		http.Error(w, "User UID is required", http.StatusBadRequest)
		return
	}

	// Get user from Firebase
	user, err := ar.firebaseService.GetUser(context.Background(), uid)
	if err != nil {
		http.Error(w, "User not found: "+err.Error(), http.StatusNotFound)
		return
	}

	response := map[string]interface{}{
		"uid":           user.UID,
		"email":         user.Email,
		"email_verified": user.EmailVerified,
		"disabled":      user.Disabled,
		"custom_claims": user.CustomClaims,
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(response)
}

// handleFirebaseCustomClaims sets custom claims for a Firebase user
func (ar *AuthRouter) handleFirebaseCustomClaims(w http.ResponseWriter, r *http.Request) {
	if !ar.firebaseEnabled {
		http.Error(w, "Firebase authentication not enabled", http.StatusNotImplemented)
		return
	}

	var request struct {
		UID    string                 `json:"uid"`
		Claims map[string]interface{} `json:"custom_claims"`
	}

	if err := json.NewDecoder(r.Body).Decode(&request); err != nil {
		http.Error(w, "Invalid request format", http.StatusBadRequest)
		return
	}

	if request.UID == "" {
		http.Error(w, "User UID is required", http.StatusBadRequest)
		return
	}

	// Set custom claims
	if err := ar.firebaseService.SetCustomClaims(context.Background(), request.UID, request.Claims); err != nil {
		http.Error(w, "Failed to set custom claims: "+err.Error(), http.StatusInternalServerError)
		return
	}

	response := map[string]interface{}{
		"success": true,
		"message": "Custom claims updated successfully",
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(response)
}

// handleFirebaseUserDeletion deletes a Firebase user
func (ar *AuthRouter) handleFirebaseUserDeletion(w http.ResponseWriter, r *http.Request) {
	if !ar.firebaseEnabled {
		http.Error(w, "Firebase authentication not enabled", http.StatusNotImplemented)
		return
	}

	uid := chi.URLParam(r, "uid")
	if uid == "" {
		http.Error(w, "User UID is required", http.StatusBadRequest)
		return
	}

	// Delete user from Firebase
	if err := ar.firebaseService.DeleteUser(context.Background(), uid); err != nil {
		http.Error(w, "Failed to delete user: "+err.Error(), http.StatusInternalServerError)
		return
	}

	response := map[string]interface{}{
		"success": true,
		"message": "User deleted successfully",
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(response)
}

// handleFirebaseUserUpdate updates a Firebase user
func (ar *AuthRouter) handleFirebaseUserUpdate(w http.ResponseWriter, r *http.Request) {
	if !ar.firebaseEnabled {
		http.Error(w, "Firebase authentication not enabled", http.StatusNotImplemented)
		return
	}

	uid := chi.URLParam(r, "uid")
	if uid == "" {
		http.Error(w, "User UID is required", http.StatusBadRequest)
		return
	}

	var request struct {
		Email    *string `json:"email,omitempty"`
		Password *string `json:"password,omitempty"`
		Disabled *bool   `json:"disabled,omitempty"`
	}

	if err := json.NewDecoder(r.Body).Decode(&request); err != nil {
		http.Error(w, "Invalid request format", http.StatusBadRequest)
		return
	}

	// Create update parameters
	params := &firebaseAuth.UserToUpdate{}
	if request.Email != nil {
		params = params.Email(*request.Email)
	}
	if request.Password != nil {
		params = params.Password(*request.Password)
	}
	if request.Disabled != nil {
		params = params.Disabled(*request.Disabled)
	}

	// Update user in Firebase
	user, err := ar.firebaseService.UpdateUser(context.Background(), uid, params)
	if err != nil {
		http.Error(w, "Failed to update user: "+err.Error(), http.StatusInternalServerError)
		return
	}

	response := map[string]interface{}{
		"uid":           user.UID,
		"email":         user.Email,
		"email_verified": user.EmailVerified,
		"disabled":      user.Disabled,
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(response)
}
