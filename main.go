package main

import (
	"context"
	"log"
	"net/http"

	"github.com/go-chi/chi/v5"
	"github.com/go-chi/chi/v5/middleware"
)

func main() {
	mongoClient = getMongoEnv()
	mongoDatabase = mongoClient.Database("surveyDB")

	r := chi.NewRouter()

	r.Use(middleware.Logger)
	r.Use(middleware.Recoverer)
	r.Use(middleware.CleanPath)
	r.Use(middleware.SetHeader("Content-Type", "application/json"))

	r.Get("/", func(w http.ResponseWriter, r *http.Request) {
		w.Write([]byte("welcome"))
	})
	r.Mount("/news", NewsRouter{}.Routes())
	r.Mount("/questions", QuestionRouter{}.Routes())
	http.ListenAndServe(":3001", r)

	// use when about to end the app
	defer func() {
		if err := mongoClient.Disconnect(context.TODO()); err != nil {
			log.Fatal(err)
		}
	}()

}
