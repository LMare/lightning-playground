// Package exception provides AppError: an error that remembers where it was created and what caused it.
package exception

import (
	"fmt"
	"runtime"
	"strings"
)

// AppError is an error enriched with the place (file:line) where it was created and an optional cause.
// It supports errors.Is / errors.As through Unwrap, and the chain can be printed with the cause of each level.
type AppError struct {
	message string
	file    string
	line    int
	cause   error
}

func (e *AppError) Error() string {
	if e.cause != nil {
		return fmt.Sprintf("(%s:%d) %s → %v", e.file, e.line, e.message, e.cause)
	}
	return fmt.Sprintf("(%s:%d) %s", e.file, e.line, e.message)
}

func (e *AppError) Unwrap() error   { return e.cause }
func (e *AppError) File() string    { return e.file }
func (e *AppError) Line() int       { return e.line }
func (e *AppError) Message() string { return e.message }

// projectPath is removed from the file path of the errors to keep them short.
var projectPath string

func ConfigureProjectBasePath(p string) {
	projectPath = p
}

// NewError creates an AppError located at the caller. cause may be nil.
//
//	return exception.NewError("cannot open channel", err)
func NewError(message string, cause error) error {
	_, file, line, _ := runtime.Caller(1)
	return &AppError{
		message: message,
		file:    strings.TrimPrefix(file, projectPath),
		line:    line,
		cause:   cause,
	}
}
