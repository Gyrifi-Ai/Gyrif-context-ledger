import { forwardRef, type ButtonHTMLAttributes, type ReactNode } from "react";
import { LoaderCircle } from "lucide-react";
import { cn } from "@/lib/utils";

export type ButtonProps = ButtonHTMLAttributes<HTMLButtonElement> & {
  variant?: "primary" | "secondary" | "ghost" | "danger";
  size?: "sm" | "md";
  loading?: boolean;
  iconLeft?: ReactNode;
};

const variants = {
  primary: "bg-primary text-primary-foreground hover:bg-primary/90",
  secondary: "border border-border bg-secondary text-secondary-foreground hover:bg-accent",
  ghost: "text-muted-foreground hover:bg-accent hover:text-accent-foreground",
  danger: "bg-destructive text-destructive-foreground hover:bg-destructive/90",
};

export const Button = forwardRef<HTMLButtonElement, ButtonProps>(function Button(
  { className, children, disabled, iconLeft, loading = false, size = "md", type = "button", variant = "primary", ...props }, ref,
) {
  return <button ref={ref} type={type} disabled={disabled || loading} aria-busy={loading || undefined} className={cn("inline-flex min-w-fit items-center justify-center gap-2 rounded-sm px-3 text-sm font-medium transition-colors disabled:cursor-not-allowed disabled:opacity-45", size === "sm" ? "h-8 text-xs" : "h-9", variants[variant], className)} {...props}>
    {loading ? <LoaderCircle className="size-3.5 animate-spin" aria-hidden="true" /> : iconLeft}
    {children}
  </button>;
});
