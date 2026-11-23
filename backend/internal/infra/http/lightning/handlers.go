package lightninghttp

import (
	"errors"
	"net/http"
	"strconv"
	"strings"

	"github.com/Lmare/lightning-playground/backend/internal/business/lightning/application"
	"github.com/Lmare/lightning-playground/backend/internal/business/lightning/domain"
	"github.com/Lmare/lightning-playground/backend/internal/infra/http/common"
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
		common.FailCheck(response, request, message, nil)
		return "", false
	}
	return id, true
}

// failUseCase answers with 400 for business validation errors and 500 for the others.
func failUseCase(response http.ResponseWriter, request *http.Request, message string, err error) {
	if errors.Is(err, domain.ErrInvalidPeerAddress) || errors.Is(err, domain.ErrInvalidAmount) {
		common.FailCheck(response, request, message, err)
		return
	}
	common.Fail(response, request, message, err)
}

// HandleUnlockWallet unlocks the wallet of a node. The password is only read from the POST body
// (never the query string) and never logged.
func (h *Handler) HandleUnlockWallet(response http.ResponseWriter, request *http.Request) {
	if err := request.ParseForm(); err != nil {
		common.FailCheck(response, request, "Invalid request.", nil)
		return
	}
	id, ok := nodeIDFromRequest(response, request, "Missing or invalid node id.")
	if !ok {
		return
	}

	err := h.nodeService.UnlockWallet(request.Context(), id, []byte(request.PostFormValue("password")))
	switch {
	case err == nil:
		response.Header().Set("HX-Trigger", "wallet-unlocked")
		if common.IsHTMX(request) {
			common.HTMXMessageOk(response, "Wallet unlocked, the node is starting.")
		} else {
			common.OkNoContent(response)
		}
	case errors.Is(err, domain.ErrWalletPasswordRequired):
		common.FailCheck(response, request, "Missing password", err)
	case errors.Is(err, domain.ErrInvalidWalletPassword):
		common.FailCheck(response, request, "Wrong wallet password.", err)
	case errors.Is(err, domain.ErrWalletNotLocked):
		common.FailCheck(response, request, "The wallet is not locked.", err)
	case errors.Is(err, domain.ErrWalletNotCreated):
		common.FailCheck(response, request, "No wallet on this node, create it first.", err)
	default:
		common.Fail(response, request, "Unable to reach the LND node.", err)
	}
}

// get the list of node
func (h *Handler) HandleListOfNodes(response http.ResponseWriter, request *http.Request) {
	nodes, err := h.nodeService.ListNodes(request.Context())
	if err != nil {
		common.Fail(response, request, "Error on listing nodes", err)
		return
	}

	dtos := toNodeSummaryDTOs(nodes)
	if common.IsHTMX(request) {
		common.HTMXResponse(response, templates, "nodes.html", dtos)
	} else {
		common.JSONResponse(response, dtos)
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
		common.Fail(response, request, "Echec de la communication avec le noeud LND", err)
		return
	}

	if common.IsHTMX(request) {
		common.HTMXResponse(response, templates, "uri.html", uri)
	} else {
		common.JSONResponse(response, uri)
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
		common.Fail(response, request, "Echec de la communication avec le noeud LND", err)
		return
	}

	dto := toNodeInfoDTO(info)
	if common.IsHTMX(request) {
		common.HTMXResponse(response, templates, "nodeInfo.html", dto)
	} else {
		common.JSONResponse(response, dto)
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

	if common.IsHTMX(request) {
		common.HTMXMessageOk(response, "Peer successfully added.")
	} else {
		common.OkNoContent(response)
	}
}

// create a channel
func (h *Handler) HandleOpenChannel(response http.ResponseWriter, request *http.Request) {
	if err := request.ParseForm(); err != nil {
		common.FailCheck(response, request, "Parse error", err)
		return
	}

	id, ok := nodeIDFromRequest(response, request, "Pas d'id transmis")
	if !ok {
		return
	}
	amount, err := strconv.ParseInt(request.FormValue("amount"), 10, 64)
	if err != nil {
		common.FailCheck(response, request, "Amount value incorrect", err)
		return
	}

	if err := h.channelService.OpenChannel(request.Context(), id, request.FormValue("pubKey"), amount); err != nil {
		failUseCase(response, request, "Fail to create the channel.", err)
		return
	}

	if common.IsHTMX(request) {
		common.HTMXMessageOk(response, "Channel successfully created.")
	} else {
		common.OkNoContent(response)
	}
}

// Create an invoice
func (h *Handler) HandleCreateInvoice(response http.ResponseWriter, request *http.Request) {
	if err := request.ParseForm(); err != nil {
		common.FailCheck(response, request, "Parse error", err)
		return
	}

	id, ok := nodeIDFromRequest(response, request, "Pas d'id transmis")
	if !ok {
		return
	}
	amount, err := strconv.ParseInt(request.FormValue("amount"), 10, 64)
	if err != nil {
		common.FailCheck(response, request, "Amount value incorrect", err)
		return
	}

	paymentRequest, err := h.channelService.CreateInvoice(request.Context(), id, request.FormValue("memo"), amount)
	if err != nil {
		failUseCase(response, request, "Fail to create the invoice.", err)
		return
	}

	if common.IsHTMX(request) {
		common.HTMXResponse(response, templates, "paymentRequest.html", paymentRequest)
	} else {
		common.JSONResponse(response, paymentRequest)
	}
}

// Make the payment
func (h *Handler) HandleMakePayment(response http.ResponseWriter, request *http.Request) {
	if err := request.ParseForm(); err != nil {
		common.FailCheck(response, request, "Error parsing form", err)
		return
	}

	id, ok := nodeIDFromRequest(response, request, "Missing ID")
	if !ok {
		return
	}
	paymentRequest := request.FormValue("paymentRequest")
	if paymentRequest == "" {
		common.FailCheck(response, request, "Missing payment request", nil)
		return
	}

	// pay the invoice (async updates via SSE)
	if err := h.channelService.Pay(request.Context(), id, paymentRequest); err != nil {
		failUseCase(response, request, "Fail to pay the invoice.", err)
		return
	}

	// TODO have a streamId to return ?
	if common.IsHTMX(request) {
		common.HTMXMessageOk(response, "Processing in progress")
	} else {
		common.OkNoContent(response)
	}
}

// Update name of the node & color
func (h *Handler) HandleUpdateNodeAlias(response http.ResponseWriter, request *http.Request) {
	if err := request.ParseForm(); err != nil {
		common.FailCheck(response, request, "Parse error", err)
		return
	}

	alias := request.FormValue("alias")
	if alias == "" {
		common.FailCheck(response, request, "Missing alias", nil)
		return
	}
	color := request.FormValue("color")
	if color == "" {
		common.FailCheck(response, request, "Missing color", nil)
		return
	}
	id, ok := nodeIDFromRequest(response, request, "Missing ID")
	if !ok {
		return
	}

	if err := h.nodeService.UpdateAppearance(request.Context(), id, alias, color); err != nil {
		common.Fail(response, request, "Modifications fail.", err)
		return
	}

	if common.IsHTMX(request) {
		common.HTMXMessageOk(response, "Modifications successfully applied.")
	} else {
		common.OkNoContent(response)
	}
}

// HandleListChannels lists the open and pending channels of a node.
func (h *Handler) HandleListChannels(response http.ResponseWriter, request *http.Request) {
	id, ok := nodeIDFromRequest(response, request, "Missing or invalid node id.")
	if !ok {
		return
	}

	channels, err := h.channelService.ListChannels(request.Context(), id)
	if err != nil {
		common.Fail(response, request, "Unable to list the channels.", err)
		return
	}

	dtos := toChannelDTOs(channels)
	if common.IsHTMX(request) {
		common.HTMXResponse(response, templates, "channels.html", channelsView{NodeID: id.String(), Channels: dtos})
	} else {
		common.JSONResponse(response, dtos)
	}
}

// HandleGetBalance returns the on-chain and channel balances of a node.
func (h *Handler) HandleGetBalance(response http.ResponseWriter, request *http.Request) {
	id, ok := nodeIDFromRequest(response, request, "Missing or invalid node id.")
	if !ok {
		return
	}

	balance, err := h.nodeService.GetBalance(request.Context(), id)
	if err != nil {
		if errors.Is(err, domain.ErrWalletNotReady) {
			common.FailCheck(response, request, "The wallet is not unlocked.", err)
			return
		}
		common.Fail(response, request, "Unable to get the balance.", err)
		return
	}

	dto := toBalanceDTO(balance)
	if common.IsHTMX(request) {
		common.HTMXResponse(response, templates, "balance.html", dto)
	} else {
		common.JSONResponse(response, dto)
	}
}

// HandleCloseChannel closes a channel, cooperatively by default or by force with force=true.
func (h *Handler) HandleCloseChannel(response http.ResponseWriter, request *http.Request) {
	if err := request.ParseForm(); err != nil {
		common.FailCheck(response, request, "Invalid request.", nil)
		return
	}
	id, ok := nodeIDFromRequest(response, request, "Missing or invalid node id.")
	if !ok {
		return
	}
	force := false
	if raw := request.FormValue("force"); raw != "" {
		parsed, err := strconv.ParseBool(raw)
		if err != nil {
			common.FailCheck(response, request, "Invalid force value.", err)
			return
		}
		force = parsed
	}

	txid, err := h.channelService.CloseChannel(request.Context(), id, request.FormValue("channelPoint"), force)
	switch {
	case err == nil:
		response.Header().Set("HX-Trigger", "channel-closed")
		if common.IsHTMX(request) {
			common.HTMXMessageOk(response, "Channel closing, tx "+truncate(txid, 8))
		} else {
			common.JSONResponse(response, closeChannelDTO{ClosingTxID: txid})
		}
	case errors.Is(err, domain.ErrInvalidChannelPoint):
		common.FailCheck(response, request, "Invalid channel point.", err)
	case errors.Is(err, domain.ErrChannelNotFound):
		common.FailCheck(response, request, "Channel not found.", err)
	case errors.Is(err, domain.ErrChannelNotClosable):
		common.FailCheck(response, request, "This channel cannot be closed.", err)
	case errors.Is(err, domain.ErrChannelPeerOffline):
		common.FailCheck(response, request, "The peer is offline, use force close.", err)
	default:
		common.Fail(response, request, "Unable to close the channel.", err)
	}
}
