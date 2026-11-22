package exception

import (
	"errors"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestNewError_RecordsTheCallerLocation(t *testing.T) {
	err := NewError("boom", nil)

	var appErr *AppError
	assert.True(t, errors.As(err, &appErr))
	assert.Equal(t, "boom", appErr.Message())
	assert.Contains(t, appErr.File(), "app_error_test.go")
	assert.Greater(t, appErr.Line(), 0)
	assert.Contains(t, err.Error(), "boom")
}

func TestNewError_KeepsTheCauseChain(t *testing.T) {
	sentinel := errors.New("root cause")
	err := NewError("outer", NewError("inner", sentinel))

	assert.ErrorIs(t, err, sentinel)
	assert.Contains(t, err.Error(), "outer")
	assert.Contains(t, err.Error(), "inner")
	assert.Contains(t, err.Error(), "root cause")
}

func TestConfigureProjectBasePath_ShortensTheFile(t *testing.T) {
	ConfigureProjectBasePath("/")
	defer ConfigureProjectBasePath("")

	var appErr *AppError
	assert.True(t, errors.As(NewError("x", nil), &appErr))
	assert.NotContains(t, appErr.File(), "/home")
}
