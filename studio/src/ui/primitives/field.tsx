import { Children, cloneElement, isValidElement, useId, type ReactElement, type ReactNode } from "react";

export function Field({ label, hint, error, children }: { label: string; hint?: ReactNode; error?: string; children: ReactElement }) {
  const id = useId();
  const descriptionId = error ? `${id}-error` : hint ? `${id}-hint` : undefined;
  const control = isValidElement(children) ? cloneElement(children, { id, "aria-invalid": Boolean(error), "aria-describedby": descriptionId } as never) : children;
  return <div className="grid gap-2"><label htmlFor={id} className="text-sm font-medium text-secondary">{label}</label>{control}{hint && !error && <p id={`${id}-hint`} className="text-xs text-muted-foreground">{hint}</p>}{error && <p id={`${id}-error`} role="alert" className="text-xs text-destructive">{error}</p>}</div>;
}
export const FieldGroup = ({ children }: { children: ReactNode }) => <div className="grid gap-4">{Children.toArray(children)}</div>;
