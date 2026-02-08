package apperr

import (
	"errors"
	"fmt"
)

const (
	CodeOK            = 0
	CodeGeneric       = 1
	CodeValidation    = 2
	CodeConnection    = 3
	CodeToolExecution = 4
	CodeConfig        = 5
	CodeInternal      = 6
)

// Error is an application-level error with a stable exit code.
type Error struct {
	Code int
	Err  error
}

func (e *Error) Error() string {
	if e == nil {
		return ""
	}
	if e.Err == nil {
		return fmt.Sprintf("application error (code=%d)", e.Code)
	}
	return e.Err.Error()
}

func (e *Error) Unwrap() error {
	if e == nil {
		return nil
	}
	return e.Err
}

func New(code int, msg string) *Error {
	return &Error{
		Code: code,
		Err:  errors.New(msg),
	}
}

func Wrap(code int, err error, format string, args ...any) *Error {
	prefix := fmt.Sprintf(format, args...)
	if err == nil {
		return New(code, prefix)
	}
	return &Error{
		Code: code,
		Err:  fmt.Errorf("%s: %w", prefix, err),
	}
}

func Code(err error) int {
	if err == nil {
		return CodeOK
	}
	var appErr *Error
	if errors.As(err, &appErr) && appErr != nil {
		if appErr.Code == 0 {
			return CodeGeneric
		}
		return appErr.Code
	}
	return CodeGeneric
}
