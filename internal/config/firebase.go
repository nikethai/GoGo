package config

import (
	"context"
	"fmt"
	"os"
	"strconv"
	"time"

	firebase "firebase.google.com/go/v4"
	"google.golang.org/api/option"
)

// FirebaseConfig holds all Firebase-related configuration
type FirebaseConfig struct {
	// Project Configuration
	ProjectID string `json:"project_id"`

	// Service Account Configuration
	ServiceAccountPath string `json:"service_account_path,omitempty"`
	ServiceAccountJSON string `json:"service_account_json,omitempty"`

	// Authentication Configuration
	AuthDomain           string        `json:"auth_domain,omitempty"`
	TokenExpirationTime  time.Duration `json:"token_expiration_time"`
	SessionCookieTimeout time.Duration `json:"session_cookie_timeout"`

	// Security Configuration
	EnforceEmailVerification bool     `json:"enforce_email_verification"`
	AllowedDomains          []string `json:"allowed_domains,omitempty"`
	RequireSecureCookies    bool     `json:"require_secure_cookies"`

	// Migration Configuration
	MigrationMode    bool `json:"migration_mode"`
	PreferFirebase   bool `json:"prefer_firebase"`
	FallbackToAzure  bool `json:"fallback_to_azure"`
	MigrationTimeout time.Duration `json:"migration_timeout"`

	// Logging and Monitoring
	EnableLogging     bool   `json:"enable_logging"`
	LogLevel          string `json:"log_level"`
	EnableMetrics     bool   `json:"enable_metrics"`
	MetricsEndpoint   string `json:"metrics_endpoint,omitempty"`

	// Rate Limiting
	RateLimitEnabled bool `json:"rate_limit_enabled"`
	RateLimitRPM     int  `json:"rate_limit_rpm"`
	RateLimitBurst   int  `json:"rate_limit_burst"`

	// Custom Claims Configuration
	MaxCustomClaimsSize int      `json:"max_custom_claims_size"`
	AllowedClaimKeys    []string `json:"allowed_claim_keys,omitempty"`
	RestrictedClaimKeys []string `json:"restricted_claim_keys,omitempty"`
}

// DefaultFirebaseConfig returns a Firebase configuration with sensible defaults
func DefaultFirebaseConfig() *FirebaseConfig {
	return &FirebaseConfig{
		// Default timeouts
		TokenExpirationTime:  time.Hour * 24,     // 24 hours
		SessionCookieTimeout: time.Hour * 24 * 7, // 7 days
		MigrationTimeout:     time.Second * 30,   // 30 seconds

		// Security defaults
		EnforceEmailVerification: true,
		RequireSecureCookies:     true,

		// Migration defaults
		MigrationMode:   true, // Start in migration mode
		PreferFirebase:  false, // Prefer Azure AD initially
		FallbackToAzure: true,  // Fallback to Azure AD if Firebase fails

		// Logging defaults
		EnableLogging: true,
		LogLevel:      "info",
		EnableMetrics: false,

		// Rate limiting defaults
		RateLimitEnabled: true,
		RateLimitRPM:     1000, // 1000 requests per minute
		RateLimitBurst:   100,  // Allow bursts of 100 requests

		// Custom claims defaults
		MaxCustomClaimsSize: 1000, // 1KB limit
		AllowedClaimKeys:    []string{"roles", "permissions", "tenant_id", "department"},
		RestrictedClaimKeys: []string{"admin", "system", "internal"},
	}
}

// LoadFirebaseConfigFromEnv loads Firebase configuration from environment variables
func LoadFirebaseConfigFromEnv() (*FirebaseConfig, error) {
	config := DefaultFirebaseConfig()

	// Required configuration
	projectID := os.Getenv("FIREBASE_PROJECT_ID")
	if projectID == "" {
		return nil, fmt.Errorf("FIREBASE_PROJECT_ID environment variable is required")
	}
	config.ProjectID = projectID

	// Service Account Configuration (either path or JSON)
	if serviceAccountPath := os.Getenv("FIREBASE_SERVICE_ACCOUNT_PATH"); serviceAccountPath != "" {
		config.ServiceAccountPath = serviceAccountPath
	} else if serviceAccountJSON := os.Getenv("FIREBASE_SERVICE_ACCOUNT_JSON"); serviceAccountJSON != "" {
		config.ServiceAccountJSON = serviceAccountJSON
	} else {
		return nil, fmt.Errorf("either FIREBASE_SERVICE_ACCOUNT_PATH or FIREBASE_SERVICE_ACCOUNT_JSON must be provided")
	}

	// Optional configuration with defaults
	if authDomain := os.Getenv("FIREBASE_AUTH_DOMAIN"); authDomain != "" {
		config.AuthDomain = authDomain
	}

	// Parse duration configurations
	if tokenExp := os.Getenv("FIREBASE_TOKEN_EXPIRATION_HOURS"); tokenExp != "" {
		if hours, err := strconv.Atoi(tokenExp); err == nil {
			config.TokenExpirationTime = time.Duration(hours) * time.Hour
		}
	}

	if sessionTimeout := os.Getenv("FIREBASE_SESSION_TIMEOUT_HOURS"); sessionTimeout != "" {
		if hours, err := strconv.Atoi(sessionTimeout); err == nil {
			config.SessionCookieTimeout = time.Duration(hours) * time.Hour
		}
	}

	// Parse boolean configurations
	if enforceEmail := os.Getenv("FIREBASE_ENFORCE_EMAIL_VERIFICATION"); enforceEmail != "" {
		config.EnforceEmailVerification = enforceEmail == "true"
	}

	if secureCookies := os.Getenv("FIREBASE_REQUIRE_SECURE_COOKIES"); secureCookies != "" {
		config.RequireSecureCookies = secureCookies == "true"
	}

	// Migration configuration
	if migrationMode := os.Getenv("FIREBASE_MIGRATION_MODE"); migrationMode != "" {
		config.MigrationMode = migrationMode == "true"
	}

	if preferFirebase := os.Getenv("FIREBASE_PREFER_FIREBASE"); preferFirebase != "" {
		config.PreferFirebase = preferFirebase == "true"
	}

	if fallbackToAzure := os.Getenv("FIREBASE_FALLBACK_TO_AZURE"); fallbackToAzure != "" {
		config.FallbackToAzure = fallbackToAzure == "true"
	}

	// Logging configuration
	if enableLogging := os.Getenv("FIREBASE_ENABLE_LOGGING"); enableLogging != "" {
		config.EnableLogging = enableLogging == "true"
	}

	if logLevel := os.Getenv("FIREBASE_LOG_LEVEL"); logLevel != "" {
		config.LogLevel = logLevel
	}

	// Rate limiting configuration
	if rateLimitRPM := os.Getenv("FIREBASE_RATE_LIMIT_RPM"); rateLimitRPM != "" {
		if rpm, err := strconv.Atoi(rateLimitRPM); err == nil {
			config.RateLimitRPM = rpm
		}
	}

	return config, nil
}

// Validate checks if the Firebase configuration is valid
func (c *FirebaseConfig) Validate() error {
	if c.ProjectID == "" {
		return fmt.Errorf("project ID is required")
	}

	if c.ServiceAccountPath == "" && c.ServiceAccountJSON == "" {
		return fmt.Errorf("either service account path or JSON must be provided")
	}

	if c.TokenExpirationTime <= 0 {
		return fmt.Errorf("token expiration time must be positive")
	}

	if c.SessionCookieTimeout <= 0 {
		return fmt.Errorf("session cookie timeout must be positive")
	}

	if c.RateLimitRPM <= 0 {
		return fmt.Errorf("rate limit RPM must be positive")
	}

	if c.MaxCustomClaimsSize <= 0 {
		return fmt.Errorf("max custom claims size must be positive")
	}

	return nil
}

// CreateFirebaseApp creates a Firebase app instance from the configuration
func (c *FirebaseConfig) CreateFirebaseApp(ctx context.Context) (*firebase.App, error) {
	if err := c.Validate(); err != nil {
		return nil, fmt.Errorf("invalid configuration: %w", err)
	}

	var opts []option.ClientOption

	// Configure service account
	if c.ServiceAccountPath != "" {
		opts = append(opts, option.WithCredentialsFile(c.ServiceAccountPath))
	} else if c.ServiceAccountJSON != "" {
		opts = append(opts, option.WithCredentialsJSON([]byte(c.ServiceAccountJSON)))
	}

	// Create Firebase configuration
	firebaseConfig := &firebase.Config{
		ProjectID: c.ProjectID,
	}

	// Initialize Firebase app
	app, err := firebase.NewApp(ctx, firebaseConfig, opts...)
	if err != nil {
		return nil, fmt.Errorf("failed to initialize Firebase app: %w", err)
	}

	return app, nil
}

// IsMigrationMode returns true if the system is in migration mode
func (c *FirebaseConfig) IsMigrationMode() bool {
	return c.MigrationMode
}

// ShouldPreferFirebase returns true if Firebase should be preferred over Azure AD
func (c *FirebaseConfig) ShouldPreferFirebase() bool {
	return c.PreferFirebase
}

// ShouldFallbackToAzure returns true if the system should fallback to Azure AD
func (c *FirebaseConfig) ShouldFallbackToAzure() bool {
	return c.FallbackToAzure
}

// GetTokenExpirationTime returns the token expiration time
func (c *FirebaseConfig) GetTokenExpirationTime() time.Duration {
	return c.TokenExpirationTime
}

// GetSessionCookieTimeout returns the session cookie timeout
func (c *FirebaseConfig) GetSessionCookieTimeout() time.Duration {
	return c.SessionCookieTimeout
}

// IsEmailVerificationRequired returns true if email verification is required
func (c *FirebaseConfig) IsEmailVerificationRequired() bool {
	return c.EnforceEmailVerification
}

// GetAllowedDomains returns the list of allowed email domains
func (c *FirebaseConfig) GetAllowedDomains() []string {
	return c.AllowedDomains
}

// IsRateLimitEnabled returns true if rate limiting is enabled
func (c *FirebaseConfig) IsRateLimitEnabled() bool {
	return c.RateLimitEnabled
}

// GetRateLimitRPM returns the rate limit requests per minute
func (c *FirebaseConfig) GetRateLimitRPM() int {
	return c.RateLimitRPM
}

// GetMaxCustomClaimsSize returns the maximum size for custom claims
func (c *FirebaseConfig) GetMaxCustomClaimsSize() int {
	return c.MaxCustomClaimsSize
}

// IsClaimKeyAllowed checks if a custom claim key is allowed
func (c *FirebaseConfig) IsClaimKeyAllowed(key string) bool {
	// Check if key is in restricted list
	for _, restricted := range c.RestrictedClaimKeys {
		if key == restricted {
			return false
		}
	}

	// If allowed list is empty, allow all non-restricted keys
	if len(c.AllowedClaimKeys) == 0 {
		return true
	}

	// Check if key is in allowed list
	for _, allowed := range c.AllowedClaimKeys {
		if key == allowed {
			return true
		}
	}

	return false
}