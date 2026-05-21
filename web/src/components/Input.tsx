import { JSX, splitProps, createSignal } from "solid-js";

interface InputProps {
  label?: string;
  icon?: JSX.Element;
  placeholder?: string;
  validate?: (value: string) => boolean;
  id?: string;
  name?: string;
  type?: string;
  value?: string | number;
  onInput?: (e: InputEvent) => void;
  onChange?: (e: Event) => void;
  required?: boolean;
  disabled?: boolean;
  error?: string;
  min?: number;
  max?: number;
  autocomplete?: string;
  class?: string;
  inputClass?: string;
}

function Input(initialProps: InputProps) {
  const [props, rest] = splitProps(initialProps, [
    "label",
    "icon",
    "placeholder",
    "validate",
    "id",
    "name",
    "type",
    "value",
    "onInput",
    "onChange",
    "required",
    "disabled",
    "error",
    "min",
    "max",
    "autocomplete",
    "class",
    "inputClass",
  ]);

  const [dirty, setDirty] = createSignal(false);

  const isInvalid = () => {
    if (props.error) return true;
    if (!dirty()) return false;
    if (!props.validate) return false;
    const val = props.value;
    if (val === undefined || val === null || val === "") return false;
    return !props.validate(String(val));
  };

  const handleInput = (e: InputEvent) => {
    if (!dirty()) setDirty(true);
    props.onInput?.(e);
  };

  const input = (
    <input
      id={props.id}
      name={props.name}
      type={props.type ?? "text"}
      placeholder={props.placeholder}
      value={props.value}
      onInput={handleInput}
      onChange={props.onChange}
      required={props.required}
      disabled={props.disabled}
      min={props.min}
      max={props.max}
      autocomplete={props.autocomplete}
      class={`w-full px-4 py-2.5 rounded-lg bg-gray-900 border text-white placeholder-gray-500 focus:border-indigo-500 focus:ring-1 focus:ring-indigo-500 outline-none transition-colors text-sm ${
        isInvalid()
          ? "border-red-500 focus:border-red-500 focus:ring-red-500"
          : "border-gray-700"
      } ${props.disabled ? "opacity-60 cursor-not-allowed" : ""} ${
        props.icon ? "pl-10" : ""
      } ${props.inputClass ?? ""}`}
      {...rest}
    />
  );

  return (
    <div class={props.class ?? ""}>
      {props.label && (
        <label for={props.id} class="block text-sm font-medium text-gray-300 mb-1.5">
          {props.label}
          {props.required && <span class="text-red-400">*</span>}
        </label>
      )}
      {props.icon ? (
        <div class="relative">
          <div class="absolute left-3 top-1/2 -translate-y-1/2 text-gray-400">
            {props.icon}
          </div>
          {input}
        </div>
      ) : (
        input
      )}
      {props.error && (
        <p class="text-red-400 text-xs mt-1">{props.error}</p>
      )}
    </div>
  );
}

export { Input };
export default Input;
