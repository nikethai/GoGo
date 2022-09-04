package router

import (
	"encoding/json"
	"main/model"
	"main/service"
	"net/http"

	"github.com/go-chi/chi/v5"
	"go.mongodb.org/mongo-driver/mongo"
)

type AuthRouter struct {
	authService *service.AuthService
	userService *service.UserService
}

func NewAuthRouter() *AuthRouter {
	return &AuthRouter{
		authService: service.NewAuthService(),
		userService: service.NewUserService(),
	}
}

func (ar *AuthRouter) Routes() chi.Router {
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
	user, usrErr := ar.userService.GetUserByID(account.ID.Hex(), true)

	if usrErr != nil {
		//TODO: incomplete information. This one should be an error
		if usrErr == mongo.ErrNoDocuments {
			w.WriteHeader(http.StatusOK)
			json.NewEncoder(w).Encode(account)
			return
		}
		w.WriteHeader(http.StatusInternalServerError)
		w.Write([]byte(usrErr.Error()))
		return
	}

	w.WriteHeader(http.StatusOK)
	json.NewEncoder(w).Encode(user)
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
