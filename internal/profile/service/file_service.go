package service

import (
	"context"
	"fmt"
	"io"
	"mime/multipart"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/google/uuid"
	"go.mongodb.org/mongo-driver/bson/primitive"

	"main/internal/profile/model"
)

// FileService handles file operations for profile management
type FileService struct {
	uploadPath   string
	maxFileSize  int64
	allowedTypes []string
}

// FileServiceInterface defines the contract for file operations
type FileServiceInterface interface {
	SaveAvatar(ctx context.Context, userID primitive.ObjectID, file multipart.File, header *multipart.FileHeader) (string, error)
	DeleteAvatar(ctx context.Context, avatarPath string) error
	ValidateFile(header *multipart.FileHeader) error
	GenerateAvatarPath(userID primitive.ObjectID, filename string) string
	GetAvatarURL(avatarPath string) string
}// NewFileService creates a new FileService instance
func NewFileService(uploadPath string, maxFileSize int64) *FileService {
	return &FileService{
		uploadPath: uploadPath,
		maxFileSize: maxFileSize,
		allowedTypes: []string{".jpg", ".jpeg", ".png", ".gif"},
	}
}

// SaveAvatar saves an uploaded avatar file to the file system
func (fs *FileService) SaveAvatar(ctx context.Context, userID primitive.ObjectID, file multipart.File, header *multipart.FileHeader) (string, error) {
	// Validate the file first
	if err := fs.ValidateFile(header); err != nil {
		return "", err
	}

	// Generate unique file path
	avatarPath := fs.GenerateAvatarPath(userID, header.Filename)
	fullPath := filepath.Join(fs.uploadPath, avatarPath)

	// Ensure the directory exists
	dir := filepath.Dir(fullPath)
	if err := os.MkdirAll(dir, 0755); err != nil {
		return "", fmt.Errorf("failed to create upload directory: %w", err)
	}

	// Create the destination file
	destFile, err := os.Create(fullPath)
	if err != nil {
		return "", fmt.Errorf("failed to create destination file: %w", err)
	}
	defer destFile.Close()

	// Copy the uploaded file to destination
	_, err = io.Copy(destFile, file)
	if err != nil {
		return "", fmt.Errorf("failed to save file: %w", err)
	}

	return avatarPath, nil
}

// DeleteAvatar removes an avatar file from the file system
func (fs *FileService) DeleteAvatar(ctx context.Context, avatarPath string) error {
	if avatarPath == "" {
		return nil // Nothing to delete
	}

	fullPath := filepath.Join(fs.uploadPath, avatarPath)
	
	// Check if file exists
	if _, err := os.Stat(fullPath); os.IsNotExist(err) {
		return nil // File doesn't exist, consider it deleted
	}

	// Remove the file
	if err := os.Remove(fullPath); err != nil {
		return fmt.Errorf("failed to delete avatar file: %w", err)
	}

	return nil
}

// ValidateFile validates the uploaded file against size and type constraints
func (fs *FileService) ValidateFile(header *multipart.FileHeader) error {
	// Check file size
	if header.Size > fs.maxFileSize {
		return model.ErrFileTooLarge
	}

	// Check file type by extension
	ext := strings.ToLower(filepath.Ext(header.Filename))
	if !fs.isAllowedType(ext) {
		return model.ErrInvalidFileType
	}

	// Additional validation can be added here:
	// - MIME type validation
	// - File content validation
	// - Virus scanning

	return nil
}

// GenerateAvatarPath generates a unique file path for the avatar
func (fs *FileService) GenerateAvatarPath(userID primitive.ObjectID, filename string) string {
	// Extract file extension
	ext := filepath.Ext(filename)
	
	// Generate unique filename using UUID and timestamp
	uniqueID := uuid.New().String()
	timestamp := time.Now().Unix()
	newFilename := fmt.Sprintf("%s_%d%s", uniqueID, timestamp, ext)
	
	// Create path structure: avatars/userID/filename
	return filepath.Join("avatars", userID.Hex(), newFilename)
}

// isAllowedType checks if the file extension is in the allowed types list
func (fs *FileService) isAllowedType(ext string) bool {
	for _, allowedType := range fs.allowedTypes {
		if ext == allowedType {
			return true
		}
	}
	return false
}

// GetAvatarURL generates the public URL for an avatar
func (fs *FileService) GetAvatarURL(avatarPath string) string {
	if avatarPath == "" {
		return ""
	}
	// In a real application, this would return the full URL
	// For now, return the relative path
	return "/uploads/" + avatarPath
}