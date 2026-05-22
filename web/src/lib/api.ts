// SSR runs on Node where there's no Vite proxy — need absolute URL.
// Client runs in browser where Vite proxy handles /api → :8080.
const BASE_URL = import.meta.env.SSR
  ? "http://localhost:8080"
  : "";

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
  getAgentModels: (id: string) => request<string[]>(`/api/agents/${id}/models`),
  getAgentModes: (id: string) => request<string[]>(`/api/agents/${id}/modes`),
};
