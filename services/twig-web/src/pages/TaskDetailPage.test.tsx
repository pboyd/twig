import { describe, it, expect, vi, beforeEach } from "vitest";
import { render, screen, fireEvent, waitFor } from "@testing-library/react";
import { MemoryRouter, Route, Routes } from "react-router";
import { QueryClient, QueryClientProvider } from "@tanstack/react-query";
import TaskDetailPage from "./TaskDetailPage";
import { ConnectError, Code } from "@connectrpc/connect";

// Stable mock schema tokens (used to differentiate useQuery/useMutation calls)
vi.mock("../gen/task/v1/task-TaskService_connectquery", () => ({
  getTask: "schema:getTask",
  listTasks: "schema:listTasks",
  updateTask: "schema:updateTask",
  completeTask: "schema:completeTask",
  uncompleteTask: "schema:uncompleteTask",
}));

vi.mock("../components/AppHeader", () => ({
  AppHeader: () => <header data-testid="app-header" />,
}));

const mockCompleteTaskFn = vi.fn();
const mockUncompleteTaskFn = vi.fn();
const mockUpdateTaskFn = vi.fn();
const mockInvalidateQueries = vi.fn();

vi.mock("@tanstack/react-query", async (importActual) => {
  const actual = await importActual<typeof import("@tanstack/react-query")>();
  return {
    ...actual,
    useQueryClient: vi.fn(() => ({
      invalidateQueries: mockInvalidateQueries,
    })),
  };
});

// Per-test configurable state
let getTaskResult: ReturnType<typeof vi.fn> = vi.fn();
let listTasksResult: ReturnType<typeof vi.fn> = vi.fn();

vi.mock("@connectrpc/connect-query", () => ({
  useQuery: vi.fn((schema: string) => {
    if (schema === "schema:getTask") return getTaskResult();
    return listTasksResult();
  }),
  useMutation: vi.fn((schema: string) => {
    if (schema === "schema:completeTask") {
      return { mutateAsync: mockCompleteTaskFn, isPending: false };
    }
    if (schema === "schema:uncompleteTask") {
      return { mutateAsync: mockUncompleteTaskFn, isPending: false };
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
      <MemoryRouter initialEntries={["/tasks/1"]}>
        <Routes>
          <Route path="/tasks/:id" element={<TaskDetailPage />} />
          <Route path="/tasks" element={<div data-testid="tasks-page" />} />
          <Route path="/login" element={<div data-testid="login-page" />} />
        </Routes>
      </MemoryRouter>
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
  mockUpdateTaskFn.mockResolvedValue({ task: mockTask });
  mockCompleteTaskFn.mockResolvedValue({ task: { ...mockTask, completedAt: { seconds: 1n, nanos: 0 } } });
  mockUncompleteTaskFn.mockResolvedValue({ task: mockTask });
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
