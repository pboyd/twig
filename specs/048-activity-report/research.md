# Research: Activity Report

**Feature**: 048-activity-report | **Date**: 2026-06-11

No external unknowns required outside research; all decisions below were resolved by
inspecting the existing codebase (proto contracts, DB queries, TUI/CLI structure).

## D1: Report composition is client-side, with one new RPC for pomodoro totals

**Decision**: The CLI and TUI build the report from the existing `ListTasks` RPC
(filtering on `completed_at` within the period) plus one new RPC,
`CountCompletedPomodoros(start, end)`, added to `task.v1.TaskService`.

**Rationale**:
- `ListTasks` already returns every task for the user — including completed ones —
  with `completed_at` and `parent_id` (`services/twig/db/queries/task.sql`:
  `SELECT * FROM tasks WHERE user_id = $1`). Both the TUI and CLI already build
  trees from this flat list (`internal/cli.TreeNode`), so parent context and
  top-level-ancestor grouping need no new server support.
- The only datum the client cannot derive is "pomodoros completed during the
  period": per-pomodoro timestamps are only exposed via `GetTask` (one task at a
  time), and `completed_pomodoro_count` on `Task` is a lifetime count. One small
  count RPC closes that gap.
- Pomodoro RPCs already live on `TaskService` (`StartPomodoro`, `CompletePomodoro`,
  …), so the new RPC follows the established home for pomodoro operations.

**Alternatives considered**:
- *Full server-side `GetReport` RPC / new `report.v1` service*: one round trip and
  better asymptotic scaling, but requires a new proto package, a recursive-CTE
  ancestor query, a new handler, and duplicate grouping logic server-side. At
  personal-todo scale (thousands of tasks), `ListTasks` is already the cost of
  every TUI session. Rejected per Principle I (YAGNI).
- *No new RPC; sum pomodoros via `GetTask` per task*: N+1 requests, and misses
  pomodoros on tasks completed outside the period or still open. Rejected.

## D2: Shared report logic lives in a new root-module package `internal/report`

**Decision**: Create `internal/report` (root module) holding period
parsing/presets, range math, and grouping (day-grouped and accomplishment-grouped
structures). Both `internal/cli` and `internal/tui` consume it.

**Rationale**: Two consumers exist today (CLI command and TUI tab), so the shared
package is not speculative. It mirrors the existing `internal/pomodoro` pattern
(domain logic separate from rendering). Rendering stays in each surface:
`internal/cli` renders plain/ANSI text, `internal/tui` renders with the shared theme.

**Alternatives considered**: putting the logic in `internal/cli` (where `TreeNode`
lives) — rejected because period/grouping logic is not CLI rendering, and
`internal/cli` is already large; a focused package keeps both surfaces honest.

## D3: Period semantics

**Decision**:
- A period is an inclusive range of local calendar days `[from, to]`. The client
  resolves it in the user's local timezone (`time.Local`) and converts it to a
  half-open UTC instant range `[startOfDay(from), startOfDay(to+1))` for filtering
  `completed_at` and for the `CountCompletedPomodoros` request.
- Presets: `today`, `yesterday`, `week` (this week), `last-week`, `month`,
  `quarter`, `year` — plus the default `recent` = yesterday through today.
- Weeks run Monday–Sunday (ISO 8601), per spec clarification.
- Layout selection: span of ≤ 14 calendar days → day-grouped; > 14 days →
  accomplishment-grouped (spec clarification; no manual override).

**Rationale**: Half-open UTC ranges make near-midnight and DST edge cases exact
(spec SC-005). Day-span is computed from calendar days, not durations, so DST
transitions cannot flip the layout.

**Alternatives considered**: sending local dates + IANA zone to the server —
unnecessary once filtering is client-side; only the pomodoro count needs a range,
and UTC instants are exact.

## D4: Accomplishment-grouped layout rules (periods > 14 days)

**Decision**:
- Every completed-in-period task is attributed to its **top-level ancestor**
  (walk `parent_id` to the root using the full `ListTasks` result).
- Section 1 — "Finished": top-level tasks completed within the period, newest
  first, each listing its completed-in-period descendants beneath it.
- Section 2 — "Progress on ongoing work": top-level tasks *not* complete (or
  completed outside the period) that have descendants completed within the period;
  each shows those descendants. This covers the spec's "subtask complete, parent
  open" scenario.
- Summary totals (tasks completed, pomodoros completed) render at the top in both
  layouts.

**Rationale**: Matches the spec's structural definition of "significant
accomplishments" (completed top-level tasks as headlines) while guaranteeing no
completed-in-period task is omitted (SC-004).

## D5: CLI surface

**Decision**: New top-level command `twig report`:

```
twig report                      # default: recent (yesterday + today)
twig report <preset>             # today|yesterday|week|last-week|month|quarter|year
twig report --from <YYYY-MM-DD> --to <YYYY-MM-DD>
```

Output styling follows the existing TTY-detection convention in
`internal/cli/render.go` (plain text when piped). Errors and empty states use the
playful tone (Principle IV). `twig help report` added alongside existing usage
functions.

**Alternatives considered**: `twig task report` subcommand — rejected; the report
spans tasks *and* pomodoros, and presets read better as a first-class command,
consistent with `plan` and `pom` being top-level.

## D6: TUI surface

**Decision**: A third tab, **Report**, joining the existing `tabTasks`/`tabPlanning`
cycle (tab / shift+tab already switch tabs). Within the tab:
- `←/→` (and `h/l`) cycle through the preset list (recent → today → yesterday →
  week → last-week → month → quarter → year); default on entry is `recent`.
- `↑/↓` scroll when content overflows; `r` refreshes (existing binding).
- Styling comes from the shared theme (`internal/tui/theme.go`); no ad-hoc colors.

**Rationale**: Reuses the established tab + keymap pattern (Principle III); single
keystrokes satisfy SC-001 (answer "what did I do yesterday?" in under 10 s).
Explicit `--from/--to` ranges remain CLI-only — arbitrary date entry in the TUI is
not needed for the stated use-cases (YAGNI); presets cover them.

## D7: Pomodoro count query semantics

**Decision**: New sqlc query in `services/twig/db/queries/pomodoro.sql`:

```sql
-- name: CountCompletedPomodorosInRange :one
SELECT count(*)::bigint AS count FROM pomodoros
WHERE user_id = $1 AND complete AND end_at >= $2 AND end_at < $3;
```

A pomodoro is "completed during the period" when `complete = TRUE` and its
`end_at` falls in the half-open UTC range. (`CompleteActivePomodoro` sets
`end_at = start_at + 25 minutes`, so `end_at` is the completion moment.)

**Rationale**: Matches the clarified semantics — effort spent during the period,
regardless of task completion. No schema migration needed.

## D8: Testing approach

**Decision**:
- `internal/report`: pure unit tests (period resolution incl. DST/midnight/ISO-week
  edges; grouping rules; layout threshold).
- `internal/cli`: report rendering tests following existing `task_test.go` patterns
  (golden-ish string assertions, TTY on/off).
- `internal/tui`: report view/update tests following existing `plan_view_test.go`
  patterns; `nowFunc`-style injection already exists in the model for time control.
- Handler: `CountCompletedPomodoros` integration test gated on `DATABASE_URL`
  (skips when unset), matching `pomodoro_test.go` conventions.
