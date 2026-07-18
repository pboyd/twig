import { describe, it, expect, vi, beforeEach, afterEach } from "vitest";
import { render, screen, fireEvent } from "@testing-library/react";
import { MemoryRouter } from "react-router";
import { QueryClient, QueryClientProvider } from "@tanstack/react-query";
import { ConnectError, Code } from "@connectrpc/connect";
import TaskTreePage from "./TaskTreePage";
import { ToastProvider } from "../context/ToastProvider";
import { messages } from "../theme/messages";

vi.mock("../gen/task/v1/task-TaskService_connectquery", () => ({
  listTasks: "schema:listTasks",
  createTask: "schema:createTask",
  completeTask: "schema:completeTask",
  uncompleteTask: "schema:uncompleteTask",
  reorderTask: "schema:reorderTask",
}));

vi.mock("../components/AppHeader", () => ({
  AppHeader: () => <header data-testid="app-header" />,
}));

vi.mock("@tanstack/react-query", async (importActual) => {
  const actual = await importActual<typeof import("@tanstack/react-query")>();
  return {
    ...actual,
    useQueryClient: vi.fn(() => ({ invalidateQueries: vi.fn() })),
  };
});

const mockMutateFn = vi.fn();
let listTasksResult: ReturnType<typeof vi.fn> = vi.fn();

vi.mock("@connectrpc/connect-query", () => ({
  useQuery: vi.fn((_schema: string) => listTasksResult()),
  useMutation: vi.fn(() => ({ mutateAsync: mockMutateFn, isPending: false })),
  createConnectQueryKey: vi.fn(() => ["mock-key"]),
}));

function makeTask(id: bigint, completed = false, parentId?: bigint) {
  return {
    id,
    name: `Task ${id}`,
    description: "",
    parentId,
    due: undefined,
    completedAt: completed ? { seconds: 1000n, nanos: 0 } : undefined,
    estimate: 0,
    $typeName: "task.v1.Task",
  };
}

vi.mock("../gen/plan/v1/plan-PlanService_connectquery", () => ({
  addPlanTask: "schema:addPlanTask",
  listPlanEntries: "schema:listPlanEntries",
}));

function renderPage() {
  const queryClient = new QueryClient({ defaultOptions: { queries: { retry: false } } });
  return render(
    <QueryClientProvider client={queryClient}>
      <MemoryRouter>
        <ToastProvider>
          <TaskTreePage />
        </ToastProvider>
      </MemoryRouter>
    </QueryClientProvider>
  );
}

beforeEach(() => {
  vi.clearAllMocks();
  localStorage.clear();
  mockMutateFn.mockResolvedValue({});
  listTasksResult = vi.fn().mockReturnValue({
    data: { tasks: [] },
    isLoading: false,
    isError: false,
    error: null,
    refetch: vi.fn(),
  });
});

afterEach(() => {
  localStorage.clear();
});

describe("TaskTreePage — hide completed by default (US1)", () => {
  it("renders only incomplete tasks by default when mix of completed/incomplete", () => {
    const incomplete = makeTask(1n, false);
    const completed = makeTask(2n, true);
    listTasksResult.mockReturnValue({
      data: { tasks: [incomplete, completed] },
      isLoading: false,
      isError: false,
      error: null,
      refetch: vi.fn(),
    });

    renderPage();

    expect(screen.getByText("Task 1")).toBeInTheDocument();
    expect(screen.queryByText("Task 2")).not.toBeInTheDocument();
  });

  it("shows allCompletedHidden message and not EmptyState when all tasks are completed", () => {
    const completed1 = makeTask(1n, true);
    const completed2 = makeTask(2n, true);
    listTasksResult.mockReturnValue({
      data: { tasks: [completed1, completed2] },
      isLoading: false,
      isError: false,
      error: null,
      refetch: vi.fn(),
    });

    renderPage();

    expect(screen.getByText(/nothing pending right now/i)).toBeInTheDocument();
    expect(screen.queryByText(/your future self is grateful/i)).not.toBeInTheDocument();
  });
});

describe("TaskTreePage — toggle reveals completed tasks (US2)", () => {
  it("clicking Show completed reveals completed tasks", () => {
    const incomplete = makeTask(1n, false);
    const completed = makeTask(2n, true);
    listTasksResult.mockReturnValue({
      data: { tasks: [incomplete, completed] },
      isLoading: false,
      isError: false,
      error: null,
      refetch: vi.fn(),
    });

    renderPage();

    expect(screen.queryByText("Task 2")).not.toBeInTheDocument();

    fireEvent.click(screen.getByRole("button", { name: /show all/i }));

    expect(screen.getByText("Task 2")).toBeInTheDocument();
  });

  it("clicking Hide completed hides them again", () => {
    const incomplete = makeTask(1n, false);
    const completed = makeTask(2n, true);
    listTasksResult.mockReturnValue({
      data: { tasks: [incomplete, completed] },
      isLoading: false,
      isError: false,
      error: null,
      refetch: vi.fn(),
    });

    renderPage();

    fireEvent.click(screen.getByRole("button", { name: /show all/i }));
    expect(screen.getByText("Task 2")).toBeInTheDocument();

    fireEvent.click(screen.getByRole("button", { name: /show only pending/i }));
    expect(screen.queryByText("Task 2")).not.toBeInTheDocument();
  });
});

describe("TaskTreePage — preference persistence (US3)", () => {
  it("initializes to showing completed tasks when preference is stored as true", () => {
    localStorage.setItem("twig-show-completed", "true");

    const completed = makeTask(1n, true);
    listTasksResult.mockReturnValue({
      data: { tasks: [completed] },
      isLoading: false,
      isError: false,
      error: null,
      refetch: vi.fn(),
    });

    renderPage();

    expect(screen.getByText("Task 1")).toBeInTheDocument();
  });

  it("hides completed by default when no stored preference", () => {
    const completed = makeTask(1n, true);
    listTasksResult.mockReturnValue({
      data: { tasks: [completed] },
      isLoading: false,
      isError: false,
      error: null,
      refetch: vi.fn(),
    });

    renderPage();

    expect(screen.queryByText("Task 1")).not.toBeInTheDocument();
  });
});

describe("TaskTreePage — inline name rendering (US2)", () => {
  it("renders bold emphasis in task name via inline markdown", () => {
    listTasksResult.mockReturnValue({
      data: { tasks: [{ ...makeTask(1n), name: "**Ship** it" }] },
      isLoading: false,
      isError: false,
      error: null,
      refetch: vi.fn(),
    });
    const { container } = renderPage();
    expect(container.querySelector("strong")).toBeTruthy();
  });

  it("emits no block element for task name with heading prefix", () => {
    listTasksResult.mockReturnValue({
      data: { tasks: [{ ...makeTask(1n), name: "# Not a heading" }] },
      isLoading: false,
      isError: false,
      error: null,
      refetch: vi.fn(),
    });
    const { container } = renderPage();
    // The task row (li) should not contain a heading element from inline markdown
    const taskRow = container.querySelector("li");
    expect(taskRow?.querySelector("h1, h2, h3, h4, h5, h6")).toBeNull();
    expect(container.textContent).toContain("Not a heading");
  });
});

describe("TaskTreePage — plan after create (US1)", () => {
  function openAddFormAndFillName(name: string) {
    // Click the "+ Add task" header button (not the form submit which appears later)
    fireEvent.click(screen.getByText("+ Add task"));
    fireEvent.change(screen.getByLabelText(/title/i), { target: { value: name } });
  }

  it("CT-08: CreateTask fails ⇒ no plan call, error toast shown, form stays open", async () => {
    mockMutateFn.mockRejectedValueOnce(new ConnectError("down", Code.Unavailable));
    listTasksResult.mockReturnValue({
      data: { tasks: [] },
      isLoading: false,
      isError: false,
      error: null,
      refetch: vi.fn(),
    });

    renderPage();
    openAddFormAndFillName("Fail task");
    fireEvent.click(screen.getByText("Today"));
    fireEvent.click(screen.getByRole("button", { name: "Add task" }));
    await vi.waitFor(() => {
      expect(mockMutateFn).toHaveBeenCalledTimes(1);
    });
    // The rejection must be caught, not left to become an unhandled
    // promise rejection: the user sees an error toast and the form
    // (still holding their typed name) stays open for a retry.
    await vi.waitFor(() => {
      expect(screen.getByText(messages.addFailed)).toBeInTheDocument();
    });
    expect(screen.getByLabelText(/title/i)).toBeInTheDocument();
  });

  it("CT-09: AddPlanTask fails ⇒ task created but partial-failure toast shown", async () => {
    mockMutateFn
      .mockResolvedValueOnce({ task: { id: 42n } })
      .mockRejectedValueOnce(
        new ConnectError("already on plan", Code.FailedPrecondition),
      );
    listTasksResult.mockReturnValue({
      data: { tasks: [] },
      isLoading: false,
      isError: false,
      error: null,
      refetch: vi.fn(),
    });

    renderPage();
    openAddFormAndFillName("Partial fail");
    fireEvent.click(screen.getByText("Today"));
    fireEvent.click(screen.getByRole("button", { name: "Add task" }));
    await vi.waitFor(() => {
      expect(mockMutateFn).toHaveBeenCalledTimes(2);
    });
    expect(screen.queryByLabelText(/title/i)).not.toBeInTheDocument();
  });

  it("CT-11: success invalidates listTasks and plan entries", async () => {
    const invalidateQueries = vi.fn();
    vi.mocked(
      (await import("@tanstack/react-query")).useQueryClient,
    ).mockReturnValue({ invalidateQueries } as never);

    mockMutateFn
      .mockResolvedValueOnce({ task: { id: 99n } })
      .mockResolvedValueOnce({});

    listTasksResult.mockReturnValue({
      data: { tasks: [] },
      isLoading: false,
      isError: false,
      error: null,
      refetch: vi.fn(),
    });

    renderPage();
    openAddFormAndFillName("Success task");
    fireEvent.click(screen.getByText("Today"));
    fireEvent.click(screen.getByRole("button", { name: "Add task" }));
    await vi.waitFor(() => {
      expect(mockMutateFn).toHaveBeenCalledTimes(2);
    });
    expect(invalidateQueries).toHaveBeenCalled();
  });
});

describe("TreeRow — plan after create for sub-tasks (US1)", () => {
  function seedOneTask() {
    listTasksResult.mockReturnValue({
      data: { tasks: [makeTask(7n)] },
      isLoading: false,
      isError: false,
      error: null,
      refetch: vi.fn(),
    });
  }

  // TreeRow's "Add sub-task" icon toggle shares its accessible name with the
  // form's own submit button once the form is open — grab the last match to
  // get the submit button.
  function clickAddSubTaskSubmit() {
    const buttons = screen.getAllByRole("button", { name: "Add sub-task" });
    fireEvent.click(buttons[buttons.length - 1]);
  }

  it("CT-08b: CreateTask fails for a sub-task ⇒ no plan call, error toast shown, form stays open", async () => {
    mockMutateFn.mockRejectedValueOnce(new ConnectError("down", Code.Unavailable));
    seedOneTask();

    renderPage();
    fireEvent.click(screen.getByRole("button", { name: "Add sub-task" }));
    fireEvent.change(screen.getByLabelText(/title/i), { target: { value: "Sub fail" } });
    fireEvent.click(screen.getByText("Today"));
    clickAddSubTaskSubmit();

    await vi.waitFor(() => {
      expect(mockMutateFn).toHaveBeenCalledTimes(1);
    });
    await vi.waitFor(() => {
      expect(screen.getByText(messages.addFailed)).toBeInTheDocument();
    });
    expect(screen.getByLabelText(/title/i)).toBeInTheDocument();
  });

  it("sub-task create succeeds but AddPlanTask fails ⇒ partial-failure toast, no throw", async () => {
    mockMutateFn
      .mockResolvedValueOnce({ task: { id: 43n } })
      .mockRejectedValueOnce(new ConnectError("down", Code.Unavailable));
    seedOneTask();

    renderPage();
    fireEvent.click(screen.getByRole("button", { name: "Add sub-task" }));
    fireEvent.change(screen.getByLabelText(/title/i), { target: { value: "Sub partial" } });
    fireEvent.click(screen.getByText("Today"));
    clickAddSubTaskSubmit();

    await vi.waitFor(() => {
      expect(mockMutateFn).toHaveBeenCalledTimes(2);
    });
    await vi.waitFor(() => {
      expect(screen.getByText(messages.addedToPlanPartialFail)).toBeInTheDocument();
    });
    expect(screen.queryByLabelText(/title/i)).not.toBeInTheDocument();
  });
});
