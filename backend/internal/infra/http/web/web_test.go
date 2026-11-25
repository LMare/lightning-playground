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

		RenderPage(rr, req, Page{Title: "Broken", ActivePage: "node"}, tmpl, "broken.html", struct{}{})

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

	RenderPage(rr, req, Page{Title: "Ok", ActivePage: "node"}, tmpl, "ok.html", nil)

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
	RenderPage(partial, req, Page{Title: "Node", ActivePage: "node"}, tmpl, "ok.html", nil)
	assert.Equal(t, 1, strings.Count(partial.Body.String(), `id="menu"`))
	assert.Regexp(t, activeNode, partial.Body.String())
	assert.NotContains(t, partial.Body.String(), "<!DOCTYPE")

	full := httptest.NewRecorder()
	RenderPage(full, httptest.NewRequest(http.MethodGet, "/node", nil), Page{Title: "Node", ActivePage: "node"}, tmpl, "ok.html", nil)
	assert.Equal(t, 1, strings.Count(full.Body.String(), `id="menu"`))
	assert.NotContains(t, full.Body.String(), "hx-swap-oob")
	assert.Regexp(t, activeNode, full.Body.String())
}
