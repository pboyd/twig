import { describe, it, expect, vi } from "vitest";
import { render, screen, fireEvent } from "@testing-library/react";
import { CompletionToggle } from "./CompletionToggle";

describe("CompletionToggle", () => {
  it("renders an empty circle when incomplete", () => {
    render(<CompletionToggle completed={false} onToggle={vi.fn()} />);
    const btn = screen.getByRole("button", { name: /mark complete/i });
    expect(btn).toBeTruthy();
  });

  it("renders a filled circle with check when complete", () => {
    render(<CompletionToggle completed={true} onToggle={vi.fn()} />);
    const btn = screen.getByRole("button", { name: /mark incomplete/i });
    expect(btn).toBeTruthy();
  });

  it("fires onToggle when clicked", () => {
    const onToggle = vi.fn();
    render(<CompletionToggle completed={false} onToggle={onToggle} />);
    fireEvent.click(screen.getByRole("button"));
    expect(onToggle).toHaveBeenCalledTimes(1);
  });

  it("disables interaction when disabled is true", () => {
    const onToggle = vi.fn();
    render(<CompletionToggle completed={false} disabled={true} onToggle={onToggle} />);
    const btn = screen.getByRole("button");
    expect(btn.hasAttribute("disabled")).toBe(true);
  });
});
