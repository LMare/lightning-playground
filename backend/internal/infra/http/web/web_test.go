package web

import (
	"bytes"
	"html/template"
	"log"
	"net/http"
	"net/http/httptest"
	"os"
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
	assert.Equal(t, "<p>ok</p>", rr.Body.String())
	assert.Contains(t, rr.Header().Get("Content-Type"), "text/html")
	assert.Contains(t, rr.Header().Values("Vary"), "HX-Request")
}
