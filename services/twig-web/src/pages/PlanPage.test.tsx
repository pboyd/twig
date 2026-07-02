import { describe, it, expect, vi, beforeEach } from "vitest";
import { render, screen, fireEvent, waitFor } from "@testing-library/react";
import { MemoryRouter } from "react-router";
import { QueryClient, QueryClientProvider } from "@tanstack/react-query";
import { ConnectError, Code } from "@connectrpc/connect";
import PlanPage from "./PlanPage";
import { ToastProvider } from "../context/ToastProvider";

vi.mock("../gen/plan/v1/plan-PlanService_connectquery", () => ({
  listPlanEntries: "schema:listPlanEntries",
  removePlanEntry: "schema:removePlanEntry",
}));

vi.mock("../gen/task/v1/task-TaskService_connectquery", () => ({
  listTasks: "schema:listTasks",
  completeTask: "schema:completeTask",
  uncompleteTask: "schema:uncompleteTask",
}));

vi.mock("../components/AppHeader", () => ({
  AppHeader: () => <header data-testid="app-header" />,
}));

const mockInvalidateQueries = vi.fn();
vi.mock("@tanstack/react-query", async (importActual) => {
  const actual = await importActual<typeof import("@tanstack/react-query")>();
  return {
    ...actual,
    useQueryClient: vi.fn(() => ({ invalidateQueries: mockInvalidateQueries })),
  };
});

let listPlanEntriesResult: ReturnType<typeof vi.fn> = vi.fn();
let listTasksResult: ReturnType<typeof vi.fn> = vi.fn();
const mockMutateAsync = vi.fn();

vi.mock("@connectrpc/connect-query", () => ({
  useQuery: vi.fn((schema: string) => {
    if (schema === "schema:listPlanEntries") return listPlanEntriesResult();
    return listTasksResult();
  }),
  useMutation: vi.fn(() => ({ mutateAsync: mockMutateAsync, isPending: false })),
  createConnectQueryKey: vi.fn(() => ["mock-key"]),
}));

function makePlanEntry(overrides: Record<string, unknown> = {}) {
  return {
    day: "2026-06-06",
    id: 1,
    taskId: 0n,
    name: "Standup",
    startMinute: 540,
    durationMinute: 30,
    completed: false,
    ...overrides,
  };
}

function makeTask(id: bigint, name: string) {
  return { id, name, description: "", parentId: undefined, due: undefined, completedAt: undefined, estimate: 0 };
}

function renderPage() {
  const queryClient = new QueryClient({ defaultOptions: { queries: { retry: false } } });
  return render(
    <QueryClientProvider client={queryClient}>
      <MemoryRouter>
        <ToastProvider>
          <PlanPage />
        </ToastProvider>
      </MemoryRouter>
    </QueryClientProvider>
  );
}

beforeEach(() => {
  vi.clearAllMocks();
  mockMutateAsync.mockResolvedValue({});
  listTasksResult = vi.fn().mockReturnValue({
    data: { tasks: [] },
    isLoading: false,
    isError: false,
    error: null,
    refetch: vi.fn(),
  });
  listPlanEntriesResult = vi.fn().mockReturnValue({
    data: { entries: [] },
    isLoading: false,
    isError: false,
    error: null,
    refetch: vi.fn(),
  });
});

// ─── US1: Loading state ───────────────────────────────────────

describe("PlanPage — US1: loading state", () => {
  it("renders a spinner when loading", () => {
    listPlanEntriesResult = vi.fn().mockReturnValue({
      data: undefined,
      isLoading: true,
      isError: false,
      error: null,
      refetch: vi.fn(),
    });
    renderPage();
    expect(screen.getByRole("status")).toBeTruthy();
  });
});

describe("PlanPage — US1: error state", () => {
  it("renders an error banner with retry when plan query fails", () => {
    listPlanEntriesResult = vi.fn().mockReturnValue({
      data: undefined,
      isLoading: false,
      isError: true,
      error: new Error("network failure"),
      refetch: vi.fn(),
    });
    renderPage();
    expect(screen.getByRole("alert")).toBeTruthy();
    expect(screen.getByRole("button", { name: /try again/i })).toBeTruthy();
  });
});

describe("PlanPage — US1: empty state", () => {
  it("shows empty state message when no entries", () => {
    renderPage();
    expect(screen.getByText(/blank slate/i)).toBeTruthy();
  });
});

// ─── US1: Timed entries render with timeline ───────────────────

describe("PlanPage — US1: timed entries render in the timeline", () => {
  it("renders timed entries with names visible", () => {
    listPlanEntriesResult = vi.fn().mockReturnValue({
      data: {
        entries: [
          makePlanEntry({ id: 1, name: "Morning meeting", startMinute: 540, durationMinute: 30 }),
          makePlanEntry({ id: 2, name: "Code review", startMinute: 600, durationMinute: 60 }),
        ],
      },
      isLoading: false,
      isError: false,
      error: null,
      refetch: vi.fn(),
    });
    renderPage();
    expect(screen.getByText("Morning meeting")).toBeTruthy();
    expect(screen.getByText("Code review")).toBeTruthy();
    expect(screen.getByTestId("timed-section")).toBeTruthy();
  });

  it("renders hour labels in the timeline", () => {
    listPlanEntriesResult = vi.fn().mockReturnValue({
      data: {
        entries: [
          makePlanEntry({ id: 1, name: "Meeting", startMinute: 540, durationMinute: 30 }),
        ],
      },
      isLoading: false,
      isError: false,
      error: null,
      refetch: vi.fn(),
    });
    renderPage();
    expect(screen.getByText("8:00 am")).toBeTruthy();
    expect(screen.getByText("9:00 am")).toBeTruthy();
  });
});

describe("PlanPage — US1: task-linked name resolution", () => {
  it("uses task name for task-linked entry with empty name", () => {
    listTasksResult = vi.fn().mockReturnValue({
      data: { tasks: [makeTask(5n, "Write tests")] },
      isLoading: false,
      isError: false,
      error: null,
      refetch: vi.fn(),
    });
    listPlanEntriesResult = vi.fn().mockReturnValue({
      data: {
        entries: [makePlanEntry({ taskId: 5n, name: "", startMinute: 540, durationMinute: 60 })],
      },
      isLoading: false,
      isError: false,
      error: null,
      refetch: vi.fn(),
    });
    renderPage();
    expect(screen.getByText("Write tests")).toBeTruthy();
  });

  it("uses generic label when both name and task name are absent", () => {
    listPlanEntriesResult = vi.fn().mockReturnValue({
      data: {
        entries: [makePlanEntry({ taskId: 99n, name: "", startMinute: 540, durationMinute: 60 })],
      },
      isLoading: false,
      isError: false,
      error: null,
      refetch: vi.fn(),
    });
    renderPage();
    expect(screen.getByText("Untitled entry")).toBeTruthy();
  });
});

describe("PlanPage — US1: event vs task visual distinction", () => {
  it("renders events and task entries both visible", () => {
    listPlanEntriesResult = vi.fn().mockReturnValue({
      data: {
        entries: [
          makePlanEntry({ id: 1, name: "Lunch", taskId: 0n, startMinute: 720, durationMinute: 60 }),
          makePlanEntry({ id: 2, name: "Code", taskId: 1n, startMinute: 600, durationMinute: 60 }),
        ],
      },
      isLoading: false,
      isError: false,
      error: null,
      refetch: vi.fn(),
    });
    renderPage();
    expect(screen.getByText("Lunch")).toBeTruthy();
    expect(screen.getByText("Code")).toBeTruthy();
  });
});

// ─── US2: Untimed entries in separate section (above timeline) ─

describe("PlanPage — US2: untimed entries in separate section above timeline", () => {
  it("renders untimed entries before timed entries in DOM order", () => {
    listPlanEntriesResult = vi.fn().mockReturnValue({
      data: {
        entries: [
          makePlanEntry({ id: 1, name: "Timed task", startMinute: 540, durationMinute: 60 }),
          makePlanEntry({ id: 2, name: "Untimed task", startMinute: undefined }),
        ],
      },
      isLoading: false,
      isError: false,
      error: null,
      refetch: vi.fn(),
    });
    renderPage();
    expect(screen.getByText("Timed task")).toBeTruthy();
    expect(screen.getByText("Untimed task")).toBeTruthy();
    expect(screen.getByTestId("untimed-section")).toBeTruthy();
    expect(screen.getByTestId("timed-section")).toBeTruthy();
  });
});

describe("PlanPage — US2: completed task entries marked done", () => {
  it("marks completed task entries with CompletionToggle filled state", () => {
    listPlanEntriesResult = vi.fn().mockReturnValue({
      data: {
        entries: [
          makePlanEntry({ id: 1, taskId: 1n, name: "Done task", completed: true, startMinute: 540, durationMinute: 60 }),
        ],
      },
      isLoading: false,
      isError: false,
      error: null,
      refetch: vi.fn(),
    });
    renderPage();
    expect(screen.getByRole("button", { name: /mark incomplete/i })).toBeTruthy();
  });
});

describe("PlanPage — US2: task entries navigate to task detail", () => {
  it("task-linked entries are rendered as links to /tasks/:taskId", () => {
    listPlanEntriesResult = vi.fn().mockReturnValue({
      data: {
        entries: [
          makePlanEntry({ id: 1, taskId: 42n, name: "My task", startMinute: 540, durationMinute: 60 }),
        ],
      },
      isLoading: false,
      isError: false,
      error: null,
      refetch: vi.fn(),
    });
    renderPage();
    const link = screen.getByRole("link", { name: /my task/i });
    expect(link.getAttribute("href")).toBe("/tasks/42");
  });
});

describe("PlanPage — US2: event entries are non-interactive", () => {
  it("event entries are not links", () => {
    listPlanEntriesResult = vi.fn().mockReturnValue({
      data: {
        entries: [
          makePlanEntry({ id: 1, taskId: 0n, name: "Lunch break", startMinute: 720, durationMinute: 60 }),
        ],
      },
      isLoading: false,
      isError: false,
      error: null,
      refetch: vi.fn(),
    });
    renderPage();
    expect(screen.queryByRole("link", { name: /lunch break/i })).toBeNull();
    expect(screen.getByText("Lunch break")).toBeTruthy();
  });
});

// ─── US3: Day navigation ───────────────────────────────────────

describe("PlanPage — US3: day navigation", () => {
  it("renders prev/next/today controls", () => {
    renderPage();
    expect(screen.getByRole("button", { name: /prev|previous|←|‹/i })).toBeTruthy();
    expect(screen.getByRole("button", { name: /next|→|›/i })).toBeTruthy();
    expect(screen.getByRole("button", { name: /today/i })).toBeTruthy();
  });

  it("shows a date label", () => {
    renderPage();
    expect(screen.getByRole("heading", { name: /today/i })).toBeTruthy();
  });

  it("a navigated day with no entries shows the empty state", () => {
    renderPage();
    const prevBtn = screen.getByRole("button", { name: /prev|previous|←|‹/i });
    fireEvent.click(prevBtn);
    expect(screen.getByText(/blank slate/i)).toBeTruthy();
  });
});

// ─── Complete / uncomplete toggle ──────────────────────────────

describe("PlanPage — toggle complete (US2 two-way toggle)", () => {
  it("renders a toggle for an incomplete task entry", () => {
    listPlanEntriesResult = vi.fn().mockReturnValue({
      data: {
        entries: [
          makePlanEntry({ id: 1, taskId: 3n, name: "Do something", startMinute: undefined, completed: false }),
        ],
      },
      isLoading: false,
      isError: false,
      error: null,
      refetch: vi.fn(),
    });
    renderPage();
    expect(screen.getByRole("button", { name: /mark complete/i })).toBeTruthy();
  });

  it("does not render a toggle for event entries (taskId == 0)", () => {
    listPlanEntriesResult = vi.fn().mockReturnValue({
      data: {
        entries: [
          makePlanEntry({ id: 1, taskId: 0n, name: "Standup", startMinute: undefined, completed: false }),
        ],
      },
      isLoading: false,
      isError: false,
      error: null,
      refetch: vi.fn(),
    });
    renderPage();
    expect(screen.queryByRole("button", { name: /mark (complete|incomplete)/i })).toBeNull();
  });

  it("renders a filled toggle for completed task entries", () => {
    listPlanEntriesResult = vi.fn().mockReturnValue({
      data: {
        entries: [
          makePlanEntry({ id: 1, taskId: 3n, name: "Done", startMinute: 540, durationMinute: 60, completed: true }),
        ],
      },
      isLoading: false,
      isError: false,
      error: null,
      refetch: vi.fn(),
    });
    renderPage();
    expect(screen.getByRole("button", { name: /mark incomplete/i })).toBeTruthy();
  });

  it("calls completeTask on incomplete entry and invalidates queries", async () => {
    listPlanEntriesResult = vi.fn().mockReturnValue({
      data: {
        entries: [makePlanEntry({ id: 1, taskId: 5n, name: "My task", startMinute: undefined, completed: false })],
      },
      isLoading: false,
      isError: false,
      error: null,
      refetch: vi.fn(),
    });
    renderPage();
    fireEvent.click(screen.getByRole("button", { name: /mark complete/i }));
    await waitFor(() => expect(mockMutateAsync).toHaveBeenCalledWith({ id: 5n }));
    expect(mockInvalidateQueries).toHaveBeenCalledTimes(2);
  });

  it("calls uncompleteTask on completed entry and invalidates queries", async () => {
    listPlanEntriesResult = vi.fn().mockReturnValue({
      data: {
        entries: [makePlanEntry({ id: 1, taskId: 5n, name: "My task", startMinute: 540, durationMinute: 60, completed: true })],
      },
      isLoading: false,
      isError: false,
      error: null,
      refetch: vi.fn(),
    });
    renderPage();
    fireEvent.click(screen.getByRole("button", { name: /mark incomplete/i }));
    await waitFor(() => expect(mockMutateAsync).toHaveBeenCalledWith({ id: 5n }));
    expect(mockInvalidateQueries).toHaveBeenCalledTimes(2);
  });

  it("shows inline complete-blocked message on FailedPrecondition", async () => {
    mockMutateAsync.mockRejectedValue(new ConnectError("blocked", Code.FailedPrecondition));
    listPlanEntriesResult = vi.fn().mockReturnValue({
      data: {
        entries: [makePlanEntry({ id: 1, taskId: 5n, name: "My task", startMinute: undefined, completed: false })],
      },
      isLoading: false,
      isError: false,
      error: null,
      refetch: vi.fn(),
    });
    renderPage();
    fireEvent.click(screen.getByRole("button", { name: /mark complete/i }));
    await waitFor(() => expect(screen.getByText(/sub-tasks/i)).toBeTruthy());
  });

  it("shows inline reopen-blocked message on FailedPrecondition for completed entry", async () => {
    mockMutateAsync.mockRejectedValue(new ConnectError("parent-blocked", Code.FailedPrecondition));
    listPlanEntriesResult = vi.fn().mockReturnValue({
      data: {
        entries: [makePlanEntry({ id: 1, taskId: 5n, name: "My task", startMinute: 540, durationMinute: 60, completed: true })],
      },
      isLoading: false,
      isError: false,
      error: null,
      refetch: vi.fn(),
    });
    renderPage();
    fireEvent.click(screen.getByRole("button", { name: /mark incomplete/i }));
    await waitFor(() => expect(screen.getByText(/parent/i)).toBeTruthy());
  });
});

// ─── Remove entry ──────────────────────────────────────────────

describe("PlanPage — remove entry from planner", () => {
  it("renders a remove button for timed task entries", () => {
    listPlanEntriesResult = vi.fn().mockReturnValue({
      data: {
        entries: [makePlanEntry({ id: 1, taskId: 3n, name: "Timed", startMinute: 540, durationMinute: 60, completed: false })],
      },
      isLoading: false,
      isError: false,
      error: null,
      refetch: vi.fn(),
    });
    renderPage();
    expect(screen.getByTestId("remove-btn")).toBeTruthy();
  });

  it("renders a remove button for untimed task entries", () => {
    listPlanEntriesResult = vi.fn().mockReturnValue({
      data: {
        entries: [makePlanEntry({ id: 1, taskId: 3n, name: "Untimed", startMinute: undefined, completed: false })],
      },
      isLoading: false,
      isError: false,
      error: null,
      refetch: vi.fn(),
    });
    renderPage();
    expect(screen.getByTestId("remove-btn")).toBeTruthy();
  });

  it("renders a remove button for event entries", () => {
    listPlanEntriesResult = vi.fn().mockReturnValue({
      data: {
        entries: [makePlanEntry({ id: 1, taskId: 0n, name: "Standup", startMinute: 540, durationMinute: 30, completed: false })],
      },
      isLoading: false,
      isError: false,
      error: null,
      refetch: vi.fn(),
    });
    renderPage();
    expect(screen.getByTestId("remove-btn")).toBeTruthy();
  });

  it("calls removePlanEntry and invalidates the day plan on success", async () => {
    listPlanEntriesResult = vi.fn().mockReturnValue({
      data: {
        entries: [makePlanEntry({ id: 7, taskId: 0n, name: "Standup", startMinute: 540, durationMinute: 30 })],
      },
      isLoading: false,
      isError: false,
      error: null,
      refetch: vi.fn(),
    });
    renderPage();
    fireEvent.click(screen.getByTestId("remove-btn"));
    await waitFor(() => expect(mockMutateAsync).toHaveBeenCalledWith(
      expect.objectContaining({ id: 7 })
    ));
    expect(mockInvalidateQueries).toHaveBeenCalled();
  });

  it("shows inline error on failed remove and leaves entry visible", async () => {
    mockMutateAsync.mockRejectedValue(new Error("network"));
    listPlanEntriesResult = vi.fn().mockReturnValue({
      data: {
        entries: [makePlanEntry({ id: 1, taskId: 0n, name: "Standup", startMinute: 540, durationMinute: 30 })],
      },
      isLoading: false,
      isError: false,
      error: null,
      refetch: vi.fn(),
    });
    renderPage();
    fireEvent.click(screen.getByTestId("remove-btn"));
    await waitFor(() => expect(screen.getByText(/couldn't reach/i)).toBeTruthy());
    expect(screen.getByText("Standup")).toBeTruthy();
  });
});

describe("PlanPage — inline name rendering (US2)", () => {
  it("renders bold in plan entry name via inline markdown", () => {
    listPlanEntriesResult = vi.fn().mockReturnValue({
      data: {
        entries: [makePlanEntry({ id: 1, name: "**Bold** task", taskId: 0n, startMinute: 540, durationMinute: 30 })],
      },
      isLoading: false,
      isError: false,
      error: null,
      refetch: vi.fn(),
    });
    const { container } = renderPage();
    expect(container.querySelector("strong")).toBeTruthy();
  });

  it("does not emit block elements for plan entry name with block syntax", () => {
    listPlanEntriesResult = vi.fn().mockReturnValue({
      data: {
        entries: [makePlanEntry({ id: 1, name: "# Not a heading", taskId: 0n, startMinute: 540, durationMinute: 30 })],
      },
      isLoading: false,
      isError: false,
      error: null,
      refetch: vi.fn(),
    });
    const { container } = renderPage();
    const timelineSection = container.querySelector('[data-testid="timed-section"]');
    expect(timelineSection?.querySelector("h1, h2, h3, h4, h5, h6")).toBeNull();
    expect(container.textContent).toContain("Not a heading");
  });
});
