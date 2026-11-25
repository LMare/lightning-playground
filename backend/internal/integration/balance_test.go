package integration

import (
	"encoding/json"
	"errors"
	"net/http"
	"testing"

	lnrpc "github.com/Lmare/lightning-playground/backend/gRPC/github.com/lightningnetwork/lnd/lnrpc"
	"github.com/Lmare/lightning-playground/backend/internal/testsupport"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
)

const balancePath = "/api/v1/lightning/balance"

func mockBalances(h *testsupport.TestHarness, locked int64) {
	h.GrpcFactory.Lightning.On("WalletBalance", mock.Anything, mock.Anything).Return(&lnrpc.WalletBalanceResponse{
		ConfirmedBalance: 1000, UnconfirmedBalance: 250, LockedBalance: locked,
	}, nil)
	h.GrpcFactory.Lightning.On("ChannelBalance", mock.Anything, mock.Anything).Return(&lnrpc.ChannelBalanceResponse{
		LocalBalance:            &lnrpc.Amount{Sat: 600},
		RemoteBalance:           &lnrpc.Amount{Sat: 390},
		PendingOpenLocalBalance: &lnrpc.Amount{Sat: 50},
		UnsettledLocalBalance:   &lnrpc.Amount{Sat: 10},
	}, nil)
}

func assertNoBalanceCall(t *testing.T, h *testsupport.TestHarness) {
	h.GrpcFactory.Lightning.AssertNotCalled(t, "WalletBalance", mock.Anything, mock.Anything)
	h.GrpcFactory.Lightning.AssertNotCalled(t, "ChannelBalance", mock.Anything, mock.Anything)
}

func TestBalance_JSON(t *testing.T) {
	h := testsupport.NewTestHarness()
	stateIs(h, lnrpc.WalletState_SERVER_ACTIVE)
	mockBalances(h, 0)

	rr := doPlain(h, http.MethodGet, balancePath+"?id=lnd-0", nil)

	assert.Equal(t, http.StatusOK, rr.Code)
	var raw map[string]any
	assert.NoError(t, json.Unmarshal(rr.Body.Bytes(), &raw))
	assert.Equal(t, map[string]any{
		"onChain": map[string]any{
			"confirmedSat": 1000.0, "unconfirmedSat": 250.0, "totalSat": 1250.0,
			"lockedSat": 0.0, "reservedAnchorSat": 0.0,
		},
		"channels": map[string]any{
			"localSat": 600.0, "remoteSat": 390.0,
			"pendingOpenLocalSat": 50.0, "pendingOpenRemoteSat": 0.0,
			"unsettledLocalSat": 10.0, "unsettledRemoteSat": 0.0,
			"totalLocalSat": 660.0,
		},
		"totalSat": 1910.0,
	}, raw)
}

func TestBalance_HTMX(t *testing.T) {
	h := testsupport.NewTestHarness()
	stateIs(h, lnrpc.WalletState_SERVER_ACTIVE)
	mockBalances(h, 0)

	rr := doHTMX(h, http.MethodGet, balancePath+"?id=lnd-0", nil)

	assert.Equal(t, http.StatusOK, rr.Code)
	assert.Contains(t, rr.Header().Get("Content-Type"), "text/html")
	body := rr.Body.String()
	for _, s := range []string{
		`data-onchain-confirmed="1000"`, `data-onchain-unconfirmed="250"`, `data-onchain-total="1250"`,
		`data-channels-local="600"`, `data-channels-remote="390"`, `data-channels-total="660"`,
		`data-total="1910"`,
	} {
		assert.Contains(t, body, s)
	}
	assert.NotContains(t, body, "Locked")

	h2 := testsupport.NewTestHarness()
	stateIs(h2, lnrpc.WalletState_SERVER_ACTIVE)
	mockBalances(h2, 42)
	rr = doHTMX(h2, http.MethodGet, balancePath+"?id=lnd-0", nil)
	assert.Contains(t, rr.Body.String(), "Locked")
}

func TestBalance_WalletNotReady(t *testing.T) {
	h := testsupport.NewTestHarness()
	stateIs(h, lnrpc.WalletState_LOCKED)
	mockBalances(h, 0)

	rr := doPlain(h, http.MethodGet, balancePath+"?id=lnd-0", nil)
	assert.Equal(t, http.StatusBadRequest, rr.Code)
	assert.Contains(t, rr.Body.String(), "The wallet is not unlocked.")

	rr = doHTMX(h, http.MethodGet, balancePath+"?id=lnd-0", nil)
	assert.Contains(t, rr.Body.String(), "The wallet is not unlocked.")
	assert.Contains(t, rr.Body.String(), "❌")
	assert.NotContains(t, rr.Body.String(), "data-total")
	assertNoBalanceCall(t, h)
}

func TestBalance_BadID(t *testing.T) {
	for _, q := range []string{"", "?id=../etc"} {
		h := testsupport.NewTestHarness()
		stateIs(h, lnrpc.WalletState_SERVER_ACTIVE)
		mockBalances(h, 0)

		rr := doPlain(h, http.MethodGet, balancePath+q, nil)

		assert.Equal(t, http.StatusBadRequest, rr.Code, q)
		assertNoBalanceCall(t, h)
		h.GrpcFactory.State.AssertNotCalled(t, "GetState", mock.Anything, mock.Anything)
	}
}

func TestBalance_LndErrors(t *testing.T) {
	boom := errors.New("lnd down")
	cases := map[string]func(h *testsupport.TestHarness){
		"state": func(h *testsupport.TestHarness) {
			h.GrpcFactory.State.On("GetState", mock.Anything, mock.Anything).Return(nil, boom)
		},
		"wallet balance": func(h *testsupport.TestHarness) {
			stateIs(h, lnrpc.WalletState_SERVER_ACTIVE)
			h.GrpcFactory.Lightning.On("WalletBalance", mock.Anything, mock.Anything).Return(nil, boom)
		},
		"channel balance": func(h *testsupport.TestHarness) {
			stateIs(h, lnrpc.WalletState_SERVER_ACTIVE)
			h.GrpcFactory.Lightning.On("WalletBalance", mock.Anything, mock.Anything).Return(&lnrpc.WalletBalanceResponse{}, nil)
			h.GrpcFactory.Lightning.On("ChannelBalance", mock.Anything, mock.Anything).Return(nil, boom)
		},
	}
	for name, setup := range cases {
		t.Run(name, func(t *testing.T) {
			h := testsupport.NewTestHarness()
			setup(h)

			rr := doPlain(h, http.MethodGet, balancePath+"?id=lnd-0", nil)
			assert.Equal(t, http.StatusInternalServerError, rr.Code)
			assert.Contains(t, rr.Body.String(), "Unable to get the balance.")

			rr = doHTMX(h, http.MethodGet, balancePath+"?id=lnd-0", nil)
			assert.Contains(t, rr.Body.String(), "Unable to get the balance.")
			assert.NotEmpty(t, rr.Body.String())
			assert.NotContains(t, rr.Body.String(), "data-total")
		})
	}
}

func TestBalance_MethodNotAllowed(t *testing.T) {
	h := testsupport.NewTestHarness()
	stateIs(h, lnrpc.WalletState_SERVER_ACTIVE)
	mockBalances(h, 0)

	rr := doPlain(h, http.MethodPost, balancePath+"?id=lnd-0", nil)

	assert.Equal(t, http.StatusMethodNotAllowed, rr.Code)
	assertNoBalanceCall(t, h)
}
