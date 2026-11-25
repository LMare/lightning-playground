package common

import (
	"encoding/json"
	"errors"
	"fmt"
	"html/template"
	"log"
	"net/http"
	"runtime"
	"strings"
)

func Fail(response http.ResponseWriter, request *http.Request, message string, exception error) {
	LogException(exception)
	if IsHTMX(request) {
		HtmxMessageKo(response, message)
	} else {
		http.Error(response, message, http.StatusInternalServerError)
	}
}

func FailCheck(response http.ResponseWriter, request *http.Request, message string, exception error) {
	LogException(exception)
	if IsHTMX(request) {
		HtmxMessageKo(response, message)
	} else {
		http.Error(response, message, http.StatusBadRequest)
	}
}

// Transform the objet as a Json and put it in the reponse
func JSONResponse(response http.ResponseWriter, objet any) {

	objetJson, err := json.Marshal(objet)

	if err != nil {
		fmt.Println("Error json Marshal : ", err)
		fmt.Fprintf(response, "Une erreur est survenue")
	}
	response.Header().Set("Content-Type", "application/json")
	response.Header().Add("Vary", "HX-Request")
	fmt.Fprintf(response, "%s", string(objetJson))
}

// render an object in Htmx with the named template
func HTMXResponse(response http.ResponseWriter, tmpl *template.Template, name string, viewObject any) {
	response.Header().Set("Content-Type", "text/html")
	response.Header().Add("Vary", "HX-Request")
	if err := tmpl.ExecuteTemplate(response, name, viewObject); err != nil {
		log.Printf("[ERROR] rendering template %s: %v", name, err)
	}
}

// return a response of success
func HTMXMessageOk(response http.ResponseWriter, message string) {
	HTMXResponse(response, messageTemplates, "success.html", message)
}

// Returns an HTML error response
func HtmxMessageKo(response http.ResponseWriter, message string) {
	HTMXResponse(response, messageTemplates, "error.html", message)
}

// return code 204
func OkNoContent(response http.ResponseWriter) {
	response.WriteHeader(http.StatusNoContent)
}

// True if the client use HTMX, in this case the response muste be in HTML
func IsHTMX(request *http.Request) bool {
	return request.Header.Get("HX-Request") == "true"
}

// format the error for the logs
func LogException(err error) {
	if err == nil {
		return
	}

	// Capture the current flags
	originalFlags := log.Flags()

	// First line: date, time, file, line
	log.SetFlags(log.Ldate | log.Ltime | log.Lshortfile)

	i := 1
	for err != nil {
		next := errors.Unwrap(err)

		currentMsg := err.Error()
		var nextMsg string
		if next != nil {
			nextMsg = next.Error()
		}

		context := strings.TrimSpace(strings.TrimSuffix(strings.TrimSuffix(currentMsg, nextMsg), "→"))

		if i == 1 {
			_, file, line, _ := runtime.Caller(1)
			log.Printf("[ERROR] at %s:%d | %s", file, line, err.Error())
			log.SetFlags(0) // Supprime les métadonnées pour les lignes suivantes
		}

		log.Printf("\t→ Error #%d | Type: %T | Message : %s", i, err, context)

		err = next
		i++
	}

	// Restore the initial flags
	log.SetFlags(originalFlags)
}
