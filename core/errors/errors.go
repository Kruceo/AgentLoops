package errors

import (
	"fmt"
	"net/http"

	stderrors "errors"
)

// AppError is the standard error type used across the application.
type AppError struct {
	Code    string `json:"code"`
	Message string `json:"message"`
	Err     error  `json:"-"`
}

func (e *AppError) Error() string { return e.Message }
func (e *AppError) Unwrap() error { return e.Err }

// Sentinel errors
var (
	ErrTaskNotFound  = &AppError{Code: "TASK_NOT_FOUND", Message: "task not found"}
	ErrRunNotFound   = &AppError{Code: "RUN_NOT_FOUND", Message: "run not found"}
	ErrAgentNotFound = &AppError{Code: "AGENT_NOT_FOUND", Message: "agent not found"}
	ErrConflict      = &AppError{Code: "CONFLICT", Message: "conflict"}
	ErrValidation    = &AppError{Code: "VALIDATION", Message: "validation error"}
	ErrInternal      = &AppError{Code: "INTERNAL", Message: "internal server error"}
	ErrUnauthorized  = &AppError{Code: "UNAUTHORIZED", Message: "unauthorized"}
)

// Required creates a validation error for a missing required field.
func Required(field string) *AppError {
	return &AppError{
		Code:    "VALIDATION_REQUIRED",
		Message: fmt.Sprintf("%s is required", field),
		Err:     ErrValidation,
	}
}

// CodeToStatus maps AppError codes to HTTP status codes.
var CodeToStatus = map[string]int{
	"TASK_NOT_FOUND":      http.StatusNotFound,
	"RUN_NOT_FOUND":       http.StatusNotFound,
	"AGENT_NOT_FOUND":     http.StatusNotFound,
	"VALIDATION":          http.StatusBadRequest,
	"VALIDATION_REQUIRED": http.StatusBadRequest,
	"CONFLICT":            http.StatusConflict,
	"INTERNAL":            http.StatusInternalServerError,
	"UNAUTHORIZED":        http.StatusUnauthorized,
}

// StatusFor returns the HTTP status code for an error.
func StatusFor(err error) int {
	var appErr *AppError
	if stderrors.As(err, &appErr) {
		if status, ok := CodeToStatus[appErr.Code]; ok {
			return status
		}
		return http.StatusInternalServerError
	}
	return http.StatusInternalServerError
}

// Is is a re-export of stdlib errors.Is for convenience.
func Is(err, target error) bool { return stderrors.Is(err, target) }

// As is a re-export of stdlib errors.As for convenience.
func As(err error, target interface{}) bool { return stderrors.As(err, target) }
