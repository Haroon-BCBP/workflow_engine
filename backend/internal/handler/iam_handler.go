package handler

import (
	"net/http"
)

func (h *Handler) GetUsers(w http.ResponseWriter, r *http.Request) {
	writeJSON(w, http.StatusOK, map[string]any{
		"departments": h.iam.AllDeptRoles(),
		"admins":      h.iam.GetAdmins(),
	})
}

func (h *Handler) GetWorkloads(w http.ResponseWriter, r *http.Request) {
	userID := r.URL.Query().Get("user_id")
	if !h.iam.IsAdmin(userID) {
		writeError(w, http.StatusForbidden, "Only admins can see workloads")
		return
	}
	workloads, err := h.svc.GetWorkloads(r.Context())
	if err != nil {
		writeError(w, http.StatusInternalServerError, err.Error())
		return
	}
	writeJSON(w, http.StatusOK, workloads)
}
