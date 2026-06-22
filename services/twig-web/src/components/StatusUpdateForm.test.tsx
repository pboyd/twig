import { describe, it, expect, vi } from "vitest";
import { render, screen, fireEvent } from "@testing-library/react";
import { StatusUpdateForm } from "./StatusUpdateForm";

describe("StatusUpdateForm", () => {
  it("renders a textarea and buttons", () => {
    const onSubmit = vi.fn();
    const onCancel = vi.fn();
    render(<StatusUpdateForm submitLabel="Save" onSubmit={onSubmit} onCancel={onCancel} />);
    expect(screen.getByRole("textbox")).toBeTruthy();
    expect(screen.getByText("Save")).toBeTruthy();
    expect(screen.getByText("Cancel")).toBeTruthy();
  });

  it("calls onSubmit with trimmed body", () => {
    const onSubmit = vi.fn();
    const onCancel = vi.fn();
    render(<StatusUpdateForm submitLabel="Save" onSubmit={onSubmit} onCancel={onCancel} />);
    const textarea = screen.getByRole("textbox");
    fireEvent.change(textarea, { target: { value: "  Hello world  " } });
    fireEvent.click(screen.getByText("Save"));
    expect(onSubmit).toHaveBeenCalledWith("Hello world");
  });

  it("rejects empty body with validation message", () => {
    const onSubmit = vi.fn();
    const onCancel = vi.fn();
    render(<StatusUpdateForm submitLabel="Save" onSubmit={onSubmit} onCancel={onCancel} />);
    fireEvent.click(screen.getByText("Save"));
    expect(onSubmit).not.toHaveBeenCalled();
    expect(screen.getByText(/substance/i)).toBeTruthy();
  });

  it("rejects whitespace-only body", () => {
    const onSubmit = vi.fn();
    const onCancel = vi.fn();
    render(<StatusUpdateForm submitLabel="Save" onSubmit={onSubmit} onCancel={onCancel} />);
    const textarea = screen.getByRole("textbox");
    fireEvent.change(textarea, { target: { value: "   " } });
    fireEvent.click(screen.getByText("Save"));
    expect(onSubmit).not.toHaveBeenCalled();
  });

  it("calls onCancel when Cancel is clicked", () => {
    const onSubmit = vi.fn();
    const onCancel = vi.fn();
    render(<StatusUpdateForm submitLabel="Save" onSubmit={onSubmit} onCancel={onCancel} />);
    fireEvent.click(screen.getByText("Cancel"));
    expect(onCancel).toHaveBeenCalled();
  });

  it("seeds with initial body when editing", () => {
    const onSubmit = vi.fn();
    const onCancel = vi.fn();
    render(<StatusUpdateForm initialBody="Existing text" submitLabel="Update" onSubmit={onSubmit} onCancel={onCancel} />);
    const textarea = screen.getByRole("textbox") as HTMLTextAreaElement;
    expect(textarea.value).toBe("Existing text");
    expect(screen.getByText("Update")).toBeTruthy();
  });
});
