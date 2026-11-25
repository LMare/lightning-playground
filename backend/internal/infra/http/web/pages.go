package web

import (
	"net/http"

	"github.com/Lmare/lightning-playground/backend/internal/business/lightning/domain"
	"github.com/Lmare/lightning-playground/backend/internal/infra/http/common"
)

type Handler struct{}

func NewHandler() *Handler { return &Handler{} }

// HandleHome renders the hello page.
func (h *Handler) HandleHome(w http.ResponseWriter, r *http.Request) {
	RenderPage(w, r, Page{Title: "Hello", ActivePage: PageHello}, templates, "hello.gohtml", nil)
}

// HandleNodePage renders the node page (full page or HTMX body).
func (h *Handler) HandleNodePage(w http.ResponseWriter, r *http.Request) {
	RenderPage(w, r, Page{Title: "Node", ActivePage: PageNode}, templates, "nodePage.gohtml", nil)
}

// HandlePersonPage renders the person page (full page or HTMX body).
func (h *Handler) HandlePersonPage(w http.ResponseWriter, r *http.Request) {
	RenderPage(w, r, Page{Title: "Person", ActivePage: PagePerson}, templates, "personPage.gohtml", nil)
}

// HandleNodePanel renders the panel of a node (sections that lazy-load, forms bound to the node).
// HTML only and without any service call: it is UI composition, not data.
func (h *Handler) HandleNodePanel(w http.ResponseWriter, r *http.Request) {
	id, err := domain.ParseNodeID(r.FormValue("id"))
	if err != nil {
		common.FailCheck(w, r, "Missing or invalid node id.", nil)
		return
	}
	w.Header().Set("HX-Trigger", "node-selected")
	common.HTMXResponse(w, templates, "nodePanel.gohtml", struct{ ID string }{ID: id.String()})
}
