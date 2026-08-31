import { renderToStaticMarkup } from "react-dom/server";
import { describe, expect, it } from "vitest";
import { Button } from "./primitives/button";
import { Input } from "./primitives/input";
import { Textarea } from "./primitives/textarea";
import { Field } from "./primitives/field";
import { Segmented } from "./primitives/segmented";
import { StatusBadge } from "./patterns/status-badge";
import { DataTable } from "./patterns/data-table";
import { HashChip } from "./patterns/hash-chip";
import { CodeBlock } from "./patterns/code-block";
import { Timeline } from "./patterns/timeline";
import { Stat } from "./patterns/stat";
import { ConfirmDialog } from "./patterns/confirm-dialog";
import { Panel } from "./layout/panel";
import { Drawer } from "./layout/drawer";
import { EmptyState } from "./feedback/empty-state";
import { ErrorState } from "./feedback/error-state";
import { Skeleton } from "./feedback/skeleton";

describe("Studio UI primitives", () => {
  it("renders the component library including loading and disabled controls", () => {
    const html = renderToStaticMarkup(<><Button loading>Save</Button><Button disabled>Disabled</Button><Input disabled /><Textarea disabled /><Field label="Name" error="Required"><Input /></Field><Segmented value="PUT" onChange={() => undefined} options={[{ value: "PUT", label: "Put" }]} /><StatusBadge label="READY" tone="neutral" dot /><DataTable loading columns={[{ key: "name", header: "Name", render: (value: { name: string }) => value.name }]} rows={[]} getRowId={(value) => value.name} empty="Empty" /><HashChip value="sha256:123456789abcdef" /><CodeBlock value={{ ok: true }} /><Timeline items={[{ id: "one", title: "Release", current: true }]} /><Stat label="Ready" value="1" /><ConfirmDialog open={false} onClose={() => undefined} title="Confirm" consequence="Continue" affectedCount={1} confirmLabel="Continue" onConfirm={() => undefined} /><Panel title="Panel">Content</Panel><Drawer open={false} onClose={() => undefined} title="Drawer">Content</Drawer><EmptyState title="Empty" description="Nothing here" /><ErrorState message="Failed" onRetry={() => undefined} /><Skeleton /></>);
    expect(html).toContain("aria-busy=\"true\"");
    expect(html).toContain("READY");
  });
});
