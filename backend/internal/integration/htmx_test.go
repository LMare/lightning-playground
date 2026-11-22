package integration

import (
	"net/http"
	"net/http/httptest"
	"net/url"
	"strings"
	"testing"

	lnrpc "github.com/Lmare/lightning-playground/backend/gRPC/github.com/lightningnetwork/lnd/lnrpc"
	"github.com/Lmare/lightning-playground/backend/internal/business/user/domain"
	"github.com/Lmare/lightning-playground/backend/internal/testsupport"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
)

// doHTMX sends the request the way htmx does.
func doHTMX(h *testsupport.TestHarness, method, path string, form url.Values) *httptest.ResponseRecorder {
	var req *http.Request
	if form != nil {
		req = httptest.NewRequest(method, path, strings.NewReader(form.Encode()))
		req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	} else {
		req = httptest.NewRequest(method, path, nil)
	}
	req.Header.Set("HX-Request", "true")
	rr := httptest.NewRecorder()
	h.Router.ServeHTTP(rr, req)
	return rr
}

func TestHTMX_ListUsers(t *testing.T) {
	h := testsupport.NewTestHarness()
	h.MockUserRepo.On("FindAll", mock.Anything).Return([]domain.UserModel{{Nom: "Doe", Prenom: "John", Age: 30}}, nil)

	rr := doHTMX(h, http.MethodGet, "/users", nil)

	assert.Equal(t, http.StatusOK, rr.Code)
	assert.Equal(t, "text/html", rr.Header().Get("Content-Type"))
	assert.Contains(t, rr.Body.String(), "<td>Doe</td>")
	assert.Contains(t, rr.Body.String(), "<td>John</td>")
}

func TestHTMX_ListNodes(t *testing.T) {
	h := testsupport.NewTestHarness()
	h.MockNodeRepo.On("GetNodesIds").Return([]string{"lnd-0"}, nil)
	h.GrpcFactory.State.On("GetState", mock.Anything, mock.Anything).Return(&lnrpc.GetStateResponse{State: lnrpc.WalletState_SERVER_ACTIVE}, nil)
	h.GrpcFactory.Lightning.On("GetInfo", mock.Anything, mock.Anything).Return(&lnrpc.GetInfoResponse{Alias: "alice", Color: "#ff0000"}, nil)

	rr := doHTMX(h, http.MethodGet, "/lightning/nodes", nil)

	assert.Equal(t, http.StatusOK, rr.Code)
	assert.Contains(t, rr.Body.String(), `data-id="lnd-0"`)
	assert.Contains(t, rr.Body.String(), "alice")
	assert.Contains(t, rr.Body.String(), `data-wallet-state="server_active"`)
	assert.Contains(t, rr.Body.String(), "Unlocked")
}

func TestHTMX_ListNodes_LockedNode(t *testing.T) {
	h := testsupport.NewTestHarness()
	h.MockNodeRepo.On("GetNodesIds").Return([]string{"lnd-0"}, nil)
	h.GrpcFactory.State.On("GetState", mock.Anything, mock.Anything).Return(&lnrpc.GetStateResponse{State: lnrpc.WalletState_LOCKED}, nil)

	rr := doHTMX(h, http.MethodGet, "/lightning/nodes", nil)

	assert.Equal(t, http.StatusOK, rr.Code)
	assert.Contains(t, rr.Body.String(), `data-id="lnd-0"`)
	assert.Contains(t, rr.Body.String(), `data-wallet-state="locked"`)
	assert.Contains(t, rr.Body.String(), "Locked")
}

func TestHTMX_NodeInfo(t *testing.T) {
	h := testsupport.NewTestHarness()
	h.GrpcFactory.Lightning.On("GetInfo", mock.Anything, mock.Anything).Return(&lnrpc.GetInfoResponse{
		Alias: "alice", Chains: []*lnrpc.Chain{{Network: "simnet"}}, BlockHeight: 42,
	}, nil)

	rr := doHTMX(h, http.MethodGet, "/lightning/nodeInfo?id=lnd-0", nil)

	assert.Equal(t, http.StatusOK, rr.Code)
	assert.Contains(t, rr.Body.String(), "Network : simnet")
	assert.Contains(t, rr.Body.String(), "Block height : 42")
}

func TestHTMX_UriUsesTheTruncateHelper(t *testing.T) {
	h := testsupport.NewTestHarness()
	uri := "0123456789abcdef@1.2.3.4:9735"
	h.GrpcFactory.Lightning.On("GetInfo", mock.Anything, mock.Anything).Return(&lnrpc.GetInfoResponse{Uris: []string{uri}}, nil)

	rr := doHTMX(h, http.MethodGet, "/lightning/uri?id=lnd-0", nil)

	assert.Equal(t, http.StatusOK, rr.Code)
	assert.Contains(t, rr.Body.String(), "0123456...9abcdef@1.2.3.4:9735")
}

func TestHTMX_Messages(t *testing.T) {
	h := testsupport.NewTestHarness()
	h.GrpcFactory.Lightning.On("ConnectPeer", mock.Anything, mock.Anything).Return(&lnrpc.ConnectPeerResponse{}, nil)

	ok := doHTMX(h, http.MethodPost, "/lightning/peer", url.Values{"id": {"lnd-0"}, "uri": {"02ab@h:9735"}})
	assert.Equal(t, http.StatusOK, ok.Code)
	assert.Contains(t, ok.Body.String(), "✅ Peer successfully added.")

	ko := doHTMX(h, http.MethodPost, "/lightning/peer", url.Values{"id": {"bad id"}})
	assert.Contains(t, ko.Body.String(), "❌")
}
