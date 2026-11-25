package userhttp

import (
	"net/http"

	"github.com/Lmare/lightning-playground/backend/internal/business/user/application"
	"github.com/Lmare/lightning-playground/backend/internal/infra/http/common"
)

type Handler struct {
	service *application.UserService
}

func NewHandler(service *application.UserService) *Handler {
	return &Handler{
		service: service,
	}
}

func (h *Handler) HandleListUsers(response http.ResponseWriter, request *http.Request) {

	users, err := h.service.List(request.Context())
	if err != nil {
		common.Fail(response, request, "Error in the user service", err)
		return
	}

	if common.IsHTMX(request) {
		vo := viewObject(users)
		common.HTMXResponse(response, templates, "user.gohtml", vo)
	} else {
		common.JSONResponse(response, toUserDTOs(users))
	}
}
