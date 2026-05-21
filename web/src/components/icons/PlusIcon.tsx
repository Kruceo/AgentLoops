import { NoHydration } from "solid-js/web";

export function PlusIcon(props: { class?: string }) {
  return (
    <NoHydration>
      <svg class={props.class ?? "w-4 h-4"} fill="none" stroke="currentColor" viewBox="0 0 24 24">
        <path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M12 4v16m8-8H4" />
      </svg>
    </NoHydration>
  );
}
