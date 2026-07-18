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
