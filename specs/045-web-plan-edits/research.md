# Phase 0 Research: Basic Planning Edits on the Web App

All spec ambiguities were resolved in `/speckit-clarify` (see spec **Clarifications**). No `NEEDS CLARIFICATION` markers remain in Technical Context. This document records the design decisions for the brownfield web work.

## 1. Server operations already cover every write

**Decision**: Consume existing RPCs unchanged; add no proto/server/DB code.

- **Add to plan (untimed)** → `plan.v1.PlanService.AddPlanTask` with `{ day, task_id }` and `start_minute` **omitted** (absent = untimed). `duration_minute: 0` lets the server pick its default.
- **Remove entry** → `RemovePlanEntry` with `{ day, id }`. Works for any entry kind (task/event, timed/untimed).
- **Complete task** → `task.v1.TaskService.CompleteTask` with `{ id }` (the entry's `task_id`).
- **Read** → `ListPlanEntries { day }` already returns `PlanEntry.completed`.

**Rationale**: Verified the generated query hooks exist (`addPlanTask`, `removePlanEntry`, `completeTask`, `listPlanEntries` in `src/gen/`). The TUI uses these same operations, so behavior parity is free.

**Alternatives considered**: Adding a web-specific batch/edit endpoint — rejected (no need; violates Principle I and II).

## 2. Duplicate-prevention is server-enforced

**Decision**: Do **not** implement a client-side "is this task already on the day" check. Call `AddPlanTask` and surface a `FailedPrecondition` rejection as the "already on that day" message.

**Rationale**: `handler/plan.go` returns `errDuplicateUntimed()` — `connect.CodeFailedPrecondition`, message *"That one's already parked here without a time — it can only wait in one spot."* — when a second untimed entry for the same (task, day) is attempted. A client check would duplicate server logic and risk drift (Principle I). On `AddPlanTask`, the only `FailedPrecondition` is the duplicate case, so mapping that code → "already on that day" copy is unambiguous.

**Alternatives considered**: Pre-checking against a cached `ListPlanEntries` for the target day — rejected (extra fetch, stale-cache races, redundant with the server guard).

## 3. Transient confirmation = a minimal toast primitive

**Decision**: Add a small `ToastProvider` (React context) + `useToast()` hook + presentational `Toast` component, mounted once at the app root around `<Routes>`. `useToast().show(message)` enqueues an auto-dismissing toast (≈3s) rendered in a fixed overlay.

**Rationale**: FR-005 requires a brief, auto-dismissing confirmation that can name the target day, and the "add" action originates on the tasks page where there is no contextual banner slot. No toast/snackbar exists today (grep confirmed). A context-based toast is the standard React pattern and the smallest thing that satisfies the requirement; it is reused by both add-from-tasks and (optionally) planner edits. Styling draws only from `theme/tokens.ts`, satisfying Principle III.

**Alternatives considered**:
- *Inline row cue* ("Added ✓" on the row) — rejected at clarification; easy to miss and tied to row context.
- *ErrorBanner-style top banner for success* — heavier and visually conflated with errors.
- *A toast library (react-hot-toast, sonner)* — rejected; a ~40-line primitive avoids a new dependency (Principle I).

## 4. Add-to-plan control: single-tap today, popover for other days

**Decision**: An `AddToPlanControl` rendered in each `TreeRow`'s action cluster:
- A primary icon button = **add to today** in one tap (→ `AddPlanTask { day: today }`, then toast). Satisfies SC-001 (single action, no day-selection step).
- A small adjacent affordance opens a lightweight popover offering **Tomorrow** (quick preset) and a native `<input type="date">` + an **Add** confirm. Future-day add = open → pick day → Add (≤ 3 actions, SC-002). Past dates are not blocked (FR-004).

**Rationale**: Splitting "today" (the dominant case, made a single tap) from "other day" (a deliberate, multi-step choice) is the only layout that satisfies both SC-001 and SC-002. A native date input needs no new dependency, is mobile-friendly, and reuses `Field` styling for consistency (Principle III).

**Alternatives considered**:
- *One menu where Today is an item* — rejected: makes today two taps and treats it as a "day-selection step," violating SC-001.
- *Adjacent-day stepping only* (mirroring planner nav) — rejected at clarification; tedious for far-future days.

## 5. Cache invalidation strategy

**Decision**: After each mutation, `await queryClient.invalidateQueries({ queryKey })` using `createConnectQueryKey`, matching the existing pattern in `TreeRow`/`TaskTreePage`.

- **Add from tasks** → invalidate `listPlanEntries` for the **target day** (so a later visit to that day is fresh). The tasks list itself does not change.
- **Complete from planner** → invalidate `listPlanEntries` for the **current day** (entry shows done / untimed entry disappears per feature 044) **and** `listTasks` (tasks tab reflects completion).
- **Remove from planner** → invalidate `listPlanEntries` for the current day.

Query keys are day-scoped: `createConnectQueryKey({ schema: listPlanEntries, input: { day }, cardinality: "finite" })`.

**Rationale**: Invalidate-then-refetch is the app's established convention (no optimistic-update machinery exists), keeps the view consistent with server truth (FR-013), and is cheap at this scale. Refetch on next render satisfies "without a manual full-page reload."

**Alternatives considered**: Optimistic updates with rollback — rejected (Principle I; the app has no such infrastructure and the data volumes don't warrant it).

## 6. Error mapping & messages (Principle IV)

**Decision**: Centralize new copy in `theme/messages.ts`; map outcomes:

| Action | Outcome | Surface | Copy source |
|--------|---------|---------|-------------|
| Add | success (today) | toast | new `addedToToday` |
| Add | success (other day) | toast | new `addedToDay(label)` |
| Add | `FailedPrecondition` (duplicate) | toast | new `alreadyOnPlan` |
| Add | other error | toast | existing `connectivityError` |
| Complete | success | (refetch; no copy needed) | — |
| Complete | `FailedPrecondition` (incomplete sub-tasks) | inline on entry | existing `completeBlockedBySubtasks` |
| Complete | other error | inline on entry | existing `connectivityError` |
| Remove | success | (refetch; optional toast) | optional new `entryRemoved` |
| Remove | error | inline on entry | existing `connectivityError` |

**Rationale**: Reuses the proven `ConnectError` + `Code.FailedPrecondition` discrimination already in `TreeRow.handleToggleComplete`. New strings stay warm/playful per Principle IV and consistent with existing entries like `completeBlockedBySubtasks` and `planEmpty`.

## 7. Remove confirmation

**Decision**: Remove takes effect immediately on tap; no confirmation dialog (spec Assumption). The underlying task is untouched (FR-009) and the entry can be re-added, so the stakes are low.

**Rationale**: Matches the spec assumption and keeps the planner interaction light. A future undo could be layered on the existing toast if desired, but is out of scope.
