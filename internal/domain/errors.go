package domain

import "fmt"

type Error struct {
	Code    string `json:"code"`
	Message string `json:"message"`
	Details any    `json:"details,omitempty"`
}

func (e *Error) Error() string { return e.Message }

func NewError(code, message string) error { return &Error{Code: code, Message: message} }
func NewDetailedError(code, message string, details any) error {
	return &Error{Code: code, Message: message, Details: details}
}

func Invalid(field, reason string) error {
	return NewError("invalid_argument", fmt.Sprintf("%s：%s", field, reason))
}

func Conflict(message string) error  { return NewError("conflict", message) }
func NotFound(message string) error  { return NewError("not_found", message) }
func Forbidden(message string) error { return NewError("forbidden", message) }

func ErrorCode(err error) string {
	if e, ok := err.(*Error); ok {
		return e.Code
	}
	return "internal_error"
}
