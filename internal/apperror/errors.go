package apperror

import (
	"fmt"
	"net/http"
)

// AppError represents a centralized error format for the application
type AppError struct {
	StatusCode int
	Status     string
	Message    string
	Err        interface{}
	Data       interface{}
}

func (e *AppError) Error() string {
	if e.Err != nil {
		if errVal, ok := e.Err.(error); ok {
			return fmt.Sprintf("%s: %v", e.Message, errVal)
		}
		return fmt.Sprintf("%s: %v", e.Message, e.Err)
	}
	return e.Message
}

// New creates a new AppError
func New(statusCode int, status, message string, err interface{}, data interface{}) *AppError {
	return &AppError{
		StatusCode: statusCode,
		Status:     status,
		Message:    message,
		Err:        err,
		Data:       data,
	}
}

// Common error constructors

func BadRequest(message string, err interface{}, data interface{}) *AppError {
	return New(http.StatusBadRequest, "error", message, err, data)
}

func Unauthorized(message string, err interface{}, data interface{}) *AppError {
	return New(http.StatusUnauthorized, "error", message, err, data)
}

func Forbidden(message string, err interface{}, data interface{}) *AppError {
	return New(http.StatusForbidden, "error", message, err, data)
}

func NotFound(message string, err interface{}, data interface{}) *AppError {
	return New(http.StatusNotFound, "error", message, err, data)
}

func UnprocessableEntity(message string, err interface{}, data interface{}) *AppError {
	return New(http.StatusUnprocessableEntity, "error", message, err, data)
}

func InternalServerError(message string, err interface{}, data interface{}) *AppError {
	return New(http.StatusInternalServerError, "error", message, err, data)
}
