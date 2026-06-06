import { describe, it, expect, vi, beforeEach, afterEach } from "vitest";
import { render, screen, fireEvent } from "@testing-library/react";
import { MemoryRouter } from "react-router";
import { QueryClient, QueryClientProvider } from "@tanstack/react-query";
import TaskTreePage from "./TaskTreePage";

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

function renderPage() {
  const queryClient = new QueryClient({ defaultOptions: { queries: { retry: false } } });
  return render(
    <QueryClientProvider client={queryClient}>
      <MemoryRouter>
        <TaskTreePage />
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
