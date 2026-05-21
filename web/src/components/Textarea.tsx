import { JSX, splitProps, createSignal } from "solid-js";

interface TextareaProps {
  label?: string;
  placeholder?: string;
  validate?: (value: string) => boolean;
  id?: string;
  name?: string;
  value?: string;
  onInput?: (e: InputEvent) => void;
  onChange?: (e: Event) => void;
  required?: boolean;
  disabled?: boolean;
  error?: string;
  rows?: number;
  class?: string;
}

function Textarea(initialProps: TextareaProps) {
  const [props, rest] = splitProps(initialProps, [
    "label",
    "placeholder",
    "validate",
    "id",
    "name",
    "value",
    "onInput",
    "onChange",
    "required",
    "disabled",
    "error",
    "rows",
    "class",
  ]);

  const [dirty, setDirty] = createSignal(false);

  const isInvalid = () => {
    if (props.error) return true;
    if (!dirty()) return false;
    if (!props.validate) return false;
    const val = props.value;
    if (val === undefined || val === null || val === "") return false;
    return !props.validate(val);
  };

  const handleInput = (e: InputEvent) => {
    if (!dirty()) setDirty(true);
    props.onInput?.(e);
  };

  return (
    <div class={props.class ?? ""}>
      {props.label && (
        <label for={props.id} class="block text-sm font-medium text-gray-300 mb-1.5">
          {props.label}
          {props.required && <span class="text-red-400">*</span>}
        </label>
      )}
      <textarea
        id={props.id}
        name={props.name}
        placeholder={props.placeholder}
        value={props.value}
        onInput={handleInput}
        onChange={props.onChange}
        required={props.required}
        disabled={props.disabled}
        rows={props.rows ?? 4}
        class={`w-full px-4 py-2.5 rounded-lg bg-gray-900 border text-white placeholder-gray-500 focus:border-indigo-500 focus:ring-1 focus:ring-indigo-500 outline-none transition-colors text-sm resize-y ${
          isInvalid()
            ? "border-red-500 focus:border-red-500 focus:ring-red-500"
            : "border-gray-700"
        } ${props.disabled ? "opacity-60 cursor-not-allowed" : ""}`}
        {...rest}
      />
      {props.error && (
        <p class="text-red-400 text-xs mt-1">{props.error}</p>
      )}
    </div>
  );
}

export default Textarea;
