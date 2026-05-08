package main

import (
	"net/http"

	"github.com/Haroon-BCBP/workflow_engine/config"
	"github.com/Haroon-BCBP/workflow_engine/internal/handler"
	"github.com/Haroon-BCBP/workflow_engine/internal/iam"
	"github.com/Haroon-BCBP/workflow_engine/internal/repository"
	"github.com/Haroon-BCBP/workflow_engine/internal/service"
	"github.com/go-chi/chi/v5"
	"github.com/go-chi/chi/v5/middleware"
	"github.com/go-chi/cors"
	"go.temporal.io/sdk/client"
)
type App struct {
	Config   *config.Config
	DB       *repository.DB
	Repo     *repository.Repository
	Temporal client.Client
	IAM      *iam.IAM
	Service  service.WorkflowService
	Handler  *handler.Handler
}

func (a *App) Cleanup() {
	if a.DB != nil {
		a.DB.Close()
	}
	if a.Temporal != nil {
		a.Temporal.Close()
	}
}

func setupApp(cfg *config.Config) (http.Handler, func(), error) {
	app := &App{Config: cfg}

	var err error
	app.Temporal, err = client.Dial(client.Options{HostPort: cfg.TemporalHost})
	if err != nil {
		return nil, nil, err
	}

	app.DB, err = repository.Connect(cfg.DBPath)
	if err != nil {
		app.Cleanup()
		return nil, nil, err
	}

	app.Repo, err = repository.New(app.DB)
	if err != nil {
		app.Cleanup()
		return nil, nil, err
	}

	app.IAM, err = iam.Load(cfg.IAMPath)
	if err != nil {
		app.Cleanup()
		return nil, nil, err
	}

	app.Service = service.New(app.Repo, app.Temporal, app.IAM)
	app.Handler = handler.New(app.Service, app.IAM)

	r := chi.NewRouter()
	r.Use(middleware.Recoverer)
	r.Use(cors.Handler(cors.Options{
		AllowedOrigins:   []string{"*"},
		AllowedMethods:   []string{"GET", "POST", "PUT", "DELETE", "OPTIONS"},
		AllowedHeaders:   []string{"Accept", "Content-Type", "Authorization"},
		AllowCredentials: false,
	}))

	app.Handler.Register(r)

	return r, app.Cleanup, nil
}

