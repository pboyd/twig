import { describe, it, expect, vi, beforeEach } from "vitest";
import { render, screen, fireEvent } from "@testing-library/react";
import { MemoryRouter, Routes, Route } from "react-router";
import { QueryClient, QueryClientProvider } from "@tanstack/react-query";
import GoalDetailPage from "./GoalDetailPage";
import { ConnectError, Code } from "@connectrpc/connect";

const { getGoalResult, listTasksResult, listStatusUpdatesResult } = vi.hoisted(() => ({
  getGoalResult: vi.fn(),
  listTasksResult: vi.fn(),
  listStatusUpdatesResult: vi.fn(),
}));

const { addMutationResult, updateMutationResult, deleteMutationResult } = vi.hoisted(() => ({
  addMutationResult: vi.fn(),
  updateMutationResult: vi.fn(),
  deleteMutationResult: vi.fn(),
}));

vi.mock("../gen/goal/v1/goal-GoalService_connectquery", () => ({
  listGoals: "schema:listGoals",
  getGoal: "schema:getGoal",
  listGoalStatusUpdates: "schema:listGoalStatusUpdates",
  addGoalStatusUpdate: "schema:addGoalStatusUpdate",
  updateGoalStatusUpdate: "schema:updateGoalStatusUpdate",
  deleteGoalStatusUpdate: "schema:deleteGoalStatusUpdate",
}));

vi.mock("../gen/task/v1/task-TaskService_connectquery", () => ({
  listTasks: "schema:listTasks",
}));

vi.mock("../components/AppHeader", () => ({
  AppHeader: () => <header data-testid="app-header" />,
}));

vi.mock("../context/ToastProvider", () => ({
  useToast: () => ({ show: vi.fn() }),
  ToastProvider: ({ children }: { children: React.ReactNode }) => children,
}));

vi.mock("@connectrpc/connect-query", () => ({
  useQuery: vi.fn((schema: unknown) => {
    if (schema === "schema:getGoal") return getGoalResult();
    if (schema === "schema:listTasks") return listTasksResult();
    if (schema === "schema:listGoalStatusUpdates") return listStatusUpdatesResult();
    return { data: undefined, isLoading: false, isError: false, error: null };
  }),
  useMutation: vi.fn((schema: unknown) => {
    if (schema === "schema:addGoalStatusUpdate") return { mutateAsync: addMutationResult, isPending: false };
    if (schema === "schema:updateGoalStatusUpdate") return { mutateAsync: updateMutationResult, isPending: false };
    if (schema === "schema:deleteGoalStatusUpdate") return { mutateAsync: deleteMutationResult, isPending: false };
    return { mutateAsync: vi.fn(), isPending: false };
  }),
  createConnectQueryKey: vi.fn(() => ["mock-key"]),
}));

function makeGoal(overrides?: Record<string, unknown>) {
  return { id: 1n, name: "My Goal", description: "", state: 2, position: 0n, $typeName: "goal.v1.Goal", ...overrides };
}

function makeUpdate(id: bigint, body: string) {
  return { id, goalId: 1n, body, createdAt: { seconds: 1719000000n, nanos: 0, $typeName: "google.protobuf.Timestamp" as const }, $typeName: "goal.v1.StatusUpdate" as const };
}

function renderPage(path = "/goals/1") {
  const queryClient = new QueryClient({ defaultOptions: { queries: { retry: false } } });
  return render(
    <QueryClientProvider client={queryClient}>
      <MemoryRouter initialEntries={[path]}>
        <Routes>
          <Route path="/goals/:id" element={<GoalDetailPage />} />
        </Routes>
      </MemoryRouter>
    </QueryClientProvider>
  );
}

beforeEach(() => {
  vi.clearAllMocks();
  getGoalResult.mockReturnValue({
    data: { goal: makeGoal() },
    isLoading: false,
    isError: false,
    error: null,
  });
  listTasksResult.mockReturnValue({
    data: { tasks: [] },
    isLoading: false,
    isError: false,
    error: null,
  });
  listStatusUpdatesResult.mockReturnValue({
    data: { updates: [] },
    isLoading: false,
    isError: false,
    error: null,
  });
  addMutationResult.mockResolvedValue({});
  updateMutationResult.mockResolvedValue({});
  deleteMutationResult.mockResolvedValue({});
});

describe("GoalDetailPage — loading state", () => {
  it("shows a spinner while loading", () => {
    getGoalResult.mockReturnValue({
      data: undefined,
      isLoading: true,
      isError: false,
      error: null,
    });
    renderPage();
    expect(document.querySelector("svg.animate-spin")).toBeTruthy();
  });
});

describe("GoalDetailPage — error state", () => {
  it("shows error banner on failure", () => {
    getGoalResult.mockReturnValue({
      data: undefined,
      isLoading: false,
      isError: true,
      error: new Error("fail"),
    });
    renderPage();
    expect(screen.getByText(/Couldn't load this goal/i)).toBeTruthy();
  });
});

describe("GoalDetailPage — not found", () => {
  it("shows a not-found message", () => {
    getGoalResult.mockReturnValue({
      data: undefined,
      isLoading: false,
      isError: true,
      error: new ConnectError("not found", Code.NotFound),
    });
    renderPage();
    expect(screen.getByText(/wandered off/i)).toBeTruthy();
  });
});

describe("GoalDetailPage — goal display", () => {
  it("renders the goal name", () => {
    renderPage();
    expect(screen.getByText("My Goal")).toBeTruthy();
  });

  it("renders the description as markdown", () => {
    getGoalResult.mockReturnValue({
      data: { goal: makeGoal({ description: "**bold** description" }) },
      isLoading: false,
      isError: false,
      error: null,
    });
    renderPage();
    expect(screen.getByText("bold")).toBeTruthy();
  });

  it("shows state label", () => {
    renderPage();
    expect(screen.getByText("In Progress")).toBeTruthy();
  });

  it("shows due date when present", () => {
    getGoalResult.mockReturnValue({
      data: { goal: makeGoal({ due: { seconds: 1719000000n, nanos: 0, $typeName: "google.protobuf.Timestamp" as const } }) },
      isLoading: false,
      isError: false,
      error: null,
    });
    renderPage();
    expect(screen.getByText(/Due/)).toBeTruthy();
  });
});

describe("GoalDetailPage — associated tasks", () => {
  it("shows associated tasks section", () => {
    listTasksResult.mockReturnValue({
      data: {
        tasks: [
          { id: 10n, name: "Sub task", goalId: 1n, parentId: undefined, $typeName: "task.v1.Task" },
        ],
      },
      isLoading: false,
      isError: false,
      error: null,
    });
    renderPage();
    expect(screen.getByText("Sub task")).toBeTruthy();
    expect(screen.getByText("Associated tasks")).toBeTruthy();
  });

  it("hides section when no associated tasks", () => {
    renderPage();
    expect(screen.queryByText("Associated tasks")).toBeNull();
  });
});

describe("GoalDetailPage — status updates", () => {
  it("shows no status updates message when empty", () => {
    renderPage();
    expect(screen.getByText(/No status updates yet/i)).toBeTruthy();
  });

  it("shows the latest status update", () => {
    listStatusUpdatesResult.mockReturnValue({
      data: { updates: [makeUpdate(1n, "Latest progress")] },
      isLoading: false,
      isError: false,
      error: null,
    });
    renderPage();
    expect(screen.getAllByText("Latest progress")).toHaveLength(2);
  });

  it("shows Add button", () => {
    renderPage();
    expect(screen.getByText("Add status update")).toBeTruthy();
  });

  it("shows add form when Add button is clicked", () => {
    renderPage();
    fireEvent.click(screen.getByText("Add status update"));
    expect(screen.getByText("Save")).toBeTruthy();
  });

  it("submits a new status update on save", async () => {
    renderPage();
    fireEvent.click(screen.getByText("Add status update"));
    const textarea = screen.getByPlaceholderText("What's the latest?");
    fireEvent.change(textarea, { target: { value: "Making great progress" } });
    fireEvent.click(screen.getByText("Save"));
    expect(addMutationResult).toHaveBeenCalledWith({ goalId: 1n, body: "Making great progress" });
    await vi.waitFor(() => expect(screen.getByText("Add status update")).toBeTruthy());
  });

  it("shows error banner when add mutation fails", async () => {
    addMutationResult.mockRejectedValue(new Error("fail"));
    renderPage();
    fireEvent.click(screen.getByText("Add status update"));
    const textarea = screen.getByPlaceholderText("What's the latest?");
    fireEvent.change(textarea, { target: { value: "update" } });
    fireEvent.click(screen.getByText("Save"));
    expect(await screen.findByText(/Couldn't save that update/i)).toBeTruthy();
  });

  it("saves an edited status update", async () => {
    listStatusUpdatesResult.mockReturnValue({
      data: { updates: [makeUpdate(1n, "Original")] },
      isLoading: false,
      isError: false,
      error: null,
    });
    renderPage();
    fireEvent.click(screen.getAllByText("Edit")[0]);
    const textarea = screen.getByPlaceholderText("What's the latest?");
    fireEvent.change(textarea, { target: { value: "Edited update" } });
    fireEvent.click(screen.getByText("Update"));
    expect(updateMutationResult).toHaveBeenCalledWith({ id: 1n, body: "Edited update" });
  });

  it("does not delete when confirm is cancelled", () => {
    vi.spyOn(window, "confirm").mockReturnValue(false);
    listStatusUpdatesResult.mockReturnValue({
      data: { updates: [makeUpdate(1n, "To delete")] },
      isLoading: false,
      isError: false,
      error: null,
    });
    renderPage();
    fireEvent.click(screen.getByText("Delete"));
    expect(deleteMutationResult).not.toHaveBeenCalled();
  });

  it("deletes when confirm is accepted", () => {
    vi.spyOn(window, "confirm").mockReturnValue(true);
    listStatusUpdatesResult.mockReturnValue({
      data: { updates: [makeUpdate(1n, "To delete")] },
      isLoading: false,
      isError: false,
      error: null,
    });
    renderPage();
    fireEvent.click(screen.getByText("Delete"));
    expect(deleteMutationResult).toHaveBeenCalledWith({ id: 1n });
  });

  it("shows error banner when status updates fail to load", () => {
    listStatusUpdatesResult.mockReturnValue({
      data: undefined,
      isLoading: false,
      isError: true,
      error: new Error("fail"),
    });
    renderPage();
    expect(screen.getByText(/Couldn't load the status updates/i)).toBeTruthy();
    expect(screen.queryByText(/No status updates yet/i)).toBeNull();
  });
});
