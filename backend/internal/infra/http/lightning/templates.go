package lightninghttp

import (
	"embed"
	"fmt"
	"html/template"
)

//go:embed templates/*.html
var templatesFS embed.FS

// templates are parsed once at startup; truncate helpers are used by the uri and payment request views.
var templates = template.Must(template.New("").
	Funcs(template.FuncMap{"truncateUri": truncateUri, "truncate": truncate, "satAttr": satAttr}).
	ParseFS(templatesFS, "templates/*.html"))

// satAttr renders a data attribute holding an integer. html/template treats "data-on*" names as
// JS event handlers and would escape the value as JavaScript, so the attribute is built here.
// The name is a template literal and the value an int64, so nothing user-controlled is emitted.
func satAttr(name string, value int64) template.HTMLAttr {
	return template.HTMLAttr(fmt.Sprintf(`%s="%d"`, name, value))
}
