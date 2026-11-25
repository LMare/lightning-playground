// Package web holds every navigable page (home, node, person), the HTML shell (layout, menu)
// and the static assets. It calls no service: pages only compose the UI.
package web

import (
	"bytes"
	"embed"
	"html/template"
	"log"
	"net/http"

	"github.com/Lmare/lightning-playground/api/internal/infra/http/common"
)

//go:embed templates/*.gohtml
var templatesFS embed.FS

var templates = template.Must(template.ParseFS(templatesFS, "templates/*.gohtml"))

// ActivePage identifies the menu entry to highlight; menu.gohtml compares on these values.
type ActivePage string

const (
	PageHello  ActivePage = "hello"
	PagePerson ActivePage = "person"
	PageNode   ActivePage = "node"
)

// Page describes the shell around a page body.
type Page struct {
	Title      string // rendered as "Lightning Playground - <Title>"
	ActivePage ActivePage
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
