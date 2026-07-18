import { describe, it, expect, vi } from "vitest";
import { render, screen, fireEvent } from "@testing-library/react";
import { TaskForm } from "./TaskForm";
import { todayIso, tomorrowIso } from "../lib/planDays";

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
      expect(onSubmit).toHaveBeenCalledWith("Task", mdDesc, undefined);
    });
  });
});

describe("TaskForm — plan control (US1)", () => {
  it("CT-01: fresh form renders No plan selected", () => {
    render(<TaskForm showPlanControl onSubmit={vi.fn()} />);
    const radios = screen.getAllByRole("radio");
    const noPlan = radios.find((r) => r.textContent === "No plan");
    expect(noPlan).toBeDefined();
    expect(noPlan).toHaveAttribute("aria-checked", "true");
  });

  it("CT-02: all four choices render in contract order", () => {
    render(<TaskForm showPlanControl onSubmit={vi.fn()} />);
    const radios = screen.getAllByRole("radio");
    expect(radios).toHaveLength(4);
    expect(radios[0].textContent).toBe("No plan");
    expect(radios[1].textContent).toBe("Today");
    expect(radios[2].textContent).toBe("Tomorrow");
    expect(radios[3].getAttribute("title")).toBe("Pick a specific date");
  });

  it("CT-03: submitting with Today passes today's ISO date as third arg", async () => {
    const onSubmit = vi.fn().mockResolvedValue(undefined);
    render(<TaskForm showPlanControl onSubmit={onSubmit} />);
    fireEvent.change(screen.getByLabelText(/title/i), { target: { value: "My task" } });
    fireEvent.click(screen.getByText("Today"));
    fireEvent.click(screen.getByRole("button", { name: /add task/i }));
    await vi.waitFor(() => {
      expect(onSubmit).toHaveBeenCalledWith("My task", "", todayIso());
    });
  });

  it("CT-05: submitting untouched passes third arg undefined", async () => {
    const onSubmit = vi.fn().mockResolvedValue(undefined);
    render(<TaskForm showPlanControl onSubmit={onSubmit} />);
    fireEvent.change(screen.getByLabelText(/title/i), { target: { value: "My task" } });
    fireEvent.click(screen.getByRole("button", { name: /add task/i }));
    await vi.waitFor(() => {
      expect(onSubmit).toHaveBeenCalledWith("My task", "", undefined);
    });
  });

  it("CT-14: radiogroup has correct ARIA semantics", () => {
    render(<TaskForm showPlanControl onSubmit={vi.fn()} />);
    expect(screen.getByRole("radiogroup")).toHaveAttribute("aria-label", "Add to plan");
    const radios = screen.getAllByRole("radio");
    radios.forEach((r) => {
      expect(r).toHaveAttribute("aria-checked");
    });
  });
});
