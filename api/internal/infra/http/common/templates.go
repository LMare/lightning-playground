package common

import (
	"embed"
	"html/template"
)

//go:embed templates/*.gohtml
var messageFS embed.FS

// messageTemplates holds the HTMX success/error toasts shared by every handler.
var messageTemplates = template.Must(template.ParseFS(messageFS, "templates/*.gohtml"))
