package main

import (
	"context"
	"log"
	"main/db"
	"main/router"
	"net/http"

	"github.com/go-chi/chi/v5"
	"github.com/go-chi/chi/v5/middleware"
	"github.com/go-chi/cors"
)

func main() {
	// look weird but haven't figured a better way yet
	db.InitConnection()

	r := chi.NewRouter()
	qRouter := router.NewQRouter()
	authRouter := router.NewAuthRouter()
	roleRouter := router.NewRoleRouter()
	userRouter := router.NewUserRouter()
	projectRouter := router.NewProjectRouter()

	r.Use(cors.Handler(cors.Options{
		AllowedOrigins:   []string{"http://localhost:*"},
		AllowedMethods:   []string{"GET", "POST", "PUT", "DELETE", "OPTIONS"},
		AllowedHeaders:   []string{"Accept", "Authorization", "Content-Type", "X-CSRF-Token"},
		ExposedHeaders:   []string{"Link"},
		AllowCredentials: false,
		MaxAge:           300, // Maximum value not ignored by any of major browsers
	}))
	r.Use(middleware.Logger)
	r.Use(middleware.Recoverer)
	r.Use(middleware.CleanPath)
	r.Use(middleware.SetHeader("Content-Type", "application/json"))

	r.Get("/", func(w http.ResponseWriter, r *http.Request) {
		w.Write([]byte("welcome"))
	})
	r.Mount("/questions", qRouter.Routes())
	r.Mount("/auth", authRouter.Routes())
	r.Mount("/roles", roleRouter.Routes())
	r.Mount("/users", userRouter.Routes())
	r.Mount("/projects", projectRouter.Routes())

	http.ListenAndServe(":3001", r)

	// use when about to end the app
	defer func() {
		if err := db.MongoClient.Disconnect(context.TODO()); err != nil {
			log.Fatal(err)
		}
	}()
}
