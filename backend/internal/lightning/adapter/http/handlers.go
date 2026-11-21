package lightninghttp

import (
	"errors"
	"html/template"
	"net/http"
	"strconv"
	"strings"

	"github.com/Lmare/lightning-playground/backend/internal/httpapi"
	"github.com/Lmare/lightning-playground/backend/internal/lightning/application"
	"github.com/Lmare/lightning-playground/backend/internal/lightning/domain"
)

func NewHandler(nodeService *application.NodeService, channelService *application.ChannelService) *Handler {
	return &Handler{nodeService: nodeService, channelService: channelService}
}

type Handler struct {
	nodeService    *application.NodeService
	channelService *application.ChannelService
}

// nodeIDFromRequest reads and validates the "id" parameter; it writes the error response when invalid.
func nodeIDFromRequest(response http.ResponseWriter, request *http.Request, message string) (domain.NodeID, bool) {
	id, err := domain.ParseNodeID(request.FormValue("id"))
	if err != nil {
		httpapi.FailCheck(response, request, message, nil)
		return "", false
	}
	return id, true
}

// failUseCase answers with 400 for business validation errors and 500 for the others.
func failUseCase(response http.ResponseWriter, request *http.Request, message string, err error) {
	if errors.Is(err, domain.ErrInvalidPeerAddress) || errors.Is(err, domain.ErrInvalidAmount) {
		httpapi.FailCheck(response, request, message, err)
		return
	}
	httpapi.Fail(response, request, message, err)
}

// get the list of node
func (h *Handler) HandleListOfNodes(response http.ResponseWriter, request *http.Request) {
	nodes, err := h.nodeService.ListNodes(request.Context())
	if err != nil {
		httpapi.Fail(response, request, "Error on listing nodes", err)
		return
	}

	dtos := toNodeSummaryDTOs(nodes)
	if httpapi.IsHTMX(request) {
		httpapi.HTMXResponse(response, "lightning/nodes.html", dtos)
	} else {
		httpapi.JSONResponse(response, dtos)
	}
}

// get the URI of the node
func (h *Handler) HandleShowUri(response http.ResponseWriter, request *http.Request) {
	id, ok := nodeIDFromRequest(response, request, "Pas d'id transmis")
	if !ok {
		return
	}

	uri, err := h.nodeService.GetURI(request.Context(), id)
	if err != nil {
		httpapi.Fail(response, request, "Echec de la communication avec le noeud LND", err)
		return
	}

	if httpapi.IsHTMX(request) {
		funcMap := template.FuncMap{"truncateUri": truncateUri}
		httpapi.HTMXResponseWithFuncs(response, "lightning/uri.html", uri, funcMap)
	} else {
		httpapi.JSONResponse(response, uri)
	}
}

// reduce an uri
func truncateUri(s string, n int) string {
	at := strings.Index(s, "@")
	if at == -1 || at < 2*n {
		return s // no @ or too short to truncate
	}

	start := s[:n]
	end := s[at-n : at]
	host := s[at:] // includes @

	return start + "..." + end + host
}

// reduce a string
func truncate(s string, n int) string {
	at := len(s)
	if at <= 2*n {
		return s
	}
	start := s[:n]
	end := s[at-n : at]
	return start + "..." + end
}

// get the info of one Node
func (h *Handler) HandleNodeInfo(response http.ResponseWriter, request *http.Request) {
	id, ok := nodeIDFromRequest(response, request, "Pas d'id transmis")
	if !ok {
		return
	}

	info, err := h.nodeService.GetInfo(request.Context(), id)
	if err != nil {
		httpapi.Fail(response, request, "Echec de la communication avec le noeud LND", err)
		return
	}

	dto := toNodeInfoDTO(info)
	if httpapi.IsHTMX(request) {
		httpapi.HTMXResponse(response, "lightning/nodeInfo.html", dto)
	} else {
		httpapi.JSONResponse(response, dto)
	}
}

// Create a connexion to a new Peer
func (h *Handler) HandleAddPeer(response http.ResponseWriter, request *http.Request) {
	if err := request.ParseForm(); err != nil {
		http.Error(response, "Erreur de parsing", http.StatusBadRequest)
		return
	}

	id, ok := nodeIDFromRequest(response, request, "Pas d'id transmis")
	if !ok {
		return
	}

	if err := h.channelService.AddPeer(request.Context(), id, request.FormValue("uri")); err != nil {
		failUseCase(response, request, "Fail to add the peer.", err)
		return
	}

	if httpapi.IsHTMX(request) {
		httpapi.HTMXMessageOk(response, "Peer successfully added.")
	} else {
		httpapi.OkNoContent(response)
	}
}

// create a channel
func (h *Handler) HandleOpenChannel(response http.ResponseWriter, request *http.Request) {
	if err := request.ParseForm(); err != nil {
		httpapi.FailCheck(response, request, "Parse error", err)
		return
	}

	id, ok := nodeIDFromRequest(response, request, "Pas d'id transmis")
	if !ok {
		return
	}
	amount, err := strconv.ParseInt(request.FormValue("amount"), 10, 64)
	if err != nil {
		httpapi.FailCheck(response, request, "Amount value incorrect", err)
		return
	}

	if err := h.channelService.OpenChannel(request.Context(), id, request.FormValue("pubKey"), amount); err != nil {
		failUseCase(response, request, "Fail to create the channel.", err)
		return
	}

	if httpapi.IsHTMX(request) {
		httpapi.HTMXMessageOk(response, "Channel successfully created.")
	} else {
		httpapi.OkNoContent(response)
	}
}

// Create an invoice
func (h *Handler) HandleCreateInvoice(response http.ResponseWriter, request *http.Request) {
	if err := request.ParseForm(); err != nil {
		httpapi.FailCheck(response, request, "Parse error", err)
		return
	}

	id, ok := nodeIDFromRequest(response, request, "Pas d'id transmis")
	if !ok {
		return
	}
	amount, err := strconv.ParseInt(request.FormValue("amount"), 10, 64)
	if err != nil {
		httpapi.FailCheck(response, request, "Amount value incorrect", err)
		return
	}

	paymentRequest, err := h.channelService.CreateInvoice(request.Context(), id, request.FormValue("memo"), amount)
	if err != nil {
		failUseCase(response, request, "Fail to create the invoice.", err)
		return
	}

	if httpapi.IsHTMX(request) {
		funcMap := template.FuncMap{"truncate": truncate}
		httpapi.HTMXResponseWithFuncs(response, "lightning/paymentRequest.html", paymentRequest, funcMap)
	} else {
		httpapi.JSONResponse(response, paymentRequest)
	}
}

// Make the payment
func (h *Handler) HandleMakePayment(response http.ResponseWriter, request *http.Request) {
	if err := request.ParseForm(); err != nil {
		httpapi.FailCheck(response, request, "Error parsing form", err)
		return
	}

	id, ok := nodeIDFromRequest(response, request, "Missing ID")
	if !ok {
		return
	}
	paymentRequest := request.FormValue("paymentRequest")
	if paymentRequest == "" {
		httpapi.FailCheck(response, request, "Missing payment request", nil)
		return
	}

	// pay the invoice (async updates via SSE)
	if err := h.channelService.Pay(request.Context(), id, paymentRequest); err != nil {
		failUseCase(response, request, "Fail to pay the invoice.", err)
		return
	}

	// TODO have a streamId to return ?
	if httpapi.IsHTMX(request) {
		httpapi.HTMXMessageOk(response, "Processing in progress")
	} else {
		httpapi.OkNoContent(response)
	}
}

// Update name of the node & color
func (h *Handler) HandleUpdateNodeAlias(response http.ResponseWriter, request *http.Request) {
	if err := request.ParseForm(); err != nil {
		httpapi.FailCheck(response, request, "Parse error", err)
		return
	}

	alias := request.FormValue("alias")
	if alias == "" {
		httpapi.FailCheck(response, request, "Missing alias", nil)
		return
	}
	color := request.FormValue("color")
	if color == "" {
		httpapi.FailCheck(response, request, "Missing color", nil)
		return
	}
	id, ok := nodeIDFromRequest(response, request, "Missing ID")
	if !ok {
		return
	}

	if err := h.nodeService.UpdateAppearance(request.Context(), id, alias, color); err != nil {
		httpapi.Fail(response, request, "Modifications fail.", err)
		return
	}

	if httpapi.IsHTMX(request) {
		httpapi.HTMXMessageOk(response, "Modifications successfully applied.")
	} else {
		httpapi.OkNoContent(response)
	}
}
