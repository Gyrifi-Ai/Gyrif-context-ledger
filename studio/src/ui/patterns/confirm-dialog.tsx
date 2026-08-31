import { useEffect, useRef, type KeyboardEvent, type ReactNode } from "react";
import { Button } from "../primitives/button";

export function ConfirmDialog({ open, onClose, title, consequence, affectedCount, confirmLabel, confirmLoading = false, confirmDisabled = false, confirmTitle, onConfirm }: { open: boolean; onClose: () => void; title: string; consequence: ReactNode; affectedCount: number; confirmLabel: string; confirmLoading?: boolean; confirmDisabled?: boolean; confirmTitle?: string; onConfirm: () => void }) {
	const ref = useRef<HTMLDialogElement>(null);
	const trigger = useRef<HTMLElement | null>(null);
	useEffect(() => {
		const dialog = ref.current;
		if (!dialog) return;
		if (open && !dialog.open) {
			trigger.current = document.activeElement as HTMLElement;
			dialog.showModal();
		}
		if (!open && dialog.open) dialog.close();
	}, [open]);
	const close = () => {
		onClose();
		trigger.current?.focus();
	};
	const keyDown = (event: KeyboardEvent<HTMLDialogElement>) => {
		if (event.key === "Escape") {
			event.preventDefault();
			close();
			return;
		}
		if (event.key !== "Tab") return;
		const focusable = Array.from(ref.current?.querySelectorAll<HTMLElement>("button:not(:disabled), [href], input:not(:disabled), select:not(:disabled), textarea:not(:disabled), [tabindex]:not([tabindex='-1'])") ?? []);
		if (focusable.length === 0) return;
		const first = focusable[0];
		const last = focusable[focusable.length - 1];
		if (event.shiftKey && document.activeElement === first) {
			event.preventDefault();
			last.focus();
		} else if (!event.shiftKey && document.activeElement === last) {
			event.preventDefault();
			first.focus();
		}
	};
	return <dialog ref={ref} onKeyDown={keyDown} onCancel={(event) => { event.preventDefault(); close(); }} onClose={() => trigger.current?.focus()} aria-labelledby="confirm-title" className="w-full max-w-md rounded-lg border border-border bg-card p-0 text-foreground backdrop:bg-black/50"><div className="p-5"><h2 id="confirm-title" className="text-lg font-semibold">{title}</h2><div className="mt-3 text-sm text-muted-foreground">{consequence}</div><p className="mt-3 font-mono text-xs">{affectedCount} affected units</p><div className="mt-5 flex justify-end gap-2"><Button variant="secondary" onClick={close}>Cancel</Button><Button variant="danger" loading={confirmLoading} disabled={confirmDisabled} title={confirmTitle} onClick={onConfirm}>{confirmLabel}</Button></div></div></dialog>;
}
