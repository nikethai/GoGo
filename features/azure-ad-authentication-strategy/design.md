# Azure AD JWT Authentication Strategy - Design Architecture

## Overview

This document provides a comprehensive architectural design for optimizing Azure AD JWT authentication in the Gogo survey management system. Based on analysis of the current implementation and industry best practices, this design establishes clear patterns for token acquisition and validation.

### Problem Statement Summary

The current Azure AD implementation supports token validation but lacks clear guidance on:
- Whether clients or backend should acquire Azure AD tokens
- Optimal authentication flows for different client types
- Security considerations for each acquisition pattern
- Performance optimization strategies

### Architecture Approach Overview

**Recommended Strategy: Hybrid Token Acquisition Pattern**

Support both client-side and server-side token acquisition based on use case:
- **Client-Side Acquisition**: For SPAs, mobile apps, and desktop applications
- **Server-Side Acquisition**: For traditional web applications and server-to-server scenarios
- **Unified Validation**: Single backend validation layer regardless of acquisition method

### Key Design Decisions

1. **Token Acquisition Flexibility**: Support multiple acquisition patterns rather than forcing a single approach
2. **Security-First Design**: Implement PKCE and secure token handling for all patterns
3. **Performance Optimization**: Cache JWKS keys and optimize validation pipeline
4. **Developer Experience**: Provide clear documentation and examples for each pattern

## Architecture Analysis

### Current Architecture Assessment

**Strengths:**
- Robust token validation using Azure AD JWKS endpoint
- Hybrid authentication supporting both regular JWT and Azure AD tokens
- Clean middleware architecture with context-based user information
- Comprehensive claims mapping from Azure AD to internal format

**Areas for Improvement:**
- No clear guidance on token acquisition patterns
- Missing OAuth 2.0 authorization code flow implementation
- Limited documentation for different client integration scenarios
- No performance optimization for JWKS key caching

**Current Component Dependencies:**
```
┌─────────────────┐    ┌──────────────────┐    ┌─────────────────┐
│   Client App    │───▶│  Gogo API        │───▶│   Azure AD      │
│                 │    │  (Token          │    │   (JWKS         │
│                 │    │   Validation)    │    │    Endpoint)    │
└─────────────────┘    └──────────────────┘    └─────────────────┘
```

### Proposed Architecture

**Enhanced Multi-Pattern Architecture:**
```
┌─────────────────┐    ┌──────────────────┐    ┌─────────────────┐
│   Client Apps   │    │    Gogo API      │    │    Azure AD     │
│                 │    │                  │    │                 │
│ ┌─────────────┐ │    │ ┌──────────────┐ │    │ ┌─────────────┐ │
│ │ SPA/Mobile  │─┼────┼▶│ Token        │ │    │ │ JWKS        │ │
│ │ (Direct)    │ │    │ │ Validation   │─┼────┼▶│ Endpoint    │ │
│ └─────────────┘ │    │ └──────────────┘ │    │ └─────────────┘ │
│                 │    │                  │    │                 │
│ ┌─────────────┐ │    │ ┌──────────────┐ │    │ ┌─────────────┐ │
│ │ Web App     │─┼────┼▶│ OAuth 2.0    │ │    │ │ Token       │ │
│ │ (Server)    │ │    │ │ Flow Handler │─┼────┼▶│ Endpoint    │ │
│ └─────────────┘ │    │ └──────────────┘ │    │ └─────────────┘ │
└─────────────────┘    └──────────────────┘    └─────────────────┘
```

## Token Acquisition Patterns

### Pattern 1: Client-Side Token Acquisition (Recommended for SPAs/Mobile)

**When to Use:**
- Single Page Applications (React, Angular, Vue)
- Mobile applications (iOS, Android)
- Desktop applications
- Scenarios requiring direct user interaction with Azure AD

**Flow Diagram:**
```
┌─────────┐    ┌─────────────┐    ┌─────────────┐    ┌─────────────┐
│  User   │    │   Client    │    │  Azure AD   │    │  Gogo API   │
└─────────┘    └─────────────┘    └─────────────┘    └─────────────┘
     │               │                   │                   │
     │ 1. Login      │                   │                   │
     │──────────────▶│                   │                   │
     │               │ 2. Redirect to    │                   │
     │               │    Azure AD       │                   │
     │               │──────────────────▶│                   │
     │               │                   │                   │
     │ 3. Authenticate                   │                   │
     │◀─────────────────────────────────▶│                   │
     │               │                   │                   │
     │               │ 4. Return Token   │                   │
     │               │◀──────────────────│                   │
     │               │                   │                   │
     │               │ 5. API Request    │                   │
     │               │   with Token      │                   │
     │               │──────────────────────────────────────▶│
     │               │                   │                   │
     │               │                   │ 6. Validate Token │
     │               │                   │◀──────────────────│
     │               │                   │                   │
     │               │ 7. Response       │                   │
     │               │◀──────────────────────────────────────│
```

**Implementation Details:**
- Client uses MSAL.js or similar library for Azure AD integration
- Implements PKCE for enhanced security
- Handles token refresh automatically
- Sends tokens in Authorization header to Gogo API

**Security Considerations:**
- Tokens stored in memory (not localStorage)
- Automatic token refresh before expiration
- HTTPS required for all communications
- PKCE implementation mandatory

### Pattern 2: Server-Side Token Acquisition (Recommended for Traditional Web Apps)

**When to Use:**
- Traditional server-rendered web applications
- Server-to-server authentication scenarios
- Applications requiring server-side session management
- Scenarios where token security is paramount

**Flow Diagram:**
```
┌─────────┐    ┌─────────────┐    ┌─────────────┐    ┌─────────────┐
│  User   │    │  Gogo Web   │    │  Azure AD   │    │  Gogo API   │
│         │    │    App      │    │             │    │             │
└─────────┘    └─────────────┘    └─────────────┘    └─────────────┘
     │               │                   │                   │
     │ 1. Login      │                   │                   │
     │──────────────▶│                   │                   │
     │               │ 2. Redirect to    │                   │
     │               │    Azure AD       │                   │
     │               │──────────────────▶│                   │
     │               │                   │                   │
     │ 3. Authenticate                   │                   │
     │◀─────────────────────────────────▶│                   │
     │               │                   │                   │
     │               │ 4. Auth Code      │                   │
     │               │◀──────────────────│                   │
     │               │                   │                   │
     │               │ 5. Exchange Code  │                   │
     │               │   for Token       │                   │
     │               │──────────────────▶│                   │
     │               │                   │                   │
     │               │ 6. Access Token   │                   │
     │               │◀──────────────────│                   │
     │               │                   │                   │
     │ 7. Session    │                   │                   │
     │    Cookie     │                   │                   │
     │◀──────────────│                   │                   │
     │               │                   │                   │
     │ 8. API Request│                   │                   │
     │──────────────▶│ 9. Internal API   │                   │
     │               │   Call with Token │                   │
     │               │──────────────────────────────────────▶│
     │               │                   │                   │
     │               │ 10. Response      │                   │
     │               │◀──────────────────────────────────────│
     │ 11. Response  │                   │                   │
     │◀──────────────│                   │                   │
```

**Implementation Details:**
- Server handles OAuth 2.0 authorization code flow
- Tokens stored securely server-side (encrypted)
- Session management using secure cookies
- Automatic token refresh handled server-side

**Security Considerations:**
- Tokens never exposed to client-side JavaScript
- Server-side token encryption at rest
- Secure session cookie configuration
- CSRF protection for state parameter

## Components and Interfaces

### Enhanced Authentication Service

```go
// AzureADService handles Azure AD authentication flows
type AzureADService interface {
    // Client-side token validation
    ValidateClientToken(token string) (*AzureADClaims, error)
    
    // Server-side OAuth 2.0 flows
    GetAuthorizationURL(state, codeChallenge string) string
    ExchangeCodeForToken(code, codeVerifier string) (*TokenResponse, error)
    RefreshToken(refreshToken string) (*TokenResponse, error)
    
    // Token management
    StoreTokenSecurely(userID string, tokens *TokenResponse) error
    GetStoredToken(userID string) (*TokenResponse, error)
    
    // JWKS management
    GetCachedJWKS() (*JWKS, error)
    RefreshJWKSCache() error
}

type TokenResponse struct {
    AccessToken  string    `json:"access_token"`
    RefreshToken string    `json:"refresh_token"`
    ExpiresIn    int       `json:"expires_in"`
    TokenType    string    `json:"token_type"`
    ExpiresAt    time.Time `json:"expires_at"`
}

type OAuthConfig struct {
    ClientID     string
    ClientSecret string
    TenantID     string
    RedirectURI  string
    Scopes       []string
}
```

### Enhanced Middleware Components

```go
// TokenAcquisitionMiddleware handles different token acquisition patterns
func TokenAcquisitionMiddleware(pattern TokenPattern) func(http.Handler) http.Handler {
    return func(next http.Handler) http.Handler {
        return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
            switch pattern {
            case ClientSidePattern:
                handleClientSideAuth(w, r, next)
            case ServerSidePattern:
                handleServerSideAuth(w, r, next)
            case HybridPattern:
                handleHybridAuth(w, r, next)
            }
        })
    }
}

type TokenPattern int

const (
    ClientSidePattern TokenPattern = iota
    ServerSidePattern
    HybridPattern
)
```

### OAuth 2.0 Handler

```go
// OAuth2Handler manages server-side OAuth 2.0 flows
type OAuth2Handler struct {
    config       *OAuthConfig
    azureService AzureADService
    sessionStore SessionStore
}

func (h *OAuth2Handler) HandleAuthCallback(w http.ResponseWriter, r *http.Request) {
    // Extract authorization code and state
    code := r.URL.Query().Get("code")
    state := r.URL.Query().Get("state")
    
    // Validate state parameter (CSRF protection)
    if !h.validateState(state, r) {
        http.Error(w, "Invalid state parameter", http.StatusBadRequest)
        return
    }
    
    // Exchange code for tokens
    tokens, err := h.azureService.ExchangeCodeForToken(code, h.getCodeVerifier(r))
    if err != nil {
        http.Error(w, "Token exchange failed", http.StatusInternalServerError)
        return
    }
    
    // Store tokens securely
    userID := h.extractUserIDFromToken(tokens.AccessToken)
    err = h.azureService.StoreTokenSecurely(userID, tokens)
    if err != nil {
        http.Error(w, "Token storage failed", http.StatusInternalServerError)
        return
    }
    
    // Create session
    h.createSecureSession(w, r, userID)
    
    // Redirect to application
    http.Redirect(w, r, "/dashboard", http.StatusFound)
}
```

## Data Models

### Enhanced Token Storage Model

```go
type StoredToken struct {
    ID           primitive.ObjectID `bson:"_id,omitempty" json:"id"`
    UserID       string            `bson:"user_id" json:"user_id"`
    AccessToken  string            `bson:"access_token" json:"-"` // Encrypted
    RefreshToken string            `bson:"refresh_token" json:"-"` // Encrypted
    ExpiresAt    time.Time         `bson:"expires_at" json:"expires_at"`
    TokenType    string            `bson:"token_type" json:"token_type"`
    Scopes       []string          `bson:"scopes" json:"scopes"`
    CreatedAt    time.Time         `bson:"created_at" json:"created_at"`
    UpdatedAt    time.Time         `bson:"updated_at" json:"updated_at"`
}

type AuthSession struct {
    ID           primitive.ObjectID `bson:"_id,omitempty" json:"id"`
    SessionID    string            `bson:"session_id" json:"session_id"`
    UserID       string            `bson:"user_id" json:"user_id"`
    AuthType     string            `bson:"auth_type" json:"auth_type"` // "azure_ad" or "jwt"
    ExpiresAt    time.Time         `bson:"expires_at" json:"expires_at"`
    CreatedAt    time.Time         `bson:"created_at" json:"created_at"`
    LastAccessed time.Time         `bson:"last_accessed" json:"last_accessed"`
}
```

### Configuration Models

```go
type AzureADConfig struct {
    TenantID       string        `json:"tenant_id"`
    ClientID       string        `json:"client_id"`
    ClientSecret   string        `json:"client_secret,omitempty"` // Only for server-side
    RedirectURI    string        `json:"redirect_uri,omitempty"`  // Only for server-side
    JWKSEndpoint   string        `json:"jwks_endpoint"`
    TokenEndpoint  string        `json:"token_endpoint"`
    AuthEndpoint   string        `json:"auth_endpoint"`
    Issuer         string        `json:"issuer"`
    CacheTimeout   time.Duration `json:"cache_timeout"`
}

type AuthenticationConfig struct {
    EnableAzureAD     bool           `json:"enable_azure_ad"`
    EnableRegularJWT  bool           `json:"enable_regular_jwt"`
    DefaultAuthType   string         `json:"default_auth_type"`
    SessionTimeout    time.Duration  `json:"session_timeout"`
    TokenRefreshTime  time.Duration  `json:"token_refresh_time"`
    AzureAD          *AzureADConfig `json:"azure_ad,omitempty"`
}
```

## Error Handling

### Comprehensive Error Strategy

```go
// Authentication-specific errors
var (
    ErrTokenAcquisitionFailed = errors.New("token acquisition failed")
    ErrInvalidAuthCode       = errors.New("invalid authorization code")
    ErrTokenRefreshFailed    = errors.New("token refresh failed")
    ErrSessionExpired        = errors.New("session expired")
    ErrInvalidState          = errors.New("invalid state parameter")
    ErrJWKSUnavailable      = errors.New("JWKS endpoint unavailable")
    ErrTokenStorageFailed   = errors.New("token storage failed")
)

// Error response structure
type AuthError struct {
    Code        string `json:"code"`
    Message     string `json:"message"`
    Description string `json:"description,omitempty"`
    Hint        string `json:"hint,omitempty"`
}

// Error handling middleware
func AuthErrorHandler(next http.Handler) http.Handler {
    return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
        defer func() {
            if err := recover(); err != nil {
                handleAuthPanic(w, err)
            }
        }()
        
        next.ServeHTTP(w, r)
    })
}
```

### Fallback Behaviors

1. **JWKS Endpoint Failure**: Use cached keys with extended TTL
2. **Token Refresh Failure**: Redirect to re-authentication
3. **Azure AD Service Outage**: Fall back to regular JWT if configured
4. **Session Storage Failure**: Use in-memory session with warning

## Testing Strategy

### Unit Test Specifications

```go
// Test token validation
func TestAzureADTokenValidation(t *testing.T) {
    tests := []struct {
        name        string
        token       string
        expectError bool
        errorType   error
    }{
        {"Valid Token", validAzureADToken, false, nil},
        {"Expired Token", expiredToken, true, ErrTokenExpired},
        {"Invalid Signature", invalidSignatureToken, true, ErrTokenValidation},
        {"Wrong Audience", wrongAudienceToken, true, ErrTokenValidation},
    }
    
    for _, tt := range tests {
        t.Run(tt.name, func(t *testing.T) {
            claims, err := ValidateAzureADToken(tt.token)
            if tt.expectError {
                assert.Error(t, err)
                assert.Equal(t, tt.errorType, err)
                assert.Nil(t, claims)
            } else {
                assert.NoError(t, err)
                assert.NotNil(t, claims)
            }
        })
    }
}

// Test OAuth 2.0 flow
func TestOAuth2Flow(t *testing.T) {
    // Test authorization URL generation
    // Test code exchange
    // Test token refresh
    // Test error scenarios
}
```

### Integration Test Scenarios

1. **End-to-End Authentication Flow**: Complete OAuth 2.0 flow with mock Azure AD
2. **Token Validation Performance**: Measure validation times with cached/uncached JWKS
3. **Concurrent Authentication**: Test multiple simultaneous authentication requests
4. **Failover Scenarios**: Test behavior during Azure AD service outages

### Performance Test Criteria

- **Token Validation**: < 100ms with cached JWKS
- **JWKS Retrieval**: < 2 seconds initial fetch
- **OAuth Flow**: < 5 seconds end-to-end
- **Concurrent Users**: Support 1000+ simultaneous authentications

## Implementation Considerations

### Security Implementation Details

1. **PKCE Implementation**:
   ```go
   func generatePKCE() (codeVerifier, codeChallenge string) {
       verifier := generateRandomString(128)
       challenge := base64URLEncode(sha256(verifier))
       return verifier, challenge
   }
   ```

2. **Token Encryption**:
   ```go
   func encryptToken(token string, key []byte) (string, error) {
       // AES-256-GCM encryption
       block, err := aes.NewCipher(key)
       if err != nil {
           return "", err
       }
       // Implementation details...
   }
   ```

3. **Secure Session Management**:
   ```go
   func createSecureSession(w http.ResponseWriter, userID string) {
       cookie := &http.Cookie{
           Name:     "session_id",
           Value:    generateSecureSessionID(),
           HttpOnly: true,
           Secure:   true,
           SameSite: http.SameSiteStrictMode,
           MaxAge:   int(sessionTimeout.Seconds()),
       }
       http.SetCookie(w, cookie)
   }
   ```

### Performance Optimization

1. **JWKS Caching Strategy**:
   - Cache keys for 24 hours
   - Implement cache-aside pattern
   - Use Redis for distributed caching
   - Automatic cache refresh before expiration

2. **Token Validation Pipeline**:
   - Pre-parse token header for quick rejection
   - Parallel signature verification
   - Claims validation optimization
   - Connection pooling for JWKS endpoint

## Migration Strategy

### Phased Rollout Approach

**Phase 1: Enhanced Backend Support**
- Implement OAuth 2.0 authorization code flow
- Add server-side token storage
- Enhance JWKS caching
- Update documentation

**Phase 2: Client Integration Examples**
- Create React/Angular/Vue examples
- Develop mobile SDK integration guides
- Implement test applications
- Performance optimization

**Phase 3: Production Deployment**
- Gradual rollout with feature flags
- Monitor authentication metrics
- Gather developer feedback
- Optimize based on usage patterns

### Backward Compatibility

- Existing Azure AD token validation continues unchanged
- Regular JWT authentication remains fully supported
- No breaking changes to existing API endpoints
- Gradual migration path for existing integrations

### Data Migration

- No existing data migration required
- New token storage collections created as needed
- Existing user accounts work with both authentication types
- Session data structure enhanced but backward compatible