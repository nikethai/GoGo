package router

import (
	"encoding/json"
	"main/internal/model"
	"main/internal/server/response"
	"main/internal/service"
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
		response.NotFound(w, "Role not found")
		return
	}
	response.Success(w, http.StatusOK, role, "Role retrieved successfully")
}

func (ar *RoleRouter) newRole(w http.ResponseWriter, r *http.Request) {
	var role model.Role
	err := json.NewDecoder(r.Body).Decode(&role)
	if err != nil {
		response.BadRequest(w, "Invalid request format: "+err.Error())
		return
	}
	rs, err := ar.roleService.NewRole(role.Name)
	if err != nil {
		response.InternalServerError(w, "Failed to create role: "+err.Error())
		return
	}
	response.Created(w, rs, "Role created successfully")
}
