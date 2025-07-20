package router

import (
	"encoding/json"
	"main/internal/model"
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
		w.WriteHeader(http.StatusInternalServerError)
		w.Write([]byte(err.Error()))
	}

	w.WriteHeader(http.StatusOK)
	json.NewEncoder(w).Encode(projects)
}

func (pr *ProjectRouter) getProjectById(w http.ResponseWriter, r *http.Request) {
	projects, err := pr.projectService.GetProjectById(chi.URLParam(r, "id"))

	if err != nil {
		w.WriteHeader(http.StatusInternalServerError)
		w.Write([]byte(err.Error()))
	}

	w.WriteHeader(http.StatusOK)
	json.NewEncoder(w).Encode(projects)
}

func (pr *ProjectRouter) createProject(w http.ResponseWriter, r *http.Request) {
	var inputProject model.Project

	err := json.NewDecoder(r.Body).Decode(&inputProject)

	if err != nil {
		w.WriteHeader(http.StatusInternalServerError)
		w.Write([]byte(err.Error()))
	}

	rs, err := pr.projectService.CreateProject(&inputProject)

	if err != nil {
		w.WriteHeader(http.StatusInternalServerError)
		w.Write([]byte(err.Error()))
	}

	w.WriteHeader(http.StatusOK)
	json.NewEncoder(w).Encode(rs)
}
