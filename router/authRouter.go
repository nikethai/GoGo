package router

import (
	"encoding/json"
	"main/model"
	"main/service"
	"net/http"

	"github.com/go-chi/chi/v5"
)

type AuthRouter struct {
	authService *service.AuthService
}

func NewAuthRouter() *AuthRouter {
	return &AuthRouter{
		authService: service.NewAuthService(),
	}
}

func (ar *AuthRouter) Routes() chi.Router {
	r := chi.NewRouter()
	r.Post("/login", ar.Login)
	r.Post("/register", ar.Register)
	return r
}

func (ar *AuthRouter) Login(w http.ResponseWriter, r *http.Request) {
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
	w.WriteHeader(http.StatusOK)
	json.NewEncoder(w).Encode(account)
}

func (ar *AuthRouter) Register(w http.ResponseWriter, r *http.Request) {
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
