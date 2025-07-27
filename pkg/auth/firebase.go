package auth

import (
	"context"
	"fmt"
	"log"
	"os"
	"strings"
	"time"

	customError "main/internal/error"

	firebase "firebase.google.com/go/v4"
	"firebase.google.com/go/v4/auth"
	"google.golang.org/api/option"
)

// FirebaseService handles Firebase authentication operations
type FirebaseService struct {
	client *auth.Client
	app    *firebase.App
}

// FirebaseClaims represents the structure of Firebase ID token claims
type FirebaseClaims struct {
	UserID   string                 `json:"user_id"`
	Email    string                 `json:"email"`
	Name     string                 `json:"name"`
	Roles    []string               `json:"roles,omitempty"`
	Custom   map[string]interface{} `json:"custom,omitempty"`
	Issuer   string                 `json:"iss"`
	Audience string                 `json:"aud"`
	Expiry   int64                  `json:"exp"`
	IssuedAt int64                  `json:"iat"`
}

// FirebaseConfig holds Firebase configuration
type FirebaseConfig struct {
	ProjectID              string
	ServiceAccountKeyPath  string
	ServiceAccountKey      string // JSON string of service account key
	DatabaseURL           string
}

// GetFirebaseConfig returns Firebase configuration from environment variables
func GetFirebaseConfig() *FirebaseConfig {
	projectID := os.Getenv("FIREBASE_PROJECT_ID")
	serviceAccountKeyPath := os.Getenv("FIREBASE_SERVICE_ACCOUNT_KEY_PATH")
	serviceAccountKey := os.Getenv("FIREBASE_SERVICE_ACCOUNT_KEY")
	databaseURL := os.Getenv("FIREBASE_DATABASE_URL")

	if projectID == "" {
		return nil
	}

	return &FirebaseConfig{
		ProjectID:             projectID,
		ServiceAccountKeyPath: serviceAccountKeyPath,
		ServiceAccountKey:     serviceAccountKey,
		DatabaseURL:          databaseURL,
	}
}

// NewFirebaseService creates a new Firebase service instance
func NewFirebaseService() (*FirebaseService, error) {
	config := GetFirebaseConfig()
	if config == nil {
		return nil, fmt.Errorf("Firebase configuration not found")
	}

	ctx := context.Background()
	var app *firebase.App
	var err error

	// Initialize Firebase app with service account
	if config.ServiceAccountKeyPath != "" {
		// Use service account key file
		opt := option.WithCredentialsFile(config.ServiceAccountKeyPath)
		app, err = firebase.NewApp(ctx, &firebase.Config{
			ProjectID:   config.ProjectID,
			DatabaseURL: config.DatabaseURL,
		}, opt)
	} else if config.ServiceAccountKey != "" {
		// Use service account key JSON string
		opt := option.WithCredentialsJSON([]byte(config.ServiceAccountKey))
		app, err = firebase.NewApp(ctx, &firebase.Config{
			ProjectID:   config.ProjectID,
			DatabaseURL: config.DatabaseURL,
		}, opt)
	} else {
		// Use default credentials (for development/testing)
		app, err = firebase.NewApp(ctx, &firebase.Config{
			ProjectID:   config.ProjectID,
			DatabaseURL: config.DatabaseURL,
		})
	}

	if err != nil {
		return nil, fmt.Errorf("failed to initialize Firebase app: %v", err)
	}

	// Get Auth client
	client, err := app.Auth(ctx)
	if err != nil {
		return nil, fmt.Errorf("failed to get Firebase Auth client: %v", err)
	}

	return &FirebaseService{
		client: client,
		app:    app,
	}, nil
}

// ValidateIDToken validates a Firebase ID token and returns the claims
func (fs *FirebaseService) ValidateIDToken(ctx context.Context, idToken string) (*FirebaseClaims, error) {
	if idToken == "" {
		return nil, customError.ErrTokenInvalid
	}

	// Verify the ID token
	token, err := fs.client.VerifyIDToken(ctx, idToken)
	if err != nil {
		if strings.Contains(err.Error(), "expired") {
			return nil, customError.ErrTokenExpired
		}
		return nil, customError.ErrTokenValidation
	}

	// Extract standard claims
	claims := &FirebaseClaims{
		UserID:   token.UID,
		Issuer:   token.Issuer,
		Audience: token.Audience,
		Expiry:   token.Expires,
		IssuedAt: token.IssuedAt,
	}

	// Extract email and name from claims
	if email, ok := token.Claims["email"].(string); ok {
		claims.Email = email
	}
	if name, ok := token.Claims["name"].(string); ok {
		claims.Name = name
	}

	// Extract custom claims (roles, permissions, etc.)
	if roles, ok := token.Claims["roles"].([]interface{}); ok {
		claims.Roles = make([]string, len(roles))
		for i, role := range roles {
			if roleStr, ok := role.(string); ok {
				claims.Roles[i] = roleStr
			}
		}
	}

	// Store all custom claims
	claims.Custom = make(map[string]interface{})
	for key, value := range token.Claims {
		if key != "email" && key != "name" && key != "roles" {
			claims.Custom[key] = value
		}
	}

	return claims, nil
}

// CreateUser creates a new user in Firebase Authentication
func (fs *FirebaseService) CreateUser(ctx context.Context, email, password, displayName string) (*auth.UserRecord, error) {
	params := (&auth.UserToCreate{}).
		Email(email).
		Password(password).
		DisplayName(displayName).
		EmailVerified(false)

	user, err := fs.client.CreateUser(ctx, params)
	if err != nil {
		return nil, fmt.Errorf("failed to create user: %v", err)
	}

	return user, nil
}

// GetUser retrieves a user by UID from Firebase Authentication
func (fs *FirebaseService) GetUser(ctx context.Context, uid string) (*auth.UserRecord, error) {
	user, err := fs.client.GetUser(ctx, uid)
	if err != nil {
		return nil, fmt.Errorf("failed to get user: %v", err)
	}

	return user, nil
}

// GetUserByEmail retrieves a user by email from Firebase Authentication
func (fs *FirebaseService) GetUserByEmail(ctx context.Context, email string) (*auth.UserRecord, error) {
	user, err := fs.client.GetUserByEmail(ctx, email)
	if err != nil {
		return nil, fmt.Errorf("failed to get user by email: %v", err)
	}

	return user, nil
}

// UpdateUser updates a user in Firebase Authentication
func (fs *FirebaseService) UpdateUser(ctx context.Context, uid string, params *auth.UserToUpdate) (*auth.UserRecord, error) {
	user, err := fs.client.UpdateUser(ctx, uid, params)
	if err != nil {
		return nil, fmt.Errorf("failed to update user: %v", err)
	}

	return user, nil
}

// DeleteUser deletes a user from Firebase Authentication
func (fs *FirebaseService) DeleteUser(ctx context.Context, uid string) error {
	err := fs.client.DeleteUser(ctx, uid)
	if err != nil {
		return fmt.Errorf("failed to delete user: %v", err)
	}

	return nil
}

// SetCustomClaims sets custom claims for a user
func (fs *FirebaseService) SetCustomClaims(ctx context.Context, uid string, claims map[string]interface{}) error {
	err := fs.client.SetCustomUserClaims(ctx, uid, claims)
	if err != nil {
		return fmt.Errorf("failed to set custom claims: %v", err)
	}

	return nil
}

// RevokeRefreshTokens revokes all refresh tokens for a user
func (fs *FirebaseService) RevokeRefreshTokens(ctx context.Context, uid string) error {
	err := fs.client.RevokeRefreshTokens(ctx, uid)
	if err != nil {
		return fmt.Errorf("failed to revoke refresh tokens: %v", err)
	}

	return nil
}

// GenerateCustomToken generates a custom token for a user
func (fs *FirebaseService) GenerateCustomToken(ctx context.Context, uid string, claims map[string]interface{}) (string, error) {
	token, err := fs.client.CustomToken(ctx, uid)
	if err != nil {
		return "", fmt.Errorf("failed to generate custom token: %v", err)
	}

	return token, nil
}

// ListUsers lists users with pagination
func (fs *FirebaseService) ListUsers(ctx context.Context, maxResults int, pageToken string) ([]*auth.ExportedUserRecord, string, error) {
	iter := fs.client.Users(ctx, pageToken)
	iter.PageInfo().MaxSize = maxResults

	var users []*auth.ExportedUserRecord
	for {
		user, err := iter.Next()
		if err != nil {
			if err.Error() == "no more items in iterator" {
				break
			}
			return nil, "", fmt.Errorf("failed to list users: %v", err)
		}
		users = append(users, user)
	}

	nextPageToken := iter.PageInfo().Token
	return users, nextPageToken, nil
}

// VerifySessionCookie verifies a session cookie
func (fs *FirebaseService) VerifySessionCookie(ctx context.Context, sessionCookie string) (*auth.Token, error) {
	token, err := fs.client.VerifySessionCookie(ctx, sessionCookie)
	if err != nil {
		return nil, fmt.Errorf("failed to verify session cookie: %v", err)
	}

	return token, nil
}

// CreateSessionCookie creates a session cookie from an ID token
func (fs *FirebaseService) CreateSessionCookie(ctx context.Context, idToken string, expiresIn time.Duration) (string, error) {
	cookie, err := fs.client.SessionCookie(ctx, idToken, expiresIn)
	if err != nil {
		return "", fmt.Errorf("failed to create session cookie: %v", err)
	}

	return cookie, nil
}

// Global Firebase service instance
var firebaseService *FirebaseService

// InitFirebaseService initializes the global Firebase service
func InitFirebaseService() error {
	service, err := NewFirebaseService()
	if err != nil {
		return err
	}

	firebaseService = service
	log.Println("Firebase service initialized successfully")
	return nil
}

// GetFirebaseService returns the global Firebase service instance
func GetFirebaseService() *FirebaseService {
	return firebaseService
}

// ConvertFirebaseClaimsToJWTClaims converts Firebase claims to internal JWT claims format
func ConvertFirebaseClaimsToJWTClaims(fbClaims *FirebaseClaims) *JWTClaims {
	return &JWTClaims{
		UserID:   fbClaims.UserID,
		Username: fbClaims.Email, // Use email as username
		Roles:    fbClaims.Roles,
	}
}