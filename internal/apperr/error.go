package apperr

import (
	"errors"
	"fmt"
)

const (
	ExitSuccess        = 0
	ExitArguments      = 2
	ExitConfiguration  = 3
	ExitAuthentication = 4
	ExitPermission     = 5
	ExitNotFound       = 6
	ExitConflict       = 7
	ExitQuota          = 8
	ExitRateLimited    = 9
	ExitNetwork        = 10
	ExitTimeout        = 11
	ExitServer         = 12
	ExitFile           = 13
	ExitInterrupted    = 130
)

type Error struct {
	Code    int
	Message string
	Cause   error
}

func (e *Error) Error() string {
	if e.Cause == nil {
		return e.Message
	}
	return fmt.Sprintf("%s: %v", e.Message, e.Cause)
}

func (e *Error) Unwrap() error { return e.Cause }

func New(code int, message string) error {
	return &Error{Code: code, Message: message}
}

func Wrap(code int, message string, cause error) error {
	return &Error{Code: code, Message: message, Cause: cause}
}

func ExitCode(err error) int {
	if err == nil {
		return ExitSuccess
	}
	var appError *Error
	if errors.As(err, &appError) {
		return appError.Code
	}
	return ExitServer
}
