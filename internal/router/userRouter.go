package router

import (
	"context"
	"encoding/json"
	"net/http"

	"github.com/go-chi/chi/v5"
	"go.mongodb.org/mongo-driver/bson/primitive"

	"main/db"
	"main/internal/model"
	"main/internal/repository/mongo"
	"main/internal/server/response"
	"main/internal/service"
)

type UserRouter struct {
	UserService *service.UserService
}

func NewUserRouter() *UserRouter {
	// Initialize repositories
	userRepo := mongo.NewMongoRepository[*model.User](db.MongoDatabase, "user")
	accountRepo := mongo.NewMongoRepository[*model.Account](db.MongoDatabase, "account")

	return &UserRouter{
		UserService: service.NewUserService(userRepo, accountRepo),
	}
}

func (ur *UserRouter) Routes() chi.Router {
	r := chi.NewRouter()
	r.Get("/{uid}", ur.getUserByID)
	r.Post("/", ur.newUser)
	return r
}

// getUserByID godoc
// @Summary Get user by ID
// @Description Retrieve a user by their ID with account information
// @Tags Users
// @Accept json
// @Produce json
// @Param uid path string true "User ID"
// @Success 200 {object} response.Response{data=model.UserResponse} "User retrieved successfully"
// @Failure 400 {object} response.Response "Invalid user ID format"
// @Failure 404 {object} response.Response "User not found"
// @Failure 500 {object} response.Response "Internal server error"
// @Router /users/{uid} [get]
func (ur *UserRouter) getUserByID(w http.ResponseWriter, r *http.Request) {
	uid := chi.URLParam(r, "uid")

	// Convert string ID to ObjectID
	userID, err := primitive.ObjectIDFromHex(uid)
	if err != nil {
		response.BadRequest(w, "Invalid user ID format")
		return
	}

	// Use GetUserWithAccount to get user with account info (similar to old behavior)
	userWithAccount, err := ur.UserService.GetUserWithAccount(context.Background(), userID)
	if err != nil {
		response.NotFound(w, "User not found")
		return
	}
	response.Success(w, http.StatusOK, userWithAccount, "User retrieved successfully")
}

// newUser godoc
// @Summary Create a new user
// @Description Create a new user for an existing account
// @Tags Users
// @Accept json
// @Produce json
// @Param request body model.UserRequest true "User creation request"
// @Success 201 {object} response.Response{data=model.UserResponse} "User created successfully"
// @Failure 400 {object} response.Response "Invalid request format"
// @Failure 500 {object} response.Response "Failed to create user"
// @Router /users [post]
func (ur *UserRouter) newUser(w http.ResponseWriter, r *http.Request) {
	var userReq model.UserRequest
	err := json.NewDecoder(r.Body).Decode(&userReq)
	if err != nil {
		response.BadRequest(w, "Invalid request format: "+err.Error())
		return
	}

	// Use the new method that creates a user for an existing account
	createdUser, err := ur.UserService.CreateUserForExistingAccount(context.Background(), &userReq)
	if err != nil {
		response.InternalServerError(w, "Failed to create user: "+err.Error())
		return
	}

	response.Created(w, createdUser, "User created successfully")
}
