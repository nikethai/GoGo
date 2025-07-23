# Gogo API

A RESTful API built with Go, Chi router, and MongoDB for managing questions, projects, users, and roles with JWT authentication and role-based access control.

## Features

- **JWT Authentication**: Secure API endpoints with JSON Web Tokens
- **Role-Based Access Control**: Restrict access to resources based on user roles
- **MongoDB Integration**: Store and retrieve data from MongoDB
- **RESTful API Design**: Follow REST principles for API design
- **Swagger Documentation**: API documentation with Swagger

## Authentication Flow

1. **Registration**: Create a new account with username and password
2. **Login**: Authenticate with username and password to receive a JWT token
3. **Protected Routes**: Access protected routes by including the JWT token in the Authorization header

## Role-Based Access Control

The API implements role-based access control to restrict access to certain endpoints based on user roles:

- **Public Routes**: `/auth/register`, `/auth/login`, `/swagger/*`
- **Authenticated Routes**: All routes under `/api/*` require a valid JWT token
- **Role-Specific Routes**:
  - **Admin**: Can manage roles and users (`/roles`, `/users`)
  - **Content Creator**: Can create questions (`/questions`)
  - **Project Manager**: Can create projects (`/projects`)

## Environment Variables

The application uses the following environment variables:

- `MONGODB_URI`: MongoDB connection string
- `JWT_SECRET_KEY`: Secret key for JWT token generation and validation
- `JWT_TOKEN_DURATION`: Duration of JWT tokens in hours (default: 24)

## API Endpoints

### Authentication

- `POST /auth/register`: Register a new user
- `POST /auth/login`: Login and receive a JWT token

### Users

- `GET /api/users/{uid}`: Get user by ID (authenticated)
- `POST /api/users`: Create a new user (admin only)

### Roles

- `GET /api/roles/{roleId}`: Get role by ID (authenticated)
- `POST /api/roles`: Create a new role (admin only)

### Questions

- `GET /api/questions`: Get all questions (authenticated)
- `POST /api/questions`: Create a new question (content_creator only)

### Projects

- `GET /api/projects`: Get all projects (authenticated)
- `GET /api/projects/{id}`: Get project by ID (authenticated)
- `POST /api/projects`: Create a new project (project_manager only)

## JWT Authentication Implementation

The JWT authentication is implemented using the following components:

1. **JWT Package**: Uses `github.com/golang-jwt/jwt/v5` for JWT token generation and validation
2. **JWT Middleware**: Validates JWT tokens and adds user information to the request context
3. **Role Middleware**: Checks if the user has the required role to access a resource

## Getting Started

1. Clone the repository
2. Set up environment variables in `.env` file
3. Run the application: `go run main.go`
4. Access the API at `http://localhost:3001`
5. Access the Swagger documentation at `http://localhost:3001/swagger/index.html`