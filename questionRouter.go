package main

import (
	"context"
	"encoding/json"
	"net/http"

	"github.com/go-chi/chi/v5"
	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/mongo"
)

type QuestionRouter struct{}

func (qr QuestionRouter) Routes() chi.Router {
	r := chi.NewRouter()
	r.Post("/", qr.setQuestionMongo)
	r.Get("/", qr.getQuestionMongo)
	return r
}

func (qr *QuestionRouter) getCollection() *mongo.Collection {
	return mongoDatabase.Collection("questions")
}

func (qr *QuestionRouter) setQuestionMongo(w http.ResponseWriter, r *http.Request) {
	mongoColl := qr.getCollection()

	var inputQuestion Question

	err := json.NewDecoder(r.Body).Decode(&inputQuestion)

	if err != nil {
		w.WriteHeader(http.StatusInternalServerError)
		w.Write([]byte(err.Error()))
	}

	// question := Question{
	// 	Content:     "What is your name?",
	// 	Description: "",
	// 	Type:        "text",
	// 	Trait: bson.M{
	// 		"options": []string{"John", "Jane", "Joe"}, // Copilot is so gud
	// 	},
	// }
	// question := Question{
	// 	Content:     "Hãy đánh giá ảnh hưởng của dự án",
	// 	Description: "",
	// 	Type:        "text",
	// 	Trait: bson.M{
	// 		"col": []string{"Ảnh hưởng tới gia đình", "Ảnh hưởng tới sức khoẻ"},
	// 		"row": []string{"Nhiều", "Trung bình", "Ít"},
	// 	},
	// }

	rs, err := mongoColl.InsertOne(context.TODO(), inputQuestion)

	if err != nil {
		w.WriteHeader(http.StatusInternalServerError)
		w.Write([]byte(err.Error()))
	}

	w.WriteHeader(http.StatusOK)
	json.NewEncoder(w).Encode(rs)
}

func (qr *QuestionRouter) getQuestionMongo(w http.ResponseWriter, r *http.Request) {
	mongoColl := qr.getCollection()

	var questions []Question
	cur, err := mongoColl.Find(context.TODO(), bson.M{})
	if err != nil {
		w.WriteHeader(http.StatusInternalServerError)
		w.Write([]byte(err.Error()))
	}
	defer cur.Close(context.TODO())
	for cur.Next(context.TODO()) {
		var question Question
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
