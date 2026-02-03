package errors

import "fmt"

type AppError struct {
	Code    string
	Message string
	Err     error
}

func (e *AppError) Error() string {
	if e.Err != nil {
		return fmt.Sprintf("%s: %s (%v)", e.Code, e.Message, e.Err)
	}
	return fmt.Sprintf("%s: %s", e.Code, e.Message)
}

func New(code, message string) *AppError {
	return &AppError{
		Code:    code,
		Message: message,
	}
}

func Wrap(err error, code, message string) *AppError {
	return &AppError{
		Code:    code,
		Message: message,
		Err:     err,
	}
}

// Common error codes
const (
	ErrCodeValidation        = "VALIDATION_ERROR"
	ErrCodeValidationFailed  = "VALIDATION_FAILED"
	ErrCodeNotFound          = "NOT_FOUND"
	ErrCodeUnauthorized      = "UNAUTHORIZED"
	ErrCodeForbidden         = "FORBIDDEN"
	ErrCodeInternalError     = "INTERNAL_ERROR"
	ErrCodeInsufficientFunds = "INSUFFICIENT_FUNDS"
	ErrCodeAlreadyExists     = "ALREADY_EXISTS"
	ErrCodeRateLimitExceeded = "RATE_LIMIT_EXCEEDED"
)
