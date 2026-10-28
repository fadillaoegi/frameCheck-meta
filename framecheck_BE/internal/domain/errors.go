package domain

import (
	"errors"
	"fmt"
)

// ErrorKind tells transport adapters how to classify a domain error without
// the domain knowing about HTTP status codes.
type ErrorKind int

const (
	KindInvalid ErrorKind = iota + 1
	KindNotFound
	KindConflict
	KindTooLarge
	KindUnavailable
)

type Error struct {
	Kind    ErrorKind
	Code    string
	Message string
}

func (e *Error) Error() string { return e.Code + ": " + e.Message }

func Invalid(code, format string, args ...any) error {
	return &Error{Kind: KindInvalid, Code: code, Message: fmt.Sprintf(format, args...)}
}

func NotFound(code, format string, args ...any) error {
	return &Error{Kind: KindNotFound, Code: code, Message: fmt.Sprintf(format, args...)}
}

func Conflict(code, format string, args ...any) error {
	return &Error{Kind: KindConflict, Code: code, Message: fmt.Sprintf(format, args...)}
}

func TooLarge(code, format string, args ...any) error {
	return &Error{Kind: KindTooLarge, Code: code, Message: fmt.Sprintf(format, args...)}
}

func Unavailable(code, format string, args ...any) error {
	return &Error{Kind: KindUnavailable, Code: code, Message: fmt.Sprintf(format, args...)}
}

// AsError extracts a domain error, if any.
func AsError(err error) (*Error, bool) {
	var de *Error
	if errors.As(err, &de) {
		return de, true
	}
	return nil, false
}
