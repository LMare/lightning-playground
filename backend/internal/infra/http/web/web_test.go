package web

import (
	"bytes"
	"html/template"
	"log"
	"net/http"
	"net/http/httptest"
	"os"
	"regexp"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
)

// 15: a template execution error yields a 500, is logged, and never a half page with 200.
func TestRenderPage_TemplateErrorReturns500(t *testing.T) {
	tmpl := template.Must(template.New("broken.html").Parse(`<p>partial {{ .Missing }}</p>`))

	var logs bytes.Buffer
	log.SetOutput(&logs)
	defer log.SetOutput(os.Stderr)

	for name, headers := range map[string]map[string]string{
		"full page": nil,
		"htmx":      {"HX-Request": "true"},
	} {
		logs.Reset()
		req := httptest.NewRequest(http.MethodGet, "/x", nil)
		for k, v := range headers {
			req.Header.Set(k, v)
		}
		rr := httptest.NewRecorder()

		RenderPage(rr, req, Page{Title: "Broken", ActivePage: PageNode}, tmpl, "broken.html", struct{}{})

		assert.Equal(t, http.StatusInternalServerError, rr.Code, name)
		assert.NotContains(t, rr.Body.String(), "partial", name)
		assert.NotContains(t, rr.Body.String(), "<!DOCTYPE", name)
		assert.NotEmpty(t, logs.String(), "%s: error must be logged", name)
	}
}

func TestRenderPage_SetsVaryAndContentType(t *testing.T) {
	tmpl := template.Must(template.New("ok.html").Parse(`<p>ok</p>`))
	req := httptest.NewRequest(http.MethodGet, "/x", nil)
	req.Header.Set("HX-Request", "true")
	rr := httptest.NewRecorder()

	RenderPage(rr, req, Page{Title: "Ok", ActivePage: PageNode}, tmpl, "ok.html", nil)

	assert.Equal(t, http.StatusOK, rr.Code)
	assert.True(t, strings.HasPrefix(rr.Body.String(), "<p>ok</p>"), "the page body comes first")
	assert.Contains(t, rr.Body.String(), `id="menu"`)
	assert.Contains(t, rr.Body.String(), `hx-swap-oob="true"`)
	assert.Contains(t, rr.Header().Get("Content-Type"), "text/html")
	assert.Contains(t, rr.Header().Values("Vary"), "HX-Request")
}

// The partial response appends the menu out-of-band with the current page active;
// the full page renders the menu once, in place.
func TestRenderPage_MenuOutOfBandOnlyForPartials(t *testing.T) {
	tmpl := template.Must(template.New("ok.html").Parse(`<p>ok</p>`))
	activeNode := regexp.MustCompile(`(?s)<a\s+class="[^"]*\bactive\b[^"]*"\s+hx-get="/node"`)

	req := httptest.NewRequest(http.MethodGet, "/node", nil)
	req.Header.Set("HX-Request", "true")
	partial := httptest.NewRecorder()
	RenderPage(partial, req, Page{Title: "Node", ActivePage: PageNode}, tmpl, "ok.html", nil)
	assert.Equal(t, 1, strings.Count(partial.Body.String(), `id="menu"`))
	assert.Regexp(t, activeNode, partial.Body.String())
	assert.NotContains(t, partial.Body.String(), "<!DOCTYPE")

	full := httptest.NewRecorder()
	RenderPage(full, httptest.NewRequest(http.MethodGet, "/node", nil), Page{Title: "Node", ActivePage: PageNode}, tmpl, "ok.html", nil)
	assert.Equal(t, 1, strings.Count(full.Body.String(), `id="menu"`))
	assert.NotContains(t, full.Body.String(), "hx-swap-oob")
	assert.Regexp(t, activeNode, full.Body.String())
}

// The menu compares on these exact string values.
func TestActivePageConstants(t *testing.T) {
	assert.Equal(t, ActivePage("hello"), PageHello)
	assert.Equal(t, ActivePage("person"), PagePerson)
	assert.Equal(t, ActivePage("node"), PageNode)
}

func activeLink(href string) *regexp.Regexp {
	return regexp.MustCompile(`(?s)<a\s+class="[^"]*\bactive\b[^"]*"\s+hx-get="` + regexp.QuoteMeta(href) + `"`)
}

// The page handlers live in web, need no service, and highlight their own menu entry.
func TestPageHandlers_FullAndHTMX(t *testing.T) {
	h := NewHandler()
	cases := []struct {
		name    string
		handler http.HandlerFunc
		path    string
		active  string
	}{
		{"home", h.HandleHome, "/", "/hello"},
		{"node", h.HandleNodePage, "/node", "/node"},
		{"person", h.HandlePersonPage, "/person", "/person"},
	}
	for _, c := range cases {
		full := httptest.NewRecorder()
		c.handler(full, httptest.NewRequest(http.MethodGet, c.path, nil))
		assert.Equal(t, http.StatusOK, full.Code, c.name)
		assert.Contains(t, full.Body.String(), "<!DOCTYPE", c.name)
		assert.Equal(t, 1, strings.Count(full.Body.String(), `id="menu"`), c.name)
		assert.NotContains(t, full.Body.String(), "hx-swap-oob", c.name)
		assert.Regexp(t, activeLink(c.active), full.Body.String(), c.name)
		assert.Contains(t, full.Header().Values("Vary"), "HX-Request", c.name)

		req := httptest.NewRequest(http.MethodGet, c.path, nil)
		req.Header.Set("HX-Request", "true")
		part := httptest.NewRecorder()
		c.handler(part, req)
		assert.Equal(t, http.StatusOK, part.Code, c.name)
		assert.NotContains(t, part.Body.String(), "<!DOCTYPE", c.name)
		assert.Equal(t, 1, strings.Count(part.Body.String(), `id="menu"`), c.name)
		assert.Contains(t, part.Body.String(), `hx-swap-oob="true"`, c.name)
		assert.Regexp(t, activeLink(c.active), part.Body.String(), c.name)
	}
}

func TestHandleNodePage_HasPanelSlots(t *testing.T) {
	rr := httptest.NewRecorder()
	NewHandler().HandleNodePage(rr, httptest.NewRequest(http.MethodGet, "/node", nil))
	assert.Contains(t, rr.Body.String(), `id="node-panel"`)
	assert.Contains(t, rr.Body.String(), `id="node-info-slot"`)
}

func TestHandlePersonPage_HasUsersTable(t *testing.T) {
	rr := httptest.NewRecorder()
	NewHandler().HandlePersonPage(rr, httptest.NewRequest(http.MethodGet, "/person", nil))
	assert.Contains(t, rr.Body.String(), `table-persons-body`)
}

func TestHandleNodePanel(t *testing.T) {
	h := NewHandler()

	req := httptest.NewRequest(http.MethodGet, "/node/panel?id=lnd-0", nil)
	req.Header.Set("HX-Request", "true")
	ok := httptest.NewRecorder()
	h.HandleNodePanel(ok, req)
	assert.Equal(t, http.StatusOK, ok.Code)
	assert.Contains(t, ok.Header().Get("HX-Trigger"), "node-selected")
	assert.Contains(t, ok.Body.String(), `hx-get="/lightning/balance?id=lnd-0"`)
	assert.Contains(t, ok.Body.String(), `id="selected-node"`)
	assert.Contains(t, ok.Body.String(), `name="id" value="lnd-0"`)

	req = httptest.NewRequest(http.MethodGet, "/node/panel?id=", nil)
	req.Header.Set("HX-Request", "true")
	bad := httptest.NewRecorder()
	h.HandleNodePanel(bad, req)
	assert.Contains(t, bad.Body.String(), "❌")
	assert.Empty(t, bad.Header().Get("HX-Trigger"))
}
