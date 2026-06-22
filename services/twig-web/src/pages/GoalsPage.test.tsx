import { describe, it, expect, vi, beforeEach, afterEach } from "vitest";
import { render, screen, fireEvent } from "@testing-library/react";
import { MemoryRouter } from "react-router";
import { QueryClient, QueryClientProvider } from "@tanstack/react-query";
import GoalsPage from "./GoalsPage";

const { listGoalsResult } = vi.hoisted(() => ({
  listGoalsResult: vi.fn(),
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

vi.mock("@connectrpc/connect-query", () => ({
  useQuery: vi.fn(() => listGoalsResult()),
  useMutation: vi.fn(() => ({ mutateAsync: vi.fn(), isPending: false })),
  createConnectQueryKey: vi.fn(() => ["mock-key"]),
}));

function makeGoal(id: bigint, state: number, name?: string) {
  return { id, name: name ?? `Goal ${id}`, description: "", state, position: 0n, $typeName: "goal.v1.Goal" };
}

function renderPage() {
  const queryClient = new QueryClient({ defaultOptions: { queries: { retry: false } } });
  return render(
    <QueryClientProvider client={queryClient}>
      <MemoryRouter>
        <GoalsPage />
      </MemoryRouter>
    </QueryClientProvider>
  );
}

beforeEach(() => {
  vi.clearAllMocks();
  localStorage.clear();
  listGoalsResult.mockReturnValue({
    data: { goals: [] },
    isLoading: false,
    isError: false,
    error: null,
  });
});

afterEach(() => {
  localStorage.clear();
});

const GOAL_STATE_COMMITTED = 2;
const GOAL_STATE_INCUBATING = 1;
const GOAL_STATE_COMPLETED = 3;
const GOAL_STATE_ARCHIVED = 4;

describe("GoalsPage — grouping", () => {
  it("shows Committed then Incubating by default", () => {
    listGoalsResult.mockReturnValue({
      data: {
        goals: [
          makeGoal(1n, GOAL_STATE_INCUBATING, "Incubating goal"),
          makeGoal(2n, GOAL_STATE_COMMITTED, "Committed goal"),
        ],
      },
      isLoading: false,
      isError: false,
      error: null,
    });
    renderPage();
    expect(screen.getByText("Committed goal")).toBeTruthy();
    expect(screen.getByText("Incubating goal")).toBeTruthy();
  });

  it("hides Completed and Archived by default", () => {
    listGoalsResult.mockReturnValue({
      data: {
        goals: [
          makeGoal(1n, GOAL_STATE_COMMITTED),
          makeGoal(2n, GOAL_STATE_COMPLETED),
          makeGoal(3n, GOAL_STATE_ARCHIVED),
        ],
      },
      isLoading: false,
      isError: false,
      error: null,
    });
    renderPage();
    expect(screen.getByText("Goal 1")).toBeTruthy();
    expect(screen.queryByText("Goal 2")).toBeNull();
    expect(screen.queryByText("Goal 3")).toBeNull();
  });
});

describe("GoalsPage — toggle hidden", () => {
  it("reveals Completed and Archived when Show all is clicked", () => {
    listGoalsResult.mockReturnValue({
      data: {
        goals: [
          makeGoal(1n, GOAL_STATE_COMMITTED),
          makeGoal(2n, GOAL_STATE_COMPLETED),
        ],
      },
      isLoading: false,
      isError: false,
      error: null,
    });
    renderPage();
    fireEvent.click(screen.getByRole("button", { name: /show all/i }));
    expect(screen.getByText("Goal 2")).toBeTruthy();
  });
});

describe("GoalsPage — loading state", () => {
  it("shows a spinner while loading", () => {
    listGoalsResult.mockReturnValue({
      data: undefined,
      isLoading: true,
      isError: false,
      error: null,
    });
    renderPage();
    expect(document.querySelector("svg.animate-spin")).toBeTruthy();
  });
});

describe("GoalsPage — error state", () => {
  it("shows an error banner on failure", () => {
    listGoalsResult.mockReturnValue({
      data: undefined,
      isLoading: false,
      isError: true,
      error: new Error("Network failure"),
    });
    renderPage();
    expect(screen.getByText(/Couldn't load your goals/i)).toBeTruthy();
  });
});

describe("GoalsPage — empty state", () => {
  it("shows empty state when no goals", () => {
    listGoalsResult.mockReturnValue({
      data: { goals: [] },
      isLoading: false,
      isError: false,
      error: null,
    });
    renderPage();
    expect(screen.getByText(/No goals yet/i)).toBeTruthy();
  });
});
