package lightninghttp

import (
	"embed"
	"fmt"
	"html/template"
	"strings"
)

//go:embed templates/*.gohtml
var templatesFS embed.FS

// templates are parsed once at startup; truncate helpers are used by the uri and payment request views.
var templates = template.Must(template.New("").
	Funcs(template.FuncMap{"truncateUri": truncateUri, "truncate": truncate, "satAttr": satAttr, "formatBTC": formatBTC}).
	ParseFS(templatesFS, "templates/*.gohtml"))

// formatBTC returns sat as BTC with 2 decimals, rounded half up (away from zero for negatives),
// using integer arithmetic only.
func formatBTC(sat int64) string {
	sign := ""
	if sat < 0 {
		sign = "-"
		sat = -sat
	}
	cents := (sat + 500000) / 1000000
	return fmt.Sprintf("%s%d.%02d", sign, cents/100, cents%100)
}

// satAttr renders a data attribute holding an integer. html/template treats "data-on*" names as
// JS event handlers and would escape the value as JavaScript, so the attribute is built here.
// The name is a template literal and the value an int64, so nothing user-controlled is emitted.
func satAttr(name string, value int64) template.HTMLAttr {
	return template.HTMLAttr(fmt.Sprintf(`%s="%d"`, name, value))
}

// reduce an uri
func truncateUri(s string, n int) string {
	at := strings.Index(s, "@")
	if at == -1 || at < 2*n {
		return s // no @ or too short to truncate
	}

	start := s[:n]
	end := s[at-n : at]
	host := s[at:] // includes @

	return start + "..." + end + host
}

// reduce a string
func truncate(s string, n int) string {
	at := len(s)
	if at <= 2*n {
		return s
	}
	start := s[:n]
	end := s[at-n : at]
	return start + "..." + end
}
