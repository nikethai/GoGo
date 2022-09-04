package router

import (
	"encoding/json"
	"main/model"
	"main/service"
	"net/http"

	"github.com/go-chi/chi/v5"
)

type RoleRouter struct {
	roleService *service.RoleService
}

func NewRoleRouter() *RoleRouter {
	return &RoleRouter{
		roleService: service.NewRoleService(),
	}
}

func (ar *RoleRouter) Routes() chi.Router {
	r := chi.NewRouter()
	r.Post("/", ar.newRole)
	r.Get("/{roleId}", ar.getRole)
	return r
}

func (ar *RoleRouter) getRole(w http.ResponseWriter, r *http.Request) {
	roleReq := chi.URLParam(r, "roleId")
	role, err := ar.roleService.GetRole(roleReq)
	if err != nil {
		w.WriteHeader(http.StatusInternalServerError)
		w.Write([]byte(err.Error()))
		return
	}
	w.WriteHeader(http.StatusOK)
	json.NewEncoder(w).Encode(role)
}

func (ar *RoleRouter) newRole(w http.ResponseWriter, r *http.Request) {
	var role model.Role
	err := json.NewDecoder(r.Body).Decode(&role)
	if err != nil {
		w.WriteHeader(http.StatusInternalServerError)
		w.Write([]byte(err.Error()))
	}
	rs, err := ar.roleService.NewRole(role.Name)
	if err != nil {
		w.WriteHeader(http.StatusInternalServerError)
		w.Write([]byte(err.Error()))
	}
	w.WriteHeader(http.StatusOK)
	json.NewEncoder(w).Encode(rs)
}
