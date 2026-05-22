import { createResource, createSignal, createEffect } from "solid-js";
import { useNavigate, useSearchParams, A } from "@solidjs/router";
import { api } from "~/lib/api";
import { Button, Input, Select, Textarea, Toggle } from "~/components";
import { BackArrowIcon } from "~/components/icons";

export default function CreateTask() {
  const [searchParams] = useSearchParams();
  const navigate = useNavigate();
  const taskId = () => searchParams.id as string | undefined;
  const isEdit = () => !!taskId();

  const [task] = createResource(
    () => (isEdit() ? taskId() : false),
    api.getTask
  );
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

  // Agent runner list (always loaded, used in both modes)
  const [agents] = createResource(() => true, api.getAgents);

  // Agent models — depends on the selected runner in the form
  const [models] = createResource(
    () => form().agentRunner || false,
    api.getAgentModels
  );

  // Agent modes — depends on the selected runner in the form
  const [modes] = createResource(
    () => form().agentRunner || false,
    api.getAgentModes
  );

  // Populate form when task loads (edit mode)
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

    // Prevent submitting while model/mode resources are still loading
    const f = form();
    if (f.agentRunner && (models.loading || modes.loading)) {
      setError("Please wait for models and modes to load before submitting.");
      setSubmitting(false);
      return;
    }

    try {
      if (isEdit()) {
        const id = taskId()!;
        await api.updateTask(id, {
          ...form(),
          intervalSeconds: Number(form().intervalSeconds),
        });
        navigate(`/tasks/${id}`);
      } else {
        const task = await api.createTask({
          ...form(),
          intervalSeconds: Number(form().intervalSeconds),
        });
        navigate(`/tasks/${task.id}`);
      }
    } catch (err: any) {
      setError(err.message);
    } finally {
      setSubmitting(false);
    }
  };

  const updateField = (field: string, value: any) => {
    setForm((prev) => {
      const next = {
        ...prev,
        [field]: field === "intervalSeconds" ? Number(value) || 60 : value,
      };
      // Reset model and mode only when runner actually changes
      if (field === "agentRunner" && value !== prev.agentRunner) {
        next.agentModel = "";
        next.agentMode = "";
      }
      return next;
    });
  };

  return (
    <div class="min-h-screen bg-gray-950 text-gray-100">
      {/* Header */}
      <header class="border-b border-gray-800 bg-gray-900/50 backdrop-blur-sm">
        <div class="max-w-3xl mx-auto px-4 sm:px-6 lg:px-8 h-16 flex items-center justify-between">
          <div class="flex items-center gap-4">
            <A
              href={isEdit() ? `/tasks/${taskId()}` : "/"}
              class="text-gray-400 hover:text-white transition-colors"
            >
              <BackArrowIcon />
            </A>
            <div class="w-8 h-8 rounded-lg bg-gradient-to-br from-indigo-500 to-purple-600 flex items-center justify-center text-white font-bold text-sm">
              AL
            </div>
            <h1 class="text-xl font-semibold text-white">
              {isEdit() ? "Edit Task" : "Create Task"}
            </h1>
          </div>
        </div>
      </header>

      <main class="max-w-3xl mx-auto px-4 sm:px-6 lg:px-8 py-8">
        {/* Loading state (edit mode only) */}
        {isEdit() && task.loading && (
          <div class="flex items-center justify-center py-24">
            <div class="flex flex-col items-center gap-3">
              <div class="w-8 h-8 border-2 border-indigo-500 border-t-transparent rounded-full animate-spin" />
              <p class="text-gray-400 text-sm">Loading task...</p>
            </div>
          </div>
        )}

        {/* Error state (edit mode only) */}
        {isEdit() && task.error && (
          <div class="bg-red-900/20 border border-red-800 rounded-xl p-6 text-center">
            <p class="text-red-400 font-medium">Failed to load task</p>
            <p class="text-red-300/70 text-sm mt-1">
              {(task.error as any)?.message || "Unknown error"}
            </p>
            <Button pattern="only-border" href="/">
              Back to Dashboard
            </Button>
          </div>
        )}

        {/* Form: always render in create mode, only when task loaded in edit mode */}
        {(!isEdit() || (task() && !task.loading)) && (
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
              <Select
                id="agentRunner"
                label="Agent Runner"
                required
                placeholder="Select a runner..."
                options={(agents() || []).map((a: any) => ({ value: a.id, label: a.name }))}
                value={form().agentRunner}
                onChange={(value) => updateField("agentRunner", value)}
                loading={agents.loading}
                error={agents.error ? "Failed to load runners" : undefined}
              />
              <Select
                id="agentModel"
                label="Agent Model"
                placeholder="Select a model..."
                disabled={!form().agentRunner}
                options={(models() || []).map((m: string) => ({ value: m, label: m }))}
                value={form().agentModel}
                onChange={(value) => updateField("agentModel", value)}
                loading={!!form().agentRunner && models.loading}
                error={models.error ? "Failed to load models" : undefined}
              />
            </div>

            {/* Agent Mode & Work Dir row */}
            <div class="grid grid-cols-1 md:grid-cols-2 gap-4">
              <Select
                id="agentMode"
                label="Agent Mode"
                placeholder="Select a mode..."
                disabled={!form().agentRunner}
                options={(modes() || []).map((m: string) => ({ value: m, label: m }))}
                value={form().agentMode}
                onChange={(value) => updateField("agentMode", value)}
                loading={!!form().agentRunner && modes.loading}
                error={modes.error ? "Failed to load modes" : undefined}
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
              <div>
                <Input
                  id="intervalSeconds"
                  label="Interval (seconds)"
                  type="number"
                  min={0}
                  value={form().intervalSeconds}
                  onInput={(e) =>
                    updateField("intervalSeconds", e.currentTarget.value)
                  }
                />
                <div class="flex flex-wrap gap-1.5 mt-2">
                  {[
                    { label: "1m", value: 60 },
                    { label: "10m", value: 600 },
                    { label: "1h", value: 3600 },
                    { label: "12h", value: 43200 },
                    { label: "24h", value: 86400 },
                  ].map((preset) => (
                    <button
                      type="button"
                      class={`px-2 py-0.5 text-xs rounded border transition-colors ${
                        form().intervalSeconds === preset.value
                          ? "border-indigo-500 bg-indigo-500/20 text-indigo-300"
                          : "border-gray-700 text-gray-400 hover:border-gray-500 hover:text-gray-300"
                      }`}
                      onClick={() => updateField("intervalSeconds", preset.value)}
                    >
                      {preset.label}
                    </button>
                  ))}
                </div>
              </div>
              <div class="flex items-end pb-2.5">
                <Toggle
                  checked={form().enabled}
                  onChange={(checked) => updateField("enabled", checked)}
                  label="Enabled"
                />
              </div>
            </div>

            {/* Submit */}
            <div class="flex items-center gap-3 pt-2">
              <Button pattern="primary" type="submit" loading={submitting()}>
                {submitting()
                  ? isEdit()
                    ? "Saving..."
                    : "Creating..."
                  : isEdit()
                    ? "Save Changes"
                    : "Create Task"}
              </Button>
              <Button
                pattern="only-border"
                href={isEdit() ? `/tasks/${taskId()}` : "/"}
              >
                Cancel
              </Button>
            </div>
          </form>
        )}
      </main>
    </div>
  );
}
