package router

import (
	"encoding/json"
	"main/internal/middleware"
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
	
	// Public routes
	r.Get("/{roleId}", ar.getRole)
	
	// Admin-only routes
	r.Group(func(r chi.Router) {
		r.Use(middleware.RequireRole("admin"))
		r.Post("/", ar.newRole)
	})
	
	return r
}

// getRole godoc
// @Summary Get a role by ID
// @Description Retrieve a role by its ID
// @Tags Roles
// @Accept json
// @Produce json
// @Param roleId path string true "Role ID"
// @Success 200 {object} response.Response{data=model.Role} "Role retrieved successfully"
// @Failure 404 {object} response.Response "Role not found"
// @Router /roles/{roleId} [get]
func (ar *RoleRouter) getRole(w http.ResponseWriter, r *http.Request) {
	roleReq := chi.URLParam(r, "roleId")
	role, err := ar.roleService.GetRole(roleReq)
	if err != nil {
		response.NotFound(w, "Role not found")
		return
	}
	response.Success(w, http.StatusOK, role, "Role retrieved successfully")
}

// newRole godoc
// @Summary Create a new role
// @Description Create a new role
// @Tags Roles
// @Accept json
// @Produce json
// @Param request body model.Role true "Role creation request"
// @Success 201 {object} response.Response{data=model.Role} "Role created successfully"
// @Failure 400 {object} response.Response "Invalid request format"
// @Failure 500 {object} response.Response "Failed to create role"
// @Router /roles [post]
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
