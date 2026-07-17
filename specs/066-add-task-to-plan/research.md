# Phase 0 Research: Add New Tasks to a Plan

**Feature**: 066-add-task-to-plan | **Date**: 2026-07-17

The spec's three UI questions were resolved with the user during `/speckit-specify`, so no
NEEDS CLARIFICATION markers reached this phase. Research here covers how the chosen design lands on
the existing code, and which existing pieces it reuses.

## R1: Does the server need any change?

**Decision**: No. No proto change, no handler change, no migration, no `sqlc generate`.

**Rationale**: `plan.v1.PlanService.AddPlanTask` already accepts `{day, task_id, start_minute?,
duration_minute}` (`api/proto/plan/v1/plan.proto:65`). Omitting `start_minute` creates an untimed
entry, which is exactly what FR-004 requires. Passing `duration_minute: 0` lets the server apply its
existing default. The feature is two client-side calls in sequence against already-shipped RPCs.

**Alternatives considered**:
- *A new `CreateTaskAndPlan` RPC making it one atomic call.* Rejected under Principle I (YAGNI). It
  would add proto surface, a handler, and a transaction to serve one UI convenience, and it
  duplicates two working RPCs. The partial-failure case it would eliminate is instead handled
  explicitly in the UI (FR-007), which is where the user can actually act on it.
- *A `plan_day` field on `CreateTaskRequest`.* Rejected for the same reason, plus it would put
  scheduling semantics on the task contract, where they do not belong — no task entity currently
  knows anything about plans.

## R2: How does the TUI form gain a cycling selector?

**Decision**: Add `focusPlan` to the existing focus-index constant block in `internal/tui/edit.go`,
carrying a `planIdx` selector plus a `planDate string` for calendar-picked days. Gate it behind a
`showPlanField bool`, set only when the form is creating a task.

**Rationale**: The form is a flat focus-index list (`focusName…focusCount`,
`internal/tui/edit.go:41`) with two existing precedents to copy rather than invent:
- **The Goal selector** (`focusGoal`) is already a left/right cycling field with a
  `showGoalField` gate and a `cycleGoal` helper doing modular arithmetic over a "none" sentinel
  (`internal/tui/edit.go:452`). The Plan selector is the same shape with a fixed choice list.
- **`cycleFocus`** already skips hidden fields, so the `showPlanField` gate needs no new mechanism.

**Alternatives considered**:
- *A `textinput` date field like Snooze.* Rejected in the spec — makes "today" cost typing.
- *A new overlay/mode.* Rejected: heavier than the form's existing idioms and would need its own
  key handling, dirty tracking, and view.

## R3: How does the calendar key work on a non-text field?

**Decision**: Extend the existing `keys.Calendar` guard from `(focusDue || focusSnooze)` to also
match `focusPlan` (`internal/tui/edit.go:355`). On pick, store the ISO date in `planDate` and set
`planIdx` to the "explicit date" position so the field renders the date.

**Rationale**: The binding is `ctrl+g` / "pick a date" (`internal/tui/keymap.go:243`) and the
`calendarModel` is already embedded in the form (`f.calendar`), opened via `f.openCalendar()`. The
only real difference is where the picked value lands: `writeDateField` currently pairs the calendar
with a `textinput.Model` (`internal/tui/edit.go:605`), and the Plan field has no text input. So the
Plan field needs its own small render path rather than reusing `writeDateField`.

**Note for implementation**: `openCalendar` seeds itself from the focused field's text value. The
Plan field must seed from `planDate` (or today when empty). This is the one place the existing
calendar plumbing does not transfer unchanged.

**Alternatives considered**:
- *Reuse `writeDateField` by backing the Plan field with a hidden `textinput`.* Rejected: a hidden
  input existing only to satisfy a render helper is indirection the reader has to unpick, and it
  would put the field into text-editing key handling it must not have.

## R4: How does the plan day travel from form to server in the TUI?

**Decision**: Add `planDay string` (empty = no plan) to `editSavedMsg`, resolved to an ISO date at
save time. Chain the plan call inside `createTaskCmd`, after `CreateTask` returns the new id.

**Rationale**: `editSavedMsg` (`internal/tui/edit.go:85`) is already the form's whole output, and
already carries derived values (`goalChanged`). `createTaskCmd` already chains a second RPC after
create for exactly this shape of problem — `SetTaskGoal` on a new task
(`internal/tui/update.go:396`) — including its error handling. The plan call follows that pattern.

**Resolving Today/Tomorrow at save time** (FR-005) uses the form's existing `nowFunc` hook
(`internal/tui/edit.go:65`), which exists to make time-dependent form behavior testable.

**Signature change**: `createTaskCmd` takes only a `TaskServiceClient` today. It needs the
`PlanServiceClient` too. `Model` already holds both (`internal/tui/model.go:141-142`), so this is a
parameter addition at one call site, not new plumbing.

**Reuse**: `addPlanTaskCmd` (`internal/tui/plan_update.go:62`) already wraps `AddPlanTask` and takes
a tab-agnostic success notice — but it returns a `planMutatedMsg` that drives the Plan tab. Creating
a task happens on the Tasks tab, which needs a `refreshedMsg` to redraw the tree. The two commands
want different terminal messages, so `createTaskCmd` calls `AddPlanTask` directly rather than
composing `addPlanTaskCmd`.

## R5: How does the web form gain the control without disturbing its other uses?

**Decision**: Add a `showPlanControl?: boolean` prop (default `false`) to `TaskForm`, and widen
`onSubmit` to `(name, description, planDay?: string)`. Enable it at the two create sites; leave the
edit site untouched.

**Rationale**: `TaskForm` has three call sites, two of which create and one of which edits:
| Site | Use | Plan control |
|---|---|---|
| `pages/TaskTreePage.tsx:165` | create root task | shown |
| `components/TreeRow.tsx:194` | create sub-task | shown |
| `pages/TaskDetailPage.tsx:257` | edit existing task | hidden (FR-011) |

An optional prop defaulting to off means the edit site needs no change at all, and TypeScript lets
its existing 2-arg `onSubmit` satisfy the widened type without modification.

**Alternatives considered**:
- *A separate `CreateTaskForm` wrapper component.* Rejected under Principle I: it would duplicate
  the form's fields and validation to vary one row.
- *Reusing `AddToPlanControl`.* Rejected in the spec on UX grounds. It is also structurally wrong
  here: it fires `AddPlanTask` on click for a task that already exists, whereas the create form must
  *defer* the call until after the task has an id.

## R6: What existing web helpers does this reuse?

**Decision**: Reuse `lib/planDays.ts` and the existing plan messages verbatim.

**Rationale**:
- `todayIso()`, `tomorrowIso()`, `dayPickerLabel()` (`services/twig-web/src/lib/planDays.ts`) already
  produce exactly the four choices this control needs, and already resolve against local time.
- `theme/messages.ts` already has tone-compliant plan copy: `addedToToday` ("On today's plan! Go get
  it. 🎉"), `addedToDay(label)`, `alreadyOnPlan`, `connectivityError`.
- `context/ToastProvider` is the established way these report success/failure.

**New copy needed**: only the partial-failure message (FR-007) — task created, planning failed —
which has no existing equivalent on either surface. Both surfaces need a tone-compliant string for
it (Principle IV).

## R7: Query invalidation on the web

**Decision**: After a successful create-and-plan, invalidate both the `listTasks` key and the
`listPlanEntries` key for the affected day.

**Rationale**: FR-008 requires the plan to reflect the entry without a manual refresh. The create
sites currently invalidate only `listTasks`. `AddToPlanControl` already shows the pattern for the
plan key, including the `cardinality: "finite"` argument that connect-query needs
(`services/twig-web/src/components/AddToPlanControl.tsx:30`).

## R8: Testing approach

**Decision**: Follow each surface's existing conventions; no new test infrastructure.

**Rationale**:
- **TUI**: table-driven Go tests over `Update`, using `export_test.go` shims. `edit_test.go`,
  `goal_filter_test.go`, and `update_test.go` are the models to follow. `nowFunc` makes
  Today/Tomorrow resolution deterministic without touching the clock.
- **Web**: Vitest + Testing Library alongside the component (`TaskForm.test.tsx`,
  `AddToPlanControl.test.tsx` are direct precedents, the latter including its mutation mocking).
- No integration infrastructure exists (per CLAUDE.md) and this feature does not justify inventing
  it; the RPCs it calls are already covered server-side.

## Summary of reuse

| Need | Existing thing reused | New code |
|---|---|---|
| Add task to a day's plan | `AddPlanTask` RPC | none |
| TUI cycling field | Goal selector pattern, `cycleFocus` | `planIdx` + choice list |
| TUI date picking | `keys.Calendar`, `calendarModel`, `openCalendar` | seed from `planDate`; own render path |
| TUI second-RPC-after-create | `SetTaskGoal` chain in `createTaskCmd` | plan call in same place |
| TUI local "today" | `nowFunc` | none |
| Web day choices | `lib/planDays.ts` | none |
| Web success/error copy | `theme/messages.ts` + `ToastProvider` | partial-failure message only |
| Web plan cache refresh | `AddToPlanControl` invalidation pattern | none |
