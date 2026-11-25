package userhttp

import (
	"embed"
	"html/template"
)

//go:embed templates/*.gohtml
var templatesFS embed.FS

var templates = template.Must(template.ParseFS(templatesFS, "templates/*.gohtml"))
