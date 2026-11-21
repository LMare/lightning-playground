package integration

import (
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/Lmare/lightning-playground/backend/internal/testsupport"
	"github.com/Lmare/lightning-playground/backend/internal/user/domain"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
)

func TestListUsers_JSON(t *testing.T) {
	h := testsupport.NewTestHarness()
	users := []domain.UserModel{
		{ID: "1", Nom: "Doe", Prenom: "John", Age: 30, Email: "john.doe@example.com"},
		{ID: "2", Nom: "Smith", Prenom: "Jane", Age: 25, Email: "jane.smith@example.com"},
	}
	h.MockUserRepo.On("FindAll", mock.Anything).Return(users, nil)

	rr := httptest.NewRecorder()
	h.Router.ServeHTTP(rr, httptest.NewRequest(http.MethodGet, "/users", nil))

	assert.Equal(t, http.StatusOK, rr.Code)
	var got []domain.UserModel
	assert.NoError(t, json.Unmarshal(rr.Body.Bytes(), &got))
	assert.Equal(t, users, got)
	h.MockUserRepo.AssertExpectations(t)
}

func TestListUsers_RepositoryError(t *testing.T) {
	h := testsupport.NewTestHarness()
	h.MockUserRepo.On("FindAll", mock.Anything).Return([]domain.UserModel(nil), errors.New("db down"))

	rr := httptest.NewRecorder()
	h.Router.ServeHTTP(rr, httptest.NewRequest(http.MethodGet, "/users", nil))

	assert.Equal(t, http.StatusInternalServerError, rr.Code)
}
