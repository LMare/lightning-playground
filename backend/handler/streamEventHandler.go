package handler


import (
	"fmt"
	"net/http"
	"strings"


	streamService "github.com/Lmare/lightning-test/backend/service/streamService"
)

// check the message from gRPC stream
func handleStreamEvent(response http.ResponseWriter, request *http.Request) {

	response.Header().Set("Content-Type", "text/event-stream")
	response.Header().Set("Cache-Control", "no-cache")
	response.Header().Set("Connection", "keep-alive")

	fmt.Fprintf(response, "event: init\ndata: initialisation de la SSE\n\n")
	response.(http.Flusher).Flush()

	notify := request.Context().Done()


	id := "uniqueSession"
	channel := streamService.GetChannel(id)

	for {
		select {
		case msg := <-channel :
			// Push SSE
			fmt.Fprintf(response, "data: %s\n\n", strings.ReplaceAll(msg, "\n", " "))
			fmt.Printf("Message brut envoyé : %#v\n", strings.ReplaceAll(msg, "\n", " "))
			response.(http.Flusher).Flush()
		case <-notify:
			fmt.Println("Client SSE déconnecté")
			return
	    }
	}
}
