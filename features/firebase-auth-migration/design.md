# Firebase Authentication Migration - Design Architecture

## 1. Architecture Overview

### 1.1 High-Level Architecture

```
┌─────────────────┐    ┌─────────────────┐    ┌─────────────────┐
│   Client Apps   │    │   Gogo API      │    │   Firebase      │
│                 │    │                 │    │   Auth Service  │
│ - Web Frontend  │◄──►│ - Auth Middleware│◄──►│                 │
│ - Mobile Apps   │    │ - Auth Service  │    │ - User Mgmt     │
│ - Third Party   │    │ - User Service  │    │ - ID Tokens     │
└─────────────────┘    └─────────────────┘    │ - Custom Claims │
                                ▲              └─────────────────┘
                                │
                                ▼
                       ┌─────────────────┐
                       │   MongoDB       │
                       │                 │
                       │ - User Profiles │
                       │ - Projects      │
                       │ - Forms         │
                       │ - Responses     │
                       └─────────────────┘
```

### 1.2 Authentication Flow

```
1. Client Authentication:
   Client → Firebase Auth → ID Token → Client

2. API Request:
   Client → Gogo API (with ID Token) → Firebase Token Validation → MongoDB → Response

3. User Management:
   Gogo API → Firebase Admin SDK → User Operations → MongoDB Sync
```

## 2. Component Design

### 2.1 Firebase Integration Layer

#### 2.1.1 Firebase Service (`pkg/auth/firebase.go`)

```go
type FirebaseService struct {
    client *auth.Client
    config *FirebaseConfig
}

type FirebaseConfig struct {
    ProjectID           string
    ServiceAccountPath  string
    DatabaseURL         string
    CustomClaimsEnabled bool
}

type FirebaseUser struct {
    UID           string            `json:"uid"`
    Email         string            `json:"email"`
    DisplayName   string            `json:"display_name"`
    EmailVerified bool              `json:"email_verified"`
    CustomClaims  map[string]interface{} `json:"custom_claims"`
    CreatedAt     time.Time         `json:"created_at"`
    LastSignIn    time.Time         `json:"last_sign_in"`
}

// Core Methods
func (fs *FirebaseService) ValidateIDToken(ctx context.Context, idToken string) (*FirebaseUser, error)
func (fs *FirebaseService) CreateUser(ctx context.Context, user *CreateUserRequest) (*FirebaseUser, error)
func (fs *FirebaseService) UpdateUser(ctx context.Context, uid string, updates *UpdateUserRequest) error
func (fs *FirebaseService) DeleteUser(ctx context.Context, uid string) error
func (fs *FirebaseService) SetCustomClaims(ctx context.Context, uid string, claims map[string]interface{}) error
func (fs *FirebaseService) GetUser(ctx context.Context, uid string) (*FirebaseUser, error)
```

#### 2.1.2 Custom Claims Management

```go
type CustomClaims struct {
    Roles       []string          `json:"roles"`
    Permissions []string          `json:"permissions"`
    Projects    []string          `json:"projects"`
    Metadata    map[string]string `json:"metadata"`
}

type ClaimsManager struct {
    firebaseService *FirebaseService
    roleService     *RoleService
}

func (cm *ClaimsManager) SetUserRoles(ctx context.Context, uid string, roles []string) error
func (cm *ClaimsManager) AddUserPermission(ctx context.Context, uid string, permission string) error
func (cm *ClaimsManager) RemoveUserPermission(ctx context.Context, uid string, permission string) error
func (cm *ClaimsManager) GetUserClaims(ctx context.Context, uid string) (*CustomClaims, error)
```

### 2.2 Authentication Middleware

#### 2.2.1 Firebase Authentication Middleware (`internal/middleware/firebase_auth.go`)

```go
type FirebaseAuthMiddleware struct {
    firebaseService *auth.FirebaseService
    logger          *logger.Logger
}

// Primary authentication middleware
func (fam *FirebaseAuthMiddleware) Authenticate(next http.Handler) http.Handler {
    return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
        // Extract ID token from Authorization header
        idToken, err := extractIDTokenFromRequest(r)
        if err != nil {
            http.Error(w, "Unauthorized: Missing or invalid token", http.StatusUnauthorized)
            return
        }

        // Validate ID token with Firebase
        user, err := fam.firebaseService.ValidateIDToken(r.Context(), idToken)
        if err != nil {
            http.Error(w, "Unauthorized: Token validation failed", http.StatusUnauthorized)
            return
        }

        // Add user info to context
        ctx := context.WithValue(r.Context(), "firebase_user", user)
        ctx = context.WithValue(ctx, "user_id", user.UID)
        ctx = context.WithValue(ctx, "user_email", user.Email)
        ctx = context.WithValue(ctx, "custom_claims", user.CustomClaims)

        next.ServeHTTP(w, r.WithContext(ctx))
    })
}

// Role-based authorization middleware
func (fam *FirebaseAuthMiddleware) RequireRoles(roles ...string) func(http.Handler) http.Handler {
    return func(next http.Handler) http.Handler {
        return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
            user, ok := r.Context().Value("firebase_user").(*auth.FirebaseUser)
            if !ok {
                http.Error(w, "Unauthorized: User not authenticated", http.StatusUnauthorized)
                return
            }

            userRoles := extractRolesFromClaims(user.CustomClaims)
            if !hasAnyRole(userRoles, roles) {
                http.Error(w, "Forbidden: Insufficient permissions", http.StatusForbidden)
                return
            }

            next.ServeHTTP(w, r)
        })
    }
}
```

#### 2.2.2 Hybrid Authentication Middleware (Migration Period)

```go
func (fam *FirebaseAuthMiddleware) HybridAuth(next http.Handler) http.Handler {
    return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
        token, err := extractTokenFromRequest(r)
        if err != nil {
            http.Error(w, "Unauthorized: Missing token", http.StatusUnauthorized)
            return
        }

        // Try Firebase ID token validation first
        if user, err := fam.firebaseService.ValidateIDToken(r.Context(), token); err == nil {
            ctx := context.WithValue(r.Context(), "firebase_user", user)
            ctx = context.WithValue(ctx, "auth_method", "firebase")
            next.ServeHTTP(w, r.WithContext(ctx))
            return
        }

        // Fallback to legacy JWT validation
        if claims, err := auth.ValidateToken(token); err == nil {
            ctx := context.WithValue(r.Context(), "user_id", claims.UserID)
            ctx = context.WithValue(ctx, "username", claims.Username)
            ctx = context.WithValue(ctx, "user_roles", claims.Roles)
            ctx = context.WithValue(ctx, "auth_method", "legacy_jwt")
            next.ServeHTTP(w, r.WithContext(ctx))
            return
        }

        http.Error(w, "Unauthorized: Invalid token", http.StatusUnauthorized)
    })
}
```

### 2.3 User Management Service

#### 2.3.1 Enhanced Auth Service (`internal/service/authService.go`)

```go
type AuthService struct {
    firebaseService   *auth.FirebaseService
    claimsManager     *auth.ClaimsManager
    userRepository    repository.UserRepository
    accountRepository repository.AccountRepository
    roleService       *RoleService
    logger            *logger.Logger
}

// Firebase-based authentication methods
func (as *AuthService) CreateFirebaseUser(ctx context.Context, req *CreateUserRequest) (*UserResponse, error) {
    // Create user in Firebase
    firebaseUser, err := as.firebaseService.CreateUser(ctx, &auth.CreateUserRequest{
        Email:       req.Email,
        Password:    req.Password,
        DisplayName: req.DisplayName,
    })
    if err != nil {
        return nil, fmt.Errorf("failed to create Firebase user: %w", err)
    }

    // Set custom claims for roles
    if len(req.Roles) > 0 {
        err = as.claimsManager.SetUserRoles(ctx, firebaseUser.UID, req.Roles)
        if err != nil {
            // Cleanup Firebase user if claims setting fails
            as.firebaseService.DeleteUser(ctx, firebaseUser.UID)
            return nil, fmt.Errorf("failed to set user roles: %w", err)
        }
    }

    // Create user profile in MongoDB
    userProfile := &model.User{
        FirebaseUID: firebaseUser.UID,
        Email:       firebaseUser.Email,
        Fullname:    firebaseUser.DisplayName,
        Status:      "active",
    }
    userProfile.SetTimestamps()

    err = as.userRepository.Create(ctx, userProfile)
    if err != nil {
        // Cleanup Firebase user if MongoDB creation fails
        as.firebaseService.DeleteUser(ctx, firebaseUser.UID)
        return nil, fmt.Errorf("failed to create user profile: %w", err)
    }

    return &UserResponse{
        ID:          userProfile.ID,
        FirebaseUID: firebaseUser.UID,
        Email:       firebaseUser.Email,
        DisplayName: firebaseUser.DisplayName,
        Roles:       req.Roles,
    }, nil
}

func (as *AuthService) ValidateFirebaseToken(ctx context.Context, idToken string) (*UserResponse, error) {
    // Validate token with Firebase
    firebaseUser, err := as.firebaseService.ValidateIDToken(ctx, idToken)
    if err != nil {
        return nil, fmt.Errorf("token validation failed: %w", err)
    }

    // Get user profile from MongoDB
    userProfile, err := as.userRepository.GetByFirebaseUID(ctx, firebaseUser.UID)
    if err != nil {
        return nil, fmt.Errorf("user profile not found: %w", err)
    }

    // Extract roles from custom claims
    roles := extractRolesFromClaims(firebaseUser.CustomClaims)

    return &UserResponse{
        ID:          userProfile.ID,
        FirebaseUID: firebaseUser.UID,
        Email:       firebaseUser.Email,
        DisplayName: firebaseUser.DisplayName,
        Roles:       roles,
    }, nil
}
```

### 2.4 Database Integration

#### 2.4.1 Updated User Model (`internal/model/userModel.go`)

```go
type User struct {
    ID          primitive.ObjectID `bson:"_id,omitempty" json:"id"`
    FirebaseUID string             `bson:"firebase_uid" json:"firebase_uid"`
    AccountId   primitive.ObjectID `bson:"account_id,omitempty" json:"account_id,omitempty"`
    Fullname    string             `bson:"fullname" json:"fullname"`
    Email       string             `bson:"email" json:"email"`
    Status      string             `bson:"status" json:"status"`
    CreatedAt   time.Time          `bson:"created_at" json:"created_at"`
    UpdatedAt   time.Time          `bson:"updated_at" json:"updated_at"`
}

// Add index for Firebase UID
// db.user.createIndex({"firebase_uid": 1}, {"unique": true})
```

#### 2.4.2 User Repository Updates (`internal/repository/mongo/userRepository.go`)

```go
type UserRepository struct {
    collection *mongo.Collection
    logger     *logger.Logger
}

func (ur *UserRepository) GetByFirebaseUID(ctx context.Context, firebaseUID string) (*model.User, error) {
    var user model.User
    err := ur.collection.FindOne(ctx, bson.M{"firebase_uid": firebaseUID}).Decode(&user)
    if err != nil {
        if errors.Is(err, mongo.ErrNoDocuments) {
            return nil, customError.ErrUserNotFound
        }
        return nil, err
    }
    return &user, nil
}

func (ur *UserRepository) UpdateFirebaseUID(ctx context.Context, userID primitive.ObjectID, firebaseUID string) error {
    update := bson.M{
        "$set": bson.M{
            "firebase_uid": firebaseUID,
            "updated_at":   time.Now(),
        },
    }
    
    result, err := ur.collection.UpdateOne(ctx, bson.M{"_id": userID}, update)
    if err != nil {
        return err
    }
    
    if result.MatchedCount == 0 {
        return customError.ErrUserNotFound
    }
    
    return nil
}
```

## 3. Migration Strategy

### 3.1 Phase 1: Infrastructure Setup

1. **Firebase Project Configuration**
   - Create Firebase project
   - Configure authentication providers
   - Set up service account and credentials
   - Configure security rules

2. **Code Implementation**
   - Implement Firebase service layer
   - Create Firebase authentication middleware
   - Update user models and repositories
   - Implement custom claims management

3. **Testing Environment Setup**
   - Deploy to staging environment
   - Configure test Firebase project
   - Set up integration tests

### 3.2 Phase 2: Parallel System Implementation

1. **Hybrid Authentication**
   - Deploy hybrid middleware supporting both systems
   - Implement feature flags for authentication method
   - Set up monitoring and logging

2. **User Migration Tools**
   - Create user export scripts
   - Implement Firebase user import
   - Develop data validation tools
   - Create rollback procedures

### 3.3 Phase 3: Gradual Migration

1. **Pilot User Migration**
   - Migrate test users
   - Validate functionality
   - Monitor performance
   - Gather feedback

2. **Batch User Migration**
   - Migrate users in batches
   - Monitor system health
   - Handle migration failures
   - Update user profiles

### 3.4 Phase 4: System Cutover

1. **Complete Migration**
   - Migrate remaining users
   - Switch default authentication to Firebase
   - Disable legacy authentication
   - Clean up old code

2. **Post-Migration Validation**
   - Verify all functionality
   - Performance testing
   - Security audit
   - Documentation updates

## 4. Security Considerations

### 4.1 Token Security

- **ID Token Validation**: Use Firebase Admin SDK for secure token validation
- **Custom Claims**: Implement secure custom claims management
- **Token Expiration**: Enforce proper token expiration and refresh
- **HTTPS Only**: Ensure all authentication traffic uses HTTPS

### 4.2 Data Protection

- **User Data Encryption**: Encrypt sensitive user data in MongoDB
- **Secure Configuration**: Use environment variables for sensitive configuration
- **Audit Logging**: Implement comprehensive audit logging
- **Data Retention**: Implement proper data retention policies

### 4.3 Access Control

- **Role-Based Access**: Implement fine-grained role-based access control
- **Permission Management**: Use custom claims for granular permissions
- **Resource Protection**: Protect API endpoints with appropriate middleware
- **Rate Limiting**: Implement rate limiting for authentication endpoints

## 5. Performance Optimization

### 5.1 Caching Strategy

- **Token Caching**: Cache validated tokens to reduce Firebase API calls
- **User Profile Caching**: Cache user profiles for frequently accessed data
- **Custom Claims Caching**: Cache custom claims to improve authorization performance

### 5.2 Connection Management

- **Firebase Admin SDK**: Optimize connection pooling and reuse
- **MongoDB Connections**: Maintain efficient database connection pooling
- **HTTP Client Optimization**: Configure optimal HTTP client settings

### 5.3 Monitoring and Metrics

- **Authentication Metrics**: Track authentication success/failure rates
- **Performance Metrics**: Monitor response times and throughput
- **Error Monitoring**: Implement comprehensive error tracking
- **Health Checks**: Set up health check endpoints for monitoring

## 6. Error Handling and Recovery

### 6.1 Error Categories

```go
var (
    ErrFirebaseTokenInvalid    = errors.New("firebase token is invalid")
    ErrFirebaseTokenExpired    = errors.New("firebase token has expired")
    ErrFirebaseUserNotFound    = errors.New("firebase user not found")
    ErrCustomClaimsInvalid     = errors.New("custom claims are invalid")
    ErrUserProfileNotFound     = errors.New("user profile not found")
    ErrUserCreationFailed      = errors.New("user creation failed")
    ErrPermissionDenied        = errors.New("permission denied")
    ErrFirebaseServiceUnavailable = errors.New("firebase service unavailable")
)
```

### 6.2 Recovery Mechanisms

- **Retry Logic**: Implement exponential backoff for transient failures
- **Circuit Breaker**: Implement circuit breaker pattern for Firebase API calls
- **Fallback Mechanisms**: Provide fallback authentication during outages
- **Data Consistency**: Ensure data consistency between Firebase and MongoDB

## 7. Testing Strategy

### 7.1 Unit Testing

- Test Firebase service methods
- Test authentication middleware
- Test custom claims management
- Test user repository operations
- Test error handling scenarios

### 7.2 Integration Testing

- Test end-to-end authentication flows
- Test API authentication and authorization
- Test user management operations
- Test migration procedures
- Test performance under load

### 7.3 Security Testing

- Test token validation security
- Test authorization bypass attempts
- Test injection attacks
- Test session management
- Perform security audit

## 8. Deployment and Configuration

### 8.1 Environment Configuration

```yaml
# Firebase Configuration
FIREBASE_PROJECT_ID: "gogo-production"
FIREBASE_SERVICE_ACCOUNT_PATH: "/path/to/service-account.json"
FIREBASE_DATABASE_URL: "https://gogo-production.firebaseio.com"

# Authentication Configuration
AUTH_METHOD: "firebase" # firebase, hybrid, legacy
TOKEN_CACHE_TTL: "300s"
MAX_TOKEN_AGE: "3600s"

# Migration Configuration
MIGRATION_ENABLED: "true"
MIGRATION_BATCH_SIZE: "100"
MIGRATION_DELAY: "1s"
```

### 8.2 Deployment Steps

1. Deploy infrastructure changes
2. Update environment configuration
3. Deploy application with hybrid authentication
4. Run migration scripts
5. Switch to Firebase-only authentication
6. Clean up legacy components

---

**Document Version**: 1.0  
**Created**: 2024-12-19  
**Status**: Draft  
**Next Review**: Implementation Phase