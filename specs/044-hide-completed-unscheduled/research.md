# Phase 0 Research: Hide Completed Unscheduled Plan Entries

No open `NEEDS CLARIFICATION` items remained from the spec (one ambiguity was resolved in the clarification session). Research here records the codebase findings that shape the approach.

## Decision 1: Completion data is already on the wire — no API/server change

**Decision**: Consume the existing `plan.v1.PlanEntry.completed` field; make no proto, server, or DB changes.

**Rationale**: `api/proto/plan/v1/plan.proto` already defines `PlanEntry.completed` (field 7): *"True iff this entry references a completed task… Populated server-side by `ListPlanEntries`; ignored on writes."* The server handler sets it (`services/twig/internal/handler/plan.go:144`) and it is covered by `TestListPlanEntries_Completed`. Both clients already read it (TUI detail pane `internal/tui/plan_view.go:316,339`; web `PlanEntryRow.tsx`). The feature only needs to *filter on* this value, which keeps it a client-side display concern and satisfies API-First (the contract pre-exists).

**Alternatives considered**:
- *New server-side "hide completed untimed" query/flag* — rejected: pushes a pure view concern into the API, adds an endpoint variant, and would diverge TUI/web rendering from a single source of truth. Violates Principle I.

## Decision 2: TUI — reuse the tasks-tab `pendingComplete` deferred-hide pattern

**Decision**: Add `pendingComplete *int32` to `planState` (entry id). When the user completes an untimed entry, set it to that entry's id; filter completed untimed entries out of the displayed list **except** the `pendingComplete` one; clear `pendingComplete` (so it disappears) on the next navigation — cursor up/down, day change, tab switch, or go-to-task.

**Rationale**: The tasks tab already implements exactly this UX with `Model.pendingComplete *int64`: `buildVisible` skips completed tasks unless the task is the pending one, and every navigation handler resets `pendingComplete = nil` then rebuilds (`internal/tui/tree.go:122-124`, `internal/tui/update.go`). Mirroring it on the plan gives an identical, already-proven interaction (Principle III) and matches the clarified requirement (retain only while it is the active highlight). The crossed-out look is already produced by `applyCompletion` → `DimStrike`, used by `RenderUntimed` (`internal/cli/plan_grid.go:460,572`), so no new styling is needed.

**Key mechanics**:
- The plan cursor indexes into the displayed entry list (`m.plan.entries[m.plan.cursor]`). The filter therefore operates on what becomes `m.plan.entries` so cursor semantics are preserved.
- Filter predicate (untimed only): drop entry when `StartMinute == nil && Completed && id != pendingComplete`. Timed completed entries are never dropped (FR-004); events have `completed == false` always (FR-005).
- On load after a completion (`completePlanTaskCmd` reloads with `highlightID = entryID` → `handlePlanEntriesMsg`), `pendingComplete` is already set, so the just-completed entry survives the filter and is re-highlighted.
- On navigation, re-running the filter with `pendingComplete == nil` drops the entry; the cursor is then clamped.

**Alternatives considered**:
- *Separate raw vs. visible lists (like the tasks tab's `m.tree`/`m.visible`)* — rejected as heavier than needed: the plan cursor already indexes the displayed list, and the pending entry is already present in that list while highlighted, so re-filtering in place removes the need to store a parallel raw list. Simpler (Principle I).
- *Hide immediately on completion (no `pendingComplete`)* — rejected: contradicts the clarified UX and would jump the highlight, inconsistent with the tasks tab.

## Decision 3: Web — filter completed untimed entries in `groupPlan`

**Decision**: In `services/twig-web/src/lib/planView.ts`, exclude completed entries from the `untimed` group: `entries.filter((e) => !e.timed && !e.completed)`. Timed entries are unchanged.

**Rationale**: `groupPlan` is the single, unit-tested place that partitions entries into `{ timed, untimed, isEmpty }` consumed by `PlanPage.tsx`. Filtering there keeps the change minimal and testable, and `isEmpty` (which already factors in `untimed.length`) automatically reflects a day whose only untimed entries are completed. The web app has no inline completion affordance in the planner — completion happens in the task list/detail — so a render/refetch-time filter is the correct and sufficient model (no deferred-highlight interaction needed, per the spec). `completed` is already present on the resolved entry (used by `PlanEntryRow.tsx`).

**Alternatives considered**:
- *Filter inside `PlanPage.tsx` JSX* — rejected: scatters view logic outside the tested helper and risks `isEmpty` drift.

## Cross-surface consistency note

For the same underlying data both surfaces exclude exactly the completed *untimed* entries and keep all timed entries (FR-003, SC-003). The only surface-specific behavior is the TUI's keyboard deferred-hide (`pendingComplete`), which the web does not need because it lacks a keyboard highlight.
