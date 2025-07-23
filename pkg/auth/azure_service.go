package auth

import (
	"crypto/aes"
	"crypto/cipher"
	"crypto/rand"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
	"sync"
	"time"

	customError "main/internal/error"
)

// AzureADService interface defines Azure AD authentication operations
type AzureADService interface {
	// OAuth 2.0 flows
	GetAuthorizationURL(state, codeChallenge string) string
	ExchangeCodeForToken(code, codeVerifier string) (*OAuth2TokenResponse, error)
	RefreshToken(refreshToken string) (*OAuth2TokenResponse, error)

	// Token management
	StoreTokenSecurely(userID string, tokens *OAuth2TokenResponse) error
	GetStoredToken(userID string) (*StoredToken, error)
	DeleteStoredToken(userID string) error

	// JWKS management
	GetCachedJWKS() (*JWKS, error)
	RefreshJWKSCache() error
}

// StoredToken represents securely stored token information
type StoredToken struct {
	UserID       string    `json:"user_id"`
	AccessToken  string    `json:"access_token"`
	RefreshToken string    `json:"refresh_token"`
	IDToken      string    `json:"id_token"`
	TokenType    string    `json:"token_type"`
	ExpiresAt    time.Time `json:"expires_at"`
	CreatedAt    time.Time `json:"created_at"`
	LastAccessed time.Time `json:"last_accessed"`
}

// azureADServiceImpl implements AzureADService
type azureADServiceImpl struct {
	config      *OAuth2Config
	tokenStore  map[string]*StoredToken
	jwksCache   *JWKS
	jwksCacheAt time.Time
	mu          sync.RWMutex
	encryptionKey []byte
}

// NewAzureADService creates a new Azure AD service instance
func NewAzureADService() (AzureADService, error) {
	config := GetOAuth2Config()
	if config == nil {
		return nil, fmt.Errorf("Azure AD OAuth2 configuration not found")
	}

	// Initialize encryption key for token storage
	encryptionKey, err := getOrCreateEncryptionKey()
	if err != nil {
		return nil, fmt.Errorf("failed to initialize encryption key: %v", err)
	}

	return &azureADServiceImpl{
		config:        config,
		tokenStore:    make(map[string]*StoredToken),
		encryptionKey: encryptionKey,
	}, nil
}

// GetAuthorizationURL generates the Azure AD authorization URL
func (s *azureADServiceImpl) GetAuthorizationURL(state, codeChallenge string) string {
	pkce := &PKCEChallenge{
		CodeChallenge: codeChallenge,
		Method:        "S256",
	}
	return BuildAuthorizationURL(s.config, state, pkce)
}

// ExchangeCodeForToken exchanges authorization code for tokens
func (s *azureADServiceImpl) ExchangeCodeForToken(code, codeVerifier string) (*OAuth2TokenResponse, error) {
	return ExchangeCodeForToken(s.config, code, codeVerifier)
}

// RefreshToken refreshes an access token using refresh token
func (s *azureADServiceImpl) RefreshToken(refreshToken string) (*OAuth2TokenResponse, error) {
	return RefreshAccessToken(s.config, refreshToken)
}

// StoreTokenSecurely stores tokens with encryption
func (s *azureADServiceImpl) StoreTokenSecurely(userID string, tokens *OAuth2TokenResponse) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	// Calculate expiration time
	expiresAt := time.Now().Add(time.Duration(tokens.ExpiresIn) * time.Second)

	// Create stored token
	storedToken := &StoredToken{
		UserID:       userID,
		AccessToken:  tokens.AccessToken,
		RefreshToken: tokens.RefreshToken,
		IDToken:      tokens.IDToken,
		TokenType:    tokens.TokenType,
		ExpiresAt:    expiresAt,
		CreatedAt:    time.Now(),
		LastAccessed: time.Now(),
	}

	// Encrypt sensitive data
	encryptedToken, err := s.encryptToken(storedToken)
	if err != nil {
		return fmt.Errorf("failed to encrypt token: %v", err)
	}

	// Store encrypted token
	s.tokenStore[userID] = encryptedToken

	return nil
}

// GetStoredToken retrieves and decrypts stored tokens
func (s *azureADServiceImpl) GetStoredToken(userID string) (*StoredToken, error) {
	s.mu.RLock()
	encryptedToken, exists := s.tokenStore[userID]
	s.mu.RUnlock()

	if !exists {
		return nil, customError.ErrTokenInvalid
	}

	// Decrypt token
	decryptedToken, err := s.decryptToken(encryptedToken)
	if err != nil {
		return nil, fmt.Errorf("failed to decrypt token: %v", err)
	}

	// Check if token is expired
	if time.Now().After(decryptedToken.ExpiresAt) {
		return nil, customError.ErrTokenExpired
	}

	// Update last accessed time
	s.mu.Lock()
	decryptedToken.LastAccessed = time.Now()
	encryptedToken, _ = s.encryptToken(decryptedToken)
	s.tokenStore[userID] = encryptedToken
	s.mu.Unlock()

	return decryptedToken, nil
}

// DeleteStoredToken removes stored tokens for a user
func (s *azureADServiceImpl) DeleteStoredToken(userID string) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	delete(s.tokenStore, userID)
	return nil
}

// GetCachedJWKS returns cached JWKS or fetches if expired
func (s *azureADServiceImpl) GetCachedJWKS() (*JWKS, error) {
	s.mu.RLock()
	// Check if cache is valid (cache for 1 hour)
	if s.jwksCache != nil && time.Since(s.jwksCacheAt) < time.Hour {
		jwks := s.jwksCache
		s.mu.RUnlock()
		return jwks, nil
	}
	s.mu.RUnlock()

	// Cache is expired or empty, refresh it
	return s.refreshJWKSCacheInternal()
}

// RefreshJWKSCache forces a refresh of the JWKS cache
func (s *azureADServiceImpl) RefreshJWKSCache() error {
	_, err := s.refreshJWKSCacheInternal()
	return err
}

// refreshJWKSCacheInternal internal method to refresh JWKS cache
func (s *azureADServiceImpl) refreshJWKSCacheInternal() (*JWKS, error) {
	s.mu.Lock()
	defer s.mu.Unlock()

	// Fetch JWKS from Azure AD
	azureConfig := GetAzureADConfig()
	if azureConfig == nil {
		return nil, fmt.Errorf("Azure AD configuration not found")
	}

	// Use existing function to get JWKS
	// We'll need to modify getPublicKeyFromJWKS to return JWKS
	jwks, err := fetchJWKS(azureConfig.JWKSEndpoint)
	if err != nil {
		return nil, fmt.Errorf("failed to fetch JWKS: %v", err)
	}

	s.jwksCache = jwks
	s.jwksCacheAt = time.Now()

	return jwks, nil
}

// encryptToken encrypts a stored token
func (s *azureADServiceImpl) encryptToken(token *StoredToken) (*StoredToken, error) {
	// Encrypt access token
	encryptedAccessToken, err := s.encrypt(token.AccessToken)
	if err != nil {
		return nil, err
	}

	// Encrypt refresh token
	encryptedRefreshToken, err := s.encrypt(token.RefreshToken)
	if err != nil {
		return nil, err
	}

	// Encrypt ID token
	encryptedIDToken, err := s.encrypt(token.IDToken)
	if err != nil {
		return nil, err
	}

	// Create encrypted token copy
	encryptedToken := *token
	encryptedToken.AccessToken = encryptedAccessToken
	encryptedToken.RefreshToken = encryptedRefreshToken
	encryptedToken.IDToken = encryptedIDToken

	return &encryptedToken, nil
}

// decryptToken decrypts a stored token
func (s *azureADServiceImpl) decryptToken(encryptedToken *StoredToken) (*StoredToken, error) {
	// Decrypt access token
	accessToken, err := s.decrypt(encryptedToken.AccessToken)
	if err != nil {
		return nil, err
	}

	// Decrypt refresh token
	refreshToken, err := s.decrypt(encryptedToken.RefreshToken)
	if err != nil {
		return nil, err
	}

	// Decrypt ID token
	idToken, err := s.decrypt(encryptedToken.IDToken)
	if err != nil {
		return nil, err
	}

	// Create decrypted token copy
	decryptedToken := *encryptedToken
	decryptedToken.AccessToken = accessToken
	decryptedToken.RefreshToken = refreshToken
	decryptedToken.IDToken = idToken

	return &decryptedToken, nil
}

// encrypt encrypts a string using AES-GCM
func (s *azureADServiceImpl) encrypt(plaintext string) (string, error) {
	if plaintext == "" {
		return "", nil
	}

	block, err := aes.NewCipher(s.encryptionKey)
	if err != nil {
		return "", err
	}

	gcm, err := cipher.NewGCM(block)
	if err != nil {
		return "", err
	}

	nonce := make([]byte, gcm.NonceSize())
	if _, err := io.ReadFull(rand.Reader, nonce); err != nil {
		return "", err
	}

	ciphertext := gcm.Seal(nonce, nonce, []byte(plaintext), nil)
	return base64.StdEncoding.EncodeToString(ciphertext), nil
}

// decrypt decrypts a string using AES-GCM
func (s *azureADServiceImpl) decrypt(ciphertext string) (string, error) {
	if ciphertext == "" {
		return "", nil
	}

	data, err := base64.StdEncoding.DecodeString(ciphertext)
	if err != nil {
		return "", err
	}

	block, err := aes.NewCipher(s.encryptionKey)
	if err != nil {
		return "", err
	}

	gcm, err := cipher.NewGCM(block)
	if err != nil {
		return "", err
	}

	nonceSize := gcm.NonceSize()
	if len(data) < nonceSize {
		return "", fmt.Errorf("ciphertext too short")
	}

	nonce, ciphertextBytes := data[:nonceSize], data[nonceSize:]
	plaintext, err := gcm.Open(nil, nonce, ciphertextBytes, nil)
	if err != nil {
		return "", err
	}

	return string(plaintext), nil
}

// getOrCreateEncryptionKey gets or creates an encryption key
func getOrCreateEncryptionKey() ([]byte, error) {
	// Try to get key from environment variable
	if keyStr := os.Getenv("AZURE_AD_ENCRYPTION_KEY"); keyStr != "" {
		key, err := base64.StdEncoding.DecodeString(keyStr)
		if err == nil && len(key) == 32 {
			return key, nil
		}
	}

	// Generate a new 32-byte key for AES-256
	key := make([]byte, 32)
	if _, err := rand.Read(key); err != nil {
		return nil, err
	}

	// In production, you should store this key securely
	// For development, we'll just use the generated key
	fmt.Printf("Generated new encryption key. Set AZURE_AD_ENCRYPTION_KEY=%s\n", 
		base64.StdEncoding.EncodeToString(key))

	return key, nil
}

// fetchJWKS fetches JWKS from Azure AD endpoint
func fetchJWKS(jwksURL string) (*JWKS, error) {
	resp, err := http.Get(jwksURL)
	if err != nil {
		return nil, fmt.Errorf("failed to fetch JWKS: %v", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("JWKS endpoint returned status: %d", resp.StatusCode)
	}

	var jwks JWKS
	if err := json.NewDecoder(resp.Body).Decode(&jwks); err != nil {
		return nil, fmt.Errorf("failed to decode JWKS: %v", err)
	}

	return &jwks, nil
}