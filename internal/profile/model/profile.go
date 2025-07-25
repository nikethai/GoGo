package model

import (
	"time"

	"go.mongodb.org/mongo-driver/bson/primitive"
)

// ProfileResponse represents the complete user profile information
type ProfileResponse struct {
	ID        primitive.ObjectID `json:"id" bson:"_id"`
	Fullname  string             `json:"fullName" bson:"fullname"`
	Email     string             `json:"email" bson:"email"`
	Phone     string             `json:"phone" bson:"phone"`
	Address   string             `json:"address" bson:"address"`
	DOB       string             `json:"dob" bson:"dob"`
	Avatar    string             `json:"avatar" bson:"avatar"`
	Status    string             `json:"status" bson:"status"`
	Account   AccountInfo        `json:"account"`
	UpdatedAt time.Time          `json:"updatedAt" bson:"updated_at"`
	CreatedAt time.Time          `json:"createdAt" bson:"created_at"`
}

// ProfileUpdateRequest represents the request payload for updating user profile
type ProfileUpdateRequest struct {
	Fullname string `json:"fullName" validate:"required,min=2,max=100"`
	Email    string `json:"email" validate:"required,email"`
	Phone    string `json:"phone" validate:"required,min=10,max=15"`
	Address  string `json:"address" validate:"max=200"`
	DOB      string `json:"dob" validate:"required"`
}

// AvatarResponse represents the response after avatar upload
type AvatarResponse struct {
	AvatarURL string    `json:"avatarUrl"`
	UpdatedAt time.Time `json:"updatedAt"`
}

// AccountInfo represents account information included in profile response
type AccountInfo struct {
	Username  string    `json:"username"`
	Roles     []string  `json:"roles"`
	CreatedAt time.Time `json:"createdAt"`
}

// ProfileError represents structured error responses for profile operations
type ProfileError struct {
	Code    string            `json:"code"`
	Message string            `json:"message"`
	Fields  map[string]string `json:"fields,omitempty"`
}

// Error implements the error interface
func (pe ProfileError) Error() string {
	return pe.Message
}

// Predefined profile errors
var (
	ErrProfileNotFound    = ProfileError{"PROFILE_NOT_FOUND", "Profile not found", nil}
	ErrEmailAlreadyExists = ProfileError{"EMAIL_EXISTS", "Email already in use", nil}
	ErrInvalidFileType    = ProfileError{"INVALID_FILE", "Invalid file type. Only JPEG, PNG, and GIF are allowed", nil}
	ErrFileTooLarge       = ProfileError{"FILE_TOO_LARGE", "File size exceeds the maximum limit of 5MB", nil}
	ErrInvalidInput       = ProfileError{"INVALID_INPUT", "Invalid input data", nil}
	ErrUnauthorized       = ProfileError{"UNAUTHORIZED", "Unauthorized to access this profile", nil}
	ErrInternalServer     = ProfileError{"INTERNAL_ERROR", "Internal server error occurred", nil}
)

// ValidationError creates a ProfileError with field-specific validation errors
func ValidationError(fields map[string]string) ProfileError {
	return ProfileError{
		Code:    "VALIDATION_ERROR",
		Message: "Validation failed",
		Fields:  fields,
	}
}