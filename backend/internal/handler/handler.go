package handler

import (
	"encoding/json"
	"io"
	"log"
	"net/http"

	"github.com/go-chi/chi/v5"

	"github.com/Haroon-BCBP/workflow_engine/internal/dsl"
	"github.com/Haroon-BCBP/workflow_engine/internal/service"
)

type Handler struct {
	svc *service.WorkflowService
}

func New(svc *service.WorkflowService) *Handler {
	return &Handler{svc: svc}
}

func (h *Handler) HealthCheck(w http.ResponseWriter, r *http.Request) {
	writeJSON(w, http.StatusOK, map[string]string{"status": "ok"})
}

// Body: { "xml": "<bpmn:definitions>...</bpmn:definitions>" }
func (h *Handler) Submit(w http.ResponseWriter, r *http.Request) {
	body, err := io.ReadAll(r.Body)
	if err != nil {
		writeError(w, http.StatusBadRequest, "cannot read body")
		return
	}
	var req struct {
		XML string `json:"xml"`
	}
	if err := json.Unmarshal(body, &req); err != nil || req.XML == "" {
		writeError(w, http.StatusBadRequest, "field 'xml' is required")
		return
	}

	result, err := h.svc.Submit(r.Context(), req.XML)
	if err != nil {
		writeError(w, http.StatusInternalServerError, err.Error())
		return
	}
	writeJSON(w, http.StatusCreated, result)
}

func (h *Handler) ListWorkflows(w http.ResponseWriter, r *http.Request) {
	runs, err := h.svc.ListRuns(r.Context())
	if err != nil {
		writeError(w, http.StatusInternalServerError, err.Error())
		return
	}
	writeJSON(w, http.StatusOK, runs)
}

func (h *Handler) GetStatus(w http.ResponseWriter, r *http.Request) {
	id := chi.URLParam(r, "id")
	state, err := h.svc.GetStatus(r.Context(), id)
	if err != nil {
		writeError(w, http.StatusInternalServerError, err.Error())
		return
	}
	writeJSON(w, http.StatusOK, state)
}

func (h *Handler) GetYAML(w http.ResponseWriter, r *http.Request) {
	id := chi.URLParam(r, "id")
	yaml, err := h.svc.GetYAML(r.Context(), id)
	if err != nil {
		writeError(w, http.StatusNotFound, err.Error())
		return
	}
	w.Header().Set("Content-Type", "text/plain; charset=utf-8")
	w.WriteHeader(http.StatusOK)
	w.Write([]byte(yaml))
}

// Body: { "dept_id": "design", "to_stage": "review", "user_id": "u-d1" }
func (h *Handler) Transition(w http.ResponseWriter, r *http.Request) {
	id := chi.URLParam(r, "id")
	var sig dsl.TransitionSignal
	if err := json.NewDecoder(r.Body).Decode(&sig); err != nil {
		writeError(w, http.StatusBadRequest,"Invalid Transition Signal")
		return
	}
	if err := h.svc.SendTransition(r.Context(), id, sig); err != nil {
		writeError(w, http.StatusInternalServerError, err.Error())
		return
	}
	writeJSON(w, http.StatusOK, map[string]string{"status": "signal sent"})
}

// Body: { "dept_id": "design", "stage": "review", "user_id": "u-d2", "text": "LGTM" }
func (h *Handler) Comment(w http.ResponseWriter, r *http.Request) {
	id := chi.URLParam(r, "id")
	var sig dsl.CommentSignal
	if err := json.NewDecoder(r.Body).Decode(&sig); err != nil {
		writeError(w, http.StatusBadRequest, "Invalid Comment Signal")
		return
	}
	if err := h.svc.SendComment(r.Context(), id, sig); err != nil {
		writeError(w, http.StatusInternalServerError, err.Error())
		return
	}
	writeJSON(w, http.StatusOK, map[string]string{"status": "comment sent"})
}

// Body: { "action": "goto", "dept_id": "design", "stage": "prep", "admin_id": "admin-1" }
func (h *Handler) AdminRoute(w http.ResponseWriter, r *http.Request) {
	id := chi.URLParam(r, "id")
	var sig dsl.AdminRoutingSignal
	if err := json.NewDecoder(r.Body).Decode(&sig); err != nil {
		writeError(w, http.StatusBadRequest, "Invalid Admin Routing Signal")
		return
	}
	log.Println("message: Admin Routing Signal", "signal: ", sig)
	if err := h.svc.SendAdminRouting(r.Context(), id, sig); err != nil {
		writeError(w, http.StatusInternalServerError, err.Error())
		log.Println("error", err.Error())
		return
	}
	log.Println("message: Admin Routing Signal sent")
	writeJSON(w, http.StatusOK, map[string]string{"status": "routing signal sent"})
}

func writeJSON(w http.ResponseWriter, code int, v any) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(code)
	json.NewEncoder(w).Encode(v)
}

func writeError(w http.ResponseWriter, code int, msg string) {
	writeJSON(w, code, map[string]string{"error": msg})
}
