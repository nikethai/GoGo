package router

import (
	"encoding/json"
	"main/internal/model"
	"main/internal/server/response"
	"main/internal/service"
	"net/http"

	"github.com/go-chi/chi/v5"
)

type QuestionRouter struct {
	questionService *service.QuestionService
}

// init question router
// init collection
func NewQRouter() *QuestionRouter {
	return &QuestionRouter{
		questionService: service.NewQuestionService(),
	}
}

func (qr QuestionRouter) Routes() chi.Router {
	r := chi.NewRouter()
	r.Post("/", qr.setQuestionMongo)
	r.Get("/", qr.getAllQuestions)
	return r
}

// setQuestionMongo godoc
// @Summary Create a new question
// @Description Create a new question in MongoDB
// @Tags Questions
// @Accept json
// @Produce json
// @Param request body model.Question true "Question creation request"
// @Success 201 {object} response.Response{data=model.Question} "Question created successfully"
// @Failure 400 {object} response.Response "Invalid request format"
// @Failure 500 {object} response.Response "Failed to create question"
// @Router /questions [post]
func (qr *QuestionRouter) setQuestionMongo(w http.ResponseWriter, r *http.Request) {
	var inputQuestion model.Question

	err := json.NewDecoder(r.Body).Decode(&inputQuestion)

	if err != nil {
		response.BadRequest(w, "Invalid request format: "+err.Error())
		return
	}

	rs, err := qr.questionService.CreateQuestion(&inputQuestion)

	if err != nil {
		response.InternalServerError(w, "Failed to create question: "+err.Error())
		return
	}

	response.Created(w, rs, "Question created successfully")
}

// getAllQuestions godoc
// @Summary Get all questions
// @Description Retrieve all questions
// @Tags Questions
// @Accept json
// @Produce json
// @Success 200 {object} response.Response{data=[]model.Question} "Questions retrieved successfully"
// @Failure 500 {object} response.Response "Failed to retrieve questions"
// @Router /questions [get]
func (qr *QuestionRouter) getAllQuestions(w http.ResponseWriter, r *http.Request) {
	questions, err := qr.questionService.GetAllQuestions()

	if err != nil {
		response.InternalServerError(w, "Failed to retrieve questions: "+err.Error())
		return
	}

	response.Success(w, http.StatusOK, questions, "Questions retrieved successfully")
}
