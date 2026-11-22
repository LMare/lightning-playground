package lightninghttp

import (
	"embed"
	"html/template"
)

//go:embed templates/*.html
var templatesFS embed.FS

// templates are parsed once at startup; truncate helpers are used by the uri and payment request views.
var templates = template.Must(template.New("").
	Funcs(template.FuncMap{"truncateUri": truncateUri, "truncate": truncate}).
	ParseFS(templatesFS, "templates/*.html"))
