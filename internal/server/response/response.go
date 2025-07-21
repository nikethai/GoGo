package response

import (
	"encoding/json"
	"net/http"
)

// Response represents the standard API response format
type Response struct {
	StatusCode int         `json:"statusCode"`
	Data       interface{} `json:"data"`
	Message    string      `json:"message"`
}

// Success sends a successful response with the provided data and message
func Success(w http.ResponseWriter, statusCode int, data interface{}, message string) {
	response := Response{
		StatusCode: statusCode,
		Data:       data,
		Message:    message,
	}
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(statusCode)
	json.NewEncoder(w).Encode(response)
}

// SuccessWithData sends a successful response with data and default success message
func SuccessWithData(w http.ResponseWriter, data interface{}) {
	Success(w, http.StatusOK, data, "Success")
}

// SuccessWithMessage sends a successful response with message and nil data
func SuccessWithMessage(w http.ResponseWriter, message string) {
	Success(w, http.StatusOK, nil, message)
}

// Fail sends a failed response with the provided status code and message
func Fail(w http.ResponseWriter, statusCode int, message string) {
	response := Response{
		StatusCode: statusCode,
		Data:       nil,
		Message:    message,
	}
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(statusCode)
	json.NewEncoder(w).Encode(response)
}

// BadRequest sends a 400 Bad Request response
func BadRequest(w http.ResponseWriter, message string) {
	Fail(w, http.StatusBadRequest, message)
}

// Unauthorized sends a 401 Unauthorized response
func Unauthorized(w http.ResponseWriter, message string) {
	Fail(w, http.StatusUnauthorized, message)
}

// Forbidden sends a 403 Forbidden response
func Forbidden(w http.ResponseWriter, message string) {
	Fail(w, http.StatusForbidden, message)
}

// NotFound sends a 404 Not Found response
func NotFound(w http.ResponseWriter, message string) {
	Fail(w, http.StatusNotFound, message)
}

// InternalServerError sends a 500 Internal Server Error response
func InternalServerError(w http.ResponseWriter, message string) {
	Fail(w, http.StatusInternalServerError, message)
}

// ValidationError sends a 422 Unprocessable Entity response for validation errors
func ValidationError(w http.ResponseWriter, message string) {
	Fail(w, http.StatusUnprocessableEntity, message)
}

// Conflict sends a 409 Conflict response
func Conflict(w http.ResponseWriter, message string) {
	Fail(w, http.StatusConflict, message)
}

// Created sends a 201 Created response with data
func Created(w http.ResponseWriter, data interface{}, message string) {
	Success(w, http.StatusCreated, data, message)
}

// NoContent sends a 204 No Content response
func NoContent(w http.ResponseWriter, message string) {
	Success(w, http.StatusNoContent, nil, message)
}
