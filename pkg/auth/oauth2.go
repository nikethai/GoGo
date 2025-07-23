package auth

import (
	"crypto/rand"
	"crypto/sha256"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"net/http"
	"net/url"
	"os"
	"strings"
	"time"

	customError "main/internal/error"
)

// OAuth2Config holds OAuth 2.0 configuration for Azure AD
type OAuth2Config struct {
	TenantID     string
	ClientID     string
	ClientSecret string
	RedirectURI  string
	Scopes       []string
}

// OAuth2TokenResponse represents the token response from Azure AD
type OAuth2TokenResponse struct {
	AccessToken  string `json:"access_token"`
	TokenType    string `json:"token_type"`
	ExpiresIn    int    `json:"expires_in"`
	RefreshToken string `json:"refresh_token"`
	Scope        string `json:"scope"`
	IDToken      string `json:"id_token"`
}

// PKCEChallenge holds PKCE challenge data
type PKCEChallenge struct {
	CodeVerifier  string
	CodeChallenge string
	Method        string
}

// AuthState holds state information for OAuth flow
type AuthState struct {
	State         string
	CodeVerifier  string
	RedirectURI   string
	Timestamp     time.Time
}

// GetOAuth2Config returns OAuth 2.0 configuration from environment variables
func GetOAuth2Config() *OAuth2Config {
	tenantID := os.Getenv("AZURE_AD_TENANT_ID")
	clientID := os.Getenv("AZURE_AD_CLIENT_ID")
	clientSecret := os.Getenv("AZURE_AD_CLIENT_SECRET")
	redirectURI := os.Getenv("AZURE_AD_REDIRECT_URI")

	if tenantID == "" || clientID == "" || redirectURI == "" {
		return nil
	}

	// Default scopes for Azure AD
	scopes := []string{"openid", "profile", "email", "User.Read"}
	if customScopes := os.Getenv("AZURE_AD_SCOPES"); customScopes != "" {
		scopes = strings.Split(customScopes, ",")
		for i, scope := range scopes {
			scopes[i] = strings.TrimSpace(scope)
		}
	}

	return &OAuth2Config{
		TenantID:     tenantID,
		ClientID:     clientID,
		ClientSecret: clientSecret,
		RedirectURI:  redirectURI,
		Scopes:       scopes,
	}
}

// GeneratePKCEChallenge generates PKCE code verifier and challenge
func GeneratePKCEChallenge() (*PKCEChallenge, error) {
	// Generate code verifier (43-128 characters)
	codeVerifier, err := generateRandomString(128)
	if err != nil {
		return nil, fmt.Errorf("failed to generate code verifier: %v", err)
	}

	// Generate code challenge using SHA256
	hash := sha256.Sum256([]byte(codeVerifier))
	codeChallenge := base64.RawURLEncoding.EncodeToString(hash[:])

	return &PKCEChallenge{
		CodeVerifier:  codeVerifier,
		CodeChallenge: codeChallenge,
		Method:        "S256",
	}, nil
}

// GenerateAuthState generates a secure state parameter for OAuth flow
func GenerateAuthState() (string, error) {
	return generateRandomString(32)
}

// BuildAuthorizationURL constructs the Azure AD authorization URL
func BuildAuthorizationURL(config *OAuth2Config, state string, pkce *PKCEChallenge) string {
	baseURL := fmt.Sprintf("https://login.microsoftonline.com/%s/oauth2/v2.0/authorize", config.TenantID)
	
	params := url.Values{
		"client_id":             {config.ClientID},
		"response_type":         {"code"},
		"redirect_uri":          {config.RedirectURI},
		"scope":                 {strings.Join(config.Scopes, " ")},
		"state":                 {state},
		"code_challenge":        {pkce.CodeChallenge},
		"code_challenge_method": {pkce.Method},
		"response_mode":         {"query"},
	}

	return baseURL + "?" + params.Encode()
}

// ExchangeCodeForToken exchanges authorization code for access token
func ExchangeCodeForToken(config *OAuth2Config, code, codeVerifier string) (*OAuth2TokenResponse, error) {
	tokenURL := fmt.Sprintf("https://login.microsoftonline.com/%s/oauth2/v2.0/token", config.TenantID)

	data := url.Values{
		"client_id":     {config.ClientID},
		"grant_type":    {"authorization_code"},
		"code":          {code},
		"redirect_uri":  {config.RedirectURI},
		"code_verifier": {codeVerifier},
	}

	// Add client secret if available (for confidential clients)
	if config.ClientSecret != "" {
		data.Set("client_secret", config.ClientSecret)
	}

	resp, err := http.PostForm(tokenURL, data)
	if err != nil {
		return nil, fmt.Errorf("failed to exchange code for token: %v", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("token exchange failed with status: %d", resp.StatusCode)
	}

	var tokenResponse OAuth2TokenResponse
	if err := json.NewDecoder(resp.Body).Decode(&tokenResponse); err != nil {
		return nil, fmt.Errorf("failed to decode token response: %v", err)
	}

	return &tokenResponse, nil
}

// RefreshAccessToken refreshes an access token using refresh token
func RefreshAccessToken(config *OAuth2Config, refreshToken string) (*OAuth2TokenResponse, error) {
	tokenURL := fmt.Sprintf("https://login.microsoftonline.com/%s/oauth2/v2.0/token", config.TenantID)

	data := url.Values{
		"client_id":     {config.ClientID},
		"grant_type":    {"refresh_token"},
		"refresh_token": {refreshToken},
		"scope":         {strings.Join(config.Scopes, " ")},
	}

	// Add client secret if available (for confidential clients)
	if config.ClientSecret != "" {
		data.Set("client_secret", config.ClientSecret)
	}

	resp, err := http.PostForm(tokenURL, data)
	if err != nil {
		return nil, fmt.Errorf("failed to refresh token: %v", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("token refresh failed with status: %d", resp.StatusCode)
	}

	var tokenResponse OAuth2TokenResponse
	if err := json.NewDecoder(resp.Body).Decode(&tokenResponse); err != nil {
		return nil, fmt.Errorf("failed to decode token response: %v", err)
	}

	return &tokenResponse, nil
}

// ValidateState validates the state parameter to prevent CSRF attacks
func ValidateState(receivedState, expectedState string) error {
	if receivedState == "" {
		return customError.ErrInvalidState
	}
	if receivedState != expectedState {
		return customError.ErrInvalidState
	}
	return nil
}

// generateRandomString generates a cryptographically secure random string
func generateRandomString(length int) (string, error) {
	bytes := make([]byte, length)
	if _, err := rand.Read(bytes); err != nil {
		return "", err
	}
	return base64.RawURLEncoding.EncodeToString(bytes)[:length], nil
}

// ExtractUserInfoFromIDToken extracts user information from ID token
func ExtractUserInfoFromIDToken(idToken string) (*AzureADClaims, error) {
	// Parse the ID token to extract user information
	return ValidateAzureADToken(idToken)
}