import { NoHydration } from "solid-js/web";

export function BackArrowIcon(props: { class?: string }) {
  return (
    <NoHydration>
      <svg class={props.class ?? "w-5 h-5"} fill="none" stroke="currentColor" viewBox="0 0 24 24">
        <path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M15 19l-7-7 7-7" />
      </svg>
    </NoHydration>
  );
}
