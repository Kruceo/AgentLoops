import { createSignal } from "solid-js";
import { useNavigate, A } from "@solidjs/router";
import { api } from "~/lib/api";

export default function CreateTask() {
  const navigate = useNavigate();
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

  const handleSubmit = async (e: Event) => {
    e.preventDefault();
    setSubmitting(true);
    setError(null);
    try {
      const task = await api.createTask({
        ...form(),
        intervalSeconds: Number(form().intervalSeconds),
      });
      navigate(`/tasks/${task.id}`);
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
            <A href="/" class="text-gray-400 hover:text-white transition-colors">
              <svg class="w-5 h-5" fill="none" stroke="currentColor" viewBox="0 0 24 24">
                <path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M15 19l-7-7 7-7" />
              </svg>
            </A>
            <div class="w-8 h-8 rounded-lg bg-gradient-to-br from-indigo-500 to-purple-600 flex items-center justify-center text-white font-bold text-sm">
              AL
            </div>
            <h1 class="text-xl font-semibold text-white">Create Task</h1>
          </div>
        </div>
      </header>

      <main class="max-w-3xl mx-auto px-4 sm:px-6 lg:px-8 py-8">
        <form onSubmit={handleSubmit} class="space-y-6">
          {/* Error banner */}
          {error() && (
            <div class="bg-red-900/20 border border-red-800 rounded-xl p-4">
              <p class="text-red-400 text-sm font-medium">{error()}</p>
            </div>
          )}

          {/* Task Name */}
          <div>
            <label for="taskName" class="block text-sm font-medium text-gray-300 mb-1.5">
              Task Name <span class="text-red-400">*</span>
            </label>
            <input
              id="taskName"
              type="text"
              required
              value={form().taskName}
              onInput={(e) => updateField("taskName", e.currentTarget.value)}
              class="w-full px-4 py-2.5 rounded-lg bg-gray-900 border border-gray-700 text-white placeholder-gray-500 focus:border-indigo-500 focus:ring-1 focus:ring-indigo-500 outline-none transition-colors text-sm"
              placeholder="My Agent Task"
            />
          </div>

          {/* Init Message */}
          <div>
            <label for="initMessage" class="block text-sm font-medium text-gray-300 mb-1.5">
              Init Message
            </label>
            <textarea
              id="initMessage"
              rows={4}
              value={form().initMessage}
              onInput={(e) => updateField("initMessage", e.currentTarget.value)}
              class="w-full px-4 py-2.5 rounded-lg bg-gray-900 border border-gray-700 text-white placeholder-gray-500 focus:border-indigo-500 focus:ring-1 focus:ring-indigo-500 outline-none transition-colors text-sm resize-y"
              placeholder="You are a helpful assistant..."
            />
          </div>

          {/* Agent Runner & Model row */}
          <div class="grid grid-cols-1 md:grid-cols-2 gap-4">
            <div>
              <label for="agentRunner" class="block text-sm font-medium text-gray-300 mb-1.5">
                Agent Runner
              </label>
              <input
                id="agentRunner"
                type="text"
                value={form().agentRunner}
                onInput={(e) => updateField("agentRunner", e.currentTarget.value)}
                class="w-full px-4 py-2.5 rounded-lg bg-gray-900 border border-gray-700 text-white placeholder-gray-500 focus:border-indigo-500 focus:ring-1 focus:ring-indigo-500 outline-none transition-colors text-sm"
                placeholder="openai"
              />
            </div>
            <div>
              <label for="agentModel" class="block text-sm font-medium text-gray-300 mb-1.5">
                Agent Model
              </label>
              <input
                id="agentModel"
                type="text"
                value={form().agentModel}
                onInput={(e) => updateField("agentModel", e.currentTarget.value)}
                class="w-full px-4 py-2.5 rounded-lg bg-gray-900 border border-gray-700 text-white placeholder-gray-500 focus:border-indigo-500 focus:ring-1 focus:ring-indigo-500 outline-none transition-colors text-sm"
                placeholder="gpt-4"
              />
            </div>
          </div>

          {/* Agent Mode & Work Dir row */}
          <div class="grid grid-cols-1 md:grid-cols-2 gap-4">
            <div>
              <label for="agentMode" class="block text-sm font-medium text-gray-300 mb-1.5">
                Agent Mode
              </label>
              <input
                id="agentMode"
                type="text"
                value={form().agentMode}
                onInput={(e) => updateField("agentMode", e.currentTarget.value)}
                class="w-full px-4 py-2.5 rounded-lg bg-gray-900 border border-gray-700 text-white placeholder-gray-500 focus:border-indigo-500 focus:ring-1 focus:ring-indigo-500 outline-none transition-colors text-sm"
                placeholder="auto"
              />
            </div>
            <div>
              <label for="workDir" class="block text-sm font-medium text-gray-300 mb-1.5">
                Work Directory
              </label>
              <input
                id="workDir"
                type="text"
                value={form().workDir}
                onInput={(e) => updateField("workDir", e.currentTarget.value)}
                class="w-full px-4 py-2.5 rounded-lg bg-gray-900 border border-gray-700 text-white placeholder-gray-500 focus:border-indigo-500 focus:ring-1 focus:ring-indigo-500 outline-none transition-colors text-sm"
                placeholder="/path/to/workdir"
              />
            </div>
          </div>

          {/* Interval & Enabled row */}
          <div class="grid grid-cols-1 md:grid-cols-2 gap-4">
            <div>
              <label for="intervalSeconds" class="block text-sm font-medium text-gray-300 mb-1.5">
                Interval (seconds)
              </label>
              <input
                id="intervalSeconds"
                type="number"
                min={0}
                value={form().intervalSeconds}
                onInput={(e) => updateField("intervalSeconds", e.currentTarget.value)}
                class="w-full px-4 py-2.5 rounded-lg bg-gray-900 border border-gray-700 text-white placeholder-gray-500 focus:border-indigo-500 focus:ring-1 focus:ring-indigo-500 outline-none transition-colors text-sm"
              />
            </div>
            <div class="flex items-end pb-2.5">
              <label class="flex items-center gap-3 cursor-pointer">
                <div class="relative">
                  <input
                    type="checkbox"
                    checked={form().enabled}
                    onChange={(e) => updateField("enabled", e.currentTarget.checked)}
                    class="sr-only peer"
                  />
                  <div class={`w-9 h-5 rounded-full transition-colors ${
                    form().enabled ? "bg-indigo-600" : "bg-gray-700"
                  }`}>
                    <div class={`h-3.5 w-3.5 rounded-full bg-white mt-[3px] transition-transform ${
                      form().enabled ? "translate-x-[18px]" : "translate-x-[3px]"
                    }`} />
                  </div>
                </div>
                <span class="text-sm text-gray-300">Enabled</span>
              </label>
            </div>
          </div>

          {/* Submit */}
          <div class="flex items-center gap-3 pt-2">
            <button
              type="submit"
              disabled={submitting()}
              class="inline-flex items-center gap-2 px-6 py-2.5 rounded-lg bg-indigo-600 hover:bg-indigo-500 disabled:bg-indigo-800 disabled:cursor-not-allowed text-white text-sm font-medium transition-colors"
            >
              {submitting() ? (
                <>
                  <div class="w-4 h-4 border-2 border-white border-t-transparent rounded-full animate-spin" />
                  Creating...
                </>
              ) : (
                "Create Task"
              )}
            </button>
            <A
              href="/"
              class="px-4 py-2.5 rounded-lg bg-gray-800 hover:bg-gray-700 text-gray-300 text-sm font-medium transition-colors"
            >
              Cancel
            </A>
          </div>
        </form>
      </main>
    </div>
  );
}
