import { createResource, createSignal, onCleanup } from "solid-js";
import { useNavigate, A } from "@solidjs/router";
import { api } from "~/lib/api";

export default function Dashboard() {
  const navigate = useNavigate();
  const [refreshKey, setRefreshKey] = createSignal(0);
  const [tasks, { refetch }] = createResource(refreshKey, () => api.getTasks());

  // Auto-refresh every 10 seconds
  const interval = setInterval(() => setRefreshKey((k) => k + 1), 10000);
  onCleanup(() => clearInterval(interval));

  const handleRunNow = async (id: string, e: Event) => {
    e.stopPropagation();
    try {
      await api.runTask(id);
      refetch();
    } catch (err: any) {
      alert(`Failed to run task: ${err.message}`);
    }
  };

  const handleDelete = async (id: string, e: Event) => {
    e.stopPropagation();
    if (!confirm("Are you sure you want to delete this task?")) return;
    try {
      await api.deleteTask(id);
      refetch();
    } catch (err: any) {
      alert(`Failed to delete task: ${err.message}`);
    }
  };

  const handleToggleEnabled = async (id: string, current: boolean, e: Event) => {
    e.stopPropagation();
    try {
      const task = await api.getTask(id);
      await api.updateTask(id, { ...task, enabled: !current });
      refetch();
    } catch (err: any) {
      alert(`Failed to toggle task: ${err.message}`);
    }
  };

  const statusColor = (status: string | null) => {
    if (!status) return "text-gray-500";
    if (status === "success") return "text-green-400";
    if (status === "error" || status === "failed") return "text-red-400";
    if (status === "running") return "text-blue-400";
    return "text-yellow-400";
  };

  return (
    <div class="min-h-screen bg-gray-950 text-gray-100">
      {/* Header */}
      <header class="border-b border-gray-800 bg-gray-900/50 backdrop-blur-sm">
        <div class="max-w-7xl mx-auto px-4 sm:px-6 lg:px-8 h-16 flex items-center justify-between">
          <div class="flex items-center gap-3">
            <div class="w-8 h-8 rounded-lg bg-gradient-to-br from-indigo-500 to-purple-600 flex items-center justify-center text-white font-bold text-sm">
              AL
            </div>
            <h1 class="text-xl font-semibold text-white">AgentLoop</h1>
          </div>
          <button
            onClick={() => navigate("/tasks/create")}
            class="inline-flex items-center gap-2 px-4 py-2 rounded-lg bg-indigo-600 hover:bg-indigo-500 text-white text-sm font-medium transition-colors"
          >
            <svg class="w-4 h-4" fill="none" stroke="currentColor" viewBox="0 0 24 24">
              <path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M12 4v16m8-8H4" />
            </svg>
            New Task
          </button>
        </div>
      </header>

      {/* Main */}
      <main class="max-w-7xl mx-auto px-4 sm:px-6 lg:px-8 py-8">
        <div class="flex items-center justify-between mb-6">
          <div>
            <h2 class="text-2xl font-bold text-white">Tasks</h2>
            <p class="text-gray-400 text-sm mt-1">Manage your agent loop tasks</p>
          </div>
          <div class="flex items-center gap-2 text-sm text-gray-400">
            <span>Auto-refreshing every 10s</span>
            <button
              onClick={() => refetch()}
              class="p-1.5 rounded-lg hover:bg-gray-800 transition-colors"
              title="Refresh now"
            >
              <svg class="w-4 h-4" fill="none" stroke="currentColor" viewBox="0 0 24 24">
                <path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M4 4v5h.582m15.356 2A8.001 8.001 0 004.582 9m0 0H9m11 11v-5h-.581m0 0a8.003 8.003 0 01-15.357-2m15.357 2H15" />
              </svg>
            </button>
          </div>
        </div>

        {/* Loading state */}
        {tasks.loading && !tasks() && (
          <div class="flex items-center justify-center py-24">
            <div class="flex flex-col items-center gap-3">
              <div class="w-8 h-8 border-2 border-indigo-500 border-t-transparent rounded-full animate-spin" />
              <p class="text-gray-400 text-sm">Loading tasks...</p>
            </div>
          </div>
        )}

        {/* Error state */}
        {tasks.error && (
          <div class="bg-red-900/20 border border-red-800 rounded-xl p-6 text-center">
            <p class="text-red-400 font-medium">Failed to load tasks</p>
            <p class="text-red-300/70 text-sm mt-1">{(tasks.error as any)?.message || "Unknown error"}</p>
            <button
              onClick={() => refetch()}
              class="mt-3 px-4 py-2 rounded-lg bg-red-800 hover:bg-red-700 text-white text-sm transition-colors"
            >
              Retry
            </button>
          </div>
        )}

        {/* Empty state */}
        {tasks() && tasks()!.length === 0 && (
          <div class="border border-dashed border-gray-700 rounded-xl p-16 text-center">
            <div class="w-16 h-16 mx-auto rounded-full bg-gray-800 flex items-center justify-center mb-4">
              <svg class="w-8 h-8 text-gray-500" fill="none" stroke="currentColor" viewBox="0 0 24 24">
                <path stroke-linecap="round" stroke-linejoin="round" stroke-width="1.5" d="M9 5H7a2 2 0 00-2 2v12a2 2 0 002 2h10a2 2 0 002-2V7a2 2 0 00-2-2h-2M9 5a2 2 0 002 2h2a2 2 0 002-2M9 5a2 2 0 012-2h2a2 2 0 012 2" />
              </svg>
            </div>
            <h3 class="text-lg font-medium text-gray-300">No tasks yet</h3>
            <p class="text-gray-500 text-sm mt-1">Create your first agent loop task to get started.</p>
            <button
              onClick={() => navigate("/tasks/create")}
              class="mt-4 inline-flex items-center gap-2 px-4 py-2 rounded-lg bg-indigo-600 hover:bg-indigo-500 text-white text-sm font-medium transition-colors"
            >
              <svg class="w-4 h-4" fill="none" stroke="currentColor" viewBox="0 0 24 24">
                <path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M12 4v16m8-8H4" />
              </svg>
              Create Task
            </button>
          </div>
        )}

        {/* Tasks table */}
        {tasks() && tasks()!.length > 0 && (
          <div class="overflow-hidden rounded-xl border border-gray-800 bg-gray-900/50">
            <table class="w-full">
              <thead>
                <tr class="border-b border-gray-800">
                  <th class="text-left px-4 py-3 text-xs font-medium text-gray-500 uppercase tracking-wider">Task Name</th>
                  <th class="text-left px-4 py-3 text-xs font-medium text-gray-500 uppercase tracking-wider">Agent</th>
                  <th class="text-left px-4 py-3 text-xs font-medium text-gray-500 uppercase tracking-wider">Model</th>
                  <th class="text-left px-4 py-3 text-xs font-medium text-gray-500 uppercase tracking-wider">Interval</th>
                  <th class="text-left px-4 py-3 text-xs font-medium text-gray-500 uppercase tracking-wider">Enabled</th>
                  <th class="text-left px-4 py-3 text-xs font-medium text-gray-500 uppercase tracking-wider">Last Run</th>
                  <th class="text-right px-4 py-3 text-xs font-medium text-gray-500 uppercase tracking-wider">Actions</th>
                </tr>
              </thead>
              <tbody class="divide-y divide-gray-800/50">
                {tasks()!.map((task: any) => (
                  <tr
                    class="hover:bg-gray-800/30 cursor-pointer transition-colors"
                    onClick={() => navigate(`/tasks/${task.id}`)}
                  >
                    <td class="px-4 py-3.5">
                      <span class="text-sm font-medium text-white">{task.taskName}</span>
                    </td>
                    <td class="px-4 py-3.5">
                      <span class="text-sm text-gray-400">{task.agentRunner || "—"}</span>
                    </td>
                    <td class="px-4 py-3.5">
                      <span class="text-sm text-gray-400">{task.agentModel || "—"}</span>
                    </td>
                    <td class="px-4 py-3.5">
                      <span class="text-sm text-gray-400">{task.intervalSeconds ?? "—"}s</span>
                    </td>
                    <td class="px-4 py-3.5">
                      <button
                        onClick={(e) => handleToggleEnabled(task.id, task.enabled ?? false, e)}
                        class={`relative inline-flex h-5 w-9 items-center rounded-full transition-colors ${
                          task.enabled ? "bg-indigo-600" : "bg-gray-700"
                        }`}
                      >
                        <span
                          class={`inline-block h-3.5 w-3.5 transform rounded-full bg-white transition-transform ${
                            task.enabled ? "translate-x-[18px]" : "translate-x-[3px]"
                          }`}
                        />
                      </button>
                    </td>
                    <td class="px-4 py-3.5">
                      <div class="flex items-center gap-2">
                        <span class={`text-sm font-medium ${statusColor(task.lastRunStatus)}`}>
                          {task.lastRunStatus || "never"}
                        </span>
                      </div>
                    </td>
                    <td class="px-4 py-3.5 text-right">
                      <div class="flex items-center justify-end gap-1.5">
                        <button
                          onClick={(e) => handleRunNow(task.id, e)}
                          class="px-3 py-1.5 rounded-md bg-emerald-700 hover:bg-emerald-600 text-emerald-100 text-xs font-medium transition-colors"
                        >
                          Run
                        </button>
                        <button
                          onClick={(e) => handleDelete(task.id, e)}
                          class="px-3 py-1.5 rounded-md bg-red-900/50 hover:bg-red-800 text-red-300 text-xs font-medium transition-colors"
                        >
                          Delete
                        </button>
                      </div>
                    </td>
                  </tr>
                ))}
              </tbody>
            </table>
          </div>
        )}
      </main>
    </div>
  );
}
