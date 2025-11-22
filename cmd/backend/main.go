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

func startServer() {
	cfg := config.Load()
	router := handler.GetRouter()
	exception.ConfigureProjectBasePath(cfg.ProjectPath)

	fmt.Printf("Server Backend started : %s:%s\n", cfg.BackendUrl, cfg.BackendPort)
	log.Fatal(http.ListenAndServe(":"+cfg.BackendPort, router))
}
