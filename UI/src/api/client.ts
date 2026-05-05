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

  listWorkflows: async () => {
    const res = await fetch(`${API_BASE}/api/v1/workflows`);
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
};
