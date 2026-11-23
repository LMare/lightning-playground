package integration

import (
	"bytes"
	"errors"
	"io"
	"log"
	"net/http"
	"net/http/httptest"
	"net/url"
	"os"
	"strings"
	"testing"

	lnrpc "github.com/Lmare/lightning-playground/backend/gRPC/github.com/lightningnetwork/lnd/lnrpc"
	"github.com/Lmare/lightning-playground/backend/internal/testsupport"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

const (
	unlockPath = "/lightning/wallet/unlock"
	pwd        = "Sup3r-S3cret-Pwd"
)

func doPlain(h *testsupport.TestHarness, method, path string, form url.Values) *httptest.ResponseRecorder {
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

func stateIs(h *testsupport.TestHarness, s lnrpc.WalletState) {
	h.GrpcFactory.State.On("GetState", mock.Anything, mock.Anything).Return(&lnrpc.GetStateResponse{State: s}, nil)
}

func unlockForm() url.Values { return url.Values{"id": {"lnd-0"}, "password": {pwd}} }

func assertNoUnlockCall(t *testing.T, h *testsupport.TestHarness) {
	h.GrpcFactory.WalletUnlocker.AssertNotCalled(t, "UnlockWallet", mock.Anything, mock.Anything)
}

func TestUnlock_Success_HTMX(t *testing.T) {
	h := testsupport.NewTestHarness()
	stateIs(h, lnrpc.WalletState_LOCKED)
	h.GrpcFactory.WalletUnlocker.On("UnlockWallet", mock.Anything, mock.MatchedBy(func(r *lnrpc.UnlockWalletRequest) bool {
		return string(r.WalletPassword) == pwd
	})).Return(&lnrpc.UnlockWalletResponse{}, nil)

	rr := doHTMX(h, http.MethodPost, unlockPath, unlockForm())

	assert.Equal(t, http.StatusOK, rr.Code)
	assert.Equal(t, "wallet-unlocked", rr.Header().Get("HX-Trigger"))
	assert.Contains(t, rr.Body.String(), "✅")
	assert.Contains(t, rr.Body.String(), "Wallet unlocked, the node is starting.")
	h.GrpcFactory.WalletUnlocker.AssertNumberOfCalls(t, "UnlockWallet", 1)
}

func TestUnlock_Success_NoHTMX(t *testing.T) {
	h := testsupport.NewTestHarness()
	stateIs(h, lnrpc.WalletState_LOCKED)
	h.GrpcFactory.WalletUnlocker.On("UnlockWallet", mock.Anything, mock.Anything).Return(&lnrpc.UnlockWalletResponse{}, nil)

	rr := doPlain(h, http.MethodPost, unlockPath, unlockForm())

	assert.Equal(t, http.StatusNoContent, rr.Code)
}

func TestUnlock_WrongPassword(t *testing.T) {
	newH := func() *testsupport.TestHarness {
		h := testsupport.NewTestHarness()
		stateIs(h, lnrpc.WalletState_LOCKED)
		h.GrpcFactory.WalletUnlocker.On("UnlockWallet", mock.Anything, mock.Anything).
			Return(nil, status.Error(codes.Unknown, "invalid passphrase for master public key"))
		return h
	}

	rr := doHTMX(newH(), http.MethodPost, unlockPath, unlockForm())
	assert.Contains(t, rr.Body.String(), "Wrong wallet password.")
	assert.Contains(t, rr.Body.String(), "❌")
	assert.Empty(t, rr.Header().Get("HX-Trigger"))

	rr = doPlain(newH(), http.MethodPost, unlockPath, unlockForm())
	assert.Equal(t, http.StatusBadRequest, rr.Code)
}

func TestUnlock_NotLocked(t *testing.T) {
	h := testsupport.NewTestHarness()
	stateIs(h, lnrpc.WalletState_SERVER_ACTIVE)

	rr := doHTMX(h, http.MethodPost, unlockPath, unlockForm())
	assert.Contains(t, rr.Body.String(), "The wallet is not locked.")
	assert.Empty(t, rr.Header().Get("HX-Trigger"))

	rr = doPlain(h, http.MethodPost, unlockPath, unlockForm())
	assert.Equal(t, http.StatusBadRequest, rr.Code)
	assertNoUnlockCall(t, h)
}

func TestUnlock_NonExisting(t *testing.T) {
	h := testsupport.NewTestHarness()
	stateIs(h, lnrpc.WalletState_NON_EXISTING)

	rr := doHTMX(h, http.MethodPost, unlockPath, unlockForm())
	assert.Contains(t, rr.Body.String(), "No wallet on this node, create it first.")

	rr = doPlain(h, http.MethodPost, unlockPath, unlockForm())
	assert.Equal(t, http.StatusBadRequest, rr.Code)
	assertNoUnlockCall(t, h)
}

func TestUnlock_Unreachable(t *testing.T) {
	h := testsupport.NewTestHarness()
	h.GrpcFactory.State.On("GetState", mock.Anything, mock.Anything).Return(nil, errors.New("connection refused"))

	rr := doHTMX(h, http.MethodPost, unlockPath, unlockForm())
	assert.Contains(t, rr.Body.String(), "Unable to reach the LND node.")
	assert.Empty(t, rr.Header().Get("HX-Trigger"))

	rr = doPlain(h, http.MethodPost, unlockPath, unlockForm())
	assert.Equal(t, http.StatusInternalServerError, rr.Code)
	assertNoUnlockCall(t, h)
}

func TestUnlock_MissingOrQueryStringPassword(t *testing.T) {
	cases := map[string]*httptest.ResponseRecorder{}
	h := testsupport.NewTestHarness()
	stateIs(h, lnrpc.WalletState_LOCKED)

	cases["missing"] = doPlain(h, http.MethodPost, unlockPath, url.Values{"id": {"lnd-0"}})
	cases["empty"] = doPlain(h, http.MethodPost, unlockPath, url.Values{"id": {"lnd-0"}, "password": {""}})
	cases["query only"] = doPlain(h, http.MethodPost, unlockPath+"?password="+pwd, url.Values{"id": {"lnd-0"}})

	for name, rr := range cases {
		assert.Equal(t, http.StatusBadRequest, rr.Code, name)
	}
	assertNoUnlockCall(t, h)

	rr := doHTMX(h, http.MethodPost, unlockPath+"?password="+pwd, url.Values{"id": {"lnd-0"}})
	assert.Contains(t, rr.Body.String(), "Missing password")
	assert.NotContains(t, rr.Body.String(), pwd)
}

func TestUnlock_InvalidID(t *testing.T) {
	h := testsupport.NewTestHarness()
	stateIs(h, lnrpc.WalletState_LOCKED)

	rr := doPlain(h, http.MethodPost, unlockPath, url.Values{"id": {"../x"}, "password": {pwd}})

	assert.Equal(t, http.StatusBadRequest, rr.Code)
	assertNoUnlockCall(t, h)
	h.GrpcFactory.State.AssertNotCalled(t, "GetState", mock.Anything, mock.Anything)
}

func TestUnlock_MethodNotAllowed(t *testing.T) {
	h := testsupport.NewTestHarness()

	rr := doPlain(h, http.MethodGet, unlockPath, nil)

	assert.Equal(t, http.StatusMethodNotAllowed, rr.Code)
}

// captureOutput records log and stdout output while f runs.
func captureOutput(t *testing.T, f func()) string {
	t.Helper()
	var logs bytes.Buffer
	oldLog, oldOut := log.Writer(), os.Stdout
	log.SetOutput(&logs)
	r, w, err := os.Pipe()
	assert.NoError(t, err)
	os.Stdout = w
	done := make(chan string)
	go func() { b, _ := io.ReadAll(r); done <- string(b) }()

	f()

	w.Close()
	os.Stdout = oldOut
	log.SetOutput(oldLog)
	return logs.String() + <-done
}

func TestUnlock_PasswordNeverLeaks(t *testing.T) {
	scenarios := map[string]func(h *testsupport.TestHarness){
		"success": func(h *testsupport.TestHarness) {
			stateIs(h, lnrpc.WalletState_LOCKED)
			h.GrpcFactory.WalletUnlocker.On("UnlockWallet", mock.Anything, mock.Anything).Return(&lnrpc.UnlockWalletResponse{}, nil)
		},
		"wrong password": func(h *testsupport.TestHarness) {
			stateIs(h, lnrpc.WalletState_LOCKED)
			h.GrpcFactory.WalletUnlocker.On("UnlockWallet", mock.Anything, mock.Anything).
				Return(nil, status.Error(codes.Unknown, "invalid passphrase for master public key"))
		},
		"unreachable": func(h *testsupport.TestHarness) {
			h.GrpcFactory.State.On("GetState", mock.Anything, mock.Anything).Return(nil, errors.New("connection refused"))
		},
	}
	for name, setup := range scenarios {
		t.Run(name, func(t *testing.T) {
			h := testsupport.NewTestHarness()
			setup(h)
			var body string
			out := captureOutput(t, func() {
				body = doHTMX(h, http.MethodPost, unlockPath, unlockForm()).Body.String()
				body += doPlain(h, http.MethodPost, unlockPath, unlockForm()).Body.String()
			})
			assert.NotContains(t, out, pwd)
			assert.NotContains(t, body, pwd)
		})
	}
}

func TestHTMX_ListNodes_UnlockButton(t *testing.T) {
	render := func(s lnrpc.WalletState) string {
		h := testsupport.NewTestHarness()
		h.MockNodeRepo.On("GetNodesIds").Return([]string{"lnd-0"}, nil)
		stateIs(h, s)
		h.GrpcFactory.Lightning.On("GetInfo", mock.Anything, mock.Anything).Return(&lnrpc.GetInfoResponse{Alias: "alice"}, nil)
		rr := doHTMX(h, http.MethodGet, "/lightning/nodes", nil)
		assert.Equal(t, http.StatusOK, rr.Code)
		return rr.Body.String()
	}

	assert.Contains(t, render(lnrpc.WalletState_LOCKED), `data-unlock-id="lnd-0"`)
	assert.NotContains(t, render(lnrpc.WalletState_SERVER_ACTIVE), "data-unlock-id")
	assert.Contains(t, render(lnrpc.WalletState_RPC_ACTIVE), "data-starting")
	assert.NotContains(t, render(lnrpc.WalletState_SERVER_ACTIVE), "data-starting")
}
