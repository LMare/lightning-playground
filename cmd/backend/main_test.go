package main

import (
    "net/http"
    "testing"
)

func TestSetupServer(t *testing.T) {
    cfg, router := setupServer()

    if cfg.BackendPort == "" {
        t.Errorf("expected BackendPort to be set, got empty string")
    }
    if cfg.BackendUrl == "" {
        t.Errorf("expected BackendUrl to be set, got empty string")
    }
    if router == nil {
        t.Errorf("expected router not to be nil")
    }

    // Vérifie que le router implémente bien http.Handler
    _, ok := router.(http.Handler)
    if !ok {
        t.Errorf("router does not implement http.Handler")
    }
}
