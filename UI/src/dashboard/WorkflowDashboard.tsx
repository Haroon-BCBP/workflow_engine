import React, { useState, useEffect, useCallback } from "react";
import { api } from "../api/client";
import "./WorkflowDashboard.css";

interface WorkflowRun {
  id: string;
  name: string;
  created_at: string;
}

interface Comment {
  user_id: string;
  text: string;
  stage: string;
}

interface DepartmentProgress {
  dept_id: string;
  label: string;
  current_stage: string;
  stage_status: string;
  assignee_id: string;
  assignee_name: string;
  has_comment: boolean;
  comments: Comment[];
}

interface WorkflowState {
  workflow_id: string;
  name: string;
  current_step: number;
  progress: Record<string, DepartmentProgress>;
  execution: {
    steps: Array<{
      sequential?: string[];
      parallel?: string[];
    }>;
  };
  status: string;
  rejected_by?: string;
}

const STAGE_ORDER = ["prep", "review", "approve"];
const STAGE_LABELS: Record<string, string> = {
  prep: "Prep",
  review: "Review",
  approve: "Approve",
};
const STATUS_NEXT: Record<string, string> = {
  prep: "review",
  review: "approve",
};

const CURRENT_USER = { id: "u-d2", name: "Bob Malik", isAdmin: false };
const ADMIN_USER = { id: "admin-1", name: "Admin User", isAdmin: true };

interface Props {
  initialWorkflowId: string | null;
}

const WorkflowDashboard: React.FC<Props> = ({ initialWorkflowId }) => {
  const [runs, setRuns] = useState<WorkflowRun[]>([]);
  const [selectedId, setSelectedId] = useState<string | null>(
    initialWorkflowId,
  );
  const [state, setState] = useState<WorkflowState | null>(null);
  const [yaml, setYaml] = useState<string | null>(null);
  const [commentText, setCommentText] = useState<Record<string, string>>({});
  const [actionLoading, setActionLoading] = useState<string | null>(null);
  const [showAdmin, setShowAdmin] = useState(false);
  const [adminGotoDept, setAdminGotoDept] = useState("");
  const [adminGotoStage, setAdminGotoStage] = useState("prep");
  const [toast, setToast] = useState<string | null>(null);
  const [showYaml, setShowYaml] = useState(false);

  const loadRuns = useCallback(async () => {
    try {
      const data = await api.listWorkflows();
      setRuns(data ?? []);
    } catch {}
  }, []);

  const loadStatus = useCallback(async (id: string) => {
    try {
      const data = await api.getStatus(id);
      setState(data);
    } catch (e: any) {
      showToast("Error loading status: " + e.message);
    }
  }, []);

  useEffect(() => {
    loadRuns();
  }, [loadRuns]);

  useEffect(() => {
    if (initialWorkflowId) {
      setSelectedId(initialWorkflowId);
      loadRuns();
    }
  }, [initialWorkflowId, loadRuns]);

  useEffect(() => {
    if (!selectedId) return;
    loadStatus(selectedId);
    const interval = setInterval(() => loadStatus(selectedId), 3000);
    return () => clearInterval(interval);
  }, [selectedId, loadStatus]);

  const showToast = (msg: string) => {
    setToast(msg);
    setTimeout(() => setToast(null), 3000);
  };

  const handleTransition = async (deptId: string, toStage: string) => {
    if (!selectedId) return;
    setActionLoading(`${deptId}-${toStage}`);
    try {
      await api.transition(selectedId, deptId, toStage, CURRENT_USER.id);
      showToast(`Transitioned ${deptId} → ${toStage}`);
      await loadStatus(selectedId);
    } catch (e: any) {
      showToast("Error: " + e.message);
    } finally {
      setActionLoading(null);
    }
  };

  const handleReject = async (deptId: string) => {
    if (!selectedId) return;
    setActionLoading(`${deptId}-reject`);
    try {
      await api.transition(selectedId, deptId, "reject", CURRENT_USER.id);
      showToast(`Rejected: ${deptId}`);
      await loadStatus(selectedId);
    } catch (e: any) {
      showToast("Error: " + e.message);
    } finally {
      setActionLoading(null);
    }
  };

  const handleComment = async (deptId: string, stage: string) => {
    if (!selectedId) return;
    const text = commentText[deptId]?.trim();
    if (!text) {
      showToast("Comment cannot be empty");
      return;
    }
    setActionLoading(`${deptId}-comment`);
    try {
      await api.comment(selectedId, deptId, stage, CURRENT_USER.id, text);
      setCommentText((prev) => ({ ...prev, [deptId]: "" }));
      showToast("Comment added");
      await loadStatus(selectedId);
    } catch (e: any) {
      showToast("Error: " + e.message);
    } finally {
      setActionLoading(null);
    }
  };

  const handleAdminRoute = async (action: string) => {
    if (!selectedId) return;
    try {
      await api.adminRoute(
        selectedId,
        action,
        adminGotoDept,
        adminGotoStage,
        ADMIN_USER.id,
      );
      showToast(
        action === "terminate"
          ? "Workflow terminated"
          : `Routed to ${adminGotoDept} / ${adminGotoStage}`,
      );
      await loadStatus(selectedId);
      setShowAdmin(false);
    } catch (e: any) {
      showToast("Error: " + e.message);
    }
  };

  const handleLoadYaml = async () => {
    if (!selectedId) return;
    try {
      const y = await api.getYAML(selectedId);
      setYaml(y);
      setShowYaml(true);
    } catch (e: any) {
      showToast("Error loading YAML: " + e.message);
    }
  };

  const deptList: DepartmentProgress[] = state
    ? state.progress
      ? (() => {
          const ordered: DepartmentProgress[] = [];
          const seen = new Set<string>();

          if (state.execution && state.execution.steps) {
            state.execution.steps.forEach((step: any) => {
              // Handle both capitalized and lowercase from backend
              const seq = step.sequential || step.Sequential || [];
              const par = step.parallel || step.Parallel || [];
              const ids = [...seq, ...par];

              ids.forEach((id) => {
                const prog = state.progress[id];
                if (prog && !seen.has(id)) {
                  ordered.push(prog);
                  seen.add(id);
                }
              });
            });
          }

          Object.keys(state.progress).forEach((id) => {
            if (!seen.has(id)) {
              ordered.push(state.progress[id]);
            }
          });
          return ordered;
        })()
      : []
    : [];

  return (
    <div className="dashboard">
      {toast && (
        <div className="toast" id="toast-msg">
          {toast}
        </div>
      )}

      {/* Sidebar — workflow list */}
      <aside className="dashboard-sidebar">
        <div className="sidebar-header">
          <h2>Workflows</h2>
          <button className="btn-ghost" onClick={loadRuns} title="Refresh">
            ↻
          </button>
        </div>
        {runs.length === 0 && (
          <p className="sidebar-empty">
            No workflows yet.
            <br />
            Submit a diagram from the Modeler.
          </p>
        )}
        <ul className="workflow-list">
          {runs.map((run) => (
            <li
              key={run.id}
              className={`workflow-item ${selectedId === run.id ? "selected" : ""}`}
              onClick={() => setSelectedId(run.id)}
              id={`workflow-item-${run.id}`}
            >
              <span className="wf-name">{run.name}</span>
              <span className="wf-date">
                {new Date(run.created_at).toLocaleDateString()}
              </span>
            </li>
          ))}
        </ul>
      </aside>

      {/* Main content */}
      <div className="dashboard-main">
        {!selectedId && (
          <div className="empty-state">
            <div className="empty-icon">📋</div>
            <h3>Select a workflow</h3>
            <p>
              Choose a workflow from the list, or submit a new diagram from the
              BPMN Modeler.
            </p>
          </div>
        )}

        {selectedId && state && (
          <>
            <div className="wf-header">
              <div className="wf-header-info">
                <h2 className="wf-title">{state.name}</h2>
                <span className={`wf-status-badge status-${state.status}`}>
                  {state.status.replace("_", " ")}
                </span>
              </div>
              <div className="wf-header-actions">
                <button
                  className="btn-ghost"
                  onClick={handleLoadYaml}
                  id="btn-view-yaml"
                >
                  View YAML
                </button>
                {state.status === "paused" && (
                  <button
                    className="btn-admin"
                    onClick={() => setShowAdmin(true)}
                    id="btn-admin-route"
                  >
                    Admin: Route Rejection
                  </button>
                )}
              </div>
            </div>

            <div className="dept-grid">
              {deptList.map((dept, idx) => {
                const isCurrent =
                  state.status === "running" &&
                  (dept.stage_status === "in_progress" ||
                    (dept.current_stage === "prep" &&
                      dept.stage_status === "pending" &&
                      idx === 0));
                const isRejected = dept.stage_status === "rejected";
                const isDone =
                  dept.stage_status === "done" &&
                  dept.current_stage === "approve";

                return (
                  <div
                    key={dept.dept_id}
                    className={`dept-card ${isCurrent ? "dept-current" : ""} ${isRejected ? "dept-rejected" : ""} ${isDone ? "dept-done" : ""}`}
                    id={`dept-card-${dept.dept_id}`}
                  >
                    <div className="dept-card-header">
                      <div>
                        <h3 className="dept-label">
                          {dept.label || dept.dept_id}
                        </h3>
                        {dept.assignee_name && (
                          <span className="dept-assignee">
                            👤 {dept.assignee_name}
                          </span>
                        )}
                      </div>
                      <span
                        className={`stage-badge stage-${dept.current_stage}`}
                      >
                        {STAGE_LABELS[dept.current_stage] ?? dept.current_stage}
                      </span>
                    </div>

                    <div className="stage-progress">
                      {STAGE_ORDER.map((s) => {
                        const currentIdx = STAGE_ORDER.indexOf(
                          dept.current_stage,
                        );
                        const sIdx = STAGE_ORDER.indexOf(s);
                        const isActive = s === dept.current_stage;
                        const isComplete =
                          sIdx < currentIdx || (isDone && sIdx <= currentIdx);
                        return (
                          <div
                            key={s}
                            className={`stage-step ${isComplete ? "complete" : ""} ${isActive && !isDone ? "active" : ""}`}
                          >
                            <div className="stage-dot" />
                            <span className="stage-step-label">
                              {STAGE_LABELS[s]}
                            </span>
                          </div>
                        );
                      })}
                    </div>

                    {dept.stage_status === "in_progress" && !isRejected && (
                      <div className="dept-actions">
                        <div className="comment-row">
                          <input
                            className="comment-input"
                            placeholder="Add a comment…"
                            value={commentText[dept.dept_id] ?? ""}
                            onChange={(e) =>
                              setCommentText((prev) => ({
                                ...prev,
                                [dept.dept_id]: e.target.value,
                              }))
                            }
                            id={`comment-input-${dept.dept_id}`}
                          />
                          <button
                            className="btn-sm btn-secondary"
                            onClick={() =>
                              handleComment(dept.dept_id, dept.current_stage)
                            }
                            disabled={
                              actionLoading === `${dept.dept_id}-comment`
                            }
                            id={`btn-comment-${dept.dept_id}`}
                          >
                            Comment
                          </button>
                        </div>

                        {dept.comments?.length > 0 && (
                          <ul className="comment-list">
                            {dept.comments.map((c, i) => (
                              <li key={i} className="comment-item">
                                <span className="comment-user">
                                  {c.user_id}
                                </span>
                                <span className="comment-text">{c.text}</span>
                              </li>
                            ))}
                          </ul>
                        )}

                        <div className="transition-buttons">
                          {dept.current_stage !== "approve" &&
                            STATUS_NEXT[dept.current_stage] && (
                              <button
                                className="btn-sm btn-primary"
                                onClick={() =>
                                  handleTransition(
                                    dept.dept_id,
                                    STATUS_NEXT[dept.current_stage],
                                  )
                                }
                                disabled={!!actionLoading}
                                id={`btn-advance-${dept.dept_id}`}
                              >
                                {dept.current_stage === "prep"
                                  ? "→ Send for Review"
                                  : "→ Send for Approval"}
                              </button>
                            )}
                          {dept.current_stage === "approve" && (
                            <button
                              className="btn-sm btn-success"
                              onClick={() =>
                                handleTransition(dept.dept_id, "approve")
                              }
                              disabled={!!actionLoading || !dept.has_comment}
                              id={`btn-approve-${dept.dept_id}`}
                              title={
                                !dept.has_comment
                                  ? "Add a comment before approving"
                                  : ""
                              }
                            >
                              ✓ Approve
                            </button>
                          )}
                          <button
                            className="btn-sm btn-danger"
                            onClick={() => handleReject(dept.dept_id)}
                            disabled={!!actionLoading}
                            id={`btn-reject-${dept.dept_id}`}
                          >
                            ✕ Reject
                          </button>
                        </div>
                      </div>
                    )}

                    {isRejected && (
                      <div className="rejected-notice">
                        ⚠ Rejected — waiting for admin routing
                      </div>
                    )}

                    {isDone && <div className="done-notice">✓ Approved</div>}
                  </div>
                );
              })}
            </div>
          </>
        )}

        {selectedId && !state && (
          <div className="loading-state">Loading workflow status…</div>
        )}
      </div>

      {showAdmin && state && (
        <div className="modal-backdrop" onClick={() => setShowAdmin(false)}>
          <div
            className="modal"
            onClick={(e) => e.stopPropagation()}
            id="admin-routing-modal"
          >
            <h3>Admin: Route After Rejection</h3>
            <p>
              Workflow rejected by: <strong>{state.rejected_by}</strong>
            </p>
            <label>Go to Department</label>
            <select
              value={adminGotoDept}
              onChange={(e) => setAdminGotoDept(e.target.value)}
              id="admin-goto-dept"
            >
              {deptList.map((d) => (
                <option key={d.dept_id} value={d.dept_id}>
                  {d.label || d.dept_id}
                </option>
              ))}
            </select>
            <label>Go to Stage</label>
            <select
              value={adminGotoStage}
              onChange={(e) => setAdminGotoStage(e.target.value)}
              id="admin-goto-stage"
            >
              {STAGE_ORDER.map((s) => (
                <option key={s} value={s}>
                  {STAGE_LABELS[s]}
                </option>
              ))}
            </select>
            <div className="modal-actions">
              <button
                className="btn-primary"
                onClick={() => handleAdminRoute("goto")}
                id="btn-admin-goto"
              >
                Route to Selected
              </button>
              <button
                className="btn-danger"
                onClick={() => handleAdminRoute("terminate")}
                id="btn-admin-terminate"
              >
                Terminate Workflow
              </button>
              <button className="btn-ghost" onClick={() => setShowAdmin(false)}>
                Cancel
              </button>
            </div>
          </div>
        </div>
      )}

      {showYaml && yaml && (
        <div className="modal-backdrop" onClick={() => setShowYaml(false)}>
          <div
            className="modal modal-wide"
            onClick={(e) => e.stopPropagation()}
            id="yaml-modal"
          >
            <h3>Generated DSL YAML</h3>
            <pre className="yaml-block">{yaml}</pre>
            <button className="btn-ghost" onClick={() => setShowYaml(false)}>
              Close
            </button>
          </div>
        </div>
      )}
    </div>
  );
};

export default WorkflowDashboard;
