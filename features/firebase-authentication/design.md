# Firebase Authentication Implementation Design

## Overview
This document outlines the design for implementing Firebase Authentication in the Gogo application router, providing a comprehensive authentication solution that integrates seamlessly with existing traditional and Azure AD authentication methods.

## Architecture Overview

### High-Level Architecture
```
┌─────────────────┐    ┌──────────────────┐    ┌─────────────────┐
│   Client App    │────│   AuthRouter     │────│ FirebaseService │
│                 │    │                  │    │                 │
│ - Web App       │    │ - Route Handler  │    │ - Token Verify  │
│ - Mobile App    │    │ - Request Valid  │    │ - User Mgmt     │
│ - API Client    │    │ - Response Form  │    │ - Claims Mgmt   │
└─────────────────┘    └──────────────────┘    └─────────────────┘
                                │
                                ▼
                       ┌──────────────────┐
                       │ Firebase Admin   │
                       │      SDK         │
                       └──────────────────┘
                                │
                                ▼
                       ┌──────────────────┐
                       │ Firebase Project │
                       │  Authentication  │
                       └──────────────────┘
```

### Component Integration
```
┌─────────────────────────────────────────────────────────────┐
│                      AuthRouter                             │
├─────────────────────────────────────────────────────────────┤
│ Traditional Auth │ Azure AD Auth │ Firebase Auth            │
│ - /login         │ - /azure/*    │ - /firebase/*           │
│ - /register      │               │                         │
└─────────────────────────────────────────────────────────────┘
                              │
                              ▼
┌─────────────────────────────────────────────────────────────┐
│                   Service Layer                             │
├─────────────────────────────────────────────────────────────┤
│ AuthService      │ AzureADService │ FirebaseService         │
│ - Traditional    │ - OAuth2 Flow  │ - Token Validation      │
│   Authentication │ - Token Mgmt   │ - User Management       │
└─────────────────────────────────────────────────────────────┘
```

## Detailed Design

### AuthRouter Enhancement

#### Struct Definition
```go
type AuthRouter struct {
    // Existing fields
    authService    service.AuthService
    userService    service.UserService
    
    // Azure AD services (optional)
    azureService   *mainAuth.AzureADService
    sessionManager *mainAuth.SessionManager
    tokenCache     *mainAuth.TokenCache
    config         *mainAuth.OAuth2Config
    azureEnabled   bool
    
    // Firebase services (optional)
    firebaseService *mainAuth.FirebaseService
    firebaseEnabled bool
}
```

#### Constructor Patterns

**1. Traditional Only**
```go
func NewAuthRouter(
    authService service.AuthService,
    userService service.UserService,
) *AuthRouter
```

**2. Traditional + Azure AD**
```go
func NewAuthRouterWithAzure(
    authService service.AuthService,
    userService service.UserService,
    azureService *mainAuth.AzureADService,
    sessionManager *mainAuth.SessionManager,
    tokenCache *mainAuth.TokenCache,
    config *mainAuth.OAuth2Config,
) *AuthRouter
```

**3. Traditional + Firebase**
```go
func NewAuthRouterWithFirebase(
    authService service.AuthService,
    userService service.UserService,
    firebaseService *mainAuth.FirebaseService,
) *AuthRouter
```

**4. All Authentication Methods**
```go
func NewAuthRouterWithAll(
    authService service.AuthService,
    userService service.UserService,
    azureService *mainAuth.AzureADService,
    sessionManager *mainAuth.SessionManager,
    tokenCache *mainAuth.TokenCache,
    config *mainAuth.OAuth2Config,
    firebaseService *mainAuth.FirebaseService,
) *AuthRouter
```

### Route Registration

#### Enhanced SetupRoutes Method
```go
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
```

### Firebase Handler Methods

#### 1. Token Verification Handler
```go
func (ar *AuthRouter) handleFirebaseTokenVerification(w http.ResponseWriter, r *http.Request) {
    // Input validation
    // Token verification using FirebaseService
    // Response formatting
    // Error handling
}
```

**Flow:**
1. Parse JSON request body
2. Validate ID token presence
3. Call `firebaseService.VerifyIDToken()`
4. Extract claims and user information
5. Return structured response

#### 2. User Registration Handler
```go
func (ar *AuthRouter) handleFirebaseUserRegistration(w http.ResponseWriter, r *http.Request) {
    // Input validation
    // User creation using FirebaseService
    // Custom claims setting (optional)
    // Response formatting
}
```

**Flow:**
1. Parse registration request
2. Validate email and password
3. Call `firebaseService.CreateUser()`
4. Set custom claims if provided
5. Return user information

#### 3. Profile Retrieval Handler
```go
func (ar *AuthRouter) handleFirebaseProfile(w http.ResponseWriter, r *http.Request) {
    // UID extraction from query/header
    // User retrieval using FirebaseService
    // Profile formatting
}
```

**Flow:**
1. Extract UID from query parameter or header
2. Call `firebaseService.GetUser()`
3. Format user profile response
4. Include custom claims

#### 4. Custom Claims Management Handler
```go
func (ar *AuthRouter) handleFirebaseCustomClaims(w http.ResponseWriter, r *http.Request) {
    // Claims validation
    // Claims setting using FirebaseService
    // Success confirmation
}
```

#### 5. User Management Handlers
```go
func (ar *AuthRouter) handleFirebaseUserDeletion(w http.ResponseWriter, r *http.Request)
func (ar *AuthRouter) handleFirebaseUserUpdate(w http.ResponseWriter, r *http.Request)
```

### Request/Response Patterns

#### Standard Request Structure
```go
type FirebaseTokenRequest struct {
    IDToken string `json:"id_token"`
}

type FirebaseRegistrationRequest struct {
    Email    string                 `json:"email"`
    Password string                 `json:"password"`
    Claims   map[string]interface{} `json:"custom_claims,omitempty"`
}

type FirebaseClaimsRequest struct {
    UID    string                 `json:"uid"`
    Claims map[string]interface{} `json:"custom_claims"`
}

type FirebaseUpdateRequest struct {
    Email    *string `json:"email,omitempty"`
    Password *string `json:"password,omitempty"`
    Disabled *bool   `json:"disabled,omitempty"`
}
```

#### Standard Response Structure
```go
type FirebaseTokenResponse struct {
    Valid   bool                   `json:"valid"`
    UID     string                 `json:"uid"`
    Email   string                 `json:"email"`
    Claims  map[string]interface{} `json:"claims"`
    Expires int64                  `json:"expires"`
}

type FirebaseUserResponse struct {
    UID           string                 `json:"uid"`
    Email         string                 `json:"email"`
    EmailVerified bool                   `json:"email_verified"`
    Disabled      bool                   `json:"disabled"`
    CustomClaims  map[string]interface{} `json:"custom_claims"`
}

type FirebaseSuccessResponse struct {
    Success bool   `json:"success"`
    Message string `json:"message"`
}
```

### Error Handling Strategy

#### Error Categories
1. **Client Errors (4xx)**
   - 400 Bad Request: Invalid request format, missing fields
   - 401 Unauthorized: Invalid tokens, authentication failures
   - 404 Not Found: User not found
   - 409 Conflict: Duplicate email during registration

2. **Server Errors (5xx)**
   - 500 Internal Server Error: Firebase service errors
   - 501 Not Implemented: Firebase disabled
   - 503 Service Unavailable: Firebase service unavailable

#### Error Response Format
```go
type ErrorResponse struct {
    Error   string `json:"error"`
    Message string `json:"message"`
    Code    int    `json:"code"`
}
```

#### Error Handling Pattern
```go
func handleFirebaseError(w http.ResponseWriter, err error, operation string) {
    log.Printf("Firebase %s error: %v", operation, err)
    
    switch {
    case strings.Contains(err.Error(), "not found"):
        http.Error(w, "User not found", http.StatusNotFound)
    case strings.Contains(err.Error(), "invalid"):
        http.Error(w, "Invalid request: "+err.Error(), http.StatusBadRequest)
    case strings.Contains(err.Error(), "expired"):
        http.Error(w, "Token expired", http.StatusUnauthorized)
    default:
        http.Error(w, "Internal server error", http.StatusInternalServerError)
    }
}
```

### Security Considerations

#### Input Validation
```go
func validateFirebaseRequest(r *http.Request) error {
    // Content-Type validation
    if r.Header.Get("Content-Type") != "application/json" {
        return errors.New("invalid content type")
    }
    
    // Request size validation
    if r.ContentLength > maxRequestSize {
        return errors.New("request too large")
    }
    
    return nil
}
```

#### Token Security
- Validate token signatures using Firebase SDK
- Check token expiration
- Verify issuer and audience claims
- Implement rate limiting for token verification

#### User Data Protection
- Sanitize user inputs
- Validate email formats
- Ensure password complexity (handled by Firebase)
- Protect sensitive user information in logs

### Performance Optimization

#### Caching Strategy
```go
type TokenCache struct {
    cache map[string]*CachedToken
    mutex sync.RWMutex
    ttl   time.Duration
}

type CachedToken struct {
    Claims    *mainAuth.FirebaseClaims
    ExpiresAt time.Time
}
```

#### Connection Pooling
- Reuse Firebase Admin SDK connections
- Implement connection pooling for database operations
- Use context timeouts for Firebase operations

#### Async Operations
```go
func (ar *AuthRouter) handleFirebaseUserRegistrationAsync(w http.ResponseWriter, r *http.Request) {
    // Immediate response for user creation
    // Async custom claims setting
    // Background user profile initialization
}
```

### Monitoring and Observability

#### Metrics Collection
```go
type FirebaseMetrics struct {
    TokenVerifications   prometheus.Counter
    UserRegistrations    prometheus.Counter
    UserDeletions        prometheus.Counter
    ErrorCount           prometheus.CounterVec
    ResponseTime         prometheus.HistogramVec
}
```

#### Logging Strategy
```go
func logFirebaseOperation(operation, uid string, duration time.Duration, err error) {
    fields := logrus.Fields{
        "operation": operation,
        "uid":       uid,
        "duration":  duration,
        "service":   "firebase_auth",
    }
    
    if err != nil {
        fields["error"] = err.Error()
        logrus.WithFields(fields).Error("Firebase operation failed")
    } else {
        logrus.WithFields(fields).Info("Firebase operation completed")
    }
}
```

### Testing Strategy

#### Unit Tests
```go
func TestHandleFirebaseTokenVerification(t *testing.T) {
    tests := []struct {
        name           string
        requestBody    string
        mockResponse   *mainAuth.FirebaseClaims
        mockError      error
        expectedStatus int
    }{
        {
            name:           "Valid token",
            requestBody:    `{"id_token":"valid_token"}`,
            mockResponse:   &mainAuth.FirebaseClaims{UserID: "test_uid"},
            expectedStatus: http.StatusOK,
        },
        {
            name:           "Invalid token",
            requestBody:    `{"id_token":"invalid_token"}`,
            mockError:      errors.New("invalid token"),
            expectedStatus: http.StatusUnauthorized,
        },
    }
    
    for _, tt := range tests {
        t.Run(tt.name, func(t *testing.T) {
            // Test implementation
        })
    }
}
```

#### Integration Tests
```go
func TestFirebaseAuthenticationFlow(t *testing.T) {
    // Setup test Firebase project
    // Test complete authentication flow
    // Verify user creation and token validation
    // Test custom claims management
}
```

#### Mock Implementation
```go
type MockFirebaseService struct {
    users  map[string]*firebaseAuth.UserRecord
    claims map[string]map[string]interface{}
}

func (m *MockFirebaseService) VerifyIDToken(ctx context.Context, token string) (*mainAuth.FirebaseClaims, error) {
    // Mock implementation
}
```

### Configuration Management

#### Environment Variables
```bash
# Firebase Configuration
FIREBASE_ENABLED=true
FIREBASE_PROJECT_ID=your-project-id
FIREBASE_SERVICE_ACCOUNT_PATH=/path/to/service-account.json

# Authentication Configuration
AUTH_METHODS=traditional,azure,firebase
DEFAULT_AUTH_METHOD=firebase
```

#### Configuration Struct
```go
type AuthConfig struct {
    Traditional struct {
        Enabled bool `env:"TRADITIONAL_AUTH_ENABLED" default:"true"`
    }
    Azure struct {
        Enabled bool `env:"AZURE_AUTH_ENABLED" default:"false"`
    }
    Firebase struct {
        Enabled bool `env:"FIREBASE_AUTH_ENABLED" default:"false"`
    }
}
```

### Deployment Considerations

#### Service Dependencies
1. Firebase Admin SDK initialization
2. Service account credentials availability
3. Network connectivity to Firebase services
4. Proper IAM permissions for Firebase operations

#### Health Checks
```go
func (ar *AuthRouter) HealthCheck() error {
    if ar.firebaseEnabled {
        // Test Firebase connectivity
        ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
        defer cancel()
        
        _, err := ar.firebaseService.GetUser(ctx, "health_check_uid")
        if err != nil && !strings.Contains(err.Error(), "not found") {
            return fmt.Errorf("firebase service unhealthy: %v", err)
        }
    }
    return nil
}
```

#### Graceful Degradation
```go
func (ar *AuthRouter) handleFirebaseUnavailable(w http.ResponseWriter, r *http.Request) {
    if !ar.firebaseEnabled {
        http.Error(w, "Firebase authentication not enabled", http.StatusNotImplemented)
        return
    }
    
    // Fallback to traditional authentication if configured
    if ar.authService != nil {
        // Redirect to traditional auth
    }
}
```

## Implementation Timeline

### Phase 1: Core Implementation (Week 1)
- AuthRouter struct enhancement
- Constructor methods implementation
- Basic route registration

### Phase 2: Handler Implementation (Week 2)
- Token verification handler
- User registration handler
- Profile retrieval handler

### Phase 3: Advanced Features (Week 3)
- Custom claims management
- User lifecycle operations
- Error handling refinement

### Phase 4: Testing and Documentation (Week 4)
- Unit test implementation
- Integration test development
- Documentation completion
- Performance optimization

## Success Metrics

1. **Functionality**: All Firebase endpoints operational
2. **Performance**: < 200ms average response time
3. **Reliability**: 99.9% uptime for Firebase operations
4. **Security**: Zero security vulnerabilities
5. **Test Coverage**: > 90% code coverage
6. **Documentation**: Complete API and integration documentation

## Future Enhancements

1. **Session Cookie Integration**: Web application session management
2. **Multi-Factor Authentication**: Enhanced security features
3. **Real-time User Management**: Live user status monitoring
4. **Advanced Analytics**: User behavior tracking
5. **Custom Token Generation**: Enhanced token management
6. **Webhook Integration**: Real-time Firebase event handling