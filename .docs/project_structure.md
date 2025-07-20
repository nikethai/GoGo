# Project Structure

This document outlines the organizational structure and architectural patterns of the Gogo project - a survey and form management system built with Go and MongoDB.

## Overview

Gogo follows a clean architecture pattern with clear separation of concerns, implementing domain-driven design principles for a scalable and maintainable codebase. The project uses Go 1.24 with full generic type support for type-safe repository operations.

## Directory Structure

```
├── .env                  # Environment variables configuration
├── .gitattributes        # Git attributes configuration
├── .gitignore            # Git ignore rules
├── .idea/                # IntelliJ IDEA configuration files
├── .trae/                # Trae-specific configuration
├── .vscode/              # VS Code configuration
├── .docs/                # Project documentation
├── db/                   # Database connection and utilities
│   ├── builder/          # MongoDB query builder utilities
│   │   └── builder.go    # Query building functions
│   └── mongo.go          # MongoDB connection setup
├── go.mod                # Go module definition
├── go.sum                # Go module checksums
├── main.go               # Application entry point
├── model/                # Data models/entities
│   ├── accountModel.go   # Account data structures
│   ├── formModel.go      # Form data structures
│   ├── projectModel.go   # Project data structures
│   ├── questionModel.go  # Question data structures
│   ├── roleModel.go      # Role data structures
│   └── userModel.go      # User data structures
├── router/               # HTTP routing definitions
│   ├── authRouter.go     # Authentication routes
│   ├── projectRouter.go  # Project management routes
│   ├── questionRouter.go # Question management routes
│   ├── roleRouter.go     # Role management routes
│   └── userRouter.go     # User management routes
└── service/              # Business logic implementation
    ├── authService.go    # Authentication services
    ├── projectService.go # Project management services
    ├── questionService.go# Question management services
    ├── roleService.go    # Role management services
    └── userService.go    # User management services
```

## Architectural Patterns

The application follows a layered architecture with the following components:

1. **Router Layer (Presentation)**
   - Handles HTTP requests and responses
   - Defines API endpoints
   - Performs basic request validation
   - Maps HTTP requests to service calls

2. **Service Layer (Business Logic)**
   - Implements core business logic
   - Orchestrates data operations
   - Handles transaction management
   - Enforces business rules

3. **Model Layer (Data)**
   - Defines data structures
   - Includes JSON/BSON mapping
   - Contains data validation logic
   - Provides data transformation methods

4. **Database Layer (Persistence)**
   - Manages database connections
   - Provides query building utilities
   - Abstracts database operations
   - Handles MongoDB-specific functionality

## Key Components

### Main Application (main.go)

- Application entry point
- Sets up HTTP server and middleware
- Configures CORS and other global settings
- Initializes database connection
- Mounts all router components

### Database (db/)

- **mongo.go**: Establishes connection to MongoDB
- **builder/builder.go**: Provides utilities for building MongoDB queries
  - Includes functions for search, pagination, lookup, etc.
  - Implements generic functions for common database operations

### Models (model/)

Defines data structures for:
- Accounts (authentication)
- Users (profile information)
- Roles (authorization)
- Projects (survey projects)
- Forms (survey forms)
- Questions (survey questions)

Each model includes:
- Struct definitions with JSON/BSON tags
- Request/response DTOs
- Data transformation methods

### Routers (router/)

Implements HTTP endpoints for:
- Authentication (login, register)
- User management
- Role management
- Project management
- Question management

Each router:
- Defines routes using Chi router
- Maps HTTP methods to handler functions
- Performs request/response serialization

### Services (service/)

Implements business logic for:
- Authentication and authorization
- User management
- Role management
- Project management
- Question management

Each service:
- Interacts with the database
- Implements domain-specific logic
- Handles error conditions

## Naming Conventions

- **Files**: Use camelCase with descriptive suffixes (e.g., `userModel.go`, `authService.go`)
- **Packages**: Use lowercase, single-word names (e.g., `model`, `service`, `router`)
- **Types/Structs**: Use PascalCase (e.g., `User`, `ProjectService`)
- **Methods**: Use camelCase (e.g., `getProjectById`, `createUser`)
- **Variables**: Use camelCase (e.g., `userService`, `projectCollection`)

## Configuration

- Environment variables are loaded from `.env` file
- MongoDB connection string is specified in the environment
- Application runs on port 3001

## Dependencies

The application uses several external dependencies:
- **chi**: Lightweight HTTP router
- **mongo-driver**: Official MongoDB driver for Go
- **godotenv**: For loading environment variables
- **cors**: For handling Cross-Origin Resource Sharing