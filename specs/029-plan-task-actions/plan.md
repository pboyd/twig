# Implementation Plan: Plan Task Actions

**Branch**: `029-plan-task-actions` | **Date**: 2026-06-01 | **Spec**: [spec.md](./spec.md)

**Input**: Feature specification from `/specs/029-plan-task-actions/spec.md`

## Summary

Let the user act on the **task behind a task-linked plan entry** directly from the TUI planning tab, using the gestures they already know from the Tasks tab:

1. **Toggle complete** (`space`) — completes the linked task (or un-completes it). The plan entry already mirrors its task's completed state, so after the mutation we reload the day's plan and the entry redraws as completed in place.
2. **Start a pomodoro** (`s`) — starts a pomodoro for the linked task, identical to starting one from the Tasks tab.

Both actions are inert on **event** entries (no linked task, `TaskId == 0`) and when no entry is selected, and surface feedback through the existing `notice`/`err` status channels in the house tone.

This is a **TUI-only wiring change**. The underlying operations (`task.v1.CompleteTask`, `UncompleteTask`, `StartPomodoro`) already exist and are already used by the Tasks tab. **No proto, no DB schema, no handler, and no generated-code changes.** `twig-web` is untouched (it consumes only `task.v1` and has no planning surface).

## Technical Context

**Language/Version**: Go 1.x (single module at `services/twig/`)

**Primary Dependencies**: ConnectRPC (existing `task.v1` + `plan.v1` clients), Bubble Tea + Lipgloss (TUI)

**Storage**: PostgreSQL — read-only here. Completion is a property of `tasks`; `plan_entries.completed` is computed/joined server-side on `ListPlan`. **No schema change.**

**Testing**: `go test ./...`; TUI tests use `export_test.go` shims and table-driven `tea.KeyMsg` dispatch tests; no running DB required.

**Target Platform**: terminal TUI (`cmd/twig`, interactive on a TTY).

**Project Type**: Single Go module, two binaries (server + CLI/TUI). React SPA at `services/twig-web/` is out of scope (no planning surface, consumes only `task.v1`).

**Performance Goals**: N/A — interactive single-user TUI; a handful of plan entries per day.

**Constraints**: No proto/DB schema change; generated code (`gen/`, `internal/db/`) only via `make proto` / `sqlc generate` (neither needed this feature); all existing planning-tab behavior preserved.

**Scale/Scope**: A handful of plan entries per day per user.

## Constitution Check

*GATE: Must pass before Phase 0 research. Re-check after Phase 1 design.*

| Principle | Status | Notes |
|-----------|--------|-------|
| I. Simplicity / YAGNI | ✅ | Pomodoro reuses `startPomCmd` verbatim. Complete adds **one** small command helper (`completePlanTaskCmd`) that calls the existing `CompleteTask`/`UncompleteTask` then re-emits the existing `planMutatedMsg` (whose reload + notice + tab-agnostic-error wiring already exists from feature 028). No new message type, no new endpoint, no new abstraction. |
| II. API-First Design | ✅ | No proto change and no new endpoint — the operations already exist and are contractually unchanged. `contracts/plan-task-actions.md` records the explicit "no API delta" decision so the gate has an artifact to check. |
| III. UI/UX Consistency | ✅ | Reuses the **exact** Tasks-tab keys (`space` toggle-complete, `s` start-pomodoro) and the same task-service operations, so behavior is identical across tabs. Feedback uses the existing `notice`/`err` status area and theme; help text is extended through the existing `PlanningMode` help groups. |
| IV. Playful User Messages | ✅ | The two new notices (event-not-actionable for complete, event-not-actionable for pomodoro) and the optional completion confirmation are authored in the warm/playful house tone and listed in `research.md` for tone review. |

No violations → Complexity Tracking table omitted.

## Project Structure

### Documentation (this feature)

```text
specs/029-plan-task-actions/
├── plan.md              # This file
├── research.md          # Phase 0 — decisions & rationale
├── data-model.md        # Phase 1 — entities/invariants (no schema change)
├── quickstart.md        # Phase 1 — manual verification script
├── contracts/
│   └── plan-task-actions.md   # Phase 1 — "no API delta" record + behavior notes
└── checklists/
    └── requirements.md  # (from /speckit-specify)
```

### Source Code (repository root)

```text
services/twig/
└── internal/
    └── tui/
        ├── update.go        # handlePlanKey (planList mode): add Complete + PomStart cases;
        │                     #   add completePlanTaskCmd helper (task-service mutation → planMutatedMsg)
        ├── keymap.go        # PlanningMode ShortHelp/FullHelp: surface Complete + PomStart
        └── update_test.go   # table-driven KeyMsg tests for the two new planning-tab actions
                             #   (incomplete→complete, complete→uncomplete, event no-op, empty-plan no-op,
                             #    pomodoro start on linked entry, pomodoro no-op on event)
# (no proto/, gen/, db/, handler/, or twig-web/ changes)
```

**Structure Decision**: Single Go module. The change is confined to the TUI's planning-tab key handler (`internal/tui/update.go`) plus help-text surfacing (`keymap.go`). Two independently-testable user stories map to two `case` branches in `handlePlanKey`; pomodoro reuses the existing `startPomCmd`, complete adds one thin command that funnels into the existing `planMutatedMsg` reload path.

## Phase 0 — Research

See [research.md](./research.md). Key decisions:

1. **Reuse Tasks-tab operations and keys, don't invent new ones.** `space` → complete/uncomplete the linked task; `s` → start a pomodoro for it. Identical keys, identical task-service calls → identical results across tabs (FR-005/FR-006, Principle III). These keys are currently unbound in `planList` mode, so there is no conflict.
2. **Pomodoro path = `startPomCmd(m.client, entry.TaskId, entry.Name)` verbatim.** It already returns `pomStartedMsg`, handled tab-agnostically (sets `m.pom`, starts the tick). No plan reload needed; the running-timer indicator already renders on both tabs.
3. **Complete path = new `completePlanTaskCmd` that funnels into `planMutatedMsg`.** It calls `CompleteTask`/`UncompleteTask` (chosen by the entry's current `Completed` flag), then returns `planMutatedMsg{notice, highlightID: entry.Id}` on success or `planMutatedMsg{err}` on failure. `planMutatedMsg` already (a) reloads the day via `listPlanHighlightCmd` so the entry's recomputed `Completed` is picked up, (b) shows a `notice`, and (c) can route errors. This is why completing one entry updates **every** entry linked to the same task (FR-004) — the whole day is re-fetched.
4. **Toggle direction is decided from `entry.Completed`** (the entry already mirrors the task's state), mirroring the Tasks-tab logic that branches on `node.Task.GetCompletedAt()`.
5. **Event guard via `entry.TaskId == 0`.** Events carry no task; both actions short-circuit with a playful `notice` and change nothing (FR-007). Empty plan / no selection guarded by `len(m.plan.entries) > 0` (FR-008), matching the existing edit/remove guards.
6. **No auto-dismiss of the notice** (consistent with feature 028): it persists until the next action replaces or clears it.

## Phase 1 — Design & Contracts

- **Data model**: [data-model.md](./data-model.md) — no schema change; documents that `PlanEntry.TaskId == 0` marks an event, `PlanEntry.Completed` mirrors the linked task's completion (server-computed on `ListPlan`), and completion is owned by the task, not the entry.
- **Contracts**: [contracts/plan-task-actions.md](./contracts/plan-task-actions.md) — explicit "no API delta": the actions reuse existing `task.v1.CompleteTask`, `task.v1.UncompleteTask`, and the existing pomodoro-start call with unchanged request/response messages.
- **Quickstart**: [quickstart.md](./quickstart.md) — manual TUI verification of complete-from-plan, uncomplete-from-plan, cross-tab consistency, pomodoro-from-plan, and the event/empty no-op cases.
- **Agent context**: CLAUDE.md SPECKIT marker updated to point at this plan.

## Post-Design Constitution Re-Check

Re-evaluated after Phase 1: still ✅ on all four principles. No proto/schema/endpoint surface added; the implementation consolidates onto existing task-service operations and the existing `planMutatedMsg` reload path (a net reuse, not new complexity); new copy is queued for tone review.
