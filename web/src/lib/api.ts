const BASE_URL = ""; // proxy handles /api routes

async function request<T>(path: string, options?: RequestInit): Promise<T> {
  const res = await fetch(`${BASE_URL}${path}`, {
    headers: { "Content-Type": "application/json" },
    ...options,
  });
  if (!res.ok) {
    const err = await res.json().catch(() => ({ error: res.statusText }));
    throw new Error(err.error || `HTTP ${res.status}`);
  }
  return res.json();
}

export const api = {
  getTasks: (enabledOnly = false) =>
    request<any[]>(`/api/tasks${enabledOnly ? "?enabled=true" : ""}`),
  getTask: (id: string) => request<any>(`/api/tasks/${id}`),
  createTask: (data: any) =>
    request<any>("/api/tasks", { method: "POST", body: JSON.stringify(data) }),
  updateTask: (id: string, data: any) =>
    request<any>(`/api/tasks/${id}`, { method: "PUT", body: JSON.stringify(data) }),
  deleteTask: (id: string) =>
    request<any>(`/api/tasks/${id}`, { method: "DELETE" }),
  runTask: (id: string) =>
    request<any>(`/api/tasks/${id}/run`, { method: "POST" }),
  getTaskRuns: (id: string) =>
    request<any[]>(`/api/tasks/${id}/runs`),
  getRuns: () => request<any[]>("/api/runs"),
  getRun: (id: string) => request<any>(`/api/runs/${id}`),
  getAgents: () => request<any[]>("/api/agents"),
};
