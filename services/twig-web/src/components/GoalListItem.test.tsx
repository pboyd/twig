import { describe, it, expect, vi } from "vitest";
import { render, screen, fireEvent } from "@testing-library/react";
import { GoalListItem } from "./GoalListItem";

function makeGoal(overrides?: Record<string, unknown>) {
  return { id: 1n, name: "Test Goal", description: "", state: 2, position: 0n, $typeName: "goal.v1.Goal", ...overrides };
}

describe("GoalListItem", () => {
  it("renders the goal name", () => {
    const onOpen = vi.fn();
    render(<GoalListItem goal={makeGoal() as any} onOpen={onOpen} />);
    expect(screen.getByText("Test Goal")).toBeTruthy();
  });

  it("shows due date when present", () => {
    const onOpen = vi.fn();
    render(<GoalListItem goal={makeGoal({ due: { seconds: 1719000000n, nanos: 0 } }) as any} onOpen={onOpen} />);
    expect(screen.getByText(/Due/)).toBeTruthy();
  });

  it("does not show due date when absent", () => {
    const onOpen = vi.fn();
    render(<GoalListItem goal={makeGoal({ due: undefined }) as any} onOpen={onOpen} />);
    expect(screen.queryByText(/Due/)).toBeNull();
  });

  it("calls onOpen with goal id on click", () => {
    const onOpen = vi.fn();
    render(<GoalListItem goal={makeGoal() as any} onOpen={onOpen} />);
    fireEvent.click(screen.getByRole("button"));
    expect(onOpen).toHaveBeenCalledWith(1n);
  });

  it("shows the state label", () => {
    const onOpen = vi.fn();
    render(<GoalListItem goal={makeGoal({ state: 1 }) as any} onOpen={onOpen} />);
    expect(screen.getByText("Incubating")).toBeTruthy();
  });
});
