package main

import (
	"fmt"
	"log"
	"net"
	"net/http"

	config "github.com/Lmare/lightning-playground"
	"github.com/Lmare/lightning-playground/api/internal/bootstrap"
	exception "github.com/Lmare/lightning-playground/api/internal/shared/exception"
)

func main() {
	cfg := config.Load()
	exception.ConfigureProjectBasePath(cfg.ProjectPath)
	db, routers, err := bootstrap.InitApp(cfg)
	if err != nil {
		log.Fatal("Failed to initialize application:", err)
	}

	executeMigrations(db)

	// Bind both listeners before serving on either: /ready (ops) must not answer before the app listens.
	appListener, err := net.Listen("tcp", ":"+cfg.APIPort)
	if err != nil {
		log.Fatal("Failed to listen on the app port:", err)
	}
	opsListener, err := net.Listen("tcp", ":"+cfg.APIOpsPort)
	if err != nil {
		log.Fatal("Failed to listen on the ops port:", err)
	}

	fmt.Printf("Server API started on :%s (ops on :%s)\n", cfg.APIPort, cfg.APIOpsPort)
	go func() {
		log.Fatal(http.Serve(opsListener, routers.Ops))
	}()
	log.Fatal(http.Serve(appListener, routers.App))
}
