# Phase 0 Research: Web Day Planner (View)

This feature surfaced no `NEEDS CLARIFICATION` items in Technical Context — the stack is fixed by the existing `services/twig-web` app and the three spec clarifications (date navigation, wall-clock display, non-interactive events) are already resolved. Research below records the key design decisions and the codebase facts they rest on.

## Decision 1: Reuse the existing `plan.v1.PlanService.ListPlanEntries` RPC — no backend changes

- **Decision**: Fetch the day's plan with the already-generated `listPlanEntries` connect-query stub (`src/gen/plan/v1/plan-PlanService_connectquery.ts`), passing `{ day: "YYYY-MM-DD" }`.
- **Rationale**:
  - The server already registers `planv1connect.NewPlanServiceHandler(&handler.Plan{...})` on the shared mux, wrapped by `auth.Middleware` (same as `TaskService`). Verified in `services/twig/cmd/server/main.go`.
  - The TypeScript stubs and message types (`PlanEntry`, `ListPlanEntriesRequest/Response`) already exist in `src/gen/plan/v1/` — `npm run gen` already pulls `api/proto/plan/v1/plan.proto`. No `make proto` / `npm run gen` change needed.
  - `ListPlanEntries` returns entries "ordered by start_minute ascending" with `completed` populated server-side — exactly the read shape this view needs.
  - Honors the hard constraint "the backend is not modified" (CLAUDE.md) and Principle II (contract already defined).
- **Alternatives considered**:
  - *Add a new aggregated "plan view" endpoint* — rejected: unnecessary backend change; `ListPlanEntries` already provides everything.
  - *Derive the plan client-side from tasks* — rejected: the plan is its own server-side concept; there is no client-derivable substitute.

## Decision 2: Wall-clock time rendering from `start_minute` (no timezone conversion)

- **Decision**: `PlanEntry.start_minute` is "minutes since local midnight on `day`". Render it directly as `HH:MM` (and end = `start_minute + duration_minute`) without constructing a `Date`/UTC value.
- **Rationale**: Per spec clarification (FR-020), times must display as the planned wall-clock time regardless of the viewing device's timezone. Because `start_minute` is already a wall-clock minute offset, simple integer→`HH:MM` formatting is both the simplest and the correct implementation — using `Date` would risk an unwanted timezone shift.
- **Alternatives considered**:
  - *Build a `Date` from day + minute and format with `toLocaleTimeString`* — rejected: introduces timezone-conversion risk and complexity for no benefit.

## Decision 3: 12-hour vs 24-hour display

- **Decision**: Display times in 12-hour format with am/pm (e.g. `9:00 am`, `1:30 pm`), matching the TUI/CLI plan presentation seen in spec 006 examples (`9:00am`, `12:00pm`).
- **Rationale**: Consistency with the established plan presentation across surfaces (Principle III). The helper centralizes formatting so this is a one-line policy.
- **Alternatives considered**: 24-hour — rejected for cross-surface consistency; not requested.

## Decision 4: Entry name fallback uses the existing `ListTasks` query

- **Decision**: When `PlanEntry.name` is empty and `task_id != 0`, fall back to the linked task's name. Resolve names by also reading the shared `listTasks` query and building an `id → name` map.
- **Rationale**: The proto explicitly states clients must fall back to the linked task's name when `name` is empty. `ListTasks` is already used elsewhere and cached by React Query, so reading it adds no meaningful cost. A final fallback label (e.g. "Untitled entry") covers the rare case of an empty name with no resolvable task (FR-019).
- **Alternatives considered**:
  - *Per-entry `GetTask` calls* — rejected: N round-trips vs one cached list.
  - *Assume server fills `name`* — rejected: contract says otherwise.

## Decision 5: Timed vs untimed grouping and gap visibility

- **Decision**: Split entries into **timed** (`start_minute` present) and **untimed** (`start_minute` absent — task entries only). Render timed entries as a chronological list showing each entry's time span; render untimed entries in a clearly separated section below. Make gaps apparent by showing each entry's own start/end (a gap is visible as a jump between one entry's end and the next entry's start), rather than reproducing the TUI's fixed hour grid.
- **Rationale**: Per Assumptions, a mobile-friendly list/timeline is preferred over the ASCII hour grid. Showing explicit start–end per entry satisfies FR-005 and FR-010 simply and reads well on a narrow screen.
- **Alternatives considered**:
  - *Replicate the fixed hour-by-hour grid* — rejected: poor fit for small screens; more complex.
  - *Insert explicit "free time" gap rows* — deferred as a possible enhancement; not required by the spec.

## Decision 6: Navigation between Tasks and Plan

- **Decision**: Add a new route `/plan` and surface navigation in the shared `AppHeader` (e.g. Tasks / Plan links), keeping the task tree (`/tasks`) as the default landing route. Task-linked plan entries navigate to `/tasks/:taskId` (the existing `TaskDetailPage`).
- **Rationale**: The web app's primary purpose remains task capture, so tasks stays the landing page (FR-002/FR-003: planner is *reachable from nav* and *defaults to today* when opened). Reusing `AppHeader` keeps navigation consistent (Principle III) and gives a natural back-path from a task opened via the plan.
- **Alternatives considered**:
  - *Make the plan the landing page* — rejected: contradicts the stated primary use case (task capture).
  - *Standalone nav bar component* — deferred: `AppHeader` already spans pages; extend it rather than add a parallel surface.

## Decision 7: Date stepping model

- **Decision**: Track the viewed day as a `YYYY-MM-DD` string in page state, defaulting to today (local). Provide previous/next-day controls (±1 day) and a "Today" control. No arbitrary date picker.
- **Rationale**: Matches spec clarification / FR-018. Pure string-based date math (parse → add days → reformat) is small and unit-testable in `planView.ts`, avoiding timezone pitfalls of `Date` arithmetic when all we need is a calendar-day step.
- **Alternatives considered**: Native `<input type="date">` picker — rejected per clarification (out of scope) and to keep the companion view focused.

## Decision 8: Empty / loading / error states

- **Decision**: Reuse `Spinner` (loading) and `ErrorBanner` with retry (load failure, FR-017). For an empty day, show a playful inline empty state naming the date (not the task-specific `EmptyState` component, which offers "add a task"). Add `planEmpty`/`planError` strings to `theme/messages.ts`.
- **Rationale**: Read-only view shouldn't invite creation, so the existing `EmptyState` (with `onAddTask`) is not a fit; an inline message keeps tone (Principle IV) without an action that doesn't belong. Spinner/ErrorBanner reuse keeps consistency (Principle III).
- **Alternatives considered**: Generalizing `EmptyState` to be action-less — possible but adds surface; an inline block is simpler for one use.

## Decision 9: Vite dev-proxy entry

- **Decision**: Add `"/plan.v1": "http://localhost:8080"` to `vite.config.ts` proxy (alongside `/task.v1`, `/health.v1`).
- **Rationale**: connect-web posts to `/plan.v1.PlanService/...` on the same origin; the dev proxy must forward it to the server so the session cookie is sent. Without it, the planner query 404s in dev.
- **Alternatives considered**: None — this mirrors the existing task/health proxy entries.
