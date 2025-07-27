package service

import (
	"context"
	"fmt"
	"log"
	"time"

	"main/internal/config"
	"main/internal/model"
	"main/internal/repository"
	"main/pkg/auth"
	firebase "firebase.google.com/go/v4"
	firebaseAuth "firebase.google.com/go/v4/auth"
	"go.mongodb.org/mongo-driver/bson/primitive"
)

// MigrationService handles user migration between Azure AD and Firebase
type MigrationService struct {
	firebaseApp    *firebase.App
	firebaseAuth   *firebaseAuth.Client
	firebaseConfig *config.FirebaseConfig
	userRepo       repository.Repository[*model.User]
	claimsManager  *auth.ClaimsManager
	logger         *log.Logger
}

// MigrationStatus represents the status of a user migration
type MigrationStatus string

const (
	MigrationStatusPending    MigrationStatus = "pending"
	MigrationStatusInProgress MigrationStatus = "in_progress"
	MigrationStatusCompleted  MigrationStatus = "completed"
	MigrationStatusFailed     MigrationStatus = "failed"
	MigrationStatusRollback   MigrationStatus = "rollback"
)

// MigrationRecord tracks the migration status of a user
type MigrationRecord struct {
	UserID           string          `json:"user_id"`
	AzureADObjectID  string          `json:"azure_ad_object_id"`
	FirebaseUID      string          `json:"firebase_uid,omitempty"`
	Status           MigrationStatus `json:"status"`
	StartedAt        time.Time       `json:"started_at"`
	CompletedAt      *time.Time      `json:"completed_at,omitempty"`
	ErrorMessage     string          `json:"error_message,omitempty"`
	RetryCount       int             `json:"retry_count"`
	LastRetryAt      *time.Time      `json:"last_retry_at,omitempty"`
	RollbackReason   string          `json:"rollback_reason,omitempty"`
	MigrationData    map[string]interface{} `json:"migration_data,omitempty"`
}

// UserMigrationData contains all user data needed for migration
type UserMigrationData struct {
	User        *model.User                `json:"user"`
	AzureClaims map[string]interface{}     `json:"azure_claims"`
	Roles       []string                   `json:"roles"`
	Permissions []string                   `json:"permissions"`
	CustomData  map[string]interface{}     `json:"custom_data,omitempty"`
}

// MigrationOptions configures the migration process
type MigrationOptions struct {
	BatchSize           int           `json:"batch_size"`
	RetryAttempts       int           `json:"retry_attempts"`
	RetryDelay          time.Duration `json:"retry_delay"`
	ValidateAfterCreate bool          `json:"validate_after_create"`
	PreserveCustomData  bool          `json:"preserve_custom_data"`
	DryRun              bool          `json:"dry_run"`
	SkipEmailVerified   bool          `json:"skip_email_verified"`
}

// DefaultMigrationOptions returns default migration options
func DefaultMigrationOptions() *MigrationOptions {
	return &MigrationOptions{
		BatchSize:           10,
		RetryAttempts:       3,
		RetryDelay:          time.Second * 5,
		ValidateAfterCreate: true,
		PreserveCustomData:  true,
		DryRun:              false,
		SkipEmailVerified:   false,
	}
}

// NewMigrationService creates a new migration service
func NewMigrationService(
	firebaseApp *firebase.App,
	firebaseConfig *config.FirebaseConfig,
	userRepo repository.Repository[*model.User],
	logger *log.Logger,
) (*MigrationService, error) {
	firebaseAuth, err := firebaseApp.Auth(context.Background())
	if err != nil {
		return nil, fmt.Errorf("failed to get Firebase auth client: %w", err)
	}

	claimsManager := auth.NewClaimsManager(&auth.FirebaseService{})

	return &MigrationService{
		firebaseApp:    firebaseApp,
		firebaseAuth:   firebaseAuth,
		firebaseConfig: firebaseConfig,
		userRepo:       userRepo,
		claimsManager:  claimsManager,
		logger:         logger,
	}, nil
}

// MigrateUser migrates a single user from Azure AD to Firebase
func (ms *MigrationService) MigrateUser(ctx context.Context, userID string, options *MigrationOptions) (*MigrationRecord, error) {
	// Convert userID string to ObjectID
	userObjectID, err := primitive.ObjectIDFromHex(userID)
	if err != nil {
		return nil, fmt.Errorf("invalid user ID format: %w", err)
	}
	if options == nil {
		options = DefaultMigrationOptions()
	}

	// Create migration record
	record := &MigrationRecord{
		UserID:    userID,
		Status:    MigrationStatusInProgress,
		StartedAt: time.Now(),
	}

	ms.logger.Printf("Starting migration for user %s", userID)

	// Get user data from database
	userData, err := ms.getUserMigrationData(ctx, userObjectID)
	if err != nil {
		return ms.failMigration(record, fmt.Sprintf("failed to get user data: %v", err)), err
	}

	record.AzureADObjectID = userData.User.AccountId.Hex()

	// Dry run check
	if options.DryRun {
		ms.logger.Printf("Dry run: would migrate user %s (%s)", userData.User.Email, userID)
		return ms.completeMigration(record, "dry-run-uid"), nil
	}

	// Create Firebase user
	firebaseUID, err := ms.createFirebaseUser(ctx, userData, options)
	if err != nil {
		return ms.failMigration(record, fmt.Sprintf("failed to create Firebase user: %v", err)), err
	}

	record.FirebaseUID = firebaseUID

	// Migrate custom claims
	if err := ms.migrateUserClaims(ctx, firebaseUID, userData); err != nil {
		ms.logger.Printf("Warning: failed to migrate claims for user %s: %v", userID, err)
		// Don't fail the migration for claims errors, just log them
	}

	// Update user record in database
	if err := ms.updateUserWithFirebaseUID(ctx, userObjectID, firebaseUID); err != nil {
		return ms.failMigration(record, fmt.Sprintf("failed to update user record: %v", err)), err
	}

	// Validate migration if requested
	if options.ValidateAfterCreate {
		if err := ms.validateMigration(ctx, firebaseUID, userData); err != nil {
			return ms.failMigration(record, fmt.Sprintf("migration validation failed: %v", err)), err
		}
	}

	ms.logger.Printf("Successfully migrated user %s to Firebase UID %s", userID, firebaseUID)
	return ms.completeMigration(record, firebaseUID), nil
}

// MigrateBatch migrates a batch of users
func (ms *MigrationService) MigrateBatch(ctx context.Context, userIDs []string, options *MigrationOptions) ([]*MigrationRecord, error) {
	if options == nil {
		options = DefaultMigrationOptions()
	}

	var records []*MigrationRecord
	successCount := 0
	failureCount := 0

	ms.logger.Printf("Starting batch migration for %d users", len(userIDs))

	for i, userID := range userIDs {
		ms.logger.Printf("Migrating user %d/%d: %s", i+1, len(userIDs), userID)

		record, err := ms.MigrateUser(ctx, userID, options)
		records = append(records, record)

		if err != nil {
			failureCount++
			ms.logger.Printf("Failed to migrate user %s: %v", userID, err)
		} else {
			successCount++
		}

		// Add delay between migrations to avoid rate limiting
		if i < len(userIDs)-1 {
			time.Sleep(time.Millisecond * 100)
		}
	}

	ms.logger.Printf("Batch migration completed: %d successful, %d failed", successCount, failureCount)
	return records, nil
}

// RollbackUser rolls back a user migration
func (ms *MigrationService) RollbackUser(ctx context.Context, userID string, reason string) error {
	ms.logger.Printf("Rolling back migration for user %s: %s", userID, reason)

	// Convert userID string to ObjectID
	userObjectID, err := primitive.ObjectIDFromHex(userID)
	if err != nil {
		return fmt.Errorf("invalid user ID format: %w", err)
	}

	// Get user data
	user, err := ms.userRepo.GetByID(ctx, userObjectID)
	if err != nil {
		return fmt.Errorf("failed to get user: %w", err)
	}

	if user.FirebaseUID == "" {
		return fmt.Errorf("user %s has no Firebase UID to rollback", userID)
	}

	// Delete Firebase user
	if err := ms.firebaseAuth.DeleteUser(ctx, user.FirebaseUID); err != nil {
		ms.logger.Printf("Warning: failed to delete Firebase user %s: %v", user.FirebaseUID, err)
	}

	// Clear Firebase UID from database
	updates := map[string]interface{}{
		"firebaseUID": "",
	}
	if _, err := ms.userRepo.Update(ctx, user.GetID(), updates); err != nil {
		return fmt.Errorf("failed to clear Firebase UID: %w", err)
	}

	ms.logger.Printf("Successfully rolled back migration for user %s", userID)
	return nil
}

// GetMigrationStatus checks if a user has been migrated
func (ms *MigrationService) GetMigrationStatus(ctx context.Context, userID string) (*MigrationRecord, error) {
	// Convert userID string to ObjectID
	userObjectID, err := primitive.ObjectIDFromHex(userID)
	if err != nil {
		return nil, fmt.Errorf("invalid user ID format: %w", err)
	}

	user, err := ms.userRepo.GetByID(ctx, userObjectID)
	if err != nil {
		return nil, fmt.Errorf("failed to get user: %w", err)
	}

	status := MigrationStatusPending
	if user.FirebaseUID != "" {
		// Verify Firebase user exists
		_, err := ms.firebaseAuth.GetUser(ctx, user.FirebaseUID)
		if err != nil {
			status = MigrationStatusFailed
		} else {
			status = MigrationStatusCompleted
		}
	}

	return &MigrationRecord{
		UserID:          userID,
		AzureADObjectID: user.AccountId.Hex(),
		FirebaseUID:     user.FirebaseUID,
		Status:          status,
	}, nil
}

// getUserMigrationData retrieves all user data needed for migration
func (ms *MigrationService) getUserMigrationData(ctx context.Context, userID primitive.ObjectID) (*UserMigrationData, error) {
	user, err := ms.userRepo.GetByID(ctx, userID)
	if err != nil {
		return nil, fmt.Errorf("failed to get user: %w", err)
	}

	if user.AccountId.IsZero() {
		return nil, fmt.Errorf("user %s has no Account ID", userID)
	}

	// Extract roles and permissions from user data
	// This would typically come from your existing auth system
	roles := []string{"user"} // Default role
	permissions := []string{"read:profile", "update:profile"} // Default permissions

	// You might want to extract these from existing claims or database
	// For now, we'll use defaults and let the application handle role assignment

	return &UserMigrationData{
		User:        user,
		AzureClaims: make(map[string]interface{}),
		Roles:       roles,
		Permissions: permissions,
		CustomData:  make(map[string]interface{}),
	}, nil
}

// createFirebaseUser creates a new Firebase user
func (ms *MigrationService) createFirebaseUser(ctx context.Context, userData *UserMigrationData, options *MigrationOptions) (string, error) {
	user := userData.User

	// Prepare user creation parameters
	params := (&firebaseAuth.UserToCreate{}).
		Email(user.Email).
		DisplayName(user.Fullname).
		EmailVerified(!options.SkipEmailVerified)

	// Set password if available (you might want to generate a random one)
	// For security, we'll let users reset their password after migration
	// params.Password("temporary-password-to-be-reset")

	// Create the user
	userRecord, err := ms.firebaseAuth.CreateUser(ctx, params)
	if err != nil {
		return "", fmt.Errorf("failed to create Firebase user: %w", err)
	}

	return userRecord.UID, nil
}

// migrateUserClaims migrates user roles and permissions to Firebase custom claims
func (ms *MigrationService) migrateUserClaims(ctx context.Context, firebaseUID string, userData *UserMigrationData) error {
	claims := make(map[string]interface{})

	// Add roles
	if len(userData.Roles) > 0 {
		claims["roles"] = userData.Roles
	}

	// Add permissions
	if len(userData.Permissions) > 0 {
		claims["permissions"] = userData.Permissions
	}

	// Add custom data if preserving
	for key, value := range userData.CustomData {
		if ms.firebaseConfig.IsClaimKeyAllowed(key) {
			claims[key] = value
		}
	}

	// Set custom claims
	if len(claims) > 0 {
		return ms.firebaseAuth.SetCustomUserClaims(ctx, firebaseUID, claims)
	}

	return nil
}

// updateUserWithFirebaseUID updates the user record with the Firebase UID
func (ms *MigrationService) updateUserWithFirebaseUID(ctx context.Context, userID primitive.ObjectID, firebaseUID string) error {
	user, err := ms.userRepo.GetByID(ctx, userID)
	if err != nil {
		return fmt.Errorf("failed to get user: %w", err)
	}

	updates := map[string]interface{}{
		"firebaseUID": firebaseUID,
	}
	_, err = ms.userRepo.Update(ctx, user.GetID(), updates)
	return err
}

// validateMigration validates that the migration was successful
func (ms *MigrationService) validateMigration(ctx context.Context, firebaseUID string, userData *UserMigrationData) error {
	// Get Firebase user
	firebaseUser, err := ms.firebaseAuth.GetUser(ctx, firebaseUID)
	if err != nil {
		return fmt.Errorf("Firebase user not found: %w", err)
	}

	// Validate email
	if firebaseUser.Email != userData.User.Email {
		return fmt.Errorf("email mismatch: expected %s, got %s", userData.User.Email, firebaseUser.Email)
	}

	// Validate display name
	expectedName := userData.User.Fullname
	if firebaseUser.DisplayName != expectedName {
		return fmt.Errorf("display name mismatch: expected %s, got %s", expectedName, firebaseUser.DisplayName)
	}

	// Validate custom claims
	if len(userData.Roles) > 0 || len(userData.Permissions) > 0 {
		token, err := ms.firebaseAuth.CustomToken(ctx, firebaseUID)
		if err != nil {
			return fmt.Errorf("failed to create custom token for validation: %w", err)
		}

		// Parse token to check claims (simplified validation)
		if token == "" {
			return fmt.Errorf("custom token is empty")
		}
	}

	return nil
}

// failMigration marks a migration as failed
func (ms *MigrationService) failMigration(record *MigrationRecord, errorMessage string) *MigrationRecord {
	now := time.Now()
	record.Status = MigrationStatusFailed
	record.ErrorMessage = errorMessage
	record.CompletedAt = &now
	return record
}

// completeMigration marks a migration as completed
func (ms *MigrationService) completeMigration(record *MigrationRecord, firebaseUID string) *MigrationRecord {
	now := time.Now()
	record.Status = MigrationStatusCompleted
	record.FirebaseUID = firebaseUID
	record.CompletedAt = &now
	return record
}

// GetMigrationStats returns statistics about the migration process
func (ms *MigrationService) GetMigrationStats(ctx context.Context) (map[string]interface{}, error) {
	// This would typically query your database for migration statistics
	// For now, we'll return a placeholder structure
	stats := map[string]interface{}{
		"total_users":     0,
		"migrated_users":  0,
		"pending_users":   0,
		"failed_users":    0,
		"migration_rate":  0.0,
		"last_migration":  nil,
		"estimated_completion": nil,
	}

	// You would implement actual statistics gathering here
	// by querying your user repository

	return stats, nil
}

// CleanupFailedMigrations removes Firebase users for failed migrations
func (ms *MigrationService) CleanupFailedMigrations(ctx context.Context, olderThan time.Duration) error {
	ms.logger.Printf("Starting cleanup of failed migrations older than %v", olderThan)

	// This would typically query your migration records
	// For now, we'll implement a basic cleanup structure
	cutoffTime := time.Now().Add(-olderThan)
	ms.logger.Printf("Cleaning up migrations failed before %v", cutoffTime)

	// Implementation would:
	// 1. Query failed migration records older than cutoffTime
	// 2. For each record, delete the Firebase user if it exists
	// 3. Update the migration record status

	return nil
}

// EstimateMigrationTime estimates how long a migration will take
func (ms *MigrationService) EstimateMigrationTime(userCount int, options *MigrationOptions) time.Duration {
	if options == nil {
		options = DefaultMigrationOptions()
	}

	// Estimate based on batch size and processing time
	processingTimePerUser := time.Second * 2 // Estimated time per user
	batchDelay := time.Millisecond * 100      // Delay between users

	totalTime := time.Duration(userCount) * (processingTimePerUser + batchDelay)
	retryOverhead := totalTime / 10 // 10% overhead for retries

	return totalTime + retryOverhead
}