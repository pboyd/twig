import { describe, it, expect, vi, beforeEach } from "vitest";
import { render, screen, fireEvent, waitFor } from "@testing-library/react";
import { MemoryRouter } from "react-router";
import { QueryClient, QueryClientProvider } from "@tanstack/react-query";
import { ConnectError, Code } from "@connectrpc/connect";
import { AddToPlanControl } from "./AddToPlanControl";
import { ToastProvider } from "../context/ToastProvider";

vi.mock("../gen/plan/v1/plan-PlanService_connectquery", () => ({
  addPlanTask: "schema:addPlanTask",
  listPlanEntries: "schema:listPlanEntries",
}));

vi.mock("../lib/planDays", () => ({
  todayIso: vi.fn(() => "2026-06-09"),
  tomorrowIso: vi.fn(() => "2026-06-10"),
  dayPickerLabel: vi.fn((iso: string) => {
    if (iso === "2026-06-09") return "Today";
    if (iso === "2026-06-10") return "Tomorrow";
    return iso;
  }),
}));

const mockInvalidateQueries = vi.fn();
vi.mock("@tanstack/react-query", async (importActual) => {
  const actual = await importActual<typeof import("@tanstack/react-query")>();
  return {
    ...actual,
    useQueryClient: vi.fn(() => ({ invalidateQueries: mockInvalidateQueries })),
  };
});

const mockMutateAsync = vi.fn();
vi.mock("@connectrpc/connect-query", () => ({
  useMutation: vi.fn(() => ({ mutateAsync: mockMutateAsync, isPending: false })),
  createConnectQueryKey: vi.fn(() => ["mock-plan-key"]),
}));

function renderControl(taskId = 42n) {
  const queryClient = new QueryClient({ defaultOptions: { queries: { retry: false } } });
  return render(
    <QueryClientProvider client={queryClient}>
      <MemoryRouter>
        <ToastProvider>
          <AddToPlanControl taskId={taskId} />
        </ToastProvider>
      </MemoryRouter>
    </QueryClientProvider>
  );
}

beforeEach(() => {
  vi.clearAllMocks();
  mockMutateAsync.mockResolvedValue({});
});

describe("AddToPlanControl — US1: add to today", () => {
  it("calls addPlanTask with today's date and taskId on add-to-today tap", async () => {
    renderControl(5n);
    fireEvent.click(screen.getByRole("button", { name: /add to today/i }));
    await waitFor(() => expect(mockMutateAsync).toHaveBeenCalledWith({
      day: "2026-06-09",
      taskId: 5n,
      durationMinute: 0,
    }));
  });

  it("does not include startMinute in the request", async () => {
    renderControl(5n);
    fireEvent.click(screen.getByRole("button", { name: /add to today/i }));
    await waitFor(() => {
      const call = mockMutateAsync.mock.calls[0][0];
      expect("startMinute" in call).toBe(false);
    });
  });

  it("shows success toast on add to today", async () => {
    renderControl(5n);
    fireEvent.click(screen.getByRole("button", { name: /add to today/i }));
    await waitFor(() => expect(screen.getByTestId("toast")).toBeTruthy());
    expect(screen.getByTestId("toast").textContent).toMatch(/today/i);
  });

  it("shows already-on-plan toast on FailedPrecondition", async () => {
    mockMutateAsync.mockRejectedValue(new ConnectError("dup", Code.FailedPrecondition));
    renderControl(5n);
    fireEvent.click(screen.getByRole("button", { name: /add to today/i }));
    await waitFor(() => expect(screen.getByTestId("toast")).toBeTruthy());
    expect(screen.getByTestId("toast").textContent).toMatch(/already/i);
  });

  it("shows connectivity error toast on transport failure", async () => {
    mockMutateAsync.mockRejectedValue(new Error("network"));
    renderControl(5n);
    fireEvent.click(screen.getByRole("button", { name: /add to today/i }));
    await waitFor(() => expect(screen.getByTestId("toast")).toBeTruthy());
    expect(screen.getByTestId("toast").textContent).toMatch(/couldn't reach/i);
  });

  it("invalidates listPlanEntries query on success", async () => {
    renderControl(5n);
    fireEvent.click(screen.getByRole("button", { name: /add to today/i }));
    await waitFor(() => expect(mockInvalidateQueries).toHaveBeenCalled());
  });
});

describe("AddToPlanControl — US3: add to future day", () => {
  it("opens the day picker popover on chevron click", () => {
    renderControl(5n);
    fireEvent.click(screen.getByRole("button", { name: /add to another day/i }));
    expect(screen.getByRole("button", { name: /tomorrow/i })).toBeTruthy();
  });

  it("calls addPlanTask with tomorrow's date on Tomorrow preset", async () => {
    renderControl(5n);
    fireEvent.click(screen.getByRole("button", { name: /add to another day/i }));
    fireEvent.click(screen.getByRole("button", { name: /tomorrow/i }));
    await waitFor(() => expect(mockMutateAsync).toHaveBeenCalledWith({
      day: "2026-06-10",
      taskId: 5n,
      durationMinute: 0,
    }));
  });

  it("shows toast naming the day after adding to tomorrow", async () => {
    renderControl(5n);
    fireEvent.click(screen.getByRole("button", { name: /add to another day/i }));
    fireEvent.click(screen.getByRole("button", { name: /tomorrow/i }));
    await waitFor(() => expect(screen.getByTestId("toast")).toBeTruthy());
    expect(screen.getByTestId("toast").textContent).toMatch(/tomorrow/i);
  });

  it("calls addPlanTask with an arbitrary picked date", async () => {
    renderControl(5n);
    fireEvent.click(screen.getByRole("button", { name: /add to another day/i }));
    const dateInput = screen.getByLabelText(/pick a date/i);
    fireEvent.change(dateInput, { target: { value: "2026-07-01" } });
    fireEvent.click(screen.getByRole("button", { name: /^add$/i }));
    await waitFor(() => expect(mockMutateAsync).toHaveBeenCalledWith({
      day: "2026-07-01",
      taskId: 5n,
      durationMinute: 0,
    }));
  });

  it("accepts a past date without blocking", async () => {
    renderControl(5n);
    fireEvent.click(screen.getByRole("button", { name: /add to another day/i }));
    const dateInput = screen.getByLabelText(/pick a date/i);
    fireEvent.change(dateInput, { target: { value: "2025-01-01" } });
    fireEvent.click(screen.getByRole("button", { name: /^add$/i }));
    await waitFor(() => expect(mockMutateAsync).toHaveBeenCalledWith({
      day: "2025-01-01",
      taskId: 5n,
      durationMinute: 0,
    }));
  });

  it("disables Add button until a date is chosen", () => {
    renderControl(5n);
    fireEvent.click(screen.getByRole("button", { name: /add to another day/i }));
    expect(screen.getByRole("button", { name: /^add$/i })).toBeDisabled();
  });
});
