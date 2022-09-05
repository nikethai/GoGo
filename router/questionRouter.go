package router

import (
	"encoding/json"
	"main/model"
	"main/service"
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
		w.WriteHeader(http.StatusInternalServerError)
		w.Write([]byte(err.Error()))
	}

	rs, err := qr.questionService.CreateQuestion(&inputQuestion)

	if err != nil {
		w.WriteHeader(http.StatusInternalServerError)
		w.Write([]byte(err.Error()))
	}

	w.WriteHeader(http.StatusOK)
	json.NewEncoder(w).Encode(rs)
}

func (qr *QuestionRouter) getAllQuestions(w http.ResponseWriter, r *http.Request) {
	questions, err := qr.questionService.GetAllQuestions()

	if err != nil {
		w.WriteHeader(http.StatusInternalServerError)
		w.Write([]byte(err.Error()))
	}

	w.WriteHeader(http.StatusOK)
	json.NewEncoder(w).Encode(questions)
}
