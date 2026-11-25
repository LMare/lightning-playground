package integration

import (
	"fmt"
	"net/http"
	"net/http/httptest"
	"net/url"
	"regexp"
	"testing"

	lnrpc "github.com/Lmare/lightning-playground/backend/gRPC/github.com/lightningnetwork/lnd/lnrpc"
	"github.com/Lmare/lightning-playground/backend/internal/business/user/domain"
	"github.com/Lmare/lightning-playground/backend/internal/testsupport"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
)

// tagContaining returns the first opening tag of body that contains needle ("" when absent).
func tagContaining(body, needle string) string {
	re := regexp.MustCompile(`<[^<>]*` + regexp.QuoteMeta(needle) + `[^<>]*>`)
	return re.FindString(body)
}

// rowHeader asserts that the <th> with the given id exists and is a row header (scope="row"),
// whatever the attribute order.
func assertRowHeader(t *testing.T, body, id string) {
	t.Helper()
	tag := regexp.MustCompile(`<th[^<>]*\sid="` + regexp.QuoteMeta(id) + `"[^<>]*>`).FindString(body)
	if assert.NotEmpty(t, tag, "row header %s", id) {
		assert.Contains(t, tag, `scope="row"`, "row header %s", id)
	}
}

// rowTag returns the opening <tr> of the node row with the given id.
func rowTag(body, id string) string {
	return tagContaining(body, `data-id="`+id+`"`)
}

// formBody returns the content of the <form> posting to path ("" when absent).
func formBody(body, verb, path string) string {
	re := regexp.MustCompile(`(?s)<form[^>]*hx-` + verb + `="` + regexp.QuoteMeta(path) + `"[^>]*>(.*?)</form>`)
	m := re.FindStringSubmatch(body)
	if m == nil {
		return ""
	}
	return m[1]
}

func mockNodes(h *testsupport.TestHarness, state lnrpc.WalletState, ids ...string) {
	h.MockNodeRepo.On("GetNodesIds").Return(ids, nil)
	h.GrpcFactory.State.On("GetState", mock.Anything, mock.Anything).Return(&lnrpc.GetStateResponse{State: state}, nil)
	h.GrpcFactory.Lightning.On("GetInfo", mock.Anything, mock.Anything).Return(&lnrpc.GetInfoResponse{Alias: "alice", Color: "#ff0000"}, nil)
}

// 16
func TestFragments_NodeRowsCarryTheirBehaviour(t *testing.T) {
	h := testsupport.NewTestHarness()
	mockNodes(h, lnrpc.WalletState_LOCKED, "lnd-0")

	body := doHTMX(h, http.MethodGet, "/lightning/nodes", nil).Body.String()

	row := rowTag(body, "lnd-0")
	assert.NotEmpty(t, row)
	assert.Contains(t, row, `hx-get="/node/panel?id=lnd-0"`)
	assert.Contains(t, row, `hx-target="#node-panel"`)
	assert.Contains(t, row, `tabindex="0"`)
	trigger := regexp.MustCompile(`hx-trigger="([^"]*)"`).FindStringSubmatch(row)
	if assert.NotNil(t, trigger, "the row needs an hx-trigger") {
		assert.Contains(t, trigger[1], "click")
		assert.Contains(t, trigger[1], "keyup")
	}

	unlock := tagContaining(body, `data-unlock-id="lnd-0"`)
	assert.NotEmpty(t, unlock, "a locked node has an unlock button")
	assert.Contains(t, unlock, "event.stopPropagation()")
}

// 17
func TestFragments_SelectedRowIsRenderedByTheServer(t *testing.T) {
	h := testsupport.NewTestHarness()
	mockNodes(h, lnrpc.WalletState_SERVER_ACTIVE, "lnd-0", "lnd-1")

	body := doHTMX(h, http.MethodGet, "/lightning/nodes?selected=lnd-1", nil).Body.String()

	selected := rowTag(body, "lnd-1")
	assert.Contains(t, selected, `class="selected"`)
	assert.Contains(t, selected, `aria-current="true"`)
	other := rowTag(body, "lnd-0")
	assert.NotEmpty(t, other)
	assert.NotContains(t, other, `class="selected"`)
	assert.NotContains(t, other, "aria-current")

	// without the parameter nothing is selected
	plain := doHTMX(h, http.MethodGet, "/lightning/nodes", nil).Body.String()
	assert.NotContains(t, plain, "aria-current")

	// JSON ignores the UI-only parameter
	with := do(h, http.MethodGet, "/lightning/nodes?selected=lnd-1", nil)
	without := do(h, http.MethodGet, "/lightning/nodes", nil)
	assert.Equal(t, http.StatusOK, with.Code)
	assert.Equal(t, without.Body.String(), with.Body.String())
}

// 18
func TestFragments_RGAA_Nodes(t *testing.T) {
	h := testsupport.NewTestHarness()
	mockNodes(h, lnrpc.WalletState_SERVER_ACTIVE, "lnd-0", "lnd-1")

	body := doHTMX(h, http.MethodGet, "/lightning/nodes", nil).Body.String()

	for i := 0; i < 2; i++ {
		row := fmt.Sprintf("table-nodes-body-row-%d", i)
		assertRowHeader(t, body, row)
		for _, col := range []string{"alias", "wallet", "unlock"} {
			assert.Contains(t, body, `headers="`+row+` `+col+`"`)
		}
		assert.NotContains(t, body, `headers="`+row+` id"`)
	}
}

// 19
func TestFragments_RGAA_Channels(t *testing.T) {
	h := testsupport.NewTestHarness()
	mockChannels(h)

	body := doHTMX(h, http.MethodGet, channelsPath+"?id=lnd-0", nil).Body.String()

	for i := 0; i < 3; i++ {
		row := fmt.Sprintf("table-channels-body-row-%d", i)
		assert.Contains(t, body, `id="`+row+`"`)
		for _, col := range []string{"capacity", "local", "remote", "status", "actions"} {
			assert.Contains(t, body, `headers="`+row+` channel-`+col+`"`, "row %d column %s", i, col)
		}
	}
}

func TestFragments_RGAA_Channels_EmptyRowHasNoHeaders(t *testing.T) {
	h := testsupport.NewTestHarness()
	h.GrpcFactory.Lightning.On("ListChannels", mock.Anything, mock.Anything).Return(&lnrpc.ListChannelsResponse{}, nil)
	h.GrpcFactory.Lightning.On("PendingChannels", mock.Anything, mock.Anything).Return(&lnrpc.PendingChannelsResponse{}, nil)

	body := doHTMX(h, http.MethodGet, channelsPath+"?id=lnd-0", nil).Body.String()

	assert.Contains(t, body, "No channel")
	assert.NotContains(t, body, "headers=")
}

// 20
func TestFragments_RGAA_Users(t *testing.T) {
	h := testsupport.NewTestHarness()
	h.MockUserRepo.On("FindAll", mock.Anything).Return([]domain.UserModel{{Nom: "Doe", Prenom: "John", Age: 30}}, nil)

	body := doHTMX(h, http.MethodGet, "/users", nil).Body.String()

	assertRowHeader(t, body, "table-persons-body-row-0")
	for _, col := range []string{"nom", "prenom", "age"} {
		assert.Contains(t, body, `headers="table-persons-body-row-0 `+col+`"`)
	}
}

// 21
func TestPanel_RendersSectionsAndForms(t *testing.T) {
	h := testsupport.NewTestHarness() // no mock: the panel makes no service call

	rr := doHTMX(h, http.MethodGet, "/node/panel?id=lnd-0", nil)

	assert.Equal(t, http.StatusOK, rr.Code)
	assert.Contains(t, rr.Header().Get("Content-Type"), "text/html")
	assert.Contains(t, rr.Header().Get("HX-Trigger"), "node-selected")
	body := rr.Body.String()
	assert.NotContains(t, body, "<!DOCTYPE")
	assert.NotContains(t, body, "hx-vals")

	for _, section := range []string{"nodeInfo", "uri", "balance", "channels"} {
		tag := tagContaining(body, `hx-get="/lightning/`+section+`?id=lnd-0"`)
		assert.NotEmpty(t, tag, section)
		assert.Regexp(t, `hx-trigger="[^"]*\bload\b`, tag, section)
	}
	assert.Contains(t, tagContaining(body, `hx-get="/lightning/nodeInfo?id=lnd-0"`), "alias-updated from:body")
	assert.Contains(t, tagContaining(body, `hx-get="/lightning/channels?id=lnd-0"`), "channel-closed from:body")
	balance := tagContaining(body, `hx-get="/lightning/balance?id=lnd-0"`)
	assert.Contains(t, balance, "channel-closed from:body")
	assert.Contains(t, balance, "click from:#balance-refresh")

	for _, path := range []string{"/lightning/peer", "/lightning/channel", "/lightning/invoice", "/lightning/paiment"} {
		form := formBody(body, "post", path)
		assert.NotEmpty(t, form, "form %s", path)
		assert.Contains(t, form, `name="id" value="lnd-0"`, "form %s", path)
	}

	assert.Contains(t, tagContaining(body, `id="selected-node"`), `value="lnd-0"`)
	slot := tagContaining(body, `id="node-info-slot"`)
	assert.NotEmpty(t, slot)
	assert.Contains(t, slot, "hx-swap-oob")
}

func TestPanel_InvalidIdAndMethod(t *testing.T) {
	h := testsupport.NewTestHarness()

	rr := doHTMX(h, http.MethodGet, "/node/panel?id="+url.QueryEscape("node!bad"), nil)
	assert.Contains(t, rr.Body.String(), "❌")
	assert.NotContains(t, rr.Body.String(), `id="selected-node"`)
	assert.Empty(t, rr.Header().Get("HX-Trigger"))

	assert.Equal(t, http.StatusMethodNotAllowed, doHTMX(h, http.MethodPost, "/node/panel?id=lnd-0", url.Values{}).Code)
}

// 22
func TestPages_Node_NoClientSidePatching(t *testing.T) {
	h := testsupport.NewTestHarness()

	body := doPlain(h, http.MethodGet, "/node", nil).Body.String()

	tbody := tagContaining(body, `id="table-nodes-body"`)
	assert.Contains(t, tbody, `hx-include="#selected-node"`)
	assert.Contains(t, tbody, "node-selected from:body")
}

// 23
func TestAlias_HTMXSuccessTriggersAliasUpdated(t *testing.T) {
	h := testsupport.NewTestHarness()
	h.GrpcFactory.Lightning.On("SetAlias", mock.Anything, mock.Anything).Return(&lnrpc.SetAliasResponse{}, nil)
	h.GrpcFactory.Lightning.On("SetColor", mock.Anything, mock.Anything).Return(&lnrpc.SetColorResponse{}, nil)
	form := url.Values{"id": {"lnd-0"}, "alias": {"bob"}, "color": {"#000000"}}

	ok := doHTMX(h, http.MethodPut, "/lightning/alias", form)
	assert.Contains(t, ok.Header().Get("HX-Trigger"), "alias-updated")

	invalid := doHTMX(h, http.MethodPut, "/lightning/alias", url.Values{"id": {"bad id"}, "alias": {"bob"}, "color": {"#000000"}})
	assert.Empty(t, invalid.Header().Get("HX-Trigger"))
	assert.Contains(t, invalid.Body.String(), "❌")

	plain := do(h, http.MethodPut, "/lightning/alias", form)
	assert.Equal(t, http.StatusNoContent, plain.Code)
}

func TestAlias_HTMXLndFailureDoesNotTrigger(t *testing.T) {
	h := testsupport.NewTestHarness()
	h.GrpcFactory.Lightning.On("SetAlias", mock.Anything, mock.Anything).Return(nil, assert.AnError)

	rr := doHTMX(h, http.MethodPut, "/lightning/alias", url.Values{"id": {"lnd-0"}, "alias": {"bob"}, "color": {"#000000"}})

	assert.Empty(t, rr.Header().Get("HX-Trigger"))
	assert.Contains(t, rr.Body.String(), "❌")
}

// 24
func TestFragments_NodeInfoRendersTheAliasForm(t *testing.T) {
	h := testsupport.NewTestHarness()
	h.GrpcFactory.Lightning.On("GetInfo", mock.Anything, mock.Anything).Return(&lnrpc.GetInfoResponse{
		Alias: "alice", Color: "#ff0000", Chains: []*lnrpc.Chain{{Network: "simnet"}}, BlockHeight: 42,
	}, nil)

	body := doHTMX(h, http.MethodGet, "/lightning/nodeInfo?id=lnd-0", nil).Body.String()

	assert.Contains(t, body, `hx-put="/lightning/alias"`)
	assert.Contains(t, body, `name="alias" value="alice"`)
	assert.Contains(t, body, `name="color" value="#ff0000"`)
	assert.Contains(t, body, `name="id" value="lnd-0"`)

	// JSON keeps its shape and carries no markup
	js := do(h, http.MethodGet, "/lightning/nodeInfo?id=lnd-0", nil).Body.String()
	assert.Contains(t, js, `"alias":"alice"`)
	assert.NotContains(t, js, "hx-put")
}

// 25
func TestDualEndpoints_VaryOnHXRequest(t *testing.T) {
	h := testsupport.NewTestHarness()
	mockNodes(h, lnrpc.WalletState_SERVER_ACTIVE, "lnd-0")
	h.MockUserRepo.On("FindAll", mock.Anything).Return([]domain.UserModel{{Nom: "Doe", Prenom: "John", Age: 30}}, nil)

	for _, path := range []string{"/lightning/nodes", "/users"} {
		html := doHTMX(h, http.MethodGet, path, nil)
		js := do(h, http.MethodGet, path, nil)
		for name, rr := range map[string]*httptest.ResponseRecorder{"html": html, "json": js} {
			assert.Equal(t, http.StatusOK, rr.Code, "%s %s", path, name)
			assert.Contains(t, rr.Header().Values("Vary"), "HX-Request", "%s %s", path, name)
		}
	}
}

// 26
func TestOps_ProbesAndMetricsOnly(t *testing.T) {
	h := testsupport.NewTestHarness()

	for _, p := range []string{"/health", "/ready", "/metrics"} {
		assert.Equal(t, http.StatusOK, doOps(h, http.MethodGet, p).Code, p)
	}
	for _, p := range []string{"/node", "/lightning/nodes", "/users", "/version", "/asset/js/generic.js"} {
		assert.Equal(t, http.StatusNotFound, doOps(h, http.MethodGet, p).Code, "%s must not be on the ops router", p)
	}
}
