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

type ProjectRouter struct {
	projectService *service.ProjectService
}

func NewProjectRouter() *ProjectRouter {
	return &ProjectRouter{
		projectService: service.NewProjectService(),
	}
}

func (pr ProjectRouter) Routes() chi.Router {
	r := chi.NewRouter()
	
	// Routes accessible to all authenticated users
	r.Get("/", pr.getAllProjects)
	r.Get("/{id}", pr.getProjectById)
	
	// Routes that require project manager privileges
	r.Group(func(r chi.Router) {
		r.Use(middleware.RequireRole("project_manager"))
		r.Post("/", pr.createProject)
	})
	
	return r
}

// getAllProjects godoc
// @Summary Get all projects
// @Description Retrieve all projects
// @Tags Projects
// @Accept json
// @Produce json
// @Success 200 {object} response.Response{data=[]model.Project} "Projects retrieved successfully"
// @Failure 500 {object} response.Response "Failed to retrieve projects"
// @Router /projects [get]
func (pr *ProjectRouter) getAllProjects(w http.ResponseWriter, r *http.Request) {
	projects, err := pr.projectService.GetProjects()

	if err != nil {
		response.InternalServerError(w, "Failed to retrieve projects: "+err.Error())
		return
	}

	response.Success(w, http.StatusOK, projects, "Projects retrieved successfully")
}

// getProjectById godoc
// @Summary Get project by ID
// @Description Retrieve a project by its ID
// @Tags Projects
// @Accept json
// @Produce json
// @Param id path string true "Project ID"
// @Success 200 {object} response.Response{data=model.Project} "Project retrieved successfully"
// @Failure 404 {object} response.Response "Project not found"
// @Router /projects/{id} [get]
func (pr *ProjectRouter) getProjectById(w http.ResponseWriter, r *http.Request) {
	project, err := pr.projectService.GetProjectById(chi.URLParam(r, "id"))

	if err != nil {
		response.NotFound(w, "Project not found")
		return
	}

	response.Success(w, http.StatusOK, project, "Project retrieved successfully")
}

// createProject godoc
// @Summary Create a new project
// @Description Create a new project
// @Tags Projects
// @Accept json
// @Produce json
// @Param request body model.Project true "Project creation request"
// @Success 201 {object} response.Response{data=model.Project} "Project created successfully"
// @Failure 400 {object} response.Response "Invalid request format"
// @Failure 500 {object} response.Response "Failed to create project"
// @Router /projects [post]
func (pr *ProjectRouter) createProject(w http.ResponseWriter, r *http.Request) {
	var inputProject model.Project

	err := json.NewDecoder(r.Body).Decode(&inputProject)

	if err != nil {
		response.BadRequest(w, "Invalid request format: "+err.Error())
		return
	}

	rs, err := pr.projectService.CreateProject(&inputProject)

	if err != nil {
		response.InternalServerError(w, "Failed to create project: "+err.Error())
		return
	}

	response.Created(w, rs, "Project created successfully")
}
