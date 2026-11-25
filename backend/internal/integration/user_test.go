package integration

import (
	"errors"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/Lmare/lightning-playground/backend/internal/business/user/domain"
	"github.com/Lmare/lightning-playground/backend/internal/testsupport"
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
	h.Router.ServeHTTP(rr, httptest.NewRequest(http.MethodGet, "/api/v1/users", nil))

	assert.Equal(t, http.StatusOK, rr.Code)
	assert.JSONEq(t, `[
		{"id":"1","nom":"Doe","prenom":"John","age":30,"email":"john.doe@example.com"},
		{"id":"2","nom":"Smith","prenom":"Jane","age":25,"email":"jane.smith@example.com"}]`, rr.Body.String())
	h.MockUserRepo.AssertExpectations(t)
}

func TestListUsers_RepositoryError(t *testing.T) {
	h := testsupport.NewTestHarness()
	h.MockUserRepo.On("FindAll", mock.Anything).Return([]domain.UserModel(nil), errors.New("db down"))

	rr := httptest.NewRecorder()
	h.Router.ServeHTTP(rr, httptest.NewRequest(http.MethodGet, "/api/v1/users", nil))

	assert.Equal(t, http.StatusInternalServerError, rr.Code)
}
