package router

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"

	"github.com/go-chi/chi/v5"

	"main/db"
	"main/internal/config"
	customError "main/internal/error"
	"main/internal/model"
	mongorepo "main/internal/repository/mongo"
	"main/internal/server/response"
	"main/internal/service"
)

type AuthRouter struct {
	authService *service.AuthService
	userService *service.UserService
}

func NewAuthRouter() *AuthRouter {
	// Initialize repositories
	userRepo := mongorepo.NewMongoRepository[*model.User](db.MongoDatabase, config.UserCollection)
	accountRepo := mongorepo.NewMongoRepository[*model.Account](db.MongoDatabase, config.AccountCollection)

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

// login godoc
// @Summary User login
// @Description Authenticate user with username and password
// @Tags Authentication
// @Accept json
// @Produce json
// @Param request body model.AccountRequest true "Login credentials"
// @Success 200 {object} model.UserResponse "User successfully authenticated"
// @Success 200 {object} model.AccountResponse "Account found but no user profile"
// @Failure 400 {object} map[string]string "Bad request"
// @Failure 401 {object} map[string]string "Invalid credentials"
// @Failure 500 {object} map[string]string "Internal server error"
// @Router /auth/login [post]
func (ar *AuthRouter) login(w http.ResponseWriter, r *http.Request) {
	var authReq model.AccountRequest
	err := json.NewDecoder(r.Body).Decode(&authReq)

	if err != nil {
		response.BadRequest(w, customError.ErrInvalidRequestFormat.Error()+": "+err.Error())
		return
	}

	// Log login attempt
	fmt.Println("===== Login attempt for username:", authReq.Username, "=====")

	account, err := ar.authService.Login(authReq.Username, authReq.Password)
	if err != nil {
		fmt.Println("===== Login failed for username:", authReq.Username, "Error:", err.Error(), "=====")
		response.Unauthorized(w, customError.ErrInvalidCredentials.Error())
		return
	}

	// Get user by account ID using the new service method
	userWithAccount, usrErr := ar.userService.GetUserByAccountID(context.Background(), account.ID)

	if usrErr != nil {
		// If user not found, return just the account (maintaining old behavior)
		if usrErr == customError.ErrUserNotFound {
			fmt.Println("===== User not found for account ID:", account.ID.Hex(), "=====")
			response.Success(w, http.StatusOK, account, "Login successful")
			return
		}
		fmt.Println("===== Error retrieving user profile:", usrErr.Error(), "=====")
		response.InternalServerError(w, customError.ErrProfileRetrieval.Error()+": "+usrErr.Error())
		return
	}

	fmt.Println("===== Login successful for username:", authReq.Username, "=====")
	response.Success(w, http.StatusOK, userWithAccount, "Login successful")
}

// register godoc
// @Summary User registration
// @Description Register a new user account with roles
// @Tags Authentication
// @Accept json
// @Produce json
// @Param request body model.AccountRegister true "Registration details"
// @Success 200 {object} model.AccountResponse "Account successfully created"
// @Failure 400 {object} map[string]string "Bad request"
// @Failure 409 {object} map[string]string "Username already exists"
// @Failure 500 {object} map[string]string "Internal server error"
// @Router /auth/register [post]
func (ar *AuthRouter) register(w http.ResponseWriter, r *http.Request) {
	var authRegis model.AccountRegister
	err := json.NewDecoder(r.Body).Decode(&authRegis)

	if err != nil {
		response.BadRequest(w, customError.ErrInvalidRequestFormat.Error()+": "+err.Error())
		return
	}

	// Validate required fields
	if authRegis.Username == "" || authRegis.Password == "" || authRegis.Email == "" {
		response.BadRequest(w, customError.ErrMissingRegistrationDetails.Error())
		return
	}

	// Register the account
	accountResponse, err := ar.authService.Register(authRegis.Username, authRegis.Password, authRegis.Email, authRegis.Roles)
	if err != nil {
		if err == customError.ErrDuplicateUsername || err == customError.ErrDuplicateEmail {
			response.Conflict(w, customError.ErrDuplicateUsernameOrEmail.Error())
			return
		}
		response.InternalServerError(w, customError.ErrRegistrationFailed.Error()+": "+err.Error())
		return
	}

	response.Created(w, accountResponse, "Account registered successfully")
}
