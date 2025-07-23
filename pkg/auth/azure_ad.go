package auth

import (
	"crypto/rsa"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"math/big"
	"net/http"
	"os"
	"strings"
	"time"

	customError "main/internal/error"

	"github.com/golang-jwt/jwt/v5"
)

// AzureADClaims represents the claims structure for Azure AD JWT tokens
type AzureADClaims struct {
	Audience          string   `json:"aud"`
	Issuer            string   `json:"iss"`
	IssuedAt          int64    `json:"iat"`
	NotBefore         int64    `json:"nbf"`
	ExpiresAt         int64    `json:"exp"`
	AppID             string   `json:"appid"`
	AppIDACR          string   `json:"appidacr"`
	IDType            string   `json:"idtyp"`
	ObjectID          string   `json:"oid"`
	Roles             []string `json:"roles"`
	Subject           string   `json:"sub"`
	TenantID          string   `json:"tid"`
	UniqueID          string   `json:"uti"`
	Version           string   `json:"ver"`
	PreferredUsername string   `json:"preferred_username"`
	Name              string   `json:"name"`
	Email             string   `json:"email"`
	jwt.RegisteredClaims
}

// JWKSKey represents a JSON Web Key from Azure AD JWKS endpoint
type JWKSKey struct {
	Kty string `json:"kty"`
	Use string `json:"use"`
	Kid string `json:"kid"`
	X5t string `json:"x5t"`
	N   string `json:"n"`
	E   string `json:"e"`
	X5c []string `json:"x5c"`
}

// JWKS represents the JSON Web Key Set from Azure AD
type JWKS struct {
	Keys []JWKSKey `json:"keys"`
}

// AzureADConfig holds Azure AD configuration
type AzureADConfig struct {
	TenantID     string
	ClientID     string
	JWKSEndpoint string
	Issuer       string
}

// GetAzureADConfig returns Azure AD configuration from environment variables
func GetAzureADConfig() *AzureADConfig {
	tenantID := os.Getenv("AZURE_AD_TENANT_ID")
	clientID := os.Getenv("AZURE_AD_CLIENT_ID")
	
	if tenantID == "" || clientID == "" {
		return nil
	}
	
	return &AzureADConfig{
		TenantID:     tenantID,
		ClientID:     clientID,
		JWKSEndpoint: fmt.Sprintf("https://login.microsoftonline.com/%s/discovery/v2.0/keys", tenantID),
		Issuer:       fmt.Sprintf("https://login.microsoftonline.com/%s/v2.0", tenantID),
	}
}

// ValidateAzureADToken validates an Azure AD JWT token
func ValidateAzureADToken(tokenString string) (*AzureADClaims, error) {
	config := GetAzureADConfig()
	if config == nil {
		return nil, fmt.Errorf("Azure AD configuration not found")
	}

	// Parse token without verification first to get the header
	token, _, err := new(jwt.Parser).ParseUnverified(tokenString, &AzureADClaims{})
	if err != nil {
		return nil, customError.ErrTokenValidation
	}

	// Get the key ID from token header
	kid, ok := token.Header["kid"].(string)
	if !ok {
		return nil, fmt.Errorf("token header missing kid")
	}

	// Get the public key from Azure AD JWKS endpoint
	publicKey, err := getPublicKeyFromJWKS(config.JWKSEndpoint, kid)
	if err != nil {
		return nil, fmt.Errorf("failed to get public key: %v", err)
	}

	// Parse and validate the token with the public key
	token, err = jwt.ParseWithClaims(tokenString, &AzureADClaims{}, func(token *jwt.Token) (interface{}, error) {
		// Validate the signing method
		if _, ok := token.Method.(*jwt.SigningMethodRSA); !ok {
			return nil, fmt.Errorf("unexpected signing method: %v", token.Header["alg"])
		}
		return publicKey, nil
	})

	if err != nil {
		if strings.Contains(err.Error(), "token is expired") {
			return nil, customError.ErrTokenExpired
		}
		return nil, customError.ErrTokenValidation
	}

	// Extract and validate the claims
	if claims, ok := token.Claims.(*AzureADClaims); ok && token.Valid {
		// Validate issuer
		if claims.Issuer != config.Issuer {
			return nil, fmt.Errorf("invalid issuer: expected %s, got %s", config.Issuer, claims.Issuer)
		}

		// Validate audience (client ID)
		if claims.Audience != config.ClientID {
			return nil, fmt.Errorf("invalid audience: expected %s, got %s", config.ClientID, claims.Audience)
		}

		// Validate expiration
		if time.Now().Unix() > claims.ExpiresAt {
			return nil, customError.ErrTokenExpired
		}

		return claims, nil
	}

	return nil, customError.ErrTokenInvalid
}

// getPublicKeyFromJWKS retrieves the public key from Azure AD JWKS endpoint
func getPublicKeyFromJWKS(jwksURL, kid string) (*rsa.PublicKey, error) {
	// Make HTTP request to JWKS endpoint
	resp, err := http.Get(jwksURL)
	if err != nil {
		return nil, fmt.Errorf("failed to fetch JWKS: %v", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("JWKS endpoint returned status: %d", resp.StatusCode)
	}

	// Parse JWKS response
	var jwks JWKS
	if err := json.NewDecoder(resp.Body).Decode(&jwks); err != nil {
		return nil, fmt.Errorf("failed to decode JWKS: %v", err)
	}

	// Find the key with matching kid
	for _, key := range jwks.Keys {
		if key.Kid == kid {
			return buildRSAPublicKey(key)
		}
	}

	return nil, fmt.Errorf("key with kid %s not found", kid)
}

// buildRSAPublicKey constructs an RSA public key from JWK parameters
func buildRSAPublicKey(key JWKSKey) (*rsa.PublicKey, error) {
	// Decode the modulus (n)
	nBytes, err := base64.RawURLEncoding.DecodeString(key.N)
	if err != nil {
		return nil, fmt.Errorf("failed to decode modulus: %v", err)
	}
	n := new(big.Int).SetBytes(nBytes)

	// Decode the exponent (e)
	eBytes, err := base64.RawURLEncoding.DecodeString(key.E)
	if err != nil {
		return nil, fmt.Errorf("failed to decode exponent: %v", err)
	}
	e := new(big.Int).SetBytes(eBytes)

	// Create RSA public key
	publicKey := &rsa.PublicKey{
		N: n,
		E: int(e.Int64()),
	}

	return publicKey, nil
}

// ConvertAzureADClaimsToJWTClaims converts Azure AD claims to internal JWT claims format
func ConvertAzureADClaimsToJWTClaims(azureClaims *AzureADClaims) *JWTClaims {
	return &JWTClaims{
		UserID:   azureClaims.ObjectID, // Use Azure AD object ID as user ID
		Username: getUsername(azureClaims),
		Roles:    azureClaims.Roles,
		RegisteredClaims: jwt.RegisteredClaims{
			ExpiresAt: jwt.NewNumericDate(time.Unix(azureClaims.ExpiresAt, 0)),
			IssuedAt:  jwt.NewNumericDate(time.Unix(azureClaims.IssuedAt, 0)),
			Issuer:    azureClaims.Issuer,
			Subject:   azureClaims.Subject,
		},
	}
}

// getUsername extracts username from Azure AD claims with fallback logic
func getUsername(claims *AzureADClaims) string {
	if claims.PreferredUsername != "" {
		return claims.PreferredUsername
	}
	if claims.Email != "" {
		return claims.Email
	}
	if claims.Name != "" {
		return claims.Name
	}
	return claims.Subject // Fallback to subject
}