import { A } from "@solidjs/router";
import { NoHydration } from "solid-js/web";

export default function NotFound() {
  return (
    <div class="min-h-screen bg-gray-950 text-gray-100 flex items-center justify-center">
      <div class="text-center">
        <h1 class="text-8xl font-bold text-gray-800 mb-4">404</h1>
        <h2 class="text-2xl font-semibold text-white mb-2">Page Not Found</h2>
        <p class="text-gray-400 mb-8 max-w-md">
          The page you're looking for doesn't exist or has been moved.
        </p>
        <A
          href="/"
          class="inline-flex items-center gap-2 px-6 py-3 rounded-lg bg-indigo-600 hover:bg-indigo-500 text-white text-sm font-medium transition-colors"
        >
          <NoHydration>
            <svg class="w-4 h-4" fill="none" stroke="currentColor" viewBox="0 0 24 24">
              <path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M15 19l-7-7 7-7" />
            </svg>
          </NoHydration>
          Back to Dashboard
        </A>
      </div>
    </div>
  );
}
