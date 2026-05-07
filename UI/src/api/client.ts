const API_BASE = import.meta.env.VITE_API_URL ?? "http://localhost:8080";

export const api = {
  submit: async (xml: string) => {
    const res = await fetch(`${API_BASE}/api/v1/workflows`, {
      method: "POST",
      headers: { "Content-Type": "application/json" },
      body: JSON.stringify({ xml }),
    });
    if (!res.ok) throw new Error(await res.text());
    return res.json();
  },

  listWorkflows: async (userId?: string) => {
    const qs = userId ? `?user_id=${userId}` : "";
    const res = await fetch(`${API_BASE}/api/v1/workflows${qs}`);
    if (!res.ok) throw new Error(await res.text());
    return res.json();
  },

  getStatus: async (id: string) => {
    const res = await fetch(`${API_BASE}/api/v1/workflows/${id}`);
    if (!res.ok) throw new Error(await res.text());
    return res.json();
  },

  getYAML: async (id: string) => {
    const res = await fetch(`${API_BASE}/api/v1/workflows/${id}/yaml`);
    if (!res.ok) throw new Error(await res.text());
    return res.text();
  },

  transition: async (id: string, deptId: string, toStage: string, userId: string) => {
    const res = await fetch(`${API_BASE}/api/v1/workflows/${id}/transition`, {
      method: "POST",
      headers: { "Content-Type": "application/json" },
      body: JSON.stringify({ dept_id: deptId, to_stage: toStage, user_id: userId }),
    });
    if (!res.ok) throw new Error(await res.text());
    return res.json();
  },

  comment: async (id: string, deptId: string, stage: string, userId: string, text: string) => {
    const res = await fetch(`${API_BASE}/api/v1/workflows/${id}/comment`, {
      method: "POST",
      headers: { "Content-Type": "application/json" },
      body: JSON.stringify({ dept_id: deptId, stage, user_id: userId, text }),
    });
    if (!res.ok) throw new Error(await res.text());
    return res.json();
  },

  adminRoute: async (id: string, action: string, deptId: string, stage: string, adminId: string) => {
    const res = await fetch(`${API_BASE}/api/v1/workflows/${id}/route`, {
      method: "POST",
      headers: { "Content-Type": "application/json" },
      body: JSON.stringify({ action, dept_id: deptId, stage, admin_id: adminId }),
    });
    if (!res.ok) throw new Error(await res.text());
    return res.json();
  },

  getUsers: async () => {
    const res = await fetch(`${API_BASE}/api/v1/users`);
    if (!res.ok) throw new Error(await res.text());
    return res.json();
  },

  getWorkloads: async (userId?: string) => {
    const qs = userId ? `?user_id=${userId}` : "";
    const res = await fetch(`${API_BASE}/api/v1/workloads${qs}`);
    if (!res.ok) throw new Error(await res.text());
    return res.json();
  },

  startWorkflow: async (id: string, assignments: Record<string, Record<string, { user_id: string; user_name: string }>>, adminId: string) => {
    const res = await fetch(`${API_BASE}/api/v1/workflows/${id}/start`, {
      method: "POST",
      headers: { "Content-Type": "application/json" },
      body: JSON.stringify({ assignments, admin_id: adminId }),
    });
    if (!res.ok) throw new Error(await res.text());
    return res.json();
  },

  uploadDocument: async (id: string, deptId: string, stage: string, filename: string, userId: string) => {
    const res = await fetch(`${API_BASE}/api/v1/workflows/${id}/documents`, {
      method: "POST",
      headers: { "Content-Type": "application/json" },
      body: JSON.stringify({ dept_id: deptId, stage, filename, user_id: userId }),
    });
    if (!res.ok) throw new Error(await res.text());
    return res.json();
  },

  getDocuments: async (workflowId: string, deptId?: string, stage?: string) => {
    const params = new URLSearchParams();
    if (deptId) params.append("dept_id", deptId);
    if (stage) params.append("stage", stage);
    const qs = params.toString() ? `?${params.toString()}` : "";
    const res = await fetch(`${API_BASE}/api/v1/workflows/${workflowId}/documents${qs}`);
    if (!res.ok) throw new Error(await res.text());
    return res.json();
  },
};
