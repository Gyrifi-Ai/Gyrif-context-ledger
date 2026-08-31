import { render, screen } from "@testing-library/react";
import userEvent from "@testing-library/user-event";
import { describe, expect, it, vi } from "vitest";
import { Badge } from "./badge";
import { Button } from "./button";
import { Card, CardContent, CardDescription, CardFooter, CardHeader, CardTitle } from "./card";
import { Checkbox } from "./checkbox";
import { Dialog, DialogClose, DialogContent, DialogDescription, DialogFooter, DialogHeader, DialogTitle, DialogTrigger } from "./dialog";
import { Input } from "./input";
import { Label } from "./label";
import { Separator } from "./separator";
import { Skeleton } from "./skeleton";
import { Table, TableBody, TableCell, TableHead, TableHeader, TableRow } from "./table";
import { Textarea } from "./textarea";
import { Tooltip, TooltipContent, TooltipProvider, TooltipTrigger } from "./tooltip";

describe("shadcn primitives", () => {
  it("renders button and badge variants with loading behavior", () => {
    render(<>{(["default", "secondary", "ghost", "destructive", "outline"] as const).map((variant) => <Button key={variant} variant={variant} size={variant === "outline" ? "icon" : variant === "ghost" ? "lg" : "sm"} aria-label={variant} loading={variant === "default"}>{variant}</Button>)}{(["neutral", "info", "review", "success", "warning", "danger", "outline"] as const).map((variant) => <Badge key={variant} variant={variant}>{variant}</Badge>)}</>);
    expect(screen.getByRole("button", { name: "default" })).toBeDisabled();
    expect(screen.getByRole("button", { name: "default" })).toHaveAttribute("aria-busy", "true");
    expect(screen.getByText("danger")).toBeInTheDocument();
  });

  it("renders composed cards and tables with semantic content", () => {
    render(<Card><CardHeader><CardTitle>Release</CardTitle><CardDescription>Verified history</CardDescription></CardHeader><CardContent><Table><TableHeader><TableRow><TableHead>Unit</TableHead></TableRow></TableHeader><TableBody><TableRow><TableCell>point/one</TableCell></TableRow></TableBody></Table></CardContent><CardFooter>Footer</CardFooter></Card>);
    expect(screen.getByRole("table")).toBeInTheDocument();
    expect(screen.getByRole("columnheader", { name: "Unit" })).toBeInTheDocument();
    expect(screen.getByRole("cell", { name: "point/one" })).toBeInTheDocument();
    expect(screen.getByText("Verified history")).toBeInTheDocument();
  });

  it("supports labelled form controls and separators", async () => {
    const user = userEvent.setup();
    const checked = vi.fn();
    render(<><Label htmlFor="name">Name</Label><Input id="name" /><Label htmlFor="notes">Notes</Label><Textarea id="notes" /><Checkbox aria-label="Select change" onCheckedChange={checked} /><Separator /><Separator orientation="vertical" decorative={false} /><Skeleton aria-label="Loading" /></>);
    await user.type(screen.getByRole("textbox", { name: "Name" }), "Ledger");
    await user.type(screen.getByRole("textbox", { name: "Notes" }), "Context");
    await user.click(screen.getByRole("checkbox", { name: "Select change" }));
    expect(checked).toHaveBeenCalledWith(true);
    expect(screen.getByRole("separator")).toBeInTheDocument();
    expect(screen.getByLabelText("Loading")).toBeInTheDocument();
  });

  it("opens and closes an accessible Radix dialog", async () => {
    const user = userEvent.setup();
    render(<Dialog><DialogTrigger>Open</DialogTrigger><DialogContent><DialogHeader><DialogTitle>Confirm</DialogTitle><DialogDescription>Review consequence</DialogDescription></DialogHeader><DialogFooter><DialogClose>Cancel</DialogClose></DialogFooter></DialogContent></Dialog>);
    await user.click(screen.getByRole("button", { name: "Open" }));
    expect(screen.getByRole("dialog", { name: "Confirm" })).toHaveAccessibleDescription("Review consequence");
    await user.click(screen.getByRole("button", { name: "Cancel" }));
    expect(screen.queryByRole("dialog", { name: "Confirm" })).not.toBeInTheDocument();
  });

  it("reveals tooltip content from keyboard focus", async () => {
    const user = userEvent.setup();
    render(<TooltipProvider delayDuration={0}><Tooltip><TooltipTrigger>Runtime</TooltipTrigger><TooltipContent>Connected</TooltipContent></Tooltip></TooltipProvider>);
    await user.tab();
    expect(await screen.findByRole("tooltip")).toHaveTextContent("Connected");
  });
});
