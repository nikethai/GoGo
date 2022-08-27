package main

import (
	"context"
	"log"
	"main/db"
	"main/router"
	"net/http"

	"github.com/go-chi/chi/v5"
	"github.com/go-chi/chi/v5/middleware"
)

func main() {
	// look weird but haven't figured a better way yet
	db.MongoClient = db.GetMongoEnv()
	db.MongoDatabase = db.MongoClient.Database("surveyDB")

	r := chi.NewRouter()
	qRouter := router.NewQRouter()
	authRouter := router.NewAuthRouter()

	r.Use(middleware.Logger)
	r.Use(middleware.Recoverer)
	r.Use(middleware.CleanPath)
	r.Use(middleware.SetHeader("Content-Type", "application/json"))

	r.Get("/", func(w http.ResponseWriter, r *http.Request) {
		w.Write([]byte("welcome"))
	})
	r.Mount("/questions", qRouter.Routes())
	r.Mount("/auth", authRouter.Routes())

	http.ListenAndServe(":3001", r)

	// use when about to end the app
	defer func() {
		if err := db.MongoClient.Disconnect(context.TODO()); err != nil {
			log.Fatal(err)
		}
	}()

}
