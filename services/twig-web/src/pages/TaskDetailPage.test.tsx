import { describe, it, expect, vi, beforeEach } from "vitest";
import { render, screen, fireEvent, waitFor } from "@testing-library/react";
import { MemoryRouter, Route, Routes } from "react-router";
import { QueryClient, QueryClientProvider } from "@tanstack/react-query";
import TaskDetailPage from "./TaskDetailPage";
import { ConnectError, Code } from "@connectrpc/connect";
import { ToastProvider } from "../context/ToastProvider";
import { messages } from "../theme/messages";
import { formatDueDate } from "../lib/formatTimestamp";

// Stable mock schema tokens (used to differentiate useQuery/useMutation calls)
vi.mock("../gen/task/v1/task-TaskService_connectquery", () => ({
  getTask: "schema:getTask",
  listTasks: "schema:listTasks",
  updateTask: "schema:updateTask",
  deleteTask: "schema:deleteTask",
  completeTask: "schema:completeTask",
  uncompleteTask: "schema:uncompleteTask",
  setTaskGoal: "schema:setTaskGoal",
}));

vi.mock("../gen/goal/v1/goal-GoalService_connectquery", () => ({
  listGoals: "schema:listGoals",
}));

vi.mock("../components/AppHeader", () => ({
  AppHeader: () => <header data-testid="app-header" />,
}));

const mockCompleteTaskFn = vi.fn();
const mockUncompleteTaskFn = vi.fn();
const mockUpdateTaskFn = vi.fn();
const mockInvalidateQueries = vi.fn();
const mockGetTaskFn = vi.fn();

vi.mock("../lib/transport", () => ({ transport: {} }));
vi.mock("@connectrpc/connect", async (importActual) => {
  const actual = await importActual<typeof import("@connectrpc/connect")>();
  return {
    ...actual,
    createClient: vi.fn(() => ({ getTask: mockGetTaskFn })),
  };
});

vi.mock("@tanstack/react-query", async (importActual) => {
  const actual = await importActual<typeof import("@tanstack/react-query")>();
  return {
    ...actual,
    useQueryClient: vi.fn(() => ({
      invalidateQueries: mockInvalidateQueries,
    })),
  };
});

const mockDeleteTaskFn = vi.fn();
const mockSetTaskGoalFn = vi.fn();

// Per-test configurable state
let getTaskResult: ReturnType<typeof vi.fn> = vi.fn();
let listTasksResult: ReturnType<typeof vi.fn> = vi.fn();
let listGoalsResult: ReturnType<typeof vi.fn> = vi.fn();

vi.mock("@connectrpc/connect-query", () => ({
  useQuery: vi.fn((schema: string) => {
    if (schema === "schema:getTask") return getTaskResult();
    if (schema === "schema:listGoals") return listGoalsResult();
    return listTasksResult();
  }),
  useMutation: vi.fn((schema: string) => {
    if (schema === "schema:completeTask") {
      return { mutateAsync: mockCompleteTaskFn, isPending: false };
    }
    if (schema === "schema:uncompleteTask") {
      return { mutateAsync: mockUncompleteTaskFn, isPending: false };
    }
    if (schema === "schema:deleteTask") {
      return { mutateAsync: mockDeleteTaskFn, isPending: false };
    }
    if (schema === "schema:setTaskGoal") {
      return { mutateAsync: mockSetTaskGoalFn, isPending: false };
    }
    // updateTask
    return { mutateAsync: mockUpdateTaskFn, isPending: false };
  }),
  createConnectQueryKey: vi.fn(() => ["mock-key"]),
}));

const mockTask = {
  id: 1n,
  name: "Test Task",
  description: "A description",
  due: undefined,
  parentId: undefined,
  completedAt: undefined,
  estimate: 0,
  $typeName: "task.v1.Task",
};

function renderDetailPage() {
  const queryClient = new QueryClient({ defaultOptions: { queries: { retry: false } } });
  return render(
    <QueryClientProvider client={queryClient}>
      <ToastProvider>
        <MemoryRouter initialEntries={["/tasks/1"]}>
          <Routes>
            <Route path="/tasks/:id" element={<TaskDetailPage />} />
            <Route path="/tasks" element={<div data-testid="tasks-page" />} />
            <Route path="/goals/:id" element={<div data-testid="goal-page" />} />
            <Route path="/login" element={<div data-testid="login-page" />} />
          </Routes>
        </MemoryRouter>
      </ToastProvider>
    </QueryClientProvider>
  );
}

beforeEach(() => {
  vi.clearAllMocks();
  getTaskResult = vi.fn().mockReturnValue({
    data: { task: mockTask },
    isLoading: false,
    isError: false,
    error: null,
  });
  listTasksResult = vi.fn().mockReturnValue({
    data: { tasks: [] },
    isLoading: false,
    isError: false,
    error: null,
  });
  listGoalsResult = vi.fn().mockReturnValue({
    data: { goals: [] },
    isLoading: false,
    isError: false,
    error: null,
  });
  mockUpdateTaskFn.mockResolvedValue({ task: mockTask });
  mockCompleteTaskFn.mockResolvedValue({ task: { ...mockTask, completedAt: { seconds: 1n, nanos: 0 } } });
  mockUncompleteTaskFn.mockResolvedValue({ task: mockTask });
  mockGetTaskFn.mockResolvedValue({ task: mockTask });
  mockDeleteTaskFn.mockResolvedValue({});
  mockSetTaskGoalFn.mockResolvedValue({ task: mockTask });
});

describe("TaskDetailPage — edit mode", () => {
  it("blocks save when title is empty", async () => {
    renderDetailPage();

    // Open edit mode
    fireEvent.click(screen.getByRole("button", { name: /edit/i }));

    // Clear the name field
    const nameInput = screen.getByLabelText(/title/i);
    fireEvent.change(nameInput, { target: { value: "" } });

    // Submit via the form element to bypass jsdom native required-field validation
    fireEvent.submit(nameInput.closest("form")!);

    // Validation error should appear; mutation should NOT be called
    await waitFor(() => {
      expect(screen.getByText("A task needs a name to live by.")).toBeInTheDocument();
    });
    expect(mockUpdateTaskFn).not.toHaveBeenCalled();
  });

  it("blocks save when title is only whitespace", async () => {
    renderDetailPage();

    fireEvent.click(screen.getByRole("button", { name: /edit/i }));

    const nameInput = screen.getByLabelText(/title/i);
    fireEvent.change(nameInput, { target: { value: "   " } });

    fireEvent.submit(nameInput.closest("form")!);

    await waitFor(() => {
      expect(screen.getByText("A task needs a name to live by.")).toBeInTheDocument();
    });
    expect(mockUpdateTaskFn).not.toHaveBeenCalled();
  });
});

describe("TaskDetailPage — inline name in header (US2)", () => {
  it("renders bold in task name header via inline markdown", () => {
    getTaskResult = vi.fn().mockReturnValue({
      data: { task: { ...mockTask, name: "**Ship** it" } },
      isLoading: false,
      isError: false,
      error: null,
    });
    const { container } = renderDetailPage();
    expect(container.querySelector("strong")).toBeTruthy();
  });

  it("does not emit h1 block for name with heading prefix", () => {
    getTaskResult = vi.fn().mockReturnValue({
      data: { task: { ...mockTask, name: "# Not a real heading" } },
      isLoading: false,
      isError: false,
      error: null,
    });
    const { container } = renderDetailPage();
    // The outer h1 wraps the inline markdown; inline mode should not produce nested h1
    const h1s = container.querySelectorAll("h1");
    expect(h1s.length).toBe(1); // only the outer wrapper h1
    expect(container.textContent).toContain("Not a real heading");
  });
});

describe("TaskDetailPage — markdown description (US1)", () => {
  it("renders a heading from description markdown", async () => {
    getTaskResult = vi.fn().mockReturnValue({
      data: {
        task: {
          ...mockTask,
          description: "# My Heading\n\n- item one\n- item two\n\n**bold** and _italic_ and ~~struck~~ and `code`\n\n> a quote\n\n---\n\n| a | b |\n|---|---|\n| 1 | 2 |\n\n[link](https://example.com)",
        },
      },
      isLoading: false,
      isError: false,
      error: null,
    });
    const { container } = renderDetailPage();
    // Heading should render as an h1 element
    await waitFor(() => {
      expect(container.querySelector("h1, h2, h3")).toBeTruthy();
    });
    // No raw markdown syntax visible
    expect(screen.queryByText(/^# My Heading/)).toBeNull();
  });

  it("renders list items from description", async () => {
    getTaskResult = vi.fn().mockReturnValue({
      data: { task: { ...mockTask, description: "- item one\n- item two" } },
      isLoading: false,
      isError: false,
      error: null,
    });
    const { container } = renderDetailPage();
    await waitFor(() => {
      expect(container.querySelector("li")).toBeTruthy();
    });
  });

  it("renders a clickable link from description", async () => {
    getTaskResult = vi.fn().mockReturnValue({
      data: { task: { ...mockTask, description: "[go here](https://example.com)" } },
      isLoading: false,
      isError: false,
      error: null,
    });
    const { container } = renderDetailPage();
    await waitFor(() => {
      const a = container.querySelector("a");
      expect(a).toBeTruthy();
      expect(a?.getAttribute("href")).toBe("https://example.com");
    });
  });

  it("renders nothing when description is empty", () => {
    getTaskResult = vi.fn().mockReturnValue({
      data: { task: { ...mockTask, description: "" } },
      isLoading: false,
      isError: false,
      error: null,
    });
    renderDetailPage();
    // No markdown wrapper should produce any extra DOM
    expect(screen.queryByRole("article")).toBeNull();
  });
});

describe("TaskDetailPage — completion toggle", () => {
  it("shows 'finish sub-tasks first' on FailedPrecondition when completing", async () => {
    mockCompleteTaskFn.mockRejectedValue(
      new ConnectError("has incomplete sub-tasks", Code.FailedPrecondition)
    );

    renderDetailPage();

    // Click the complete toggle button
    fireEvent.click(screen.getByRole("button", { name: /mark complete/i }));

    await waitFor(() => {
      expect(screen.getByText("Hold on — finish its sub-tasks first.")).toBeInTheDocument();
    });
  });
});

describe("TaskDetailPage — save refetches before building the payload", () => {
  it("sends the freshly-fetched parentId, not the stale cached one", async () => {
    // Cached getTask result: top-level (no parentId). A fresh fetch reveals
    // the task was moved under parent 99 elsewhere while this page was open.
    mockGetTaskFn.mockResolvedValue({ task: { ...mockTask, parentId: 99n } });

    renderDetailPage();
    fireEvent.click(screen.getByRole("button", { name: /edit/i }));

    const nameInput = screen.getByLabelText(/title/i);
    fireEvent.change(nameInput, { target: { value: "Renamed" } });
    fireEvent.submit(nameInput.closest("form")!);

    await waitFor(() => {
      expect(mockUpdateTaskFn).toHaveBeenCalled();
    });
    expect(mockGetTaskFn).toHaveBeenCalledWith({ id: 1n });
    expect(mockUpdateTaskFn.mock.calls[0][0].parentId).toBe(99n);
  });

  it("surfaces a connectivity error and does not save if the refetch fails", async () => {
    mockGetTaskFn.mockRejectedValue(new Error("network down"));

    renderDetailPage();
    fireEvent.click(screen.getByRole("button", { name: /edit/i }));

    const nameInput = screen.getByLabelText(/title/i);
    fireEvent.change(nameInput, { target: { value: "Renamed" } });
    fireEvent.submit(nameInput.closest("form")!);

    await waitFor(() => {
      expect(screen.getByText("Couldn't reach the server. Want to try again?")).toBeInTheDocument();
    });
    expect(mockUpdateTaskFn).not.toHaveBeenCalled();
  });
});

describe("TaskDetailPage — FR-009: editing only due leaves other fields untouched", () => {
  it("sends unchanged name, description, snoozeUntil, parentId in the UpdateTask payload", async () => {
    const snoozeUntil = {
      seconds: BigInt(Date.UTC(2026, 6, 15) / 1000),
      nanos: 0,
      $typeName: "google.protobuf.Timestamp" as const,
    };
    getTaskResult = vi.fn().mockReturnValue({
      data: {
        task: {
          ...mockTask,
          description: "A description",
          parentId: 42n,
          snoozeUntil,
        },
      },
      isLoading: false,
      isError: false,
      error: null,
    });
    mockGetTaskFn.mockResolvedValue({
      task: {
        ...mockTask,
        description: "A description",
        parentId: 42n,
        snoozeUntil,
      },
    });

    renderDetailPage();
    fireEvent.click(screen.getByRole("button", { name: /edit/i }));

    const dueInput = screen.getByLabelText(/^due$/i);
    fireEvent.change(dueInput, { target: { value: "2026-09-01" } });
    fireEvent.submit(dueInput.closest("form")!);

    await waitFor(() => {
      expect(mockUpdateTaskFn).toHaveBeenCalled();
    });
    const payload = mockUpdateTaskFn.mock.calls[0][0];
    expect(payload.name).toBe(mockTask.name);
    expect(payload.description).toBe("A description");
    expect(payload.parentId).toBe(42n);
    expect(payload.snoozeUntil).toEqual(snoozeUntil);
    expect("goalId" in payload).toBe(false);
  });
});

describe("TaskDetailPage — day-unchanged edits preserve time-of-day", () => {
  it("does not truncate a CLI-set time-of-day when the calendar day is unchanged", async () => {
    const due = {
      seconds: BigInt(Date.UTC(2026, 7, 3, 17, 0, 0) / 1000),
      nanos: 0,
      $typeName: "google.protobuf.Timestamp" as const,
    };
    getTaskResult = vi.fn().mockReturnValue({
      data: { task: { ...mockTask, due } },
      isLoading: false,
      isError: false,
      error: null,
    });
    mockGetTaskFn.mockResolvedValue({ task: { ...mockTask, due } });

    renderDetailPage();
    fireEvent.click(screen.getByRole("button", { name: /edit/i }));

    // Only rename; the due date field is untouched.
    const nameInput = screen.getByLabelText(/title/i);
    fireEvent.change(nameInput, { target: { value: "Renamed" } });
    fireEvent.submit(nameInput.closest("form")!);

    await waitFor(() => {
      expect(mockUpdateTaskFn).toHaveBeenCalled();
    });
    expect(mockUpdateTaskFn.mock.calls[0][0].due).toEqual(due);
  });
});

describe("TaskDetailPage — failed save", () => {
  it("shows an error and keeps the edit form open when UpdateTask fails", async () => {
    mockUpdateTaskFn.mockRejectedValue(new ConnectError("boom", Code.Internal));
    renderDetailPage();
    fireEvent.click(screen.getByRole("button", { name: /edit/i }));

    const nameInput = screen.getByLabelText(/title/i);
    fireEvent.change(nameInput, { target: { value: "Renamed" } });
    fireEvent.submit(nameInput.closest("form")!);

    await waitFor(() => {
      expect(screen.getByText(messages.saveError)).toBeInTheDocument();
    });
    expect(screen.getByLabelText(/title/i)).toBeInTheDocument();
  });
});

describe("TaskDetailPage — CT-07: no plan control on edit form (FR-011)", () => {
  it("edit form does not render the Add to plan radiogroup", () => {
    renderDetailPage();
    fireEvent.click(screen.getByRole("button", { name: /edit/i }));
    expect(screen.queryByRole("radiogroup", { name: /add to plan/i })).not.toBeInTheDocument();
  });

  it("edit form does not render a date input for plan", () => {
    renderDetailPage();
    fireEvent.click(screen.getByRole("button", { name: /edit/i }));
    expect(screen.queryByLabelText(/pick a date/i)).not.toBeInTheDocument();
  });
});

describe("TaskDetailPage — due date display (US2)", () => {
  it("shows the no-due-date copy when due is unset", () => {
    renderDetailPage();
    expect(document.body.textContent).toContain(messages.noDueDate);
  });

  it("shows the formatted due date when set", () => {
    const due = { seconds: BigInt(Date.UTC(2026, 7, 2) / 1000), nanos: 0 };
    getTaskResult = vi.fn().mockReturnValue({
      data: { task: { ...mockTask, due } },
      isLoading: false,
      isError: false,
      error: null,
    });
    renderDetailPage();
    expect(document.body.textContent).toContain(formatDueDate(due as never));
  });
});

describe("TaskDetailPage — snooze display (US3)", () => {
  function tomorrowSeconds(): bigint {
    const d = new Date();
    d.setUTCDate(d.getUTCDate() + 1);
    return BigInt(Math.floor(Date.UTC(d.getUTCFullYear(), d.getUTCMonth(), d.getUTCDate()) / 1000));
  }

  function yesterdaySeconds(): bigint {
    const d = new Date();
    d.setUTCDate(d.getUTCDate() - 1);
    return BigInt(Math.floor(Date.UTC(d.getUTCFullYear(), d.getUTCMonth(), d.getUTCDate()) / 1000));
  }

  it("shows the Snoozed until line for a task snoozed to a future day", () => {
    getTaskResult = vi.fn().mockReturnValue({
      data: { task: { ...mockTask, snoozeUntil: { seconds: tomorrowSeconds(), nanos: 0 } } },
      isLoading: false,
      isError: false,
      error: null,
    });
    renderDetailPage();
    expect(document.body.textContent).toContain("Snoozed until");
  });

  it("shows no snooze marker when there is no snooze", () => {
    renderDetailPage();
    expect(document.body.textContent).not.toContain("Snoozed until");
  });

  it("shows no snooze marker when the snooze day is today or past", () => {
    getTaskResult = vi.fn().mockReturnValue({
      data: { task: { ...mockTask, snoozeUntil: { seconds: yesterdaySeconds(), nanos: 0 } } },
      isLoading: false,
      isError: false,
      error: null,
    });
    renderDetailPage();
    expect(document.body.textContent).not.toContain("Snoozed until");
  });
});

const mockGoal = { id: 5n, name: "Ship it", state: 1, $typeName: "goal.v1.Goal" };

describe("TaskDetailPage — goal section (US4)", () => {
  it("no goal shows a picker of non-hidden goals; selecting one links it", async () => {
    listGoalsResult = vi.fn().mockReturnValue({
      data: { goals: [mockGoal] },
      isLoading: false,
      isError: false,
      error: null,
    });
    renderDetailPage();
    const select = screen.getByRole("combobox", { name: /goal/i });
    fireEvent.change(select, { target: { value: "5" } });

    await waitFor(() => {
      expect(mockSetTaskGoalFn).toHaveBeenCalledWith({ taskId: 1n, goalId: 5n });
    });
    expect(mockInvalidateQueries).toHaveBeenCalled();
  });

  it("a direct goal shows the goal name linked, plus switch/unlink controls; unlink clears goalId", async () => {
    getTaskResult = vi.fn().mockReturnValue({
      data: { task: { ...mockTask, goalId: 5n } },
      isLoading: false,
      isError: false,
      error: null,
    });
    listTasksResult = vi.fn().mockReturnValue({
      data: { tasks: [{ ...mockTask, goalId: 5n }] },
      isLoading: false,
      isError: false,
      error: null,
    });
    listGoalsResult = vi.fn().mockReturnValue({
      data: { goals: [mockGoal] },
      isLoading: false,
      isError: false,
      error: null,
    });
    renderDetailPage();
    expect(screen.getByRole("link", { name: /ship it/i })).toHaveAttribute("href", "/goals/5");

    fireEvent.click(screen.getByRole("button", { name: /unlink/i }));
    await waitFor(() => {
      expect(mockSetTaskGoalFn).toHaveBeenCalledWith({ taskId: 1n });
    });
  });

  it("an inherited goal shows a read-only explanation with no controls", () => {
    listTasksResult = vi.fn().mockReturnValue({
      data: { tasks: [{ ...mockTask, id: 1n, goalId: undefined, parentId: 9n }, { ...mockTask, id: 9n, goalId: 5n }] },
      isLoading: false,
      isError: false,
      error: null,
    });
    listGoalsResult = vi.fn().mockReturnValue({
      data: { goals: [mockGoal] },
      isLoading: false,
      isError: false,
      error: null,
    });
    renderDetailPage();
    expect(document.body.textContent).toContain(messages.goalInheritedFrom("Ship it"));
    expect(screen.queryByRole("button", { name: /unlink/i })).not.toBeInTheDocument();
    expect(screen.queryByRole("combobox", { name: /goal/i })).not.toBeInTheDocument();
  });

  it("a task linked to an archived goal still shows it as the selected option", () => {
    const archivedGoal = { id: 7n, name: "Old goal", state: 4, $typeName: "goal.v1.Goal" };
    getTaskResult = vi.fn().mockReturnValue({
      data: { task: { ...mockTask, goalId: 7n } },
      isLoading: false,
      isError: false,
      error: null,
    });
    listTasksResult = vi.fn().mockReturnValue({
      data: { tasks: [{ ...mockTask, goalId: 7n }] },
      isLoading: false,
      isError: false,
      error: null,
    });
    listGoalsResult = vi.fn().mockReturnValue({
      data: { goals: [archivedGoal, mockGoal] },
      isLoading: false,
      isError: false,
      error: null,
    });
    renderDetailPage();
    const select = screen.getByRole("combobox", { name: /goal/i }) as HTMLSelectElement;
    expect(select.value).toBe("7");
    expect(screen.getByRole("link", { name: /old goal/i })).toHaveAttribute("href", "/goals/7");
  });

  it("waits for listGoals to load before rendering the goal card", () => {
    listGoalsResult = vi.fn().mockReturnValue({
      data: undefined,
      isLoading: true,
      isError: false,
      error: null,
    });
    renderDetailPage();
    expect(screen.queryByText(messages.noGoalLinked)).not.toBeInTheDocument();
    expect(document.body.textContent).not.toContain("Inherited from  —");
  });

  it("an empty listGoals result shows the friendly no-goals message", () => {
    listGoalsResult = vi.fn().mockReturnValue({
      data: { goals: [] },
      isLoading: false,
      isError: false,
      error: null,
    });
    renderDetailPage();
    expect(document.body.textContent).toContain(messages.noGoalsToPick);
    expect(screen.queryByRole("combobox", { name: /goal/i })).not.toBeInTheDocument();
  });
});

describe("TaskDetailPage — goal error mapping (US4)", () => {
  it("Code.FailedPrecondition shows the rule explanation and leaves the goal unchanged", async () => {
    mockSetTaskGoalFn.mockRejectedValue(new ConnectError("blocked", Code.FailedPrecondition));
    listGoalsResult = vi.fn().mockReturnValue({
      data: { goals: [mockGoal] },
      isLoading: false,
      isError: false,
      error: null,
    });
    renderDetailPage();
    const select = screen.getByRole("combobox", { name: /goal/i });
    fireEvent.change(select, { target: { value: "5" } });

    await waitFor(() => {
      expect(screen.getByText(messages.goalLinkBlocked)).toBeInTheDocument();
    });
    expect(document.body.textContent).toContain(messages.noGoalLinked);
  });

  it("Code.NotFound shows the gone message and refreshes", async () => {
    mockSetTaskGoalFn.mockRejectedValue(new ConnectError("gone", Code.NotFound));
    listGoalsResult = vi.fn().mockReturnValue({
      data: { goals: [mockGoal] },
      isLoading: false,
      isError: false,
      error: null,
    });
    renderDetailPage();
    const select = screen.getByRole("combobox", { name: /goal/i });
    fireEvent.change(select, { target: { value: "5" } });

    await waitFor(() => {
      expect(screen.getByText(messages.taskNotFound)).toBeInTheDocument();
    });
    expect(mockInvalidateQueries).toHaveBeenCalled();
  });

  it("other codes show the generic connectivity message", async () => {
    mockSetTaskGoalFn.mockRejectedValue(new ConnectError("boom", Code.Internal));
    listGoalsResult = vi.fn().mockReturnValue({
      data: { goals: [mockGoal] },
      isLoading: false,
      isError: false,
      error: null,
    });
    renderDetailPage();
    const select = screen.getByRole("combobox", { name: /goal/i });
    fireEvent.change(select, { target: { value: "5" } });

    await waitFor(() => {
      expect(screen.getByText(messages.connectivityError)).toBeInTheDocument();
    });
  });
});

describe("TaskDetailPage — delete flow (US1)", () => {
  it("clicking Delete shows the inline confirmation without calling DeleteTask", () => {
    renderDetailPage();
    fireEvent.click(screen.getByRole("button", { name: /^delete$/i }));
    expect(screen.getByText(messages.deleteTaskConfirm)).toBeInTheDocument();
    expect(mockDeleteTaskFn).not.toHaveBeenCalled();
  });

  it("confirmation copy mentions subtasks when the task has children", () => {
    listTasksResult = vi.fn().mockReturnValue({
      data: { tasks: [{ ...mockTask, id: 2n, parentId: 1n }] },
      isLoading: false,
      isError: false,
      error: null,
    });
    renderDetailPage();
    fireEvent.click(screen.getByRole("button", { name: /^delete$/i }));
    expect(screen.getByText(messages.deleteTaskConfirmWithSubtasks)).toBeInTheDocument();
  });

  it("Cancel dismisses the panel and leaves the task view unchanged", () => {
    renderDetailPage();
    fireEvent.click(screen.getByRole("button", { name: /^delete$/i }));
    expect(screen.getByText(messages.deleteTaskConfirm)).toBeInTheDocument();
    fireEvent.click(screen.getByRole("button", { name: /cancel/i }));
    expect(screen.queryByText(messages.deleteTaskConfirm)).not.toBeInTheDocument();
    expect(screen.getByText("Test Task")).toBeInTheDocument();
  });

  it("confirming calls DeleteTask, invalidates listTasks, and navigates to /tasks with a success toast", async () => {
    renderDetailPage();
    fireEvent.click(screen.getByRole("button", { name: /^delete$/i }));
    fireEvent.click(screen.getByRole("button", { name: /^confirm/i }));

    await waitFor(() => {
      expect(mockDeleteTaskFn).toHaveBeenCalledWith({ id: 1n });
    });
    await waitFor(() => {
      expect(screen.getByTestId("tasks-page")).toBeInTheDocument();
    });
    expect(mockInvalidateQueries).toHaveBeenCalled();
    expect(screen.getByText(messages.taskDeleted)).toBeInTheDocument();
  });

  it("a Code.NotFound failure shows the already-gone message and still navigates/refreshes", async () => {
    mockDeleteTaskFn.mockRejectedValue(new ConnectError("gone", Code.NotFound));
    renderDetailPage();
    fireEvent.click(screen.getByRole("button", { name: /^delete$/i }));
    fireEvent.click(screen.getByRole("button", { name: /^confirm/i }));

    await waitFor(() => {
      expect(screen.getByTestId("tasks-page")).toBeInTheDocument();
    });
    expect(mockInvalidateQueries).toHaveBeenCalled();
    expect(screen.getByText(messages.taskAlreadyGone)).toBeInTheDocument();
  });

  it("any other error shows the generic connectivity message and stays on the page", async () => {
    mockDeleteTaskFn.mockRejectedValue(new ConnectError("boom", Code.Internal));
    renderDetailPage();
    fireEvent.click(screen.getByRole("button", { name: /^delete$/i }));
    fireEvent.click(screen.getByRole("button", { name: /^confirm/i }));

    await waitFor(() => {
      expect(screen.getByText(messages.connectivityError)).toBeInTheDocument();
    });
    expect(screen.queryByTestId("tasks-page")).not.toBeInTheDocument();
  });
});
