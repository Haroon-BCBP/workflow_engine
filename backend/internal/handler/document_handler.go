package handler

import (
	"encoding/json"
	"net/http"

	"github.com/go-chi/chi/v5"
	"github.com/Haroon-BCBP/workflow_engine/internal/repository"
)

func (h *Handler) UploadDocument(w http.ResponseWriter, r *http.Request) {
	id := chi.URLParam(r, "id")
	var req struct {
		DeptID   string `json:"dept_id"`
		Stage    string `json:"stage"`
		Filename string `json:"filename"`
		UserID   string `json:"user_id"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeError(w, http.StatusBadRequest, "Invalid request body")
		return
	}
	doc, err := h.svc.UploadDocument(r.Context(), id, req.DeptID, req.Stage, req.Filename, req.UserID)
	if err != nil {
		writeError(w, http.StatusInternalServerError, err.Error())
		return
	}
	writeJSON(w, http.StatusCreated, doc)
}

func (h *Handler) GetDocuments(w http.ResponseWriter, r *http.Request) {
	id := chi.URLParam(r, "id")
	deptID := r.URL.Query().Get("dept_id")
	stage := r.URL.Query().Get("stage")
	docs, err := h.svc.GetDocuments(r.Context(), id, deptID, stage)
	if err != nil {
		writeError(w, http.StatusInternalServerError, err.Error())
		return
	}
	if docs == nil {
		docs = make([]repository.Document, 0)
	}
	writeJSON(w, http.StatusOK, docs)
}
