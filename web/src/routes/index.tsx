import { createResource, createSignal, onCleanup } from "solid-js";
import { useNavigate } from "@solidjs/router";
import { api } from "~/lib/api";
import { Button, PageHeader, Toggle } from "~/components";
import { PlusIcon, RefreshIcon, TaskIcon } from "~/components/icons";

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

  const handleToggleEnabled = async (id: string, current: boolean) => {
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
          <Button pattern="primary" onClick={() => navigate("/tasks/create")} icon={
            <PlusIcon />
          }>New Task</Button>
        </div>
      </header>

      {/* Main */}
      <main class="max-w-7xl mx-auto px-4 sm:px-6 lg:px-8 py-8">
        <PageHeader title="Tasks" description="Manage your agent loop tasks">
          <span>Auto-refreshing every 10s</span>
          <Button pattern="ghost" onClick={() => refetch()} title="Refresh now" icon={
            <RefreshIcon />
          } />
        </PageHeader>

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
            <Button pattern="secondary" onClick={() => refetch()}>Retry</Button>
          </div>
        )}

        {/* Empty state */}
        {tasks() && tasks()!.length === 0 && (
          <div class="border border-dashed border-gray-700 rounded-xl p-16 text-center">
            <div class="w-16 h-16 mx-auto rounded-full bg-gray-800 flex items-center justify-center mb-4">
              <TaskIcon class="text-gray-500" />
            </div>
            <h3 class="text-lg font-medium text-gray-300">No tasks yet</h3>
            <p class="text-gray-500 text-sm mt-1">Create your first agent loop task to get started.</p>
            <Button pattern="primary" onClick={() => navigate("/tasks/create")} icon={
              <PlusIcon />
            }>Create Task</Button>
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
                      <span onClick={(e) => e.stopPropagation()}>
                        <Toggle
                          checked={task.enabled ?? false}
                          onChange={() => handleToggleEnabled(task.id, task.enabled ?? false)}
                        />
                      </span>
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
                        <Button pattern="success" size="sm" onClick={(e) => handleRunNow(task.id, e)}>Run</Button>
                        <Button pattern="danger" size="sm" onClick={(e) => handleDelete(task.id, e)}>Delete</Button>
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
