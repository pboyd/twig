import { describe, it, expect, vi } from "vitest";
import { render, screen, fireEvent } from "@testing-library/react";
import { StatusUpdateList } from "./StatusUpdateList";

function makeUpdate(id: bigint, body: string, createdAt = { seconds: 1719000000n, nanos: 0, $typeName: "google.protobuf.Timestamp" as const }) {
  return { id, goalId: 1n, body, createdAt, $typeName: "goal.v1.StatusUpdate" as const };
}

describe("StatusUpdateList", () => {
  it("renders updates newest-first", () => {
    const updates = [
      makeUpdate(1n, "First update"),
      makeUpdate(2n, "Second update"),
    ];
    render(<StatusUpdateList updates={updates} />);
    const items = screen.getAllByText(/update/i);
    expect(items).toHaveLength(2);
  });

  it("renders the body as markdown", () => {
    const updates = [makeUpdate(1n, "**bold** text")];
    render(<StatusUpdateList updates={updates} />);
    expect(screen.getByText("bold")).toBeTruthy();
  });

  it("shows Edit button when onEdit is provided", () => {
    const onEdit = vi.fn();
    render(<StatusUpdateList updates={[makeUpdate(1n, "test")]} onEdit={onEdit} />);
    expect(screen.getByText("Edit")).toBeTruthy();
  });

  it("shows Delete button when onDelete is provided", () => {
    const onDelete = vi.fn();
    render(<StatusUpdateList updates={[makeUpdate(1n, "test")]} onDelete={onDelete} />);
    expect(screen.getByText("Delete")).toBeTruthy();
  });

  it("calls onEdit with the update", () => {
    const onEdit = vi.fn();
    const update = makeUpdate(1n, "test");
    render(<StatusUpdateList updates={[update]} onEdit={onEdit} />);
    fireEvent.click(screen.getByText("Edit"));
    expect(onEdit).toHaveBeenCalledWith(update);
  });

  it("calls onDelete with the update id", () => {
    const onDelete = vi.fn();
    render(<StatusUpdateList updates={[makeUpdate(1n, "test")]} onDelete={onDelete} />);
    fireEvent.click(screen.getByText("Delete"));
    expect(onDelete).toHaveBeenCalledWith(1n);
  });

  it("returns null for empty updates", () => {
    const { container } = render(<StatusUpdateList updates={[]} />);
    expect(container.innerHTML).toBe("");
  });
});
