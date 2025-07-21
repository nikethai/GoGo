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

func (qr *QuestionRouter) getAllQuestions(w http.ResponseWriter, r *http.Request) {
	questions, err := qr.questionService.GetAllQuestions()

	if err != nil {
		response.InternalServerError(w, "Failed to retrieve questions: "+err.Error())
		return
	}

	response.Success(w, http.StatusOK, questions, "Questions retrieved successfully")
}
