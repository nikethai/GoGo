package main

import (
	"context"
	"log"
	"main/db"
	customMiddleware "main/internal/middleware"
	"main/internal/router"
	profileRouter "main/internal/profile/router"
	"main/pkg/auth"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

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
// @BasePath /api/v1
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

	// Initialize auth router with conditional Azure AD support
	var authRouter *router.AuthRouter

	// Check if Azure AD is configured
	if os.Getenv("AZURE_AD_TENANT_ID") != "" && os.Getenv("AZURE_AD_CLIENT_ID") != "" {
		// Initialize Azure AD services
		azureService, err := auth.NewAzureADService()
		if err != nil {
			log.Printf("Failed to initialize Azure AD service: %v", err)
			authRouter = router.NewAuthRouter()
		} else {
			// Initialize Azure AD components
			sessionConfig := &auth.SessionConfig{
				DefaultTTL:         24 * time.Hour,
				MaxTTL:             7 * 24 * time.Hour,
				CleanupInterval:    time.Hour,
				MaxSessionsPerUser: 5,
				SecureCookies:      true,
				SameSite:           "Strict",
			}
			sessionManager := auth.NewSessionManager(sessionConfig)

			tokenCacheConfig := &auth.TokenCacheConfig{
				DefaultTTL:       time.Hour,
				MaxTTL:           24 * time.Hour,
				CleanupInterval:  15 * time.Minute,
				MaxCacheSize:     1000,
				EncryptTokens:    true,
				CompressionLevel: 6,
				PersistToDisk:    false,
				CacheFilePath:    "/tmp/token_cache.json",
			}
			tokenCache, err := auth.NewTokenCache(tokenCacheConfig)
			if err != nil {
				log.Fatalf("Failed to initialize token cache: %v", err)
			}
			oauth2Config := auth.GetOAuth2Config()

			if oauth2Config != nil {
				authRouter = router.NewAuthRouterWithAzure(azureService, sessionManager, tokenCache, oauth2Config)
				log.Println("Azure AD authentication enabled")
			} else {
				log.Println("Azure AD OAuth2 config not found, using regular auth router")
				authRouter = router.NewAuthRouter()
			}
		}
	} else {
		log.Println("Azure AD not configured, using regular authentication only")
		authRouter = router.NewAuthRouter()
	}

	roleRouter := router.NewRoleRouter()
	userRouter := router.NewUserRouter()
	projectRouter := router.NewProjectRouter()
	profileRouterInstance := profileRouter.NewProfileRouter()

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
	// Group all API routes under /api/v1
	r.Route("/api/v1", func(r chi.Router) {
		// Swagger documentation
		r.Get("/swagger/*", httpSwagger.Handler(
			httpSwagger.URL("http://localhost:3001/api/v1/swagger/doc.json"),
		))

		// Welcome route
		r.Get("/", func(w http.ResponseWriter, r *http.Request) {
			w.Write([]byte("welcome to Gogo API v1"))
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
		protectedRouter.Mount("/profile", profileRouterInstance.Routes())

		// Mount the protected router
		r.Mount("/", protectedRouter)
	})

	// Create HTTP server
	server := &http.Server{
		Addr:    ":3001",
		Handler: r,
	}

	// Channel to listen for interrupt signal
	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)

	// Start server in a goroutine
	go func() {
		log.Println("Server starting on :3001")
		log.Println("Swagger docs available at: http://localhost:3001/swagger/")
		if err := server.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			log.Fatalf("Server failed to start: %v", err)
		}
	}()

	// Wait for interrupt signal
	<-quit
	log.Println("Shutting down server...")

	// Create a context with timeout for graceful shutdown
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	// Shutdown server gracefully
	if err := server.Shutdown(ctx); err != nil {
		log.Printf("Server forced to shutdown: %v", err)
	}

	// Disconnect from MongoDB
	if err := db.MongoClient.Disconnect(context.TODO()); err != nil {
		log.Printf("Error disconnecting from MongoDB: %v", err)
	} else {
		log.Println("Disconnected from MongoDB")
	}

	log.Println("Server shutdown complete")
}
