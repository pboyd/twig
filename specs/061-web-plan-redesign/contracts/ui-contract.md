# UI Contract: Web Plan Tab Day-Planner Redesign

**Date**: 2026-07-02 | **Spec**: [spec.md](./spec.md)

This is a presentation-layer feature: **no proto, endpoint, or backend changes**. The contract has two parts: the (existing, frozen) RPC surface the page consumes, and the component contracts the implementation must conform to.

## Consumed RPCs (existing — MUST NOT change)

| RPC | Used for | Notes |
|---|---|---|
| `plan.v1.PlanService/ListPlanEntries` | Day's entries (`day: YYYY-MM-DD`) | Source of timed/untimed groups |
| `plan.v1.PlanService/RemovePlanEntry` | Trash action | Per-entry removal |
| `task.v1.TaskService/ListTasks` | Task-name fallback for entries | Existing join in `resolveEntries` |
| `task.v1.TaskService/CompleteTask` | Toggle: incomplete → complete | `FailedPrecondition` → `completeBlockedBySubtasks` |
| `task.v1.TaskService/UncompleteTask` | Toggle: complete → incomplete (**newly wired on this page**, RPC already exists and is used by the Tasks tab) | `FailedPrecondition` → `reopenBlockedByParent` |

Server guarantee relied upon: timed plan entries never overlap (rejected at creation), so the timeline renders at most one entry per 15-minute slot.

## Component contracts

### `CompletionToggle` (new, shared)

```ts
interface CompletionToggleProps {
  completed: boolean;
  disabled?: boolean;        // pending mutation
  onToggle: () => void;
}
```

- Visual: 44px (`h-11 w-11`) button containing a 20px (`h-5 w-5`) circle — empty with gray border when incomplete; green-filled with white check when complete. Byte-for-byte the markup currently inline in `TreeRow.tsx`.
- Accessibility: `aria-label` "Mark complete" / "Mark incomplete" based on state.
- Consumers: `TreeRow` (Tasks tab — no visual change), `PlanTimeline` entry blocks, `PlanEntryRow` (untimed rows). Task-detail page may adopt later; out of scope.

### `PlanTimeline` (new)

```ts
interface PlanTimelineProps {
  entries: ResolvedEntry[];            // timed only
  day: string;                          // YYYY-MM-DD, for the today check
  onToggleComplete: (entry: ResolvedEntry) => void;
  onRemove: (entry: ResolvedEntry) => void;
  pendingIds: Set<number>;              // entries with an in-flight mutation
  entryErrors: Map<number, string>;     // per-entry inline error text
}
```

Rendering contract (testable assertions):
- Hour-labeled gutter using existing `formatMinute`; ruled line at each hour boundary.
- One grid row per 15-minute slot, 44px tall; window per `computeWindow` (auto-fit, 8:00–17:00 minimum).
- Each entry renders one block spanning its snapped slots: `CompletionToggle` (task entries only), name (link to `/tasks/:id` for tasks; italic plain text for events), trash button, strikethrough when completed.
- `data-testid="now-indicator"` line rendered iff `day` is today and now ∈ window; positioned by minute offset.
- Empty timed set: component renders the default-window ruled grid only when untimed entries exist; the page-level empty state covers fully empty days.

### `PlanEntryRow` (repurposed: untimed row)

Drops the time-label branch and the old check-button/badge; renders `CompletionToggle` + name + trash in the existing list styling. Same handlers/props shape as today otherwise.

### `PlanPage` (modified)

- Section order: day nav → untimed section (heading "Untimed") → timeline.
- Adds `uncompleteTask` mutation beside `completeTask`; toggle dispatches on `entry.completed`.
- Loading (`Spinner`), error (`ErrorBanner` + retry), empty (`messages.planEmpty`) states unchanged (FR-010).

## Non-functional contract

- All tap targets ≥ 44px (`spacing.touchTarget`).
- Colors/styles from existing Tailwind utility conventions + `src/theme/tokens.ts`; dark-mode variants for every new style.
- Any new user-visible copy goes through `src/theme/messages.ts` (Principle IV); none is currently planned.
