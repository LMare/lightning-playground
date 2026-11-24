package integration

import (
	"context"
	"net/http"
	"net/http/httptest"
	"net/url"
	"regexp"
	"strings"
	"testing"

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

func assertFullPage(t *testing.T, rr *httptest.ResponseRecorder, title string) {
	t.Helper()
	assert.Equal(t, http.StatusOK, rr.Code)
	assert.Contains(t, rr.Header().Get("Content-Type"), "text/html")
	body := rr.Body.String()
	assert.True(t, strings.HasPrefix(body, "<!DOCTYPE html>"), "full page must start with the doctype")
	assert.Contains(t, body, "<title>Lightning Playground - "+title+"</title>")
	assert.Contains(t, body, `id="menu"`)
}

func assertFragmentOnly(t *testing.T, rr *httptest.ResponseRecorder) {
	t.Helper()
	assert.Equal(t, http.StatusOK, rr.Code)
	body := rr.Body.String()
	assert.NotContains(t, body, "<!DOCTYPE")
	assert.NotContains(t, body, "<html")
	assert.NotContains(t, body, `id="menu"`)
}

// activeMenuLink reports whether the menu link targeting href has the "active" class.
func activeMenuLink(body, href string) bool {
	re := regexp.MustCompile(`(?s)<a\s+class="([^"]*)"\s+hx-get="` + regexp.QuoteMeta(href) + `"`)
	m := re.FindStringSubmatch(body)
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
	assert.Contains(t, body, `hx-get="/lightning/nodes"`)
}

// 2
func TestPages_Node_HTMXFragmentOnly(t *testing.T) {
	h := testsupport.NewTestHarness()

	rr := doHTMX(h, http.MethodGet, "/node", nil)

	assertFragmentOnly(t, rr)
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
		assertFragmentOnly(t, frag)
		assert.Contains(t, frag.Body.String(), "HTMX Hello World")
	}
}

// 6
func TestPages_Person(t *testing.T) {
	h := testsupport.NewTestHarness()

	full := doPlain(h, http.MethodGet, "/person", nil)
	assertFullPage(t, full, "Person")
	assert.Contains(t, full.Body.String(), `hx-get="/users"`)
	assert.True(t, activeMenuLink(full.Body.String(), "/person"))

	frag := doHTMX(h, http.MethodGet, "/person", nil)
	assertFragmentOnly(t, frag)
	assert.Contains(t, frag.Body.String(), `hx-get="/users"`)
}

// 7
func TestPages_LayoutContent(t *testing.T) {
	h := testsupport.NewTestHarness()

	body := doPlain(h, http.MethodGet, "/node", nil).Body.String()

	assert.Contains(t, body, `sse-connect="/stream-event"`)
	assert.Contains(t, body, `hx-get="/version"`)
	assert.Contains(t, body, `id="toaster"`)
	assert.Contains(t, body, `id="confirm-dialog"`)
	assert.Contains(t, body, `src="/asset/js/generic.js"`)
	assert.Contains(t, body, `href="/asset/css/generic.css"`)
}

// 8
func TestPages_NoLegacyIndirection(t *testing.T) {
	h := testsupport.NewTestHarness()
	for _, path := range []string{"/", "/person", "/node"} {
		rr := doPlain(h, http.MethodGet, path, nil)
		assert.Equal(t, http.StatusOK, rr.Code, path)
		body := rr.Body.String()
		assert.NotContains(t, body, "/api/", path)
		assert.NotContains(t, body, "__API_PREFIX__", path)
	}
}

var hxLinkRe = regexp.MustCompile(`\bhx-(get|post|put|delete)="([^"]+)"`)
var sseLinkRe = regexp.MustCompile(`\bsse-connect="([^"]+)"`)

// 9: every hypermedia link rendered in a full page resolves to a registered route.
func TestPages_EveryLinkResolves(t *testing.T) {
	h := testsupport.NewTestHarness()
	// empty answers, only for handlers that reach their repository without prior validation
	h.MockUserRepo.On("FindAll", mock.Anything).Return([]domain.UserModel{}, nil).Maybe()
	h.MockNodeRepo.On("GetNodesIds").Return([]string{}, nil).Maybe()

	seen := map[string]bool{}
	for _, page := range []string{"/node", "/person", "/"} {
		body := doPlain(h, http.MethodGet, page, nil).Body.String()
		links := 0
		for _, m := range hxLinkRe.FindAllStringSubmatch(body, -1) {
			method := strings.ToUpper(m[1])
			path := m[2]
			if i := strings.IndexAny(path, "?#"); i >= 0 {
				path = path[:i]
			}
			if !strings.HasPrefix(path, "/") {
				t.Errorf("%s: link %q is not an absolute same-origin path", page, m[2])
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
			assert.NotEqual(t, http.StatusNotFound, rr.Code, "%s linked from %s is not routed", key, page)
			assert.NotEqual(t, http.StatusMethodNotAllowed, rr.Code, "%s linked from %s has wrong method", key, page)
		}
		if page == "/node" {
			assert.GreaterOrEqual(t, links, 10, "node page should expose its hx-* links")
			for _, want := range []string{
				"GET /lightning/nodes", "GET /lightning/nodeInfo", "GET /lightning/uri", "GET /lightning/balance",
				"GET /lightning/channels", "POST /lightning/wallet/unlock", "PUT /lightning/alias",
				"POST /lightning/peer", "POST /lightning/channel", "POST /lightning/invoice", "POST /lightning/paiment",
			} {
				assert.True(t, seen[want], "node page should link %s", want)
			}
		}
		if page == "/person" {
			assert.True(t, seen["GET /users"])
		}

		for _, m := range sseLinkRe.FindAllStringSubmatch(body, -1) {
			assert.Equal(t, "/stream-event", m[1])
			// blocking handler: run it with an already-cancelled context so it returns at once
			ctx, cancel := context.WithCancel(context.Background())
			cancel()
			req := httptest.NewRequest(http.MethodGet, m[1], nil).WithContext(ctx)
			rr := httptest.NewRecorder()
			h.Router.ServeHTTP(rr, req)
			assert.Equal(t, "text/event-stream", rr.Header().Get("Content-Type"))
		}
	}
	assert.Equal(t, http.StatusOK, doHTMX(h, http.MethodGet, "/version", nil).Code)
}

// 10
func TestStatic_Assets(t *testing.T) {
	h := testsupport.NewTestHarness()

	js := doPlain(h, http.MethodGet, "/asset/js/generic.js", nil)
	assert.Equal(t, http.StatusOK, js.Code)
	assert.Contains(t, js.Header().Get("Content-Type"), "javascript")
	assert.Contains(t, js.Body.String(), "enrichRgaaTableRows")

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
	assert.Equal(t, http.StatusNotFound, doPlain(h, http.MethodGet, "/api/lightning/nodes", nil).Code)
	assert.Equal(t, http.StatusNotFound, doPlain(h, http.MethodGet, "/api/users", nil).Code)
}

// 13
func TestMetrics_PagesAndStaticLabels(t *testing.T) {
	h := testsupport.NewTestHarness()
	assert.Equal(t, http.StatusOK, doPlain(h, http.MethodGet, "/node", nil).Code)
	assert.Equal(t, http.StatusOK, doPlain(h, http.MethodGet, "/asset/js/generic.js", nil).Code)

	rr := doPlain(h, http.MethodGet, "/metrics", nil)

	assert.Equal(t, http.StatusOK, rr.Code)
	body := rr.Body.String()
	assert.Contains(t, body, `http_requests_total{handler="/node",method="GET"} 1`)
	assert.Contains(t, body, `handler="/asset/"`)
	assert.NotContains(t, body, "generic.js")
}
