package router

import (
	"encoding/json"
	"main/model"
	"main/service"
	"net/http"

	"github.com/go-chi/chi/v5"
)

type UserRouter struct {
	UserService *service.UserService
}

func NewUserRouter() *UserRouter {
	return &UserRouter{
		UserService: service.NewUserService(),
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
	user, err := ur.UserService.GetUserByID(uid, false)
	if err != nil {
		w.WriteHeader(http.StatusInternalServerError)
		w.Write([]byte(err.Error()))
		return
	}
	w.WriteHeader(http.StatusOK)
	json.NewEncoder(w).Encode(user)
}

func (ur *UserRouter) newUser(w http.ResponseWriter, r *http.Request) {
	var user model.UserRequest
	err := json.NewDecoder(r.Body).Decode(&user)
	if err != nil {
		w.WriteHeader(http.StatusInternalServerError)
		w.Write([]byte(err.Error()))
	}
	urs, err := ur.UserService.NewUser(&user, user.AccountId)
	if err != nil {
		w.WriteHeader(http.StatusInternalServerError)
		w.Write([]byte(err.Error()))
	}
	w.WriteHeader(http.StatusOK)
	json.NewEncoder(w).Encode(urs)
}
