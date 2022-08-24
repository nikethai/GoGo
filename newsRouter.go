package main

import (
	"context"
	"encoding/json"
	"net/http"

	"github.com/go-chi/chi/v5"
	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/mongo"
)

// Why the fuck need this?
// Isn't this shit a class?
type NewsRouter struct {
}

// Router of news
func (nr NewsRouter) Routes() chi.Router {
	r := chi.NewRouter()
	r.Get("/", nr.getNews)
	r.Post("/", nr.setNewsMongo)
	r.Get("/mongo", nr.getNewsMongo)
	return r
}

func (nr NewsRouter) getNews(w http.ResponseWriter, r *http.Request) {
	// dateParam := chi.URLParam(r, "date")
	// slugParam := chi.URLParam(r, "slug")

	news := News{
		Date:    "2020-01-01",
		Id:      "1",
		Title:   "Hello World",
		Content: "",
	}

	w.WriteHeader(http.StatusOK)
	json.NewEncoder(w).Encode(news)
}

func (nr *NewsRouter) setNewsMongo(w http.ResponseWriter, r *http.Request) {
	mongoColl := mongoDatabase.Collection("news")

	news := News{
		Date:  "2020-01-01",
		Id:    "1",
		Title: "Hello World",
	}

	rs, err := mongoColl.InsertOne(context.TODO(), news)

	if err != nil {
		w.WriteHeader(http.StatusInternalServerError)
		w.Write([]byte(err.Error()))
	}

	w.WriteHeader(http.StatusOK)
	json.NewEncoder(w).Encode(rs)
}

func (nr *NewsRouter) getNewsMongo(w http.ResponseWriter, r *http.Request) {
	mongoColl := mongoDatabase.Collection("news")

	respCursor, err := mongoColl.Find(context.TODO(), bson.D{{Key: "id", Value: "1"}})

	if err != nil {
		w.WriteHeader(http.StatusInternalServerError)
		w.Write([]byte(err.Error()))
	}

	if err == mongo.ErrNoDocuments {
		w.WriteHeader(http.StatusNotFound)
		w.Write([]byte("No news found"))
	}

	var news []News

	for respCursor.TryNext(context.TODO()) {
		var new News
		respCursor.Decode(&new)
		news = append(news, new)
	}

	if err := respCursor.Err(); err != nil {
		w.WriteHeader(http.StatusInternalServerError)
		w.Write([]byte(err.Error()))
	}

	w.WriteHeader(http.StatusOK)
	json.NewEncoder(w).Encode(news)
}
