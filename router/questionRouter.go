package router

import (
	"context"
	"encoding/json"
	"main/db"
	"main/model"
	"net/http"

	"github.com/go-chi/chi/v5"
	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/mongo"
)

type QuestionRouter struct {
	questionCollection *mongo.Collection
}

// init question router
// init collection
func NewQRouter() *QuestionRouter {
	return &QuestionRouter{
		questionCollection: db.MongoDatabase.Collection("question"),
	}
}

func (qr QuestionRouter) Routes() chi.Router {
	r := chi.NewRouter()
	r.Post("/", qr.setQuestionMongo)
	r.Get("/", qr.getQuestionMongo)
	return r
}

func (qr *QuestionRouter) setQuestionMongo(w http.ResponseWriter, r *http.Request) {
	quesCol := qr.questionCollection

	var inputQuestion model.Question

	err := json.NewDecoder(r.Body).Decode(&inputQuestion)

	if err != nil {
		w.WriteHeader(http.StatusInternalServerError)
		w.Write([]byte(err.Error()))
	}

	rs, err := quesCol.InsertOne(context.TODO(), inputQuestion)

	if err != nil {
		w.WriteHeader(http.StatusInternalServerError)
		w.Write([]byte(err.Error()))
	}

	w.WriteHeader(http.StatusOK)
	json.NewEncoder(w).Encode(rs)
}

func (qr *QuestionRouter) getQuestionMongo(w http.ResponseWriter, r *http.Request) {
	quesCol := qr.questionCollection

	var questions []model.Question
	cur, err := quesCol.Find(context.TODO(), bson.M{})
	if err != nil {
		w.WriteHeader(http.StatusInternalServerError)
		w.Write([]byte(err.Error()))
	}
	defer cur.Close(context.TODO())
	for cur.Next(context.TODO()) {
		var question model.Question
		err := cur.Decode(&question)
		if err != nil {
			w.WriteHeader(http.StatusInternalServerError)
			w.Write([]byte(err.Error()))
		}
		questions = append(questions, question)
	}

	w.WriteHeader(http.StatusOK)
	json.NewEncoder(w).Encode(questions)
}
