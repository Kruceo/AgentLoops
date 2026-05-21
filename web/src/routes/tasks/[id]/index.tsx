import { createResource } from "solid-js";
import { useParams, useNavigate, A } from "@solidjs/router";
import { api } from "~/lib/api";
import { Button } from "~/components";
import { BackArrowIcon, PlayIcon, EditIcon, TrashIcon } from "~/components/icons";

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
              <BackArrowIcon />
            </A>
            <div class="w-8 h-8 rounded-lg bg-gradient-to-br from-indigo-500 to-purple-600 flex items-center justify-center text-white font-bold text-sm">
              AL
            </div>
            <h1 class="text-xl font-semibold text-white truncate">
              {task() ? task().taskName : "Loading..."}
            </h1>
          </div>
          <div class="flex items-center gap-2">
            <Button pattern="success" onClick={handleRunNow} icon={
              <PlayIcon />
            }>Run Now</Button>
            <Button pattern="secondary" onClick={() => navigate(`/tasks/${id()}/edit`)} icon={
              <EditIcon />
            }>Edit</Button>
            <Button pattern="danger" onClick={handleDelete} icon={
              <TrashIcon />
            }>Delete</Button>
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
            <Button pattern="only-border" href="/">Back to Dashboard</Button>
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
                <Button pattern="ghost" onClick={() => refetchRuns()}>Refresh</Button>
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
