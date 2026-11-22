package integration

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"net/url"
	"strings"
	"testing"

	lnrpc "github.com/Lmare/lightning-playground/backend/gRPC/github.com/lightningnetwork/lnd/lnrpc"
	"github.com/Lmare/lightning-playground/backend/internal/testsupport"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

func do(h *testsupport.TestHarness, method, path string, form url.Values) *httptest.ResponseRecorder {
	var req *http.Request
	if form != nil {
		req = httptest.NewRequest(method, path, strings.NewReader(form.Encode()))
		req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	} else {
		req = httptest.NewRequest(method, path, nil)
	}
	rr := httptest.NewRecorder()
	h.Router.ServeHTTP(rr, req)
	return rr
}

func stateResp(s lnrpc.WalletState) *lnrpc.GetStateResponse {
	return &lnrpc.GetStateResponse{State: s}
}

func TestListNodes_JSON(t *testing.T) {
	h := testsupport.NewTestHarness()
	h.MockNodeRepo.On("GetNodesIds").Return([]string{"lnd-0"}, nil)
	h.GrpcFactory.State.On("GetState", mock.Anything, mock.Anything).Return(stateResp(lnrpc.WalletState_SERVER_ACTIVE), nil)
	h.GrpcFactory.Lightning.On("GetInfo", mock.Anything, mock.Anything).Return(&lnrpc.GetInfoResponse{Alias: "alice", Color: "#ff0000"}, nil)

	rr := do(h, http.MethodGet, "/lightning/nodes", nil)

	assert.Equal(t, http.StatusOK, rr.Code)
	assert.JSONEq(t, `[{"id":"lnd-0","alias":"alice","color":"#ff0000","walletState":"server_active","walletLabel":"Unlocked"}]`, rr.Body.String())
	assert.Equal(t, "lnd-0.lnd.test:10009", h.GrpcFactory.Connections[0].Address)
}

func TestListNodes_LockedNodeIsListedWithoutGetInfo(t *testing.T) {
	h := testsupport.NewTestHarness()
	h.MockNodeRepo.On("GetNodesIds").Return([]string{"lnd-0"}, nil)
	h.GrpcFactory.State.On("GetState", mock.Anything, mock.Anything).Return(stateResp(lnrpc.WalletState_LOCKED), nil)

	rr := do(h, http.MethodGet, "/lightning/nodes", nil)

	assert.Equal(t, http.StatusOK, rr.Code)
	assert.JSONEq(t, `[{"id":"lnd-0","alias":"","color":"","walletState":"locked","walletLabel":"Locked"}]`, rr.Body.String())
	h.GrpcFactory.Lightning.AssertNotCalled(t, "GetInfo", mock.Anything, mock.Anything)
}

func TestListNodes_StartingStates(t *testing.T) {
	tests := []struct {
		state       lnrpc.WalletState
		wantState   string
		getInfoCall bool
	}{
		{lnrpc.WalletState_UNLOCKED, "unlocked", false},
		{lnrpc.WalletState_WAITING_TO_START, "waiting_to_start", false},
		{lnrpc.WalletState_RPC_ACTIVE, "rpc_active", true},
	}
	for _, tt := range tests {
		t.Run(tt.wantState, func(t *testing.T) {
			h := testsupport.NewTestHarness()
			h.MockNodeRepo.On("GetNodesIds").Return([]string{"lnd-0"}, nil)
			h.GrpcFactory.State.On("GetState", mock.Anything, mock.Anything).Return(stateResp(tt.state), nil)
			h.GrpcFactory.Lightning.On("GetInfo", mock.Anything, mock.Anything).Return(&lnrpc.GetInfoResponse{}, nil)

			rr := do(h, http.MethodGet, "/lightning/nodes", nil)

			assert.Equal(t, http.StatusOK, rr.Code)
			var got []map[string]any
			assert.NoError(t, json.Unmarshal(rr.Body.Bytes(), &got))
			if assert.Len(t, got, 1) {
				assert.Equal(t, tt.wantState, got[0]["walletState"])
				assert.Equal(t, "Starting", got[0]["walletLabel"])
			}
			if tt.getInfoCall {
				h.GrpcFactory.Lightning.AssertCalled(t, "GetInfo", mock.Anything, mock.Anything)
			} else {
				h.GrpcFactory.Lightning.AssertNotCalled(t, "GetInfo", mock.Anything, mock.Anything)
			}
		})
	}
}

func TestListNodes_UnreachableWhenDialFails(t *testing.T) {
	h := testsupport.NewTestHarness()
	h.MockNodeRepo.On("GetNodesIds").Return([]string{"lnd-0"}, nil)
	h.GrpcFactory.Err = assert.AnError

	rr := do(h, http.MethodGet, "/lightning/nodes", nil)

	assert.Equal(t, http.StatusOK, rr.Code)
	assert.JSONEq(t, `[{"id":"lnd-0","alias":"","color":"","walletState":"unreachable","walletLabel":"Unreachable"}]`, rr.Body.String())
}

func TestListNodes_UnreachableWhenGetStateFails(t *testing.T) {
	h := testsupport.NewTestHarness()
	h.MockNodeRepo.On("GetNodesIds").Return([]string{"lnd-0"}, nil)
	h.GrpcFactory.State.On("GetState", mock.Anything, mock.Anything).Return(nil, status.Error(codes.Unavailable, "down"))

	rr := do(h, http.MethodGet, "/lightning/nodes", nil)

	assert.Equal(t, http.StatusOK, rr.Code)
	assert.JSONEq(t, `[{"id":"lnd-0","alias":"","color":"","walletState":"unreachable","walletLabel":"Unreachable"}]`, rr.Body.String())
}

func TestListNodes_ReadyButGetInfoFailsKeepsTheNode(t *testing.T) {
	h := testsupport.NewTestHarness()
	h.MockNodeRepo.On("GetNodesIds").Return([]string{"lnd-0"}, nil)
	h.GrpcFactory.State.On("GetState", mock.Anything, mock.Anything).Return(stateResp(lnrpc.WalletState_SERVER_ACTIVE), nil)
	h.GrpcFactory.Lightning.On("GetInfo", mock.Anything, mock.Anything).Return(nil, assert.AnError)

	rr := do(h, http.MethodGet, "/lightning/nodes", nil)

	assert.Equal(t, http.StatusOK, rr.Code)
	assert.JSONEq(t, `[{"id":"lnd-0","alias":"","color":"","walletState":"server_active","walletLabel":"Unlocked"}]`, rr.Body.String())
}

func TestListNodes_MixedStatesKeepRepositoryOrder(t *testing.T) {
	h := testsupport.NewTestHarness()
	h.MockNodeRepo.On("GetNodesIds").Return([]string{"lnd-0", "lnd-1"}, nil)
	locked := &testsupport.MockStateClient{}
	locked.On("GetState", mock.Anything, mock.Anything).Return(stateResp(lnrpc.WalletState_LOCKED), nil)
	h.GrpcFactory.State.On("GetState", mock.Anything, mock.Anything).Return(stateResp(lnrpc.WalletState_SERVER_ACTIVE), nil)
	h.GrpcFactory.StateByAddress = map[string]*testsupport.MockStateClient{"lnd-1.lnd.test:10009": locked}
	h.GrpcFactory.Lightning.On("GetInfo", mock.Anything, mock.Anything).Return(&lnrpc.GetInfoResponse{Alias: "alice", Color: "#ff0000"}, nil)

	rr := do(h, http.MethodGet, "/lightning/nodes", nil)

	assert.Equal(t, http.StatusOK, rr.Code)
	assert.JSONEq(t, `[
		{"id":"lnd-0","alias":"alice","color":"#ff0000","walletState":"server_active","walletLabel":"Unlocked"},
		{"id":"lnd-1","alias":"","color":"","walletState":"locked","walletLabel":"Locked"}]`, rr.Body.String())
}

func TestListNodes_InvalidIdIsStillSkipped(t *testing.T) {
	h := testsupport.NewTestHarness()
	h.MockNodeRepo.On("GetNodesIds").Return([]string{"bad id", "lnd-0"}, nil)
	h.GrpcFactory.State.On("GetState", mock.Anything, mock.Anything).Return(stateResp(lnrpc.WalletState_LOCKED), nil)

	rr := do(h, http.MethodGet, "/lightning/nodes", nil)

	assert.Equal(t, http.StatusOK, rr.Code)
	assert.JSONEq(t, `[{"id":"lnd-0","alias":"","color":"","walletState":"locked","walletLabel":"Locked"}]`, rr.Body.String())
}

func TestNodeInfoAndURI_JSON(t *testing.T) {
	h := testsupport.NewTestHarness()
	h.GrpcFactory.Lightning.On("GetInfo", mock.Anything, mock.Anything).Return(&lnrpc.GetInfoResponse{
		Alias: "alice", NumPeers: 1, Uris: []string{"k@h:9735"}, Chains: []*lnrpc.Chain{{Network: "simnet"}},
	}, nil)

	rr := do(h, http.MethodGet, "/lightning/nodeInfo?id=lnd-0", nil)
	assert.Equal(t, http.StatusOK, rr.Code)
	var info map[string]any
	assert.NoError(t, json.Unmarshal(rr.Body.Bytes(), &info))
	assert.Equal(t, "alice", info["alias"])
	assert.Equal(t, "simnet", info["network"])
	assert.EqualValues(t, 1, info["numPeers"])

	rr = do(h, http.MethodGet, "/lightning/uri?id=lnd-0", nil)
	assert.Equal(t, http.StatusOK, rr.Code)
	assert.JSONEq(t, `"k@h:9735"`, rr.Body.String())
}

func TestNodeInfo_LndDown(t *testing.T) {
	h := testsupport.NewTestHarness()
	h.GrpcFactory.Lightning.On("GetInfo", mock.Anything, mock.Anything).Return(nil, assert.AnError)

	assert.Equal(t, http.StatusInternalServerError, do(h, http.MethodGet, "/lightning/nodeInfo?id=lnd-0", nil).Code)
}

func TestAddPeer(t *testing.T) {
	h := testsupport.NewTestHarness()
	h.GrpcFactory.Lightning.On("ConnectPeer", mock.Anything, mock.Anything).Return(&lnrpc.ConnectPeerResponse{}, nil)

	rr := do(h, http.MethodPost, "/lightning/peer", url.Values{"id": {"lnd-0"}, "uri": {"02ab@lnd-1:9735"}})
	assert.Equal(t, http.StatusNoContent, rr.Code)

	// a malformed URI is a client error (it used to panic)
	rr = do(h, http.MethodPost, "/lightning/peer", url.Values{"id": {"lnd-0"}, "uri": {"garbage"}})
	assert.Equal(t, http.StatusBadRequest, rr.Code)
}

func TestOpenChannel_NonPositiveAmount(t *testing.T) {
	h := testsupport.NewTestHarness()

	rr := do(h, http.MethodPost, "/lightning/channel", url.Values{"id": {"lnd-0"}, "pubKey": {"02ab"}, "amount": {"0"}})

	assert.Equal(t, http.StatusBadRequest, rr.Code)
}

func TestOpenChannel(t *testing.T) {
	h := testsupport.NewTestHarness()
	h.GrpcFactory.Lightning.On("OpenChannelSync", mock.Anything, mock.Anything).Return(&lnrpc.ChannelPoint{}, nil)

	rr := do(h, http.MethodPost, "/lightning/channel", url.Values{"id": {"lnd-0"}, "pubKey": {"02ab"}, "amount": {"1000"}})

	assert.Equal(t, http.StatusNoContent, rr.Code)
}

func TestCreateInvoice_JSON(t *testing.T) {
	h := testsupport.NewTestHarness()
	h.GrpcFactory.Lightning.On("AddInvoice", mock.Anything, mock.Anything).Return(&lnrpc.AddInvoiceResponse{PaymentRequest: "lnbc1"}, nil)

	rr := do(h, http.MethodPost, "/lightning/invoice", url.Values{"id": {"lnd-0"}, "memo": {"m"}, "amount": {"10"}})

	assert.Equal(t, http.StatusOK, rr.Code)
	assert.JSONEq(t, `"lnbc1"`, rr.Body.String())
}

func TestUpdateAlias(t *testing.T) {
	h := testsupport.NewTestHarness()
	h.GrpcFactory.Lightning.On("SetAlias", mock.Anything, mock.Anything).Return(&lnrpc.SetAliasResponse{}, nil)
	h.GrpcFactory.Lightning.On("SetColor", mock.Anything, mock.Anything).Return(&lnrpc.SetColorResponse{}, nil)

	rr := do(h, http.MethodPut, "/lightning/alias", url.Values{"id": {"lnd-0"}, "alias": {"bob"}, "color": {"#000000"}})
	assert.Equal(t, http.StatusNoContent, rr.Code)

	// regression: a failing LND used to answer an error AND a success
	h = testsupport.NewTestHarness()
	h.GrpcFactory.Lightning.On("SetAlias", mock.Anything, mock.Anything).Return(nil, assert.AnError)
	rr = do(h, http.MethodPut, "/lightning/alias", url.Values{"id": {"lnd-0"}, "alias": {"bob"}, "color": {"#000000"}})
	assert.Equal(t, http.StatusInternalServerError, rr.Code)
	assert.NotContains(t, rr.Body.String(), "success")
}

func TestMakePayment(t *testing.T) {
	h := testsupport.NewTestHarness()
	h.GrpcFactory.Router.On("EstimateRouteFee", mock.Anything, mock.Anything).Return(nil, assert.AnError)
	h.GrpcFactory.Router.On("SendPaymentV2", mock.Anything, mock.Anything).Return(&testsupport.PaymentStream{
		Payments: []*lnrpc.Payment{{ValueSat: 10, Status: lnrpc.Payment_SUCCEEDED}},
	}, nil)

	rr := do(h, http.MethodPost, "/lightning/paiment", url.Values{"id": {"lnd-0"}, "paymentRequest": {"lnbc1"}})

	assert.Equal(t, http.StatusNoContent, rr.Code)
}
