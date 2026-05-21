import { createResource, createSignal, createEffect } from "solid-js";
import { useParams, useNavigate, A } from "@solidjs/router";
import { api } from "~/lib/api";
import { Button, Input, Textarea, Toggle } from "~/components";
import { BackArrowIcon } from "~/components/icons";

export default function EditTask() {
  const params = useParams();
  const navigate = useNavigate();
  const id = () => params.id as string;
  const [task] = createResource(id, api.getTask);
  const [submitting, setSubmitting] = createSignal(false);
  const [error, setError] = createSignal<string | null>(null);

  const [form, setForm] = createSignal({
    taskName: "",
    initMessage: "",
    agentRunner: "",
    agentModel: "",
    agentMode: "",
    workDir: "",
    intervalSeconds: 60,
    enabled: true,
  });

  // Populate form when task loads
  createEffect(() => {
    const t = task();
    if (t) {
      setForm({
        taskName: t.taskName || "",
        initMessage: t.initMessage || "",
        agentRunner: t.agentRunner || "",
        agentModel: t.agentModel || "",
        agentMode: t.agentMode || "",
        workDir: t.workDir || "",
        intervalSeconds: t.intervalSeconds ?? 60,
        enabled: t.enabled ?? true,
      });
    }
  });

  const handleSubmit = async (e: Event) => {
    e.preventDefault();
    setSubmitting(true);
    setError(null);
    try {
      await api.updateTask(id(), {
        ...form(),
        intervalSeconds: Number(form().intervalSeconds),
      });
      navigate(`/tasks/${id()}`);
    } catch (err: any) {
      setError(err.message);
    } finally {
      setSubmitting(false);
    }
  };

  const updateField = (field: string, value: any) => {
    setForm((prev) => ({ ...prev, [field]: value }));
  };

  return (
    <div class="min-h-screen bg-gray-950 text-gray-100">
      {/* Header */}
      <header class="border-b border-gray-800 bg-gray-900/50 backdrop-blur-sm">
        <div class="max-w-3xl mx-auto px-4 sm:px-6 lg:px-8 h-16 flex items-center justify-between">
          <div class="flex items-center gap-4">
            <A href={`/tasks/${id()}`} class="text-gray-400 hover:text-white transition-colors">
              <BackArrowIcon />
            </A>
            <div class="w-8 h-8 rounded-lg bg-gradient-to-br from-indigo-500 to-purple-600 flex items-center justify-center text-white font-bold text-sm">
              AL
            </div>
            <h1 class="text-xl font-semibold text-white">Edit Task</h1>
          </div>
        </div>
      </header>

      <main class="max-w-3xl mx-auto px-4 sm:px-6 lg:px-8 py-8">
        {task.loading && (
          <div class="flex items-center justify-center py-24">
            <div class="flex flex-col items-center gap-3">
              <div class="w-8 h-8 border-2 border-indigo-500 border-t-transparent rounded-full animate-spin" />
              <p class="text-gray-400 text-sm">Loading task...</p>
            </div>
          </div>
        )}

        {task.error && (
          <div class="bg-red-900/20 border border-red-800 rounded-xl p-6 text-center">
            <p class="text-red-400 font-medium">Failed to load task</p>
            <p class="text-red-300/70 text-sm mt-1">{(task.error as any)?.message || "Unknown error"}</p>
            <Button pattern="only-border" href="/">Back to Dashboard</Button>
          </div>
        )}

        {task() && (
          <form onSubmit={handleSubmit} class="space-y-6">
            {/* Error banner */}
            {error() && (
              <div class="bg-red-900/20 border border-red-800 rounded-xl p-4">
                <p class="text-red-400 text-sm font-medium">{error()}</p>
              </div>
            )}

            {/* Task Name */}
            <Input
              id="taskName"
              label="Task Name"
              required
              placeholder="My Agent Task"
              value={form().taskName}
              onInput={(e) => updateField("taskName", e.currentTarget.value)}
            />

            {/* Init Message */}
            <Textarea
              id="initMessage"
              label="Init Message"
              rows={4}
              placeholder="You are a helpful assistant..."
              value={form().initMessage}
              onInput={(e) => updateField("initMessage", e.currentTarget.value)}
            />

            {/* Agent Runner & Model row */}
            <div class="grid grid-cols-1 md:grid-cols-2 gap-4">
              <Input
                id="agentRunner"
                label="Agent Runner"
                placeholder="openai"
                value={form().agentRunner}
                onInput={(e) => updateField("agentRunner", e.currentTarget.value)}
              />
              <Input
                id="agentModel"
                label="Agent Model"
                placeholder="gpt-4"
                value={form().agentModel}
                onInput={(e) => updateField("agentModel", e.currentTarget.value)}
              />
            </div>

            {/* Agent Mode & Work Dir row */}
            <div class="grid grid-cols-1 md:grid-cols-2 gap-4">
              <Input
                id="agentMode"
                label="Agent Mode"
                placeholder="auto"
                value={form().agentMode}
                onInput={(e) => updateField("agentMode", e.currentTarget.value)}
              />
              <Input
                id="workDir"
                label="Work Directory"
                placeholder="/path/to/workdir"
                value={form().workDir}
                onInput={(e) => updateField("workDir", e.currentTarget.value)}
              />
            </div>

            {/* Interval & Enabled row */}
            <div class="grid grid-cols-1 md:grid-cols-2 gap-4">
              <Input
                id="intervalSeconds"
                label="Interval (seconds)"
                type="number"
                min={0}
                value={form().intervalSeconds}
                onInput={(e) => updateField("intervalSeconds", e.currentTarget.value)}
              />
              <div class="flex items-end pb-2.5">
                <Toggle
                  checked={form().enabled}
                  onChange={(checked) => updateField("enabled", checked)}
                  label="Enabled"
                />
              </div>
            </div>

            {/* Actions */}
            <div class="flex items-center gap-3 pt-2">
              <Button pattern="primary" type="submit" loading={submitting()}>
                {submitting() ? "Saving..." : "Save Changes"}
              </Button>
              <Button pattern="only-border" href={`/tasks/${id()}`}>Cancel</Button>
            </div>
          </form>
        )}
      </main>
    </div>
  );
}
