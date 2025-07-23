package main

import (
	"context"
	"log"
	"main/db"
	customMiddleware "main/internal/middleware"
	"main/internal/router"
	"net/http"

	_ "main/docs" // Import generated docs

	"github.com/go-chi/chi/v5"
	"github.com/go-chi/chi/v5/middleware"
	"github.com/go-chi/cors"
	httpSwagger "github.com/swaggo/http-swagger"
)

// @title Gogo API
// @version 1.0
// @description API Server for Gogo application with user management, projects, questions, and forms
// @termsOfService http://swagger.io/terms/

// @contact.name API Support
// @contact.url http://www.swagger.io/support
// @contact.email support@swagger.io

// @license.name MIT
// @license.url https://opensource.org/licenses/MIT

// @host localhost:3001
// @BasePath /
// @schemes http https

// @securityDefinitions.apikey BearerAuth
// @in header
// @name Authorization
// @description Type "Bearer" followed by a space and JWT token.

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

	// Swagger documentation
	r.Get("/swagger/*", httpSwagger.Handler(
		httpSwagger.URL("http://localhost:3001/swagger/doc.json"),
	))

	r.Get("/", func(w http.ResponseWriter, r *http.Request) {
		w.Write([]byte("welcome"))
	})
	// Public routes (no authentication required)
	r.Mount("/auth", authRouter.SetupRoutes())

	// Create a router group with hybrid authentication (supports both JWT and Azure AD)
	protectedRouter := chi.NewRouter()
	protectedRouter.Use(customMiddleware.HybridAuth)

	// Protected routes (authentication required)
	protectedRouter.Mount("/questions", qRouter.Routes())
	protectedRouter.Mount("/roles", roleRouter.Routes())
	protectedRouter.Mount("/users", userRouter.Routes())
	protectedRouter.Mount("/projects", projectRouter.Routes())

	// Mount the protected router
	r.Mount("/api", protectedRouter)

	log.Println("Server starting on :3001")
	log.Println("Swagger docs available at: http://localhost:3001/swagger/")
	http.ListenAndServe(":3001", r)

	// use when about to end the app
	defer func() {
		if err := db.MongoClient.Disconnect(context.TODO()); err != nil {
			log.Fatal(err)
		}
	}()
}
