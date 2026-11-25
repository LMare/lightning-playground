package integration

import (
	"context"
	"net/http"
	"net/http/httptest"
	"net/url"
	"regexp"
	"strings"
	"testing"

	lnrpc "github.com/Lmare/lightning-playground/backend/gRPC/github.com/lightningnetwork/lnd/lnrpc"
	"github.com/Lmare/lightning-playground/backend/internal/business/user/domain"
	"github.com/Lmare/lightning-playground/backend/internal/testsupport"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
)

// doWithHeaders sends a bodyless request with the given headers.
func doWithHeaders(h *testsupport.TestHarness, method, path string, headers map[string]string) *httptest.ResponseRecorder {
	req := httptest.NewRequest(method, path, nil)
	for k, v := range headers {
		req.Header.Set(k, v)
	}
	rr := httptest.NewRecorder()
	h.Router.ServeHTTP(rr, req)
	return rr
}

// doHistoryRestore sends the request htmx issues on a history cache miss.
func doHistoryRestore(h *testsupport.TestHarness, path string) *httptest.ResponseRecorder {
	return doWithHeaders(h, http.MethodGet, path, map[string]string{
		"HX-Request":                 "true",
		"HX-History-Restore-Request": "true",
	})
}

// doOps sends a plain request to the ops router (health, ready, metrics).
func doOps(h *testsupport.TestHarness, method, path string) *httptest.ResponseRecorder {
	req := httptest.NewRequest(method, path, nil)
	rr := httptest.NewRecorder()
	h.OpsRouter.ServeHTTP(rr, req)
	return rr
}

var menuTagRe = regexp.MustCompile(`<[^<>]*\bid="menu"[^<>]*>`)

func assertFullPage(t *testing.T, rr *httptest.ResponseRecorder, title string) {
	t.Helper()
	assert.Equal(t, http.StatusOK, rr.Code)
	assert.Contains(t, rr.Header().Get("Content-Type"), "text/html")
	body := rr.Body.String()
	assert.True(t, strings.HasPrefix(body, "<!DOCTYPE html>"), "full page must start with the doctype")
	assert.Contains(t, body, "<title>Lightning Playground - "+title+"</title>")
	tags := menuTagRe.FindAllString(body, -1)
	assert.Len(t, tags, 1, "the full page renders the menu once")
	for _, tag := range tags {
		assert.NotContains(t, tag, "hx-swap-oob", "the menu of a full page is not out-of-band")
	}
}

// assertFragmentOnly checks an HTMX page response: no layout, and the menu only as an
// out-of-band element whose active link is activeHref.
func assertFragmentOnly(t *testing.T, rr *httptest.ResponseRecorder, activeHref string) {
	t.Helper()
	assert.Equal(t, http.StatusOK, rr.Code)
	body := rr.Body.String()
	assert.NotContains(t, body, "<!DOCTYPE")
	assert.NotContains(t, body, "<html")
	tags := menuTagRe.FindAllString(body, -1)
	assert.Len(t, tags, 1, "the fragment carries the menu once, out-of-band")
	for _, tag := range tags {
		assert.Contains(t, tag, `hx-swap-oob="true"`)
	}
	assert.True(t, activeMenuLink(body, activeHref), "menu link %s must be active in the OOB menu", activeHref)
}

// activeMenuLink reports whether the menu link targeting href has the "active" class.
func activeMenuLink(body, href string) bool {
	tag := regexp.MustCompile(`<a\b[^<>]*\shx-get="` + regexp.QuoteMeta(href) + `"[^<>]*>`).FindString(body)
	m := regexp.MustCompile(`\sclass="([^"]*)"`).FindStringSubmatch(tag)
	if m == nil {
		return false
	}
	for _, c := range strings.Fields(m[1]) {
		if c == "active" {
			return true
		}
	}
	return false
}

// 1
func TestPages_Node_FullPage(t *testing.T) {
	h := testsupport.NewTestHarness()

	rr := doPlain(h, http.MethodGet, "/node", nil)

	assertFullPage(t, rr, "Node")
	body := rr.Body.String()
	assert.True(t, activeMenuLink(body, "/node"), "node menu link must be active")
	assert.False(t, activeMenuLink(body, "/person"))
	assert.False(t, activeMenuLink(body, "/hello"))
	assert.Contains(t, body, `id="table-nodes-body"`)
	assert.Contains(t, body, `hx-get="/api/v1/lightning/nodes"`)
	assert.Contains(t, body, `id="node-panel"`)
	assert.Contains(t, body, `id="node-info-slot"`)
}

// 2
func TestPages_Node_HTMXFragmentOnly(t *testing.T) {
	h := testsupport.NewTestHarness()

	rr := doHTMX(h, http.MethodGet, "/node", nil)

	assertFragmentOnly(t, rr, "/node")
	assert.Contains(t, rr.Body.String(), `id="table-nodes-body"`)
}

// 3
func TestPages_HistoryRestoreGetsFullPage(t *testing.T) {
	for path, title := range map[string]string{"/node": "Node", "/person": "Person", "/hello": "Hello", "/": "Hello"} {
		h := testsupport.NewTestHarness()
		rr := doHistoryRestore(h, path)
		assertFullPage(t, rr, title)
	}
}

// 4
func TestPages_VaryHeader(t *testing.T) {
	for _, path := range []string{"/", "/hello", "/person", "/node"} {
		h := testsupport.NewTestHarness()
		for name, rr := range map[string]*httptest.ResponseRecorder{
			"full":     doPlain(h, http.MethodGet, path, nil),
			"fragment": doHTMX(h, http.MethodGet, path, nil),
		} {
			assert.Contains(t, rr.Header().Values("Vary"), "HX-Request", "%s %s", path, name)
		}
	}
}

// 5
func TestPages_Hello(t *testing.T) {
	for _, path := range []string{"/", "/hello"} {
		h := testsupport.NewTestHarness()

		full := doPlain(h, http.MethodGet, path, nil)
		assertFullPage(t, full, "Hello")
		assert.Contains(t, full.Body.String(), "HTMX Hello World")
		assert.True(t, activeMenuLink(full.Body.String(), "/hello"))

		frag := doHTMX(h, http.MethodGet, path, nil)
		assertFragmentOnly(t, frag, "/hello")
		assert.Contains(t, frag.Body.String(), "HTMX Hello World")
	}
}

// 6
func TestPages_Person(t *testing.T) {
	h := testsupport.NewTestHarness()

	full := doPlain(h, http.MethodGet, "/person", nil)
	assertFullPage(t, full, "Person")
	assert.Contains(t, full.Body.String(), `hx-get="/api/v1/users"`)
	assert.True(t, activeMenuLink(full.Body.String(), "/person"))

	frag := doHTMX(h, http.MethodGet, "/person", nil)
	assertFragmentOnly(t, frag, "/person")
	assert.Contains(t, frag.Body.String(), `hx-get="/api/v1/users"`)
}

// 7
func TestPages_LayoutContent(t *testing.T) {
	h := testsupport.NewTestHarness()

	body := doPlain(h, http.MethodGet, "/node", nil).Body.String()

	assert.Contains(t, body, `sse-connect="/api/v1/stream-event"`)
	assert.Contains(t, body, `hx-get="/api/v1/version"`)
	assert.Contains(t, body, `id="toaster"`)
	assert.Contains(t, body, `id="confirm-dialog"`)
	assert.Regexp(t, `<script[^>]*src="[^"]*htmx\.org@[^"/]+/dist/ext/sse\.js"`, body, "sse extension must be pinned")
	assert.Contains(t, body, `src="/asset/js/generic.js"`)
	assert.Contains(t, body, `href="/asset/css/generic.css"`)
}

var hxLinkRe = regexp.MustCompile(`\bhx-(get|post|put|delete)="([^"]+)"`)
var sseLinkRe = regexp.MustCompile(`\bsse-connect="([^"]+)"`)

// 9: every hypermedia link rendered in a full page or in a fragment resolves to a registered route.
func TestPages_EveryLinkResolves(t *testing.T) {
	h := testsupport.NewTestHarness()
	// answers for the fragments that are crawled and for handlers that reach their repository without prior validation
	h.MockUserRepo.On("FindAll", mock.Anything).Return([]domain.UserModel{}, nil).Maybe()
	h.MockNodeRepo.On("GetNodesIds").Return([]string{"lnd-0"}, nil).Maybe()
	h.GrpcFactory.State.On("GetState", mock.Anything, mock.Anything).Return(&lnrpc.GetStateResponse{State: lnrpc.WalletState_LOCKED}, nil).Maybe()
	h.GrpcFactory.Lightning.On("GetInfo", mock.Anything, mock.Anything).Return(&lnrpc.GetInfoResponse{Alias: "alice", Color: "#ff0000"}, nil).Maybe()

	type source struct{ name, body string }
	var sources []source
	pages := []string{"/node", "/person", "/"}
	for _, page := range pages {
		sources = append(sources, source{"page " + page, doPlain(h, http.MethodGet, page, nil).Body.String()})
	}
	for _, frag := range []string{"/api/v1/lightning/nodes", "/node/panel?id=lnd-0", "/api/v1/lightning/nodeInfo?id=lnd-0"} {
		rr := doHTMX(h, http.MethodGet, frag, nil)
		assert.Equal(t, http.StatusOK, rr.Code, frag)
		sources = append(sources, source{"fragment " + frag, rr.Body.String()})
	}

	seen := map[string]bool{}
	links := 0
	for _, src := range sources {
		for _, m := range hxLinkRe.FindAllStringSubmatch(src.body, -1) {
			method := strings.ToUpper(m[1])
			path := m[2]
			if i := strings.IndexAny(path, "?#"); i >= 0 {
				path = path[:i]
			}
			if !strings.HasPrefix(path, "/") {
				t.Errorf("%s: link %q is not an absolute same-origin path", src.name, m[2])
				continue
			}
			links++
			key := method + " " + path
			if seen[key] {
				continue
			}
			seen[key] = true

			var rr *httptest.ResponseRecorder
			if method == http.MethodGet {
				rr = doHTMX(h, method, path, nil)
			} else {
				rr = doHTMX(h, method, path, url.Values{})
			}
			assert.NotEqual(t, http.StatusNotFound, rr.Code, "%s linked from %s is not routed", key, src.name)
			assert.NotEqual(t, http.StatusMethodNotAllowed, rr.Code, "%s linked from %s has wrong method", key, src.name)
		}
	}
	assert.GreaterOrEqual(t, links, 10, "the node screens should expose their hx-* links")
	for _, want := range []string{
		"GET /node/panel", "GET /api/v1/lightning/nodes", "GET /api/v1/lightning/nodeInfo", "GET /api/v1/lightning/uri", "GET /api/v1/lightning/balance",
		"GET /api/v1/lightning/channels", "POST /api/v1/lightning/wallet/unlock", "PUT /api/v1/lightning/alias",
		"POST /api/v1/lightning/peer", "POST /api/v1/lightning/channel", "POST /api/v1/lightning/invoice", "POST /api/v1/lightning/payment",
		"GET /api/v1/users",
	} {
		assert.True(t, seen[want], "the crawled screens should link %s", want)
	}

	for _, src := range sources {
		if !strings.HasPrefix(src.name, "page ") {
			continue
		}
		body := src.body
		for _, m := range sseLinkRe.FindAllStringSubmatch(body, -1) {
			assert.Equal(t, "/api/v1/stream-event", m[1])
			// blocking handler: run it with an already-cancelled context so it returns at once
			ctx, cancel := context.WithCancel(context.Background())
			cancel()
			req := httptest.NewRequest(http.MethodGet, m[1], nil).WithContext(ctx)
			rr := httptest.NewRecorder()
			h.Router.ServeHTTP(rr, req)
			assert.Equal(t, "text/event-stream", rr.Header().Get("Content-Type"))
		}
	}
	assert.Equal(t, http.StatusOK, doHTMX(h, http.MethodGet, "/api/v1/version", nil).Code)
}

// 10
func TestStatic_Assets(t *testing.T) {
	h := testsupport.NewTestHarness()

	js := doPlain(h, http.MethodGet, "/asset/js/generic.js", nil)
	assert.Equal(t, http.StatusOK, js.Code)
	assert.Contains(t, js.Header().Get("Content-Type"), "javascript")

	css := doPlain(h, http.MethodGet, "/asset/css/generic.css", nil)
	assert.Equal(t, http.StatusOK, css.Code)
	assert.Contains(t, css.Header().Get("Content-Type"), "text/css")

	png := doPlain(h, http.MethodGet, "/asset/img/logo1.png", nil)
	assert.Equal(t, http.StatusOK, png.Code)
	assert.Equal(t, "image/png", png.Header().Get("Content-Type"))
}

// 11
func TestStatic_Errors(t *testing.T) {
	h := testsupport.NewTestHarness()

	assert.Equal(t, http.StatusNotFound, doPlain(h, http.MethodGet, "/asset/nope.js", nil).Code)

	for _, dir := range []string{"/asset/", "/asset/js/", "/asset/js", "/asset/css/"} {
		rr := doPlain(h, http.MethodGet, dir, nil)
		assert.Equal(t, http.StatusNotFound, rr.Code, dir)
		assert.NotContains(t, rr.Body.String(), "generic.js", "no directory listing for %s", dir)
	}

	assert.NotEqual(t, http.StatusOK, doPlain(h, http.MethodGet, "/asset/../go.mod", nil).Code)
	assert.NotEqual(t, http.StatusOK, doPlain(h, http.MethodGet, "/asset/..%2f..%2fgo.mod", nil).Code)

	assert.Equal(t, http.StatusMethodNotAllowed, doPlain(h, http.MethodPost, "/asset/js/generic.js", url.Values{}).Code)
}

// 12
func TestRouting_PagesAndLegacyPrefix(t *testing.T) {
	h := testsupport.NewTestHarness()

	assert.Equal(t, http.StatusNotFound, doPlain(h, http.MethodGet, "/nope", nil).Code)
	assert.Equal(t, http.StatusMethodNotAllowed, doPlain(h, http.MethodPost, "/node", url.Values{}).Code)
	assert.Equal(t, http.StatusMethodNotAllowed, doPlain(h, http.MethodPost, "/person", url.Values{}).Code)
	// probes and metrics live on the ops listener only
	for _, p := range []string{"/metrics", "/health", "/ready"} {
		assert.Equal(t, http.StatusNotFound, doPlain(h, http.MethodGet, p, nil).Code, "%s must not be on the app router", p)
	}
}

// every resource route answers under /api/v1 with its own verb
func TestRouting_APIRoutesUnderPrefix(t *testing.T) {
	h := testsupport.NewTestHarness()
	h.MockUserRepo.On("FindAll", mock.Anything).Return([]domain.UserModel{}, nil).Maybe()
	h.MockNodeRepo.On("GetNodesIds").Return([]string{"lnd-0"}, nil).Maybe()
	h.GrpcFactory.State.On("GetState", mock.Anything, mock.Anything).Return(&lnrpc.GetStateResponse{State: lnrpc.WalletState_LOCKED}, nil).Maybe()
	h.GrpcFactory.Lightning.On("GetInfo", mock.Anything, mock.Anything).Return(&lnrpc.GetInfoResponse{Alias: "alice"}, nil).Maybe()

	routes := []struct{ method, path string }{
		{http.MethodGet, "/api/v1/lightning/nodes"},
		{http.MethodGet, "/api/v1/lightning/nodeInfo"},
		{http.MethodGet, "/api/v1/lightning/uri"},
		{http.MethodGet, "/api/v1/lightning/balance"},
		{http.MethodGet, "/api/v1/lightning/channels"},
		{http.MethodPost, "/api/v1/lightning/channel"},
		{http.MethodPost, "/api/v1/lightning/channel/close"},
		{http.MethodPost, "/api/v1/lightning/peer"},
		{http.MethodPost, "/api/v1/lightning/invoice"},
		{http.MethodPost, "/api/v1/lightning/payment"},
		{http.MethodPost, "/api/v1/lightning/wallet/unlock"},
		{http.MethodPut, "/api/v1/lightning/alias"},
		{http.MethodGet, "/api/v1/users"},
		{http.MethodGet, "/api/v1/version"},
	}
	for _, r := range routes {
		var form url.Values
		if r.method != http.MethodGet {
			form = url.Values{}
		}
		rr := doPlain(h, r.method, r.path, form)
		assert.NotEqual(t, http.StatusNotFound, rr.Code, "%s %s is not routed", r.method, r.path)
		assert.NotEqual(t, http.StatusMethodNotAllowed, rr.Code, "%s %s has the wrong method", r.method, r.path)
	}
}

// the metrics label of an API route is its full registered path
func TestMetrics_APIRouteLabel(t *testing.T) {
	h := testsupport.NewTestHarness()
	h.MockNodeRepo.On("GetNodesIds").Return([]string{}, nil).Maybe()
	doPlain(h, http.MethodGet, "/api/v1/lightning/nodes", nil)

	body := doOps(h, http.MethodGet, "/metrics").Body.String()

	assert.Contains(t, body, `http_requests_total{handler="/api/v1/lightning/nodes",method="GET"} 1`)
}

// 13
func TestMetrics_PagesAndStaticLabels(t *testing.T) {
	h := testsupport.NewTestHarness()
	assert.Equal(t, http.StatusOK, doPlain(h, http.MethodGet, "/node", nil).Code)
	assert.Equal(t, http.StatusOK, doPlain(h, http.MethodGet, "/asset/js/generic.js", nil).Code)

	// probes hit the ops router and must not be counted
	assert.Equal(t, http.StatusOK, doOps(h, http.MethodGet, "/health").Code)
	assert.Equal(t, http.StatusOK, doOps(h, http.MethodGet, "/ready").Code)

	rr := doOps(h, http.MethodGet, "/metrics")

	assert.Equal(t, http.StatusOK, rr.Code)
	body := rr.Body.String()
	assert.Contains(t, body, `http_requests_total{handler="/node",method="GET"} 1`)
	assert.Contains(t, body, `handler="/asset/"`)
	assert.NotContains(t, body, "generic.js")
	assert.NotContains(t, body, `handler="/health"`)
	assert.NotContains(t, body, `handler="/ready"`)
}
