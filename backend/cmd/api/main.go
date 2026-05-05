package main

import (
	"log"
	"net/http"
	"os"

	"github.com/go-chi/chi/v5"
	"github.com/go-chi/chi/v5/middleware"
	"github.com/go-chi/cors"
	"go.temporal.io/sdk/client"

	"github.com/Haroon-BCBP/workflow_engine/internal/handler"
	"github.com/Haroon-BCBP/workflow_engine/internal/repository"
	"github.com/Haroon-BCBP/workflow_engine/internal/service"
)

func main() {
	temporalHost := getEnv("TEMPORAL_HOST", "localhost:7233")
	dbPath := getEnv("DB_PATH", "./workflow_engine.db")
	listenAddr := getEnv("LISTEN_ADDR", ":8080")

	tc, err := client.Dial(client.Options{HostPort: temporalHost})
	if err != nil {
		log.Fatalf("Failed to connect to Temporal: %v", err)
	}
	defer tc.Close()

	repo, err := repository.New(dbPath)
	if err != nil {
		log.Fatalf("Failed to open DB: %v", err)
	}

	svc := service.New(repo, tc)

	h := handler.New(svc)

	r := chi.NewRouter()
	r.Use(middleware.Logger)
	r.Use(middleware.Recoverer)
	r.Use(cors.Handler(cors.Options{
		AllowedOrigins:   []string{"*"},
		AllowedMethods:   []string{"GET", "POST", "PUT", "DELETE", "OPTIONS"},
		AllowedHeaders:   []string{"Accept", "Content-Type", "Authorization"},
		AllowCredentials: false,
	}))

	r.Get("/health", h.HealthCheck)

	r.Route("/api/v1/workflows", func(r chi.Router) {
		r.Post("/", h.Submit)
		r.Get("/", h.ListWorkflows)
		r.Get("/{id}", h.GetStatus)
		r.Get("/{id}/yaml", h.GetYAML)
		r.Post("/{id}/transition", h.Transition)
		r.Post("/{id}/comment", h.Comment)
		r.Post("/{id}/route", h.AdminRoute)
	})

	log.Printf("API server listening on %s", listenAddr)
	if err := http.ListenAndServe(listenAddr, r); err != nil {
		log.Fatalf("Server error: %v", err)
	}
}

func getEnv(key, fallback string) string {
	if v := os.Getenv(key); v != "" {
		return v
	}
	return fallback
}
