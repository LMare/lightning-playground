package main

import (
	"fmt"
	"log"
	"net/http"

	config "github.com/Lmare/lightning-playground"
	handler "github.com/Lmare/lightning-playground/backend/handler"
	exception "github.com/Lmare/lightning-playground/backend/exception"
)

func main() {
	startServer()
}

//Set the conf of the serveur
func setupServer() (*config.Config, http.Handler) {
    cfg := config.Load()
    router := handler.GetRouter()
    exception.ConfigureProjectBasePath(cfg.ProjectPath)
    return cfg, router
}

// startServer launch the server HTTP
func startServer() {
    cfg, router := setupServer()
    fmt.Printf("Server Backend started : %s:%s\n", cfg.BackendUrl, cfg.BackendPort)
    log.Fatal(http.ListenAndServe(":"+cfg.BackendPort, router))
}
