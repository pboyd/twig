# Surface Contract: `twig report` (CLI) and Report tab (TUI)

**Feature**: 048-activity-report | **Status**: contract — commit before implementation

## CLI command

```
twig report                                  # default period: recent (yesterday + today)
twig report <preset>                         # one of the presets below
twig report --from <YYYY-MM-DD> --to <YYYY-MM-DD>
twig help report                             # usage
```

Presets: `today`, `yesterday`, `week`, `last-week`, `month`, `quarter`, `year`
(resolution rules in `data-model.md`; weeks are Monday–Sunday; all boundaries in
the user's local timezone).

Argument rules:
- A preset and `--from/--to` are mutually exclusive.
- `--from` and `--to` must be given together; both inclusive calendar days.
- Unknown preset, unparseable date, or `--to` before `--from` → exit code 1 with a
  playful, actionable error naming the accepted formats; no partial output (FR-013).

Exit codes: `0` success (including an empty period), `1` usage/validation or
server/connection error — matching existing command conventions.

### Output — day-grouped (period span ≤ 14 days)

```
What you got done — Yesterday & Today

Thu, Jun 11
  ✓ Write quarterly summary

Wed, Jun 10
  ✓ Wire up the download page        (under: Web app polish)
  ✓ Fix proxy config

3 tasks done · 5 pomodoros burned
```

- Days ordered most recent first (FR-003).
- Subtasks show `(under: <parent name>)` context (FR-002).
- When attached to a TTY, the day headers and ✓ marks use the existing styled
  rendering conventions (`internal/cli/render.go`); when piped, output is plain
  text with no ANSI sequences (FR-009).

### Output — accomplishment-grouped (period span > 14 days)

```
What you got done — Apr 1 – Jun 11

42 tasks done · 87 pomodoros burned

Finished
  ✓ Ship alternate profiles            (done Jun 2)
      ✓ Config schema                  (done May 20)
      ✓ Profile flag parsing           (done May 28)

Progress on ongoing work
  … Web app polish
      ✓ Wire up the download page      (done Jun 10)
```

- "Finished" lists top-level tasks completed in the period, newest first, with
  their completed-in-period descendants indented beneath (FR-004).
- "Progress on ongoing work" lists in-period completed descendants of top-level
  tasks that are not themselves finished in the period.

### Empty state (both layouts)

```
Nothing checked off between Jun 10 and Jun 11 — a blank page, full of potential.
```

Exact copy may be tuned, but MUST be playful, name the period, and exit 0 (FR-012,
Principle IV).

## TUI Report tab

- Third tab in the existing cycle: **Tasks → Planning → Report** via
  `tab`/`shift+tab` (existing `NextTab`/`PrevTab` bindings).
- Period presets cycle with `←/→` (`h`/`l`); order:
  `recent → today → yesterday → week → last-week → month → quarter → year`.
  Default on entry: `recent`.
- `↑/↓` (`k`/`j`) scroll overflowing content; `r` refreshes; `?` help; `q` quit —
  all existing bindings.
- Header shows the period label and summary totals; body uses the same layouts as
  the CLI (day-grouped vs accomplishment-grouped by the 14-day rule).
- All colors/styles come from the shared theme in `internal/tui/theme.go`
  (Principle III). Empty and error states reuse the playful copy conventions.
- No explicit `--from/--to` equivalent in the TUI (presets cover the stated
  use-cases; CLI handles arbitrary ranges).

## Shared behavior

- Data sources: `ListTasks` (filter `completed_at` in period) +
  `CountCompletedPomodoros` (see `report-rpc.md`). Both authenticated as the
  calling user (FR-010).
- Connection failures surface the same error treatment as existing commands/tabs.
- Both surfaces are read-only; no mutations are issued by the report.
