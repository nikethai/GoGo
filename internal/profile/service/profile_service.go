package service

import (
	"context"
	"fmt"
	"mime/multipart"
	"regexp"
	"strings"
	"time"

	"go.mongodb.org/mongo-driver/bson/primitive"
	"go.mongodb.org/mongo-driver/mongo"

	"main/internal/profile/model"
	"main/internal/repository"
	userModel "main/internal/model"
	accountModel "main/internal/model"
)

// ProfileService handles profile management business logic
type ProfileService struct {
	userRepo    repository.Repository[*userModel.User]
	accountRepo repository.Repository[*accountModel.Account]
	fileService FileServiceInterface
}

// ProfileServiceInterface defines the contract for profile operations
type ProfileServiceInterface interface {
	GetProfile(ctx context.Context, userID primitive.ObjectID) (*model.ProfileResponse, error)
	UpdateProfile(ctx context.Context, userID primitive.ObjectID, req *model.ProfileUpdateRequest) (*model.ProfileResponse, error)
	UploadAvatar(ctx context.Context, userID primitive.ObjectID, file multipart.File, header *multipart.FileHeader) (*model.AvatarResponse, error)
	DeleteAvatar(ctx context.Context, userID primitive.ObjectID) error
	ValidateProfileUpdate(req *model.ProfileUpdateRequest) error
}

// NewProfileService creates a new ProfileService instance
func NewProfileService(
	userRepo repository.Repository[*userModel.User],
	accountRepo repository.Repository[*accountModel.Account],
	fileService FileServiceInterface,
) *ProfileService {
	return &ProfileService{
		userRepo:    userRepo,
		accountRepo: accountRepo,
		fileService: fileService,
	}
}

// GetProfile retrieves complete profile information for a user
func (ps *ProfileService) GetProfile(ctx context.Context, userID primitive.ObjectID) (*model.ProfileResponse, error) {
	// Get user information
	user, err := ps.userRepo.GetByID(ctx, userID)
	if err != nil {
		if err == mongo.ErrNoDocuments {
			return nil, model.ErrProfileNotFound
		}
		return nil, fmt.Errorf("failed to get user: %w", err)
	}

	// Get account information
	account, err := ps.accountRepo.GetByID(ctx, user.AccountId)
	if err != nil {
		if err == mongo.ErrNoDocuments {
			return nil, model.ErrProfileNotFound
		}
		return nil, fmt.Errorf("failed to get account: %w", err)
	}

	// Build profile response
	profile := &model.ProfileResponse{
		ID:        user.ID,
		Fullname:  user.Fullname,
		Email:     user.Email,
		Phone:     user.Phone,
		Address:   user.Address,
		DOB:       user.DOB,
		Avatar:    ps.fileService.GetAvatarURL(user.Avatar),
		Status:    user.Status,
		UpdatedAt: user.UpdatedAt,
		CreatedAt: user.CreatedAt,
		Account: model.AccountInfo{
			Username:  account.Username,
			Roles:     ps.convertRolesToStrings(account.Roles),
			CreatedAt: account.CreatedAt,
		},
	}

	return profile, nil
}

// UpdateProfile updates user profile information
func (ps *ProfileService) UpdateProfile(ctx context.Context, userID primitive.ObjectID, req *model.ProfileUpdateRequest) (*model.ProfileResponse, error) {
	// Validate the update request
	if err := ps.ValidateProfileUpdate(req); err != nil {
		return nil, err
	}

	// Check if email is already in use by another user
	if err := ps.validateEmailUniqueness(ctx, userID, req.Email); err != nil {
		return nil, err
	}

	// Get current user
	user, err := ps.userRepo.GetByID(ctx, userID)
	if err != nil {
		if err == mongo.ErrNoDocuments {
			return nil, model.ErrProfileNotFound
		}
		return nil, fmt.Errorf("failed to get user: %w", err)
	}

	// Update user fields
	user.Fullname = req.Fullname
	user.Email = req.Email
	user.Phone = req.Phone
	user.Address = req.Address
	user.DOB = req.DOB
	user.UpdatedAt = time.Now()

	// Save updated user
	if _, err := ps.userRepo.Update(ctx, userID, user); err != nil {
		return nil, fmt.Errorf("failed to update user: %w", err)
	}

	// Return updated profile
	return ps.GetProfile(ctx, userID)
}

// UploadAvatar handles avatar file upload for a user
func (ps *ProfileService) UploadAvatar(ctx context.Context, userID primitive.ObjectID, file multipart.File, header *multipart.FileHeader) (*model.AvatarResponse, error) {
	// Get current user to check for existing avatar
	user, err := ps.userRepo.GetByID(ctx, userID)
	if err != nil {
		if err == mongo.ErrNoDocuments {
			return nil, model.ErrProfileNotFound
		}
		return nil, fmt.Errorf("failed to get user: %w", err)
	}

	// Save the new avatar file
	avatarPath, err := ps.fileService.SaveAvatar(ctx, userID, file, header)
	if err != nil {
		return nil, err
	}

	// Delete old avatar if exists
	if user.Avatar != "" {
		if err := ps.fileService.DeleteAvatar(ctx, user.Avatar); err != nil {
			// Log error but don't fail the operation
			fmt.Printf("Warning: failed to delete old avatar: %v\n", err)
		}
	}

	// Update user with new avatar path
	user.Avatar = avatarPath
	user.UpdatedAt = time.Now()

	if _, err := ps.userRepo.Update(ctx, userID, user); err != nil {
		// If database update fails, clean up the uploaded file
		ps.fileService.DeleteAvatar(ctx, avatarPath)
		return nil, fmt.Errorf("failed to update user avatar: %w", err)
	}

	return &model.AvatarResponse{
		AvatarURL: ps.fileService.GetAvatarURL(avatarPath),
		UpdatedAt: user.UpdatedAt,
	}, nil
}

// DeleteAvatar removes the user's avatar
func (ps *ProfileService) DeleteAvatar(ctx context.Context, userID primitive.ObjectID) error {
	// Get current user
	user, err := ps.userRepo.GetByID(ctx, userID)
	if err != nil {
		if err == mongo.ErrNoDocuments {
			return model.ErrProfileNotFound
		}
		return fmt.Errorf("failed to get user: %w", err)
	}

	// Delete avatar file if exists
	if user.Avatar != "" {
		if err := ps.fileService.DeleteAvatar(ctx, user.Avatar); err != nil {
			return fmt.Errorf("failed to delete avatar file: %w", err)
		}
	}

	// Update user to remove avatar reference
	user.Avatar = ""
	user.UpdatedAt = time.Now()

	if _, err := ps.userRepo.Update(ctx, userID, user); err != nil {
		return fmt.Errorf("failed to update user: %w", err)
	}

	return nil
}

// ValidateProfileUpdate validates profile update request data
func (ps *ProfileService) ValidateProfileUpdate(req *model.ProfileUpdateRequest) error {
	validationErrors := make(map[string]string)

	// Validate fullname
	if len(strings.TrimSpace(req.Fullname)) < 2 {
		validationErrors["fullName"] = "Full name must be at least 2 characters long"
	}
	if len(req.Fullname) > 100 {
		validationErrors["fullName"] = "Full name must not exceed 100 characters"
	}

	// Validate email
	if !ps.isValidEmail(req.Email) {
		validationErrors["email"] = "Invalid email format"
	}

	// Validate phone
	if !ps.isValidPhone(req.Phone) {
		validationErrors["phone"] = "Invalid phone number format"
	}

	// Validate address
	if len(req.Address) > 200 {
		validationErrors["address"] = "Address must not exceed 200 characters"
	}

	// Validate date of birth
	if !ps.isValidDOB(req.DOB) {
		validationErrors["dob"] = "Invalid date of birth format (expected YYYY-MM-DD)"
	}

	if len(validationErrors) > 0 {
		return model.ValidationError(validationErrors)
	}

	return nil
}

// validateEmailUniqueness checks if email is already in use by another user
func (ps *ProfileService) validateEmailUniqueness(ctx context.Context, userID primitive.ObjectID, email string) error {
	// Find user with the same email
	existingUser, err := ps.userRepo.GetByField(ctx, "email", email)
	if err != nil && err != mongo.ErrNoDocuments {
		return fmt.Errorf("failed to check email uniqueness: %w", err)
	}

	if existingUser != nil && existingUser.ID != userID {
		return model.ErrEmailAlreadyExists
	}

	return nil
}

// isValidEmail validates email format using regex
func (ps *ProfileService) isValidEmail(email string) bool {
	emailRegex := regexp.MustCompile(`^[a-zA-Z0-9._%+-]+@[a-zA-Z0-9.-]+\.[a-zA-Z]{2,}$`)
	return emailRegex.MatchString(email)
}

// isValidPhone validates phone number format
func (ps *ProfileService) isValidPhone(phone string) bool {
	// Remove common phone number characters
	cleanPhone := strings.ReplaceAll(phone, " ", "")
	cleanPhone = strings.ReplaceAll(cleanPhone, "-", "")
	cleanPhone = strings.ReplaceAll(cleanPhone, "(", "")
	cleanPhone = strings.ReplaceAll(cleanPhone, ")", "")
	cleanPhone = strings.ReplaceAll(cleanPhone, "+", "")

	// Check if it's between 10-15 digits
	phoneRegex := regexp.MustCompile(`^\d{10,15}$`)
	return phoneRegex.MatchString(cleanPhone)
}

// isValidDOB validates date of birth format (YYYY-MM-DD)
func (ps *ProfileService) isValidDOB(dob string) bool {
	_, err := time.Parse("2006-01-02", dob)
	return err == nil
}

// convertRolesToStrings converts Role slice to string slice
func (ps *ProfileService) convertRolesToStrings(roles []accountModel.Role) []string {
	result := make([]string, len(roles))
	for i, role := range roles {
		result[i] = role.Name
	}
	return result
}