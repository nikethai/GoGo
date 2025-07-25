package router

import (
	"encoding/json"
	"net/http"

	"github.com/go-chi/chi/v5"
	"go.mongodb.org/mongo-driver/bson/primitive"

	"main/internal/middleware"
	"main/internal/profile/model"
	"main/internal/profile/service"
	"main/internal/repository/mongo"
	"main/internal/server/response"
	userModel "main/internal/model"
	accountModel "main/internal/model"
	"main/db"
)

// ProfileRouter handles profile-related HTTP requests
type ProfileRouter struct {
	profileService *service.ProfileService
}

// NewProfileRouter creates a new ProfileRouter instance
func NewProfileRouter() *ProfileRouter {
	// Initialize repositories
	userRepo := mongo.NewMongoRepository[*userModel.User](db.MongoDatabase, "users")
	accountRepo := mongo.NewMongoRepository[*accountModel.Account](db.MongoDatabase, "accounts")
	
	// Initialize file service
	fileService := service.NewFileService("./uploads", 5*1024*1024) // 5MB max file size
	
	// Initialize profile service
	profileService := service.NewProfileService(userRepo, accountRepo, fileService)
	
	return &ProfileRouter{
		profileService: profileService,
	}
}

// Routes sets up the profile routes
func (pr *ProfileRouter) Routes() chi.Router {
	r := chi.NewRouter()
	
	// All profile routes require authentication
	r.Use(middleware.JWTAuth)
	
	// Profile management routes
	r.Get("/", pr.getProfile)
	r.Put("/", pr.updateProfile)
	
	// Avatar management routes
	r.Post("/avatar", pr.uploadAvatar)
	r.Delete("/avatar", pr.deleteAvatar)
	
	return r
}

// getProfile godoc
// @Summary Get user profile
// @Description Retrieve the authenticated user's profile information
// @Tags Profile
// @Accept json
// @Produce json
// @Success 200 {object} response.Response{data=model.ProfileResponse} "Profile retrieved successfully"
// @Failure 401 {object} response.Response "Unauthorized"
// @Failure 404 {object} response.Response "User not found"
// @Failure 500 {object} response.Response "Internal server error"
// @Router /profile [get]
func (pr *ProfileRouter) getProfile(w http.ResponseWriter, r *http.Request) {
	// Get user ID from context (set by auth middleware)
	userID, ok := r.Context().Value(middleware.UserIDKey).(primitive.ObjectID)
	if !ok {
		response.Unauthorized(w, "User ID not found in context")
		return
	}
	
	// Get profile
	profile, err := pr.profileService.GetProfile(r.Context(), userID)
	if err != nil {
		if profileErr, ok := err.(*model.ProfileError); ok {
			switch profileErr.Code {
			case "PROFILE_NOT_FOUND":
				response.NotFound(w, profileErr.Message)
			default:
				response.InternalServerError(w, profileErr.Message)
			}
			return
		}
		response.InternalServerError(w, "Failed to retrieve profile")
		return
	}
	
	response.Success(w, http.StatusOK, profile, "Profile retrieved successfully")
}

// updateProfile godoc
// @Summary Update user profile
// @Description Update the authenticated user's profile information
// @Tags Profile
// @Accept json
// @Produce json
// @Param request body model.ProfileUpdateRequest true "Profile update request"
// @Success 200 {object} response.Response{data=model.ProfileResponse} "Profile updated successfully"
// @Failure 400 {object} response.Response "Invalid request format or validation error"
// @Failure 401 {object} response.Response "Unauthorized"
// @Failure 404 {object} response.Response "User not found"
// @Failure 500 {object} response.Response "Internal server error"
// @Router /profile [put]
func (pr *ProfileRouter) updateProfile(w http.ResponseWriter, r *http.Request) {
	// Get user ID from context
	userID, ok := r.Context().Value(middleware.UserIDKey).(primitive.ObjectID)
	if !ok {
		response.Unauthorized(w, "User ID not found in context")
		return
	}
	
	// Parse request body
	var req model.ProfileUpdateRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		response.BadRequest(w, "Invalid request format: "+err.Error())
		return
	}
	
	// Update profile
	profile, err := pr.profileService.UpdateProfile(r.Context(), userID, &req)
	if err != nil {
		if profileErr, ok := err.(*model.ProfileError); ok {
			switch profileErr.Code {
			case "PROFILE_NOT_FOUND":
				response.NotFound(w, profileErr.Message)
			case "VALIDATION_ERROR":
				response.BadRequest(w, profileErr.Message)
			default:
				response.InternalServerError(w, profileErr.Message)
			}
			return
		}
		response.InternalServerError(w, "Failed to update profile")
		return
	}
	
	response.Success(w, http.StatusOK, profile, "Profile updated successfully")
}

// uploadAvatar godoc
// @Summary Upload user avatar
// @Description Upload a new avatar image for the authenticated user
// @Tags Profile
// @Accept multipart/form-data
// @Produce json
// @Param avatar formData file true "Avatar image file"
// @Success 200 {object} response.Response{data=model.AvatarResponse} "Avatar uploaded successfully"
// @Failure 400 {object} response.Response "Invalid file or validation error"
// @Failure 401 {object} response.Response "Unauthorized"
// @Failure 404 {object} response.Response "User not found"
// @Failure 500 {object} response.Response "Internal server error"
// @Router /profile/avatar [post]
func (pr *ProfileRouter) uploadAvatar(w http.ResponseWriter, r *http.Request) {
	// Get user ID from context
	userID, ok := r.Context().Value(middleware.UserIDKey).(primitive.ObjectID)
	if !ok {
		response.Unauthorized(w, "User ID not found in context")
		return
	}
	
	// Parse multipart form
	err := r.ParseMultipartForm(10 << 20) // 10MB max memory
	if err != nil {
		response.BadRequest(w, "Failed to parse multipart form: "+err.Error())
		return
	}
	
	// Get file from form
	file, header, err := r.FormFile("avatar")
	if err != nil {
		response.BadRequest(w, "Avatar file is required")
		return
	}
	defer file.Close()
	
	// Upload avatar
	avatarResponse, err := pr.profileService.UploadAvatar(r.Context(), userID, file, header)
	if err != nil {
		if profileErr, ok := err.(*model.ProfileError); ok {
			switch profileErr.Code {
			case "PROFILE_NOT_FOUND":
				response.NotFound(w, profileErr.Message)
			case "VALIDATION_ERROR", "INVALID_FILE", "FILE_TOO_LARGE":
				response.BadRequest(w, profileErr.Message)
			default:
				response.InternalServerError(w, profileErr.Message)
			}
			return
		}
		response.InternalServerError(w, "Failed to upload avatar")
		return
	}
	
	response.Success(w, http.StatusOK, avatarResponse, "Avatar uploaded successfully")
}

// deleteAvatar godoc
// @Summary Delete user avatar
// @Description Remove the authenticated user's avatar image
// @Tags Profile
// @Accept json
// @Produce json
// @Success 200 {object} response.Response "Avatar deleted successfully"
// @Failure 401 {object} response.Response "Unauthorized"
// @Failure 404 {object} response.Response "User not found"
// @Failure 500 {object} response.Response "Internal server error"
// @Router /profile/avatar [delete]
func (pr *ProfileRouter) deleteAvatar(w http.ResponseWriter, r *http.Request) {
	// Get user ID from context
	userID, ok := r.Context().Value(middleware.UserIDKey).(primitive.ObjectID)
	if !ok {
		response.Unauthorized(w, "User ID not found in context")
		return
	}
	
	// Delete avatar
	err := pr.profileService.DeleteAvatar(r.Context(), userID)
	if err != nil {
		if profileErr, ok := err.(*model.ProfileError); ok {
			switch profileErr.Code {
			case "PROFILE_NOT_FOUND":
				response.NotFound(w, profileErr.Message)
			default:
				response.InternalServerError(w, profileErr.Message)
			}
			return
		}
		response.InternalServerError(w, "Failed to delete avatar")
		return
	}
	
	response.Success(w, http.StatusOK, nil, "Avatar deleted successfully")
}