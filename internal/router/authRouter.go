package router

import (
	"context"
	"encoding/json"
	"net/http"

	"github.com/go-chi/chi/v5"

	"main/db"
	"main/internal/repository/mongo"
	"main/internal/service"
	"main/internal/config"
	"main/internal/model"
)

type AuthRouter struct {
	authService *service.AuthService
	userService *service.UserService
}

func NewAuthRouter() *AuthRouter {
	// Initialize repositories
	userRepo := mongo.NewMongoRepository[*model.User](db.MongoDatabase, config.UserCollection)
	accountRepo := mongo.NewMongoRepository[*model.Account](db.MongoDatabase, config.AccountCollection)
	
	return &AuthRouter{
		authService: service.NewAuthService(),
		userService: service.NewUserService(userRepo, accountRepo),
	}
}

func (ar *AuthRouter) SetupRoutes() chi.Router {
	r := chi.NewRouter()
	r.Post("/login", ar.login)
	r.Post("/register", ar.register)
	return r
}

func (ar *AuthRouter) login(w http.ResponseWriter, r *http.Request) {
	var authReq model.AccountRequest
	err := json.NewDecoder(r.Body).Decode(&authReq)

	if err != nil {
		w.WriteHeader(http.StatusInternalServerError)
		w.Write([]byte(err.Error()))
		return
	}

	account, err := ar.authService.Login(authReq.Username, authReq.Password)
	if err != nil {
		w.WriteHeader(http.StatusInternalServerError)
		w.Write([]byte(err.Error()))
		return
	}
	
	// Get user by account ID using the new service method
	userWithAccount, usrErr := ar.userService.GetUserByAccountID(context.Background(), account.ID)
	
	if usrErr != nil {
		// If user not found, return just the account (maintaining old behavior)
		if usrErr.Error() == "user not found" {
			w.WriteHeader(http.StatusOK)
			json.NewEncoder(w).Encode(account)
			return
		}
		w.WriteHeader(http.StatusInternalServerError)
		w.Write([]byte(usrErr.Error()))
		return
	}

	w.WriteHeader(http.StatusOK)
	json.NewEncoder(w).Encode(userWithAccount)
}

func (ar *AuthRouter) register(w http.ResponseWriter, r *http.Request) {
	var authRegis model.AccountRegister
	err := json.NewDecoder(r.Body).Decode(&authRegis)

	if err != nil {
		w.WriteHeader(http.StatusInternalServerError)
		w.Write([]byte("Decode err: " + err.Error()))
		return
	}

	rs, err := ar.authService.Register(authRegis.Username, authRegis.Password, authRegis.Roles)
	if err != nil {
		w.WriteHeader(http.StatusInternalServerError)
		w.Write([]byte(err.Error()))
		return
	}
	w.WriteHeader(http.StatusOK)
	json.NewEncoder(w).Encode(rs)
}
