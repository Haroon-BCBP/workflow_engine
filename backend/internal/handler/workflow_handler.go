package handler

import (
	"encoding/json"
	"io"
	"log"
	"net/http"

	"github.com/go-chi/chi/v5"
	engine "github.com/Haroon-BCBP/workflow_engine/internal/workflow"
)

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
	userID := r.URL.Query().Get("user_id")
	isAdmin := h.iam.IsAdmin(userID)
	runs, err := h.svc.ListRuns(r.Context(), userID, isAdmin)
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

func (h *Handler) StartWorkflow(w http.ResponseWriter, r *http.Request) {
	id := chi.URLParam(r, "id")
	var sig engine.AdminStartSignal
	if err := json.NewDecoder(r.Body).Decode(&sig); err != nil {
		writeError(w, http.StatusBadRequest, "Invalid Admin Start Signal")
		return
	}
	if !h.iam.IsAdmin(sig.AdminID) {
		writeError(w, http.StatusForbidden, "Only admins can start workflows")
		return
	}
	if err := h.svc.SendAdminStart(r.Context(), id, sig); err != nil {
		writeError(w, http.StatusInternalServerError, err.Error())
		return
	}
	writeJSON(w, http.StatusOK, map[string]string{"status": "workflow started"})
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

func (h *Handler) Transition(w http.ResponseWriter, r *http.Request) {
	id := chi.URLParam(r, "id")
	var sig engine.TransitionSignal
	if err := json.NewDecoder(r.Body).Decode(&sig); err != nil {
		writeError(w, http.StatusBadRequest, "Invalid Transition Signal")
		return
	}

	state, err := h.svc.GetStatus(r.Context(), id)
	if err == nil {
		if dept, ok := state.Progress[sig.DeptID]; ok {
			assignees := dept.StageAssignees[dept.CurrentStage]
			isAssigned := false
			for _, a := range assignees {
				if a == sig.UserID {
					isAssigned = true
					break
				}
			}
			if !isAssigned && !h.iam.IsAdmin(sig.UserID) {
				writeError(w, http.StatusForbidden, "User is not assigned to this stage")
				return
			}
		}
	}

	if err := h.svc.SendTransition(r.Context(), id, sig); err != nil {
		writeError(w, http.StatusInternalServerError, err.Error())
		return
	}
	writeJSON(w, http.StatusOK, map[string]string{"status": "signal sent"})
}

func (h *Handler) Comment(w http.ResponseWriter, r *http.Request) {
	id := chi.URLParam(r, "id")
	var sig engine.CommentSignal
	if err := json.NewDecoder(r.Body).Decode(&sig); err != nil {
		writeError(w, http.StatusBadRequest, "Invalid Comment Signal")
		return
	}

	state, err := h.svc.GetStatus(r.Context(), id)
	if err == nil {
		if dept, ok := state.Progress[sig.DeptID]; ok {
			assignees := dept.StageAssignees[dept.CurrentStage]
			isAssigned := false
			for _, a := range assignees {
				if a == sig.UserID {
					isAssigned = true
					break
				}
			}
			if !isAssigned && !h.iam.IsAdmin(sig.UserID) {
				writeError(w, http.StatusForbidden, "User is not assigned to this stage")
				return
			}
		}
	}

	if err := h.svc.SendComment(r.Context(), id, sig); err != nil {
		writeError(w, http.StatusInternalServerError, err.Error())
		return
	}
	writeJSON(w, http.StatusOK, map[string]string{"status": "comment sent"})
}

func (h *Handler) AdminRoute(w http.ResponseWriter, r *http.Request) {
	id := chi.URLParam(r, "id")
	var sig engine.AdminRoutingSignal
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
