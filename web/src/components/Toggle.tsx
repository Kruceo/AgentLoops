interface ToggleProps {
  checked: boolean;
  onChange: (checked: boolean) => void;
  label?: string;
  disabled?: boolean;
  class?: string;
}

export default function Toggle(props: ToggleProps) {
  return (
    <label
      class={`flex items-center gap-3 ${props.disabled ? "opacity-50 cursor-not-allowed" : "cursor-pointer"} ${props.class ?? ""}`}
    >
      <div class="relative">
        <input
          type="checkbox"
          class="absolute inset-0 opacity-0 cursor-pointer z-10 invisible"
          checked={props.checked}
          onChange={(e) => props.onChange(e.currentTarget.checked)}
          disabled={props.disabled}
        />
        <div
          class={`w-9 h-5 p-1 flex rounded-full transition-colors ${props.checked ? "bg-indigo-600" : "bg-gray-700"}`}
        >
          <div
            class={`h-1/2 aspect-square rounded-full bg-white  transition-[left] top-1/2 -translate-1/2 absolute ${props.checked ? "left-[75%]" : "left-[25%]"}`}
          />
        </div>
      </div>
      {props.label && <span class="text-sm text-gray-300">{props.label}</span>}
    </label>
  );
}
