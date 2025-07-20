package router

import (
	"context"
	"encoding/json"
	"net/http"

	"github.com/go-chi/chi/v5"
	"go.mongodb.org/mongo-driver/bson/primitive"

	"main/db"
	"main/internal/repository/mongo"
	"main/internal/service"
	"main/internal/model"
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

func (ur *UserRouter) getUserByID(w http.ResponseWriter, r *http.Request) {
	uid := chi.URLParam(r, "uid")
	
	// Convert string ID to ObjectID
	userID, err := primitive.ObjectIDFromHex(uid)
	if err != nil {
		w.WriteHeader(http.StatusBadRequest)
		w.Write([]byte("Invalid user ID format"))
		return
	}
	
	// Use GetUserWithAccount to get user with account info (similar to old behavior)
	userWithAccount, err := ur.UserService.GetUserWithAccount(context.Background(), userID)
	if err != nil {
		w.WriteHeader(http.StatusInternalServerError)
		w.Write([]byte(err.Error()))
		return
	}
	w.WriteHeader(http.StatusOK)
	json.NewEncoder(w).Encode(userWithAccount)
}

func (ur *UserRouter) newUser(w http.ResponseWriter, r *http.Request) {
	var userReq model.UserRequest
	err := json.NewDecoder(r.Body).Decode(&userReq)
	if err != nil {
		w.WriteHeader(http.StatusBadRequest)
		w.Write([]byte(err.Error()))
		return
	}
	
	// Use the new method that creates a user for an existing account
	createdUser, err := ur.UserService.CreateUserForExistingAccount(context.Background(), &userReq)
	if err != nil {
		w.WriteHeader(http.StatusInternalServerError)
		w.Write([]byte(err.Error()))
		return
	}
	
	w.WriteHeader(http.StatusOK)
	json.NewEncoder(w).Encode(createdUser)
}
