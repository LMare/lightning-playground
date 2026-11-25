// Package web holds the HTML shell (layout), the static assets and the home page.
package web

import (
	"bytes"
	"embed"
	"html/template"
	"io/fs"
	"log"
	"net/http"
	"strings"

	"github.com/Lmare/lightning-playground/backend/internal/infra/http/common"
)

//go:embed templates/*.gohtml
var templatesFS embed.FS

//go:embed static
var staticFS embed.FS

var templates = template.Must(template.ParseFS(templatesFS, "templates/*.gohtml"))

// Page describes the shell around a page body.
type Page struct {
	Title      string // rendered as "Lightning Playground - <Title>"
	ActivePage string // "hello" | "person" | "node" (menu highlighting)
}

// RenderPage executes the template `name` of tmpl with data. HTMX navigation (not a history
// restore) gets the body only; otherwise the body is wrapped in layout.gohtml.
func RenderPage(w http.ResponseWriter, r *http.Request, page Page, tmpl *template.Template, name string, data any) {
	var body bytes.Buffer
	if err := tmpl.ExecuteTemplate(&body, name, data); err != nil {
		log.Printf("[ERROR] rendering page %s: %v", name, err)
		http.Error(w, "Internal server error", http.StatusInternalServerError)
		return
	}

	partial := common.IsHTMX(r) && r.Header.Get("HX-History-Restore-Request") != "true"
	var out bytes.Buffer
	if partial {
		out = body
		// the menu follows the page: sent out-of-band so the server owns the active state
		if err := templates.ExecuteTemplate(&out, "menu", map[string]any{"ActivePage": page.ActivePage, "OOB": true}); err != nil {
			log.Printf("[ERROR] rendering menu for %s: %v", name, err)
			http.Error(w, "Internal server error", http.StatusInternalServerError)
			return
		}
	} else if err := templates.ExecuteTemplate(&out, "layout.gohtml", map[string]any{
		"Title":      page.Title,
		"ActivePage": page.ActivePage,
		"Content":    template.HTML(body.String()),
	}); err != nil {
		log.Printf("[ERROR] rendering layout for %s: %v", name, err)
		http.Error(w, "Internal server error", http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	w.Header().Add("Vary", "HX-Request")
	w.Write(out.Bytes())
}

type Handler struct{}

func NewHandler() *Handler { return &Handler{} }

// HandleHome renders the hello page.
func (h *Handler) HandleHome(w http.ResponseWriter, r *http.Request) {
	RenderPage(w, r, Page{Title: "Hello", ActivePage: "hello"}, templates, "hello.gohtml", nil)
}

// StaticHandler serves the embedded assets under /asset/ (no directory listing).
func StaticHandler() http.Handler {
	sub, err := fs.Sub(staticFS, "static")
	if err != nil {
		panic(err)
	}
	files := http.FileServerFS(sub)
	return http.StripPrefix("/asset/", http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		p := r.URL.Path
		if p == "" || strings.HasSuffix(p, "/") {
			http.NotFound(w, r)
			return
		}
		if info, err := fs.Stat(sub, strings.TrimPrefix(p, "/")); err == nil && info.IsDir() {
			http.NotFound(w, r)
			return
		}
		files.ServeHTTP(w, r)
	}))
}
