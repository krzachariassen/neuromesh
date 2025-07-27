package responses

import (
	"encoding/json"
	"net/http"
	"neuromesh/internal/api/rest/v1/domain"
)

// JSON writes a JSON response with the given status code
func JSON(w http.ResponseWriter, statusCode int, data interface{}) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(statusCode)

	if data != nil {
		if err := json.NewEncoder(w).Encode(data); err != nil {
			// If encoding fails, try to send a basic error response
			http.Error(w, "Internal server error", http.StatusInternalServerError)
		}
	}
}

// Success writes a successful JSON response
func Success(w http.ResponseWriter, data interface{}) {
	JSON(w, http.StatusOK, data)
}

// Created writes a 201 Created JSON response
func Created(w http.ResponseWriter, data interface{}) {
	JSON(w, http.StatusCreated, data)
}

// Error writes an error JSON response
func Error(w http.ResponseWriter, statusCode int, message string) {
	errorResponse := domain.ErrorResponse{
		Error: message,
	}
	JSON(w, statusCode, errorResponse)
}

// BadRequest writes a 400 Bad Request error response
func BadRequest(w http.ResponseWriter, message string) {
	Error(w, http.StatusBadRequest, message)
}

// NotFound writes a 404 Not Found error response
func NotFound(w http.ResponseWriter, message string) {
	Error(w, http.StatusNotFound, message)
}

// InternalError writes a 500 Internal Server Error response
func InternalError(w http.ResponseWriter, message string) {
	Error(w, http.StatusInternalServerError, message)
}

// MethodNotAllowed writes a 405 Method Not Allowed error response
func MethodNotAllowed(w http.ResponseWriter, message string) {
	Error(w, http.StatusMethodNotAllowed, message)
}
