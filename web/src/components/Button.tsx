import { JSX, splitProps } from "solid-js";
import { A } from "@solidjs/router";

type ButtonPattern = "primary" | "secondary" | "ghost" | "only-border" | "danger" | "success";
type ButtonSize = "sm" | "md" | "xl";

interface ButtonProps {
  pattern: ButtonPattern;
  size?: ButtonSize;
  icon?: JSX.Element;
  children?: JSX.Element | string;
  type?: "button" | "submit" | "reset";
  disabled?: boolean;
  loading?: boolean;
  onClick?: (e: MouseEvent) => void;
  class?: string;
  href?: string;
  title?: string;
}

const patternClasses: Record<ButtonPattern, { base: string; disabled: string }> = {
  primary: {
    base: "bg-indigo-600 hover:bg-indigo-500 text-white",
    disabled: "bg-indigo-800",
  },
  secondary: {
    base: "bg-gray-800 hover:bg-gray-700 text-gray-200",
    disabled: "bg-gray-800",
  },
  ghost: {
    base: "text-gray-400 hover:text-white",
    disabled: "text-gray-600",
  },
  "only-border": {
    base: "border border-gray-700 text-gray-300 hover:border-gray-600 hover:text-white bg-transparent",
    disabled: "border-gray-700 text-gray-600 bg-transparent",
  },
  danger: {
    base: "bg-red-900/50 hover:bg-red-800 text-red-300",
    disabled: "bg-red-900/50",
  },
  success: {
    base: "bg-emerald-700 hover:bg-emerald-600 text-emerald-100",
    disabled: "bg-emerald-700",
  },
};

const sizeClasses: Record<ButtonSize, string> = {
  sm: "px-3 py-1.5 text-xs",
  md: "px-4 py-2 text-sm",
  xl: "px-6 py-2.5 text-sm",
};

function Button(props: ButtonProps) {
  const [local, rest] = splitProps(props, [
    "pattern",
    "size",
    "icon",
    "children",
    "type",
    "disabled",
    "loading",
    "onClick",
    "class",
    "href",
    "title",
  ]);

  const disabled = () => local.disabled || local.loading;
  const patternStyle = () => patternClasses[local.pattern];
  const size = () => local.size ?? "md";
  const isGhost = () => local.pattern === "ghost";

  const commonClasses = () => {
    const p = patternStyle();
    const base = disabled() ? p.disabled : p.base;
    if (isGhost()) {
      return `inline-flex items-center gap-2 transition-colors ${base} ${local.class ?? ""}`;
    }
    return `inline-flex items-center gap-2 rounded-lg font-medium transition-colors ${base} ${
      disabled() ? "opacity-50 cursor-not-allowed" : ""
    } ${!isGhost() ? sizeClasses[size()] : ""} ${local.class ?? ""}`;
  };

  const content = () => (
    <>
      {local.loading && (
        <div class="w-4 h-4 border-2 border-white border-t-transparent rounded-full animate-spin" />
      )}
      {!local.loading && local.icon && local.icon}
      {local.children}
    </>
  );

  if (local.href) {
    return (
      <A
        href={local.href}
        title={local.title}
        class={commonClasses()}
        aria-disabled={disabled() || undefined}
      >
        {content()}
      </A>
    );
  }

  return (
    <button
      type={local.type ?? "button"}
      disabled={disabled()}
      onClick={local.onClick}
      title={local.title}
      class={commonClasses()}
      {...rest}
    >
      {content()}
    </button>
  );
}

export default Button;
