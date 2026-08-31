import { useEffect, useState, type FormEvent } from "react";
import { api } from "../../api/client";
import type { Change } from "../../api/types";
import { useAppState } from "../../app/providers";
import { EmptyState } from "../../ui/feedback/empty-state";
import { StatusBadge } from "../../ui/patterns/status-badge";
import { changeTone } from "../shared/status";
import { Button } from "@/components/ui/button";
import {
  Card,
  CardContent,
  CardDescription,
  CardHeader,
  CardTitle,
} from "@/components/ui/card";
import { Input } from "@/components/ui/input";
import { Label } from "@/components/ui/label";
import { Textarea } from "@/components/ui/textarea";
import {
  Table,
  TableBody,
  TableCell,
  TableHead,
  TableHeader,
  TableRow,
} from "@/components/ui/table";

export function ChangesPage() {
  const { ledgerId } = useAppState();
  const [items, setItems] = useState<Change[]>([]);
  const [unit, setUnit] = useState("");
  const [desired, setDesired] = useState("{}");
  const [error, setError] = useState("");
  const load = () => {
    if (ledgerId) {
      void api.changes(ledgerId).then((value) => setItems(value.items ?? [])).catch((value: Error) => setError(value.message));
    }
  };
  useEffect(() => { load(); }, [ledgerId]);
  const create = async (event: FormEvent) => {
    event.preventDefault(); setError("");
    try { await api.createChange(ledgerId, { unit, action: "PUT", desired: JSON.parse(desired), idempotencyKey: crypto.randomUUID() }); setUnit(""); await load(); } catch (value) { setError((value as Error).message); }
  };
  if (!ledgerId) return (
    <Card>
      <EmptyState title="Select a ledger" description="Choose or create a ledger before submitting Changes." />
    </Card>
  );
  return (
    <div className="grid items-start gap-5 xl:grid-cols-[minmax(0,1fr)_360px]">
      <Card>
        <CardHeader className="flex-row items-center justify-between space-y-0 border-b">
          <div>
            <CardTitle className="text-base">Received changes</CardTitle>
            <CardDescription className="mt-1">Durable inbox for this ledger</CardDescription>
          </div>
          <span className="text-sm text-muted-foreground">{items.length} received</span>
        </CardHeader>
        <CardContent className="p-0">
          {items.length === 0 ? (
            <EmptyState title="No Changes yet" description="Submit desired state here or through the versioned API." />
          ) : (
            <Table>
              <TableHeader>
                <TableRow className="hover:bg-transparent">
                  <TableHead>Change</TableHead>
                  <TableHead>Unit</TableHead>
                  <TableHead>Action</TableHead>
                  <TableHead className="text-right">Status</TableHead>
                </TableRow>
              </TableHeader>
              <TableBody>
                {items.map((change) => (
                  <TableRow key={change.id}>
                    <TableCell>
                      <code className="text-xs text-muted-foreground">
                        {change.id.slice(0, 12)}
                      </code>
                    </TableCell>
                    <TableCell className="font-medium">{change.unit}</TableCell>
                    <TableCell className="text-muted-foreground">
                      {change.action}
                    </TableCell>
                    <TableCell className="text-right">
                      <StatusBadge label={change.status} tone={changeTone(change.status)} />
                    </TableCell>
                  </TableRow>
                ))}
              </TableBody>
            </Table>
          )}
        </CardContent>
      </Card>
      <Card>
        <CardHeader className="border-b">
          <CardTitle className="text-base">Submit desired state</CardTitle>
          <CardDescription>Accepted into the inbox; nothing touches the target yet.</CardDescription>
        </CardHeader>
        <CardContent className="pt-5">
          <form onSubmit={create} className="grid gap-4">
            <div className="grid gap-2">
              <Label htmlFor="change-unit">Logical unit</Label>
              <Input id="change-unit" value={unit} onChange={(event) => setUnit(event.target.value)} placeholder="point/42" required />
            </div>
            <div className="grid gap-2">
              <Label htmlFor="change-desired">JSON value</Label>
              <Textarea id="change-desired" value={desired} onChange={(event) => setDesired(event.target.value)} rows={8} spellCheck={false} />
            </div>
            {error && <p role="alert" className="rounded-md border border-destructive/30 bg-destructive/10 px-3 py-2 font-mono text-xs text-destructive">{error}</p>}
            <Button type="submit" className="w-full">Accept Change</Button>
          </form>
        </CardContent>
      </Card>
    </div>
  );
}
