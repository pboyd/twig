import { describe, it, expect, vi } from "vitest";
import { render, screen, fireEvent } from "@testing-library/react";
import { TaskForm } from "./TaskForm";

describe("TaskForm — US3 round-trip safety", () => {
  it("shows raw markdown source in name field, not rendered HTML", () => {
    const mdName = "**Ship** the ~~old~~ report";
    render(<TaskForm initialName={mdName} initialDescription="" onSubmit={vi.fn()} />);
    const nameInput = screen.getByLabelText(/title/i) as HTMLInputElement;
    expect(nameInput.value).toBe(mdName);
    // Should not have rendered <strong> or <del> inside the input
    expect(nameInput.tagName.toLowerCase()).toBe("input");
  });

  it("shows raw markdown source in description textarea, not rendered HTML", () => {
    const mdDesc = "# Heading\n\n- item\n\n**bold**";
    render(<TaskForm initialName="Task" initialDescription={mdDesc} onSubmit={vi.fn()} />);
    const textarea = screen.getByLabelText(/description/i) as HTMLTextAreaElement;
    expect(textarea.value).toBe(mdDesc);
    expect(textarea.tagName.toLowerCase()).toBe("textarea");
  });

  it("submits unchanged description byte-for-byte", async () => {
    const mdDesc = "# Heading\n\n- item\n\n**bold**";
    const onSubmit = vi.fn().mockResolvedValue(undefined);
    render(<TaskForm initialName="Task" initialDescription={mdDesc} onSubmit={onSubmit} />);
    fireEvent.click(screen.getByRole("button", { name: /add task/i }));
    await vi.waitFor(() => {
      expect(onSubmit).toHaveBeenCalledWith("Task", mdDesc);
    });
  });
});
