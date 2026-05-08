package main

import (
	"log"
	"net/http"

	"github.com/Haroon-BCBP/workflow_engine/config"
)

func main() {
	cfg := config.Load()

	h, cleanup, err := setupApp(cfg)
	if err != nil {
		log.Fatalf("Failed to setup app: %v", err)
	}
	defer cleanup()

	log.Printf("API server listening on %s", cfg.ListenAddr)
	if err := http.ListenAndServe(cfg.ListenAddr, h); err != nil {
		log.Fatalf("Server error: %v", err)
	}
}

