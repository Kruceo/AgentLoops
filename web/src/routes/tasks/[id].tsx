import { createResource } from "solid-js";
import { useParams, useNavigate, A } from "@solidjs/router";
import { api } from "~/lib/api";

export default function TaskDetail() {
  const params = useParams();
  const navigate = useNavigate();
  const id = () => params.id as string;
  const [task] = createResource(id, api.getTask);
  const [runs, { refetch: refetchRuns }] = createResource(id, api.getTaskRuns);

  const handleRunNow = async () => {
    try {
      await api.runTask(id());
      refetchRuns();
    } catch (err: any) {
      alert(`Failed to run task: ${err.message}`);
    }
  };

  const handleDelete = async () => {
    if (!confirm("Are you sure you want to delete this task?")) return;
    try {
      await api.deleteTask(id());
      navigate("/");
    } catch (err: any) {
      alert(`Failed to delete task: ${err.message}`);
    }
  };



  const statusColor = (status: string | null) => {
    if (!status) return "text-gray-500";
    if (status === "success") return "text-green-400";
    if (status === "error" || status === "failed") return "text-red-400";
    if (status === "running") return "text-blue-400";
    return "text-yellow-400";
  };

  const formatDate = (dateStr: string | null | undefined) => {
    if (!dateStr) return "—";
    return new Date(dateStr).toLocaleString();
  };

  return (
    <div class="min-h-screen bg-gray-950 text-gray-100">
      {/* Header */}
      <header class="border-b border-gray-800 bg-gray-900/50 backdrop-blur-sm">
        <div class="max-w-7xl mx-auto px-4 sm:px-6 lg:px-8 h-16 flex items-center justify-between">
          <div class="flex items-center gap-4">
            <A href="/" class="text-gray-400 hover:text-white transition-colors">
              <svg class="w-5 h-5" fill="none" stroke="currentColor" viewBox="0 0 24 24">
                <path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M15 19l-7-7 7-7" />
              </svg>
            </A>
            <div class="w-8 h-8 rounded-lg bg-gradient-to-br from-indigo-500 to-purple-600 flex items-center justify-center text-white font-bold text-sm">
              AL
            </div>
            <h1 class="text-xl font-semibold text-white truncate">
              {task() ? task().taskName : "Loading..."}
            </h1>
          </div>
          <div class="flex items-center gap-2">
            <button
              onClick={handleRunNow}
              class="inline-flex items-center gap-2 px-4 py-2 rounded-lg bg-emerald-700 hover:bg-emerald-600 text-emerald-100 text-sm font-medium transition-colors"
            >
              <svg class="w-4 h-4" fill="none" stroke="currentColor" viewBox="0 0 24 24">
                <path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M14.752 11.168l-3.197-2.132A1 1 0 0010 9.87v4.263a1 1 0 001.555.832l3.197-2.132a1 1 0 000-1.664z" />
                <path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M21 12a9 9 0 11-18 0 9 9 0 0118 0z" />
              </svg>
              Run Now
            </button>
            <button
              onClick={() => navigate(`/tasks/${id()}/edit`)}
              class="inline-flex items-center gap-2 px-4 py-2 rounded-lg bg-gray-800 hover:bg-gray-700 text-gray-200 text-sm font-medium transition-colors"
            >
              <svg class="w-4 h-4" fill="none" stroke="currentColor" viewBox="0 0 24 24">
                <path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M11 5H6a2 2 0 00-2 2v11a2 2 0 002 2h11a2 2 0 002-2v-5m-1.414-9.414a2 2 0 112.828 2.828L11.828 15H9v-2.828l8.586-8.586z" />
              </svg>
              Edit
            </button>
            <button
              onClick={handleDelete}
              class="inline-flex items-center gap-2 px-4 py-2 rounded-lg bg-red-900/50 hover:bg-red-800 text-red-300 text-sm font-medium transition-colors"
            >
              <svg class="w-4 h-4" fill="none" stroke="currentColor" viewBox="0 0 24 24">
                <path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M19 7l-.867 12.142A2 2 0 0116.138 21H7.862a2 2 0 01-1.995-1.858L5 7m5 4v6m4-6v6m1-10V4a1 1 0 00-1-1h-4a1 1 0 00-1 1v3M4 7h16" />
              </svg>
              Delete
            </button>
          </div>
        </div>
      </header>

      <main class="max-w-7xl mx-auto px-4 sm:px-6 lg:px-8 py-8">
        {/* Loading */}
        {task.loading && (
          <div class="flex items-center justify-center py-24">
            <div class="flex flex-col items-center gap-3">
              <div class="w-8 h-8 border-2 border-indigo-500 border-t-transparent rounded-full animate-spin" />
              <p class="text-gray-400 text-sm">Loading task...</p>
            </div>
          </div>
        )}

        {/* Error */}
        {task.error && (
          <div class="bg-red-900/20 border border-red-800 rounded-xl p-6 text-center">
            <p class="text-red-400 font-medium">Failed to load task</p>
            <p class="text-red-300/70 text-sm mt-1">{(task.error as any)?.message || "Unknown error"}</p>
            <A href="/" class="mt-3 inline-block px-4 py-2 rounded-lg bg-gray-800 hover:bg-gray-700 text-white text-sm transition-colors">
              Back to Dashboard
            </A>
          </div>
        )}

        {task() && (
          <>
            {/* Task Details Card */}
            <div class="rounded-xl border border-gray-800 bg-gray-900/50 p-6 mb-8">
              <h3 class="text-lg font-semibold text-white mb-4">Task Details</h3>
              <div class="grid grid-cols-1 md:grid-cols-2 lg:grid-cols-3 gap-6">
                <div>
                  <label class="block text-xs font-medium text-gray-500 uppercase tracking-wider mb-1">Task Name</label>
                  <p class="text-sm text-white">{task().taskName}</p>
                </div>
                <div>
                  <label class="block text-xs font-medium text-gray-500 uppercase tracking-wider mb-1">Agent Runner</label>
                  <p class="text-sm text-gray-300">{task().agentRunner || "—"}</p>
                </div>
                <div>
                  <label class="block text-xs font-medium text-gray-500 uppercase tracking-wider mb-1">Agent Model</label>
                  <p class="text-sm text-gray-300">{task().agentModel || "—"}</p>
                </div>
                <div>
                  <label class="block text-xs font-medium text-gray-500 uppercase tracking-wider mb-1">Agent Mode</label>
                  <p class="text-sm text-gray-300">{task().agentMode || "—"}</p>
                </div>
                <div>
                  <label class="block text-xs font-medium text-gray-500 uppercase tracking-wider mb-1">Interval</label>
                  <p class="text-sm text-gray-300">{task().intervalSeconds ?? "—"} seconds</p>
                </div>
                <div>
                  <label class="block text-xs font-medium text-gray-500 uppercase tracking-wider mb-1">Enabled</label>
                  <p class={`text-sm font-medium ${task().enabled ? "text-green-400" : "text-gray-500"}`}>
                    {task().enabled ? "Yes" : "No"}
                  </p>
                </div>
                <div>
                  <label class="block text-xs font-medium text-gray-500 uppercase tracking-wider mb-1">Work Directory</label>
                  <p class="text-sm text-gray-300 font-mono text-xs">{task().workDir || "—"}</p>
                </div>
                <div>
                  <label class="block text-xs font-medium text-gray-500 uppercase tracking-wider mb-1">Last Run Status</label>
                  <p class={`text-sm font-medium ${statusColor(task().lastRunStatus)}`}>
                    {task().lastRunStatus || "never run"}
                  </p>
                </div>
                <div>
                  <label class="block text-xs font-medium text-gray-500 uppercase tracking-wider mb-1">Created At</label>
                  <p class="text-sm text-gray-300">{formatDate(task().createdAt)}</p>
                </div>
              </div>
              {task().initMessage && (
                <div class="mt-6">
                  <label class="block text-xs font-medium text-gray-500 uppercase tracking-wider mb-2">Init Message</label>
                  <pre class="text-sm text-gray-300 bg-gray-950 rounded-lg p-4 border border-gray-800 whitespace-pre-wrap font-mono text-xs">
                    {task().initMessage}
                  </pre>
                </div>
              )}
            </div>

            {/* Runs */}
            <div>
              <div class="flex items-center justify-between mb-4">
                <h3 class="text-lg font-semibold text-white">Runs</h3>
                <button
                  onClick={() => refetchRuns()}
                  class="text-sm text-gray-400 hover:text-white transition-colors"
                >
                  Refresh
                </button>
              </div>

              {runs.loading && (
                <div class="flex items-center justify-center py-12">
                  <div class="w-6 h-6 border-2 border-indigo-500 border-t-transparent rounded-full animate-spin" />
                </div>
              )}

              {runs() && runs()!.length === 0 && (
                <div class="border border-dashed border-gray-700 rounded-xl p-12 text-center">
                  <p class="text-gray-500">No runs yet. Click "Run Now" to execute this task.</p>
                </div>
              )}

              {runs() && runs()!.length > 0 && (
                <div class="space-y-3">
                  {runs()!.map((run: any) => (
                    <div class="rounded-xl border border-gray-800 bg-gray-900/50 p-4">
                      <div class="flex items-center justify-between mb-2">
                        <div class="flex items-center gap-3">
                          <span class={`inline-flex items-center px-2.5 py-0.5 rounded-full text-xs font-medium ${
                            run.status === "success"
                              ? "bg-green-900/50 text-green-300"
                              : run.status === "error" || run.status === "failed"
                              ? "bg-red-900/50 text-red-300"
                              : run.status === "running"
                              ? "bg-blue-900/50 text-blue-300"
                              : "bg-gray-800 text-gray-400"
                          }`}>
                            {run.status || "unknown"}
                          </span>
                          <span class="text-xs text-gray-500">
                            {formatDate(run.startedAt)}
                          </span>
                        </div>
                        {run.finishedAt && (
                          <span class="text-xs text-gray-500">
                            Finished: {formatDate(run.finishedAt)}
                          </span>
                        )}
                      </div>
                      {run.output && (
                        <pre class="text-xs text-gray-400 bg-gray-950 rounded-lg p-3 border border-gray-800 whitespace-pre-wrap font-mono max-h-32 overflow-y-auto">
                          {run.output.length > 500
                            ? run.output.slice(0, 500) + "\n..."
                            : run.output}
                        </pre>
                      )}
                    </div>
                  ))}
                </div>
              )}
            </div>
          </>
        )}
      </main>
    </div>
  );
}
