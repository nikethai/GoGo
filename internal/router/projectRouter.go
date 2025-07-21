package router

import (
	"encoding/json"
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
	r.Post("/", pr.createProject)
	r.Get("/", pr.getAllProjects)
	r.Get("/{id}", pr.getProjectById)
	return r
}

func (pr *ProjectRouter) getAllProjects(w http.ResponseWriter, r *http.Request) {
	projects, err := pr.projectService.GetProjects()

	if err != nil {
		response.InternalServerError(w, "Failed to retrieve projects: "+err.Error())
		return
	}

	response.Success(w, http.StatusOK, projects, "Projects retrieved successfully")
}

func (pr *ProjectRouter) getProjectById(w http.ResponseWriter, r *http.Request) {
	project, err := pr.projectService.GetProjectById(chi.URLParam(r, "id"))

	if err != nil {
		response.NotFound(w, "Project not found")
		return
	}

	response.Success(w, http.StatusOK, project, "Project retrieved successfully")
}

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
