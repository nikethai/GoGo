# Project Structure

This document details the organizational structure, architectural patterns, and module relationships of the Gogo project. Gogo is a survey and form management system built with Go and MongoDB, designed for scalability, maintainability, and clear separation of concerns.

## Overview

Gogo adheres to a **Clean Architecture** pattern, emphasizing domain-driven design principles. This approach ensures a highly modular, testable, and maintainable codebase by separating business logic from external concerns like databases and UI. The project leverages Go 1.24, making extensive use of its generic type support for robust and type-safe operations, particularly within the repository layer.

## Directory Structure

The project's root directory contains several key folders and files, organized to reflect its architectural layers and functional domains:

```
├── .docs/                  # Project documentation (this folder)
├── .gitattributes          # Git attributes configuration
├── .gitignore              # Git ignore rules
├── .promptx/               # Trae AI specific configuration and memory
├── .trae/                  # Trae IDE specific rules and configurations
├── .vercel/               # Vercel deployment configuration
├── .vscode/                # VS Code editor configuration
├── Makefile                # Project build and utility commands
├── README.md               # Main project README file
├── README_AZURE_AD.md      # Documentation for Azure AD integration
├── cmd/                    # Command-line applications
│   └── api/                # Entry point for the main API server
├── db/                     # Database connection and low-level utilities
│   ├── builder/            # MongoDB query builder utilities
│   │   └── builder.go      # Generic query building functions
│   └── mongo.go            # MongoDB connection setup and client initialization
├── docs/                   # Additional project documentation (e.g., Swagger, design docs)
│   ├── AZURE_AD_JWT.md     # Detailed Azure AD JWT authentication documentation
│   ├── GENERIC_REPOSITORY.md # Documentation on the generic repository pattern
│   ├── docs.go             # Go file for embedding documentation (e.g., Swagger UI)
│   ├── swagger.json        # OpenAPI/Swagger specification (JSON)
│   └── swagger.yaml        # OpenAPI/Swagger specification (YAML)
├── features/               # Feature-specific documentation and design (e.g., requirements, design, tasks)
│   └── azure-ad-authentication-strategy/
│       ├── design.md       # Design document for Azure AD authentication
│       ├── requirements.md # Requirements for Azure AD authentication
│       └── tasks.md        # Task breakdown for Azure AD authentication implementation
├── go.mod                  # Go module definition and direct dependencies
├── go.sum                  # Go module checksums for reproducible builds
├── internal/               # Internal application logic (not exposed externally)
│   ├── app/                # Core application setup and initialization
│   ├── config/             # Application configuration structures
│   │   └── collections.go  # MongoDB collection configurations
│   ├── domain/             # Core business entities and interfaces (domain layer)
│   │   ├── account/        # Account domain models and logic
│   │   ├── form/           # Form domain models and logic
│   │   ├── project/        # Project domain models and logic
│   │   ├── question/       # Question domain models and logic
│   │   ├── role/           # Role domain models and logic
│   │   └── user/           # User domain models and logic
│   ├── error/              # Custom error definitions and handling utilities
│   │   └── error.go        # Centralized error types
│   ├── middleware/         # HTTP middleware functions
│   │   ├── auth.go         # General authentication middleware
│   │   ├── azure_auth.go   # Azure AD specific authentication middleware
│   │   └── enhanced_azure_auth.go # Enhanced Azure AD authentication middleware
│   ├── model/              # Data Transfer Objects (DTOs) and database models
│   │   ├── accountModel.go # Account data model
│   │   ├── baseModel.go    # Base model for common fields (e.g., timestamps)
│   │   ├── formModel.go    # Form data model
│   │   ├── projectModel.go # Project data model
│   │   ├── questionModel.go# Question data model
│   │   ├── roleModel.go    # Role data model
│   │   └── userModel.go    # User data model
│   ├── repository/         # Data access interfaces and implementations
│   │   ├── mongo/          # MongoDB specific repository implementations
│   │   └── repository.go   # Generic repository interfaces
│   ├── router/             # HTTP route definitions and handlers
│   │   ├── authRouter.go   # Authentication routes
│   │   ├── projectRouter.go# Project management routes
│   │   ├── questionRouter.go # Question management routes
│   │   ├── roleRouter.go   # Role management routes
│   │   └── userRouter.go   # User management routes
│   ├── server/             # HTTP server setup and request/response handling
│   │   ├── handler/        # Request handlers (controllers)
│   │   ├── request/        # Request payload structures
│   │   └── response/       # Response payload structures
│   └── service/            # Business logic implementations (application layer)
│       ├── authService.go  # Authentication business logic
│       ├── projectService.go # Project management business logic
│       ├── questionService.go# Question management business logic
│       ├── roleService.go  # Role management business logic
│       └── userService.go  # User management business logic
├── main                    # Compiled executable (after build)
├── main.go                 # Main application entry point (source file)
├── pkg/                    # Publicly reusable packages (external to internal logic)
│   ├── auth/               # Authentication related utilities and services
│   │   ├── azure_ad.go     # Azure AD specific authentication logic
│   │   ├── azure_service.go# Azure AD service integration
│   │   ├── config.go       # Authentication configuration
│   │   ├── jwt.go          # JWT token handling
│   │   ├── oauth2.go       # OAuth2 related utilities
│   │   ├── session_manager.go # Session management logic
│   │   └── token_cache.go  # Token caching for authentication
│   ├── database/           # Database related utilities (e.g., MongoDB client setup)
│   │   └── mongodb/        # MongoDB specific package
│   ├── errors/             # Common error handling utilities
│   ├── logger/             # Centralized logging utilities
│   └── validator/          # Data validation utilities
└── scripts/                # Shell scripts for build, test, and deployment automation
    ├── build.sh            # Build script
    ├── test_auth.sh        # Authentication testing script
    └── test_azure_auth.sh  # Azure AD authentication testing script
```

## Architectural Layers and Components

Gogo's architecture is structured into distinct layers, promoting modularity, testability, and maintainability:

1.  **`cmd/api` (Entry Point/Application Layer)**
    -   The `cmd/api` directory contains the main entry point for the API server. It is responsible for initializing the application, setting up the HTTP server, configuring middleware, and orchestrating the dependencies.

2.  **`internal/` (Core Application Logic)**
    -   This directory encapsulates the core business logic and application-specific components that are not intended for external consumption. It strictly adheres to the Clean Architecture principles:
        -   **`internal/domain/`**: Defines the core business entities, value objects, and interfaces. This is the heart of the application's business rules, independent of any frameworks or databases.
        -   **`internal/service/`**: Implements the application's use cases and business logic. Services orchestrate interactions between domain entities and repositories, enforcing business rules.
        -   **`internal/repository/`**: Defines interfaces for data persistence (e.g., `UserRepository`). The `internal/repository/mongo/` subdirectory provides the concrete MongoDB implementations of these interfaces.
        -   **`internal/router/`**: Contains the HTTP route definitions and handlers (controllers). These handlers receive requests, call the appropriate services, and return responses. They are part of the presentation layer.
        -   **`internal/model/`**: Houses Data Transfer Objects (DTOs) used for request/response payloads and the structures that map directly to database documents.
        -   **`internal/middleware/`**: Contains HTTP middleware functions for cross-cutting concerns like authentication, logging, and error handling.
        -   **`internal/config/`**: Manages application configurations.
        -   **`internal/error/`**: Defines custom error types for consistent error handling across the application.

3.  **`pkg/` (Publicly Reusable Packages)**
    -   This directory contains packages that are generally reusable across different projects or parts of a larger monorepo. They are external-facing and provide common utilities or integrations.
        -   **`pkg/auth/`**: Authentication-related utilities, including JWT handling, OAuth2, and Azure AD specific logic.
        -   **`pkg/database/mongodb/`**: Generic MongoDB client setup and connection management.
        -   **`pkg/logger/`**: Centralized logging utility.
        -   **`pkg/validator/`**: General-purpose data validation utilities.
        -   **`pkg/errors/`**: Common error handling patterns and utilities.

4.  **`db/` (Low-Level Database Utilities)**
    -   Contains low-level database connection setup and specialized query builders that might be specific to the chosen database technology (MongoDB in this case).

5.  **`docs/` (Supplementary Documentation)**
    -   Holds additional technical documentation, API specifications (Swagger/OpenAPI), and detailed design documents that are not part of the core `.docs` folder.

6.  **`features/` (Feature-Specific Documentation)**
    -   Dedicated folders for each major feature, containing its `requirements.md`, `design.md`, and `tasks.md` documents. This ensures that feature development is well-documented from conception to implementation.

## Key Architectural Patterns

-   **Clean Architecture**: Enforces separation of concerns, making the domain layer independent of external frameworks and databases. This promotes testability and maintainability.
-   **Domain-Driven Design (DDD)**: Focuses on modeling the core business domain, ensuring that the software reflects the real-world concepts and logic.
-   **Repository Pattern**: Provides an abstraction layer over data persistence. The `internal/repository` interfaces define contracts for data access, while `internal/repository/mongo` provides the concrete implementations.
-   **Service Layer Pattern**: Encapsulates business logic and orchestrates operations between different components. Services are responsible for enforcing business rules and managing transactions.
-   **Dependency Injection**: Used throughout the application to manage dependencies, making components loosely coupled and easier to test.
-   **API-First Design**: The application is designed with its API as the primary interface, ensuring consistency and ease of integration for various clients.

## Module Relationships and Data Flow

-   **`cmd/api`** initializes the HTTP server and injects dependencies into the `internal/router` handlers.
-   **`internal/router`** receives HTTP requests, validates input using `pkg/validator`, and calls methods on `internal/service`.
-   **`internal/service`** contains the core business logic. It interacts with `internal/repository` interfaces to perform data operations and uses `internal/domain` entities.
-   **`internal/repository/mongo`** implements the `internal/repository` interfaces, using the `db/mongo.go` client and `db/builder` for MongoDB interactions. It maps `internal/model` DTOs to database documents.
-   **`pkg/auth`** provides authentication mechanisms used by `internal/middleware/auth.go`.
-   **`pkg/logger`** is used across all layers for consistent logging.
-   **`internal/error`** defines custom errors that are propagated and handled consistently throughout the application, often transformed into appropriate HTTP responses by `internal/router`.

## Naming Conventions

-   **Directories**: Use `kebab-case` for feature folders (e.g., `azure-ad-authentication-strategy`) and `snake_case` or `lowercase` for internal directories (e.g., `internal/domain/account`).
-   **Files**: Use `camelCase` with descriptive suffixes (e.g., `accountModel.go`, `authService.go`).
-   **Packages**: Use `lowercase`, single-word names (e.g., `model`, `service`, `router`).
-   **Types/Structs**: Use `PascalCase` (e.g., `User`, `ProjectService`).
-   **Methods**: Use `camelCase` (e.g., `getProjectByID`, `createUser`).
-   **Variables**: Use `camelCase` (e.g., `userService`, `projectCollection`).

## Configuration Management

-   Environment variables are loaded from `.env` files using `godotenv` (from `pkg/auth/config.go` or similar).
-   Sensitive configurations like MongoDB connection strings are managed via environment variables.
-   Application settings (e.g., port, timeouts) are configured in `internal/config`.

## Dependencies

The project manages its dependencies using Go Modules (`go.mod` and `go.sum`). Key direct dependencies include:

-   **`github.com/go-chi/chi/v5`**: Lightweight, idiomatic HTTP router.
-   **`github.com/go-chi/cors`**: CORS middleware for HTTP requests.
-   **`github.com/google/uuid`**: For generating UUIDs.
-   **`github.com/joho/godotenv`**: For loading environment variables from `.env` files.
-   **`go.mongodb.org/mongo-driver`**: Official MongoDB driver for Go.

Indirect dependencies are managed by `go.sum` to ensure reproducible builds.