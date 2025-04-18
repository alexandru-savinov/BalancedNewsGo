package apperrors

import (
	"fmt"
)

// AppError represents an application error with a code and message
type AppError struct {
	Code    string `json:"code"`
	Message string `json:"message"`
}

// Error implements the error interface
func (e *AppError) Error() string {
	return fmt.Sprintf("%s: %s", e.Code, e.Message)
}

// Is returns true if the target error is an AppError and has the same code
func (e *AppError) Is(target error) bool {
	if t, ok := target.(*AppError); ok {
		return e.Code == t.Code
	}
	return false
}

// New creates a new AppError
func New(message string, code string) *AppError {
	return &AppError{
		Code:    code,
		Message: message,
	}
}

// HandleError wraps a standard error into an AppError with default error code if not already an AppError
func HandleError(err error, defaultMessage string) *AppError {
	if err == nil {
		return nil
	}
	if appErr, ok := err.(*AppError); ok {
		return appErr
	}
	message := defaultMessage
	if message == "" {
		message = err.Error()
	}
	return &AppError{
		Code:    "internal_error",
		Message: message,
	}
}

// Join combines multiple errors into a single AppError
func Join(errs ...error) *AppError {
	if len(errs) == 0 {
		return nil
	}
	messages := make([]string, 0, len(errs))
	code := "internal_error"
	for _, err := range errs {
		if err == nil {
			continue
		}
		if appErr, ok := err.(*AppError); ok {
			messages = append(messages, appErr.Message)
			code = appErr.Code
		} else {
			messages = append(messages, err.Error())
		}
	}
	if len(messages) == 0 {
		return nil
	}
	return &AppError{
		Code:    code,
		Message: fmt.Sprintf("Multiple errors occurred: %v", messages),
	}
}
