package handler

import (
	"github.com/go-chi/chi/v5"
	"github.com/go-chi/chi/v5/middleware"

	"github.com/Haroon-BCBP/workflow_engine/internal/iam"
	"github.com/Haroon-BCBP/workflow_engine/internal/service"
)

type Handler struct {
	svc service.WorkflowService
	iam *iam.IAM
}

func New(svc service.WorkflowService, iamSvc *iam.IAM) *Handler {
	return &Handler{svc: svc, iam: iamSvc}
}

func (h *Handler) Register(r chi.Router) {
	r.Get("/health", h.HealthCheck)
	r.Get("/api/v1/users", h.GetUsers)
	r.Get("/api/v1/workloads", h.GetWorkloads)
	
	r.Route("/api/v1/workflows", func(r chi.Router) {
		r.Use(middleware.Logger)
		r.Post("/", h.Submit)
		r.Get("/", h.ListWorkflows)
		r.Get("/{id}/yaml", h.GetYAML)
		r.Post("/{id}/start", h.StartWorkflow)
		r.Post("/{id}/transition", h.Transition)
		r.Post("/{id}/comment", h.Comment)
		r.Post("/{id}/route", h.AdminRoute)
		r.Post("/{id}/documents", h.UploadDocument)
		r.Get("/{id}/documents", h.GetDocuments)
	})
	
	r.Get("/api/v1/workflows/{id}", h.GetStatus)
}
