import React, { useState, useEffect, useCallback, useMemo } from "react";
import { api } from "../api/client";
import "./WorkflowDashboard.css";

interface WorkflowRun {
  id: string;
  name: string;
  created_at: string;
  status: string;
}

interface Comment {
  user_id: string;
  text: string;
  stage: string;
}

interface Document {
  id: string;
  filename: string;
  user_id: string;
  dept_id: string;
  stage: string;
  created_at: string;
}

interface DepartmentProgress {
  dept_id: string;
  label: string;
  current_stage: string;
  stage_status: string;
  stage_assignees: Record<string, string>;
  stage_assignee_names: Record<string, string>;
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
      exclusive?: string[];
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

interface User {
  id: string;
  name: string;
  isAdmin: boolean;
  dept?: string;
  role?: string;
}

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
  const [users, setUsers] = useState<User[]>([]);
  const [currentUser, setCurrentUser] = useState<User | null>(null);
  const [deptDocs, setDeptDocs] = useState<Record<string, Document[]>>({});
  const [viewDoc, setViewDoc] = useState<Document | null>(null);

  const [workloads, setWorkloads] = useState<Record<string, number>>({});
  const [activeTab, setActiveTab] = useState<"all" | "admin">("all");
  const [showAssignModal, setShowAssignModal] = useState(false);
  const [assignments, setAssignments] = useState<
    Record<string, Record<string, { user_id: string; user_name: string }>>
  >({});

  const loadWorkloads = useCallback(async () => {
    try {
      const w = await api.getWorkloads(currentUser?.id);
      setWorkloads(w || {});
    } catch (e: any) {
      console.error("Failed to load workloads", e);
    }
  }, [currentUser?.id]);

  const loadUsers = useCallback(async () => {
    try {
      const data = await api.getUsers();
      const flat: User[] = [];
      data.admins?.forEach((u: any) =>
        flat.push({ id: u.user_id, name: u.name, isAdmin: true }),
      );
      Object.entries(data.departments || {}).forEach(([dept, roles]: any) => {
        Object.entries(roles).forEach(([role, rUsers]: any) => {
          rUsers.forEach((u: any) => {
            flat.push({
              id: u.user_id,
              name: `${u.name} (${dept} ${role})`,
              isAdmin: false,
              dept,
              role,
            });
          });
        });
      });
      setUsers(flat);
      if (flat.length > 0) setCurrentUser(flat[0]);
    } catch (e: any) {
      showToast("Error loading users: " + e.message);
    }
  }, []);

  useEffect(() => {
    loadUsers();
  }, [loadUsers]);

  const loadRuns = useCallback(async () => {
    try {
      const data = await api.listWorkflows(currentUser?.id);
      setRuns(data ?? []);
    } catch {}
  }, [currentUser?.id]);

  const loadStatus = useCallback(async (id: string) => {
    try {
      const data = await api.getStatus(id);
      setState(data);
      if (data.progress) {
        const docsMap: Record<string, Document[]> = {};
        const allDocs = await api.getDocuments(id);
        allDocs.forEach((d: Document) => {
          if (!docsMap[d.dept_id]) docsMap[d.dept_id] = [];
          docsMap[d.dept_id].push(d);
        });
        setDeptDocs(docsMap);
      }
    } catch (e: any) {
      showToast("Error loading status: " + e.message);
    }
  }, []);

  useEffect(() => {
    loadRuns();
  }, [loadRuns]);

  useEffect(() => {
    if (activeTab === "admin") {
      loadWorkloads();
    }
  }, [activeTab, loadWorkloads]);

  useEffect(() => {
    if (initialWorkflowId) {
      setSelectedId(initialWorkflowId);
      loadRuns();
    }
  }, [initialWorkflowId, loadRuns]);

  useEffect(() => {
    if (!selectedId) return;
    loadStatus(selectedId);
    const interval = setInterval(() => loadStatus(selectedId), 2000);
    return () => clearInterval(interval);
  }, [selectedId, loadStatus]);

  useEffect(() => {
    if (showAdmin && state?.rejected_by) {
      setAdminGotoDept(state.rejected_by);
    }
  }, [showAdmin, state?.rejected_by]);

  const showToast = (msg: string) => {
    setToast(msg);
    setTimeout(() => setToast(null), 2000);
  };

  const handleTransition = async (deptId: string, toStage: string) => {
    if (!selectedId || !currentUser) return;
    setActionLoading(`${deptId}-${toStage}`);
    try {
      await api.transition(selectedId, deptId, toStage, currentUser.id);
      showToast(`Transitioned ${deptId} → ${toStage}`);
      await loadStatus(selectedId);
    } catch (e: any) {
      showToast("Error: " + e.message);
    } finally {
      setActionLoading(null);
    }
  };

  const handleReject = async (deptId: string) => {
    if (!selectedId || !currentUser) return;
    setActionLoading(`${deptId}-reject`);
    try {
      await api.transition(selectedId, deptId, "reject", currentUser.id);
      showToast(`Rejected: ${deptId}`);
      await loadStatus(selectedId);
    } catch (e: any) {
      showToast("Error: " + e.message);
    } finally {
      setActionLoading(null);
    }
  };

  const handleComment = async (deptId: string, stage: string) => {
    if (!selectedId || !currentUser) return;
    const text = commentText[deptId]?.trim();
    if (!text) {
      showToast("Comment cannot be empty");
      return;
    }
    setActionLoading(`${deptId}-comment`);
    try {
      await api.comment(selectedId, deptId, stage, currentUser.id, text);
      setCommentText((prev) => ({ ...prev, [deptId]: "" }));
      showToast("Comment added");
      await loadStatus(selectedId);
    } catch (e: any) {
      showToast("Error: " + e.message);
    } finally {
      setActionLoading(null);
    }
  };

  const handleUploadDocument = async (deptId: string, stage: string) => {
    if (!selectedId || !currentUser) return;
    const filename = prompt("Enter mock document filename:", "document_v1.pdf");
    if (!filename) return;
    setActionLoading(`${deptId}-upload`);
    try {
      await api.uploadDocument(
        selectedId,
        deptId,
        stage,
        filename,
        currentUser.id,
      );
      showToast("Document uploaded");
      await loadStatus(selectedId);
    } catch (e: any) {
      showToast("Error: " + e.message);
    } finally {
      setActionLoading(null);
    }
  };

  const handleAdminRoute = async (action: string) => {
    if (!selectedId || !currentUser?.isAdmin) return;
    try {
      await api.adminRoute(
        selectedId,
        action,
        adminGotoDept,
        adminGotoStage,
        currentUser.id,
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

  const deptList = useMemo(() => {
    if (!state?.progress) return [];

    const ordered: DepartmentProgress[] = [];
    const seen = new Set<string>();

    if (state?.execution?.steps) {
      state.execution.steps.forEach((step: any) => {
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

    // 2. Add any remaining departments not in execution plan (fallback)
    Object.keys(state.progress).forEach((id) => {
      if (!seen.has(id)) {
        ordered.push(state.progress[id]);
      }
    });

    return ordered;
  }, [state]);

  const hasPermission = (deptId: string, stage: string) => {
    if (currentUser?.isAdmin) return true;
    if (!state?.progress[deptId]) return false;
    const assignee = state.progress[deptId].stage_assignees?.[stage];
    return currentUser?.id === assignee;
  };

  const filteredRuns = runs.filter((r) => {
    if (activeTab === "admin") {
      return r.status === "pending_assignment";
    }
    return true;
  });

  const getAvailableUsers = (deptId: string, stage: string) => {
    const roleMap: Record<string, string> = {
      prep: "preparer",
      review: "reviewer",
      approve: "approver",
    };
    return users.filter((u) => u.dept === deptId && u.role === roleMap[stage]);
  };

  const handleOpenAssignModal = () => {
    // Build auto-assignments based on workloads
    const initial: Record<
      string,
      Record<string, { user_id: string; user_name: string }>
    > = {};
    deptList.forEach((dept) => {
      initial[dept.dept_id] = {};
      STAGE_ORDER.forEach((stage) => {
        const avail = getAvailableUsers(dept.dept_id, stage);
        if (avail.length > 0) {
          const sorted = [...avail].sort(
            (a, b) => (workloads[a.id] || 0) - (workloads[b.id] || 0),
          );
          initial[dept.dept_id][stage] = {
            user_id: sorted[0].id,
            user_name: sorted[0].name,
          };
        }
      });
    });
    setAssignments(initial);
    setShowAssignModal(true);
  };

  const groupedUsers = useMemo(() => {
    const groups: Record<string, User[]> = {};
    users.forEach((u) => {
      const groupName = u.isAdmin ? "Admins" : u.dept || "Other";
      if (!groups[groupName]) groups[groupName] = [];
      groups[groupName].push(u);
    });
    return groups;
  }, [users]);

  const handleStartWorkflow = async () => {
    if (!selectedId || !currentUser?.isAdmin) return;
    setActionLoading("start");
    try {
      await api.startWorkflow(selectedId, assignments, currentUser.id);
      showToast("Workflow started with assignments!");
      setShowAssignModal(false);
      await loadStatus(selectedId);
      await loadRuns();
    } catch (e: any) {
      showToast("Error: " + e.message);
    } finally {
      setActionLoading(null);
    }
  };

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

        {currentUser?.isAdmin && (
          <div
            className="tab-switcher"
            style={{
              display: "flex",
              margin: "0 1rem 1rem",
              borderBottom: "1px solid #ddd",
            }}
          >
            <button
              className={`tab-btn ${activeTab === "all" ? "active" : ""}`}
              onClick={() => setActiveTab("all")}
              style={{
                flex: 1,
                padding: "0.5rem",
                background: "none",
                border: "none",
                borderBottom:
                  activeTab === "all"
                    ? "2px solid #0066cc"
                    : "2px solid transparent",
                cursor: "pointer",
                fontWeight: activeTab === "all" ? "bold" : "normal",
              }}
            >
              All
            </button>
            <button
              className={`tab-btn ${activeTab === "admin" ? "active" : ""}`}
              onClick={() => setActiveTab("admin")}
              style={{
                flex: 1,
                padding: "0.5rem",
                background: "none",
                border: "none",
                borderBottom:
                  activeTab === "admin"
                    ? "2px solid #0066cc"
                    : "2px solid transparent",
                cursor: "pointer",
                fontWeight: activeTab === "admin" ? "bold" : "normal",
              }}
            >
              Admin Queue
            </button>
          </div>
        )}

        <div className="user-switcher" style={{ padding: "0 1rem 1rem" }}>
          <label
            style={{
              display: "block",
              fontSize: "0.8rem",
              marginBottom: "4px",
            }}
          >
            Logged in as:
          </label>
          <select
            value={currentUser?.id || ""}
            onChange={(e) =>
              setCurrentUser(users.find((u) => u.id === e.target.value) || null)
            }
            style={{ width: "100%", padding: "4px" }}
          >
            {Object.entries(groupedUsers).map(([groupName, groupUsers]) => (
              <optgroup key={groupName} label={groupName}>
                {groupUsers.map((u) => (
                  <option key={u.id} value={u.id}>
                    {u.name}
                  </option>
                ))}
              </optgroup>
            ))}
          </select>
        </div>
        {filteredRuns.length === 0 && (
          <p className="sidebar-empty">No workflows match the current view.</p>
        )}
        <ul className="workflow-list">
          {filteredRuns.map((run) => (
            <li key={run.id}>
              <button
                type="button"
                className={`workflow-item ${selectedId === run.id ? "selected" : ""}`}
                onClick={() => setSelectedId(run.id)}
                id={`workflow-item-${run.id}`}
              >
                <div className="wf-item-info">
                  <span className="wf-name">{run.name}</span>
                  <span className={`wf-status-pill status-${run.status}`}>
                    {run.status?.replace("_", " ")}
                  </span>
                </div>
                <span className="wf-date">
                  {new Date(run.created_at).toLocaleDateString()}
                </span>
              </button>
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
                {state.status === "pending_assignment" &&
                  currentUser?.isAdmin && (
                    <button
                      className="btn-admin"
                      onClick={handleOpenAssignModal}
                      id="btn-admin-assign"
                    >
                      Assign & Start Workflow
                    </button>
                  )}
                {state.status === "paused" && (
                  <button
                    className="btn-admin"
                    onClick={() => setShowAdmin(true)}
                    id="btn-admin-route"
                  >
                    Admin: Route Rejection
                  </button>
                )}
                {state.status === "paused_xor" && currentUser?.isAdmin && (
                  <button
                    className="btn-admin"
                    onClick={() =>
                      setAdminGotoDept(
                        state.execution.steps[state.current_step]
                          ?.exclusive?.[0] || "",
                      )
                    }
                    id="btn-admin-xor"
                  >
                    Admin: Select Path
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

                let advanceButtonLabel = "→ Send for Approval";
                if (dept.current_stage === "prep") {
                  advanceButtonLabel = "→ Send for Review";
                } else if (
                  dept.current_stage === "review" &&
                  dept.has_comment
                ) {
                  advanceButtonLabel = "→ Return to Prep (Address Comments)";
                }

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
                        {dept.stage_assignee_names?.[dept.current_stage] && (
                          <span className="dept-assignee">
                            👤 {dept.stage_assignee_names[dept.current_stage]}
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
                              {STAGE_LABELS[s] || s}
                            </span>
                          </div>
                        );
                      })}
                    </div>

                    {dept.stage_status === "in_progress" &&
                      !isRejected &&
                      state.status !== "paused" && (
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

                          {hasPermission(dept.dept_id, dept.current_stage) && (
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
                                    {advanceButtonLabel}
                                  </button>
                                )}
                              {dept.current_stage === "approve" && (
                                <button
                                  className="btn-sm btn-success"
                                  onClick={() =>
                                    handleTransition(
                                      dept.dept_id,
                                      dept.current_stage,
                                    )
                                  }
                                  disabled={
                                    !!actionLoading ||
                                    (dept.current_stage === "approve" &&
                                      !dept.has_comment)
                                  }
                                  id={`btn-approve-${dept.dept_id}`}
                                  title={
                                    dept.current_stage === "approve" &&
                                    !dept.has_comment
                                      ? "Add a comment before approving"
                                      : ""
                                  }
                                >
                                  ✓{" "}
                                  {dept.current_stage === "approve"
                                    ? "Approve"
                                    : "Complete"}
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
                          )}
                        </div>
                      )}

                    {/* Documents Section */}
                    {state.status !== "paused" && (
                      <div
                        className="dept-actions"
                        style={{
                          marginTop: "1rem",
                          borderTop: "1px dashed #ccc",
                          paddingTop: "0.5rem",
                        }}
                      >
                        <h4
                          style={{
                            margin: "0 0 0.5rem",
                            fontSize: "0.85rem",
                            color: "#666",
                          }}
                        >
                          Stage Documents
                        </h4>
                        {deptDocs[dept.dept_id]?.length > 0 ? (
                          <ul className="comment-list">
                            {deptDocs[dept.dept_id].map((d) => {
                              const canOpen =
                                currentUser?.isAdmin ||
                                currentUser?.dept === d.dept_id;
                              return (
                                <li
                                  key={d.id}
                                  className="comment-item"
                                  style={{
                                    background: "#f8f9fa",
                                    border: "1px solid #ddd",
                                    opacity: canOpen ? 1 : 0.6,
                                    padding: 0,
                                  }}
                                >
                                  <button
                                    onClick={() => {
                                      if (canOpen) {
                                        setViewDoc(d);
                                      } else {
                                        showToast(
                                          "You do not have permission to open documents from other departments.",
                                        );
                                      }
                                    }}
                                    style={{
                                      fontSize: "0.8rem",
                                      cursor: canOpen
                                        ? "pointer"
                                        : "not-allowed",
                                      background: "transparent",
                                      border: "none",
                                      width: "100%",
                                      textAlign: "left",
                                      padding: "0.5rem",
                                      display: "flex",
                                      gap: "0.5rem",
                                    }}
                                  >
                                    <span
                                      className="comment-user"
                                      style={{
                                        color: canOpen ? "#0066cc" : "#666",
                                        textDecoration: canOpen
                                          ? "underline"
                                          : "none",
                                      }}
                                    >
                                      📄 {d.filename}
                                    </span>
                                    <span className="comment-text">
                                      by {d.user_id} ({d.stage})
                                    </span>
                                  </button>
                                </li>
                              );
                            })}
                          </ul>
                        ) : (
                          <p
                            style={{
                              fontSize: "0.8rem",
                              color: "#999",
                              margin: "0 0 0.5rem",
                            }}
                          >
                            No documents yet.
                          </p>
                        )}
                        {hasPermission(dept.dept_id, dept.current_stage) &&
                          dept.stage_status === "in_progress" &&
                          !isRejected && (
                            <button
                              className="btn-sm btn-secondary"
                              onClick={() =>
                                handleUploadDocument(
                                  dept.dept_id,
                                  dept.current_stage,
                                )
                              }
                              disabled={
                                actionLoading === `${dept.dept_id}-upload`
                              }
                            >
                              + Upload Mock Doc
                            </button>
                          )}
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
        <div className="modal-overlay">
          <button
            type="button"
            className="modal-backdrop"
            onClick={() => setShowAdmin(false)}
            aria-label="Close modal"
          />
          <div className="modal" id="admin-routing-modal">
            <h3>Admin: Route After Rejection</h3>
            <p>
              Workflow rejected by: <strong>{state.rejected_by}</strong>
            </p>
            <label htmlFor="admin-goto-dept">Go to Department</label>
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
            <label htmlFor="admin-goto-stage">Go to Stage</label>
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

      {state?.status === "paused_xor" &&
        currentUser?.isAdmin &&
        adminGotoDept !== "" && (
          <div className="modal-overlay">
            <button
              type="button"
              className="modal-backdrop"
              onClick={() => setAdminGotoDept("")}
            />
            <div className="modal" id="admin-xor-modal">
              <h3>Admin: Select XOR Path</h3>
              <p>
                Workflow is at an Exclusive Gateway. Choose which department
                path to execute:
              </p>
              <label htmlFor="admin-xor-select">Path</label>
              <select
                value={adminGotoDept}
                onChange={(e) => setAdminGotoDept(e.target.value)}
                id="admin-xor-select"
              >
                {(
                  state.execution.steps[state.current_step]?.exclusive || []
                ).map((deptId) => (
                  <option key={deptId} value={deptId}>
                    {state.progress[deptId]?.label || deptId}
                  </option>
                ))}
              </select>
              <div className="modal-actions">
                <button
                  className="btn-primary"
                  onClick={async () => {
                    await handleAdminRoute("xor_route");
                    setAdminGotoDept("");
                  }}
                >
                  Execute Selected
                </button>
                <button
                  className="btn-ghost"
                  onClick={() => setAdminGotoDept("")}
                >
                  Cancel
                </button>
              </div>
            </div>
          </div>
        )}

      {viewDoc && (
        <div className="modal-overlay">
          <button
            type="button"
            className="modal-backdrop"
            onClick={() => setViewDoc(null)}
          />
          <div className="modal">
            <h3>Document Viewer: {viewDoc.filename}</h3>
            <div
              style={{
                backgroundColor: "#1e1e1e",
                color: "#d4d4d4",
                padding: "1rem",
                borderRadius: "4px",
                fontFamily: "monospace",
                whiteSpace: "pre-wrap",
                margin: "1rem 0",
              }}
            >
              {JSON.stringify(
                {
                  id: viewDoc.id,
                  title: viewDoc.filename,
                  author: viewDoc.user_id,
                  stage: viewDoc.stage,
                  department: viewDoc.dept_id,
                  content:
                    "This is a mock document content for demonstration purposes.",
                  verified: true,
                },
                null,
                2,
              )}
            </div>
            <div className="modal-actions">
              <button className="btn-ghost" onClick={() => setViewDoc(null)}>
                Close
              </button>
            </div>
          </div>
        </div>
      )}

      {showYaml && yaml && (
        <div className="modal-overlay">
          <button
            type="button"
            className="modal-backdrop"
            onClick={() => setShowYaml(false)}
            aria-label="Close YAML view"
          />
          <div className="modal modal-wide" id="yaml-modal">
            <h3>Generated DSL YAML</h3>
            <pre className="yaml-block">{yaml}</pre>
            <button className="btn-ghost" onClick={() => setShowYaml(false)}>
              Close
            </button>
          </div>
        </div>
      )}

      {showAssignModal && state && (
        <div className="modal-overlay">
          <button
            type="button"
            className="modal-backdrop"
            onClick={() => setShowAssignModal(false)}
            aria-label="Close modal"
          />
          <div className="modal modal-wide" id="assign-modal">
            <h3>Admin: Assign Users & Start</h3>
            <p>
              Review the auto-assigned users based on workloads, or change them
              manually before starting the workflow.
            </p>
            <div
              className="assignments-form"
              style={{
                maxHeight: "60vh",
                overflowY: "auto",
                paddingRight: "1rem",
              }}
            >
              {deptList.map((dept) => (
                <div
                  key={dept.dept_id}
                  className="dept-assignment-group"
                  style={{
                    marginBottom: "1.5rem",
                    border: "1px solid #eee",
                    padding: "1rem",
                    borderRadius: "8px",
                  }}
                >
                  <h4 style={{ margin: "0 0 1rem 0" }}>
                    {dept.label || dept.dept_id}
                  </h4>
                  {STAGE_ORDER.map((stage) => {
                    const avail = getAvailableUsers(dept.dept_id, stage);
                    const currentAssign =
                      assignments[dept.dept_id]?.[stage]?.user_id || "";
                    return (
                      <div
                        key={stage}
                        style={{
                          marginBottom: "0.8rem",
                          display: "flex",
                          flexDirection: "column",
                        }}
                      >
                        <label
                          style={{
                            fontSize: "0.85rem",
                            fontWeight: "bold",
                            marginBottom: "0.3rem",
                          }}
                        >
                          {STAGE_LABELS[stage]} ({avail.length} available)
                        </label>
                        <select
                          value={currentAssign}
                          onChange={(e) => {
                            const selectedUser = avail.find(
                              (u) => u.id === e.target.value,
                            );
                            if (selectedUser) {
                              setAssignments((prev) => ({
                                ...prev,
                                [dept.dept_id]: {
                                  ...prev[dept.dept_id],
                                  [stage]: {
                                    user_id: selectedUser.id,
                                    user_name: selectedUser.name,
                                  },
                                },
                              }));
                            }
                          }}
                          style={{ padding: "0.5rem" }}
                        >
                          {avail.map((u) => (
                            <option key={u.id} value={u.id}>
                              {u.name.replace(/\s\(.*\)/, "")} (Workload:{" "}
                              {workloads[u.id] || 0})
                            </option>
                          ))}
                          {avail.length === 0 && (
                            <option value="" disabled>
                              No users match role
                            </option>
                          )}
                        </select>
                      </div>
                    );
                  })}
                </div>
              ))}
            </div>
            <div className="modal-actions" style={{ marginTop: "1rem" }}>
              <button
                className="btn-primary"
                onClick={handleStartWorkflow}
                disabled={!!actionLoading}
              >
                {actionLoading === "start" ? "Starting..." : "Start Workflow"}
              </button>
              <button
                className="btn-ghost"
                onClick={() => setShowAssignModal(false)}
                disabled={!!actionLoading}
              >
                Cancel
              </button>
            </div>
          </div>
        </div>
      )}
    </div>
  );
};

export default WorkflowDashboard;
