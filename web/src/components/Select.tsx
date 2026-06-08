import { JSX, splitProps } from "solid-js";
import { NoHydration } from "solid-js/web";

interface SelectProps {
  label?: string;
  options: { value: string; label: string }[];
  value: string;
  onChange: (value: string) => void;
  placeholder?: string;
  loading?: boolean;
  disabled?: boolean;
  error?: string;
  required?: boolean;
  id?: string;
  name?: string;
  class?: string;
}

function Select(initialProps: SelectProps) {
  const [props, rest] = splitProps(initialProps, [
    "label",
    "options",
    "value",
    "onChange",
    "placeholder",
    "loading",
    "disabled",
    "error",
    "required",
    "id",
    "name",
    "class",
  ]);

  const handleChange: JSX.ChangeEventHandler<HTMLSelectElement, Event> = (e) => {
    props.onChange(e.currentTarget.value);
  };

  const select = (
    <div class="relative">
      <select
        id={props.id}
        name={props.name}
        value={props.value}
        onChange={handleChange}
        disabled={props.disabled || props.loading}
        required={props.required}
        class={`w-full px-4 py-2.5 rounded-lg bg-gray-900 border text-white placeholder-gray-500 focus:border-indigo-500 focus:ring-1 focus:ring-indigo-500 outline-none transition-colors text-sm appearance-none ${
          props.placeholder && !props.value ? "text-gray-500" : ""
        } ${
          props.error
            ? "border-red-500 focus:border-red-500 focus:ring-red-500"
            : "border-gray-700"
        } ${props.disabled ? "opacity-60 cursor-not-allowed" : ""} ${
          props.loading ? "opacity-60 cursor-wait" : ""
        }`}
        {...rest}
      >
        {props.loading ? (
          <option value="" disabled>
            Loading...
          </option>
        ) : props.options.length === 0 ? (
          <option value="" disabled>
            No options available
          </option>
        ) : (
          <>
            {props.placeholder && (
              <option value="" disabled>
                {props.placeholder}
              </option>
            )}
            {props.options.map((opt) => (
              <option value={opt.value}>{opt.label}</option>
            ))}
          </>
        )}
      </select>
      <div class="pointer-events-none absolute inset-y-0 right-0 flex items-center pr-3">
        {props.loading ? (
          <div class="w-4 h-4 border-2 border-indigo-500 border-t-transparent rounded-full animate-spin" />
        ) : (
          <NoHydration>
            <svg
              class="h-4 w-4 text-gray-400"
              xmlns="http://www.w3.org/2000/svg"
              viewBox="0 0 20 20"
              fill="currentColor"
            >
              <path
                fill-rule="evenodd"
                d="M5.23 7.21a.75.75 0 011.06.02L10 11.168l3.71-3.938a.75.75 0 111.08 1.04l-4.25 4.5a.75.75 0 01-1.08 0l-4.25-4.5a.75.75 0 01.02-1.06z"
                clip-rule="evenodd"
              />
            </svg>
          </NoHydration>
        )}
      </div>
    </div>
  );

  return (
    <div class={props.class ?? ""}>
      {props.label && (
        <label for={props.id} class="block text-sm font-medium text-gray-300 mb-1.5">
          {props.label}
          {props.required && <span class="text-red-400">*</span>}
        </label>
      )}
      {select}
      {props.error && (
        <p class="text-red-400 text-xs mt-1">{props.error}</p>
      )}
    </div>
  );
}

export { Select };
export default Select;
