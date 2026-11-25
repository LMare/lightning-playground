package common

import (
	"fmt"
	"net/http"

	"github.com/Lmare/lightning-playground/api/internal/infra/sse"
)

func NewStreamHandler() *StreamHandler {
	return &StreamHandler{}
}

type StreamHandler struct{}

// HandleStreamEvent handles the SSE endpoint; clients subscribe to the shared stream hub.
func (h *StreamHandler) HandleStreamEvent(response http.ResponseWriter, request *http.Request) {

	response.Header().Set("Content-Type", "text/event-stream")
	response.Header().Set("Cache-Control", "no-cache")
	response.Header().Set("Connection", "keep-alive")

	fmt.Fprintf(response, "event: init\ndata: SSE initialization\n\n")
	response.(http.Flusher).Flush()

	notify := request.Context().Done()
	// subscribe to the notification stream
	sse.SubscribeSse(response)

	// wait until the connection is interrupted
	<-notify
	fmt.Println("SSE client disconnected")
	sse.RevoqueSse(response)
}
