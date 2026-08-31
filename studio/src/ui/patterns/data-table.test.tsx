import { render, screen } from "@testing-library/react";
import userEvent from "@testing-library/user-event";
import { describe, expect, it, vi } from "vitest";
import { DataTable, type Column } from "./data-table";

type Row = { id: string; name: string; ready: boolean };
const rows: Row[] = [{ id: "one", name: "First", ready: true }, { id: "two", name: "Second", ready: false }];
const columns: Column<Row>[] = [{ key: "name", header: "Name", render: (row) => row.name }];

describe("DataTable", () => {
  it("renders loading, empty, and populated states", () => {
    const { rerender, container } = render(<DataTable loading columns={columns} rows={[]} getRowId={(row) => row.id} empty="No rows" />);
    expect(container.querySelectorAll("tbody [aria-busy='true']")).toHaveLength(4);
    rerender(<DataTable columns={columns} rows={[]} getRowId={(row) => row.id} empty="No rows" />);
    expect(screen.getByText("No rows")).toBeInTheDocument();
    rerender(<DataTable columns={columns} rows={rows} getRowId={(row) => row.id} />);
    expect(screen.getByRole("table")).toBeInTheDocument();
    expect(screen.getByRole("cell", { name: "First" })).toBeInTheDocument();
  });

  it("selects eligible rows and explains disabled selection", async () => {
    const user = userEvent.setup();
    const onSelectionChange = vi.fn();
    render(<DataTable selectable columns={columns} rows={rows} getRowId={(row) => row.id} selectedIds={[]} onSelectionChange={onSelectionChange} isRowSelectable={(row) => row.ready} getSelectionDisabledReason={() => "Only ready rows can be selected"} />);
    await user.click(screen.getByRole("checkbox", { name: "Select one" }));
    expect(onSelectionChange).toHaveBeenCalledWith(["one"]);
    expect(screen.getByRole("checkbox", { name: "Select two" })).toBeDisabled();
    expect(screen.getByRole("checkbox", { name: "Select two" })).toHaveAttribute("title", "Only ready rows can be selected");
  });

  it("supports row activation, keyboard selection, and arrow navigation", async () => {
    const user = userEvent.setup();
    const onSelectionChange = vi.fn();
    const onRowClick = vi.fn();
    render(<DataTable selectable columns={columns} rows={rows} getRowId={(row) => row.id} selectedIds={[]} onSelectionChange={onSelectionChange} onRowClick={onRowClick} highlightedId="one" />);
    const first = screen.getByRole("row", { name: /First/ });
    const second = screen.getByRole("row", { name: /Second/ });
    first.focus();
    await user.keyboard(" ");
    expect(onSelectionChange).toHaveBeenCalledWith(["one"]);
    await user.keyboard("{ArrowDown}");
    expect(second).toHaveFocus();
    await user.keyboard("{Enter}");
    expect(onRowClick).toHaveBeenCalledWith(rows[1]);
    await user.keyboard("{ArrowUp}");
    expect(first).toHaveFocus();
  });
});
