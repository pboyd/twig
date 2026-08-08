# Phase 0 Research: Plan Objectives and Notes

**Feature**: 074-plan-objective-notes | **Date**: 2026-08-06

The spec left no `[NEEDS CLARIFICATION]` markers. The open questions here are design
decisions the spec deliberately left to implementation, plus the codebase facts each
one depends on.

---

## D1: Where do day-level fields live?

**Decision**: A new `plan_days` table keyed `(user_id, day)` with `objective VARCHAR(255)`
and `notes TEXT`, both `NOT NULL DEFAULT ''`. Writes upsert; a row is never deleted.
Empty string is the canonical "not set" value at every layer.

**Rationale**: There is no day-level record today — `services/twig/db/migrations/000006_plan_entries.up.sql`
defines `plan_entries` keyed `(user_id, day, id)`, and a "day" exists only as the set of
entries pointing at it. Spec FR-001 requires both fields to exist for a day with zero
entries, so a table of its own is unavoidable. Making the columns `NOT NULL DEFAULT ''`
means one nullability story instead of two (`NULL` vs `''` both meaning absent), which
matters because FR-005 says whitespace-only input clears the field: the handler trims,
and `''` is what lands in the column.

Not deleting the row when both fields are cleared trades a handful of empty rows for
strictly simpler write logic (Principle I). The rows are bounded by days-the-user-touched
and carry no meaning when empty.

**Alternatives considered**:

- *Columns on a widened `plan_entries`* — rejected: the fields belong to the day, not an
  entry, and a day with no entries has nowhere to put them.
- *Nullable columns, `NULL` = unset* — rejected: adds a second way to spell "absent" that
  every reader would have to collapse anyway.
- *Delete the row when both fields go empty* — rejected as premature optimisation; it also
  forces every write to read the sibling field first.

**Objective length**: 255 runes, validated in the handler with a friendly error, matching
`validatePlanName` in `services/twig/internal/handler/plan.go:55` and the `VARCHAR(255)`
used for `plan_entries.name`. The spec calls the objective "short" without a number;
reusing the existing convention beats inventing a second one (Principle III). Notes are
`TEXT` and unbounded, like task descriptions.

---

## D2: API shape

**Decision**: Add to `plan.v1.PlanService`:

- `message PlanDay { string day; string objective; string notes; }`
- `ListPlanEntriesResponse` gains `PlanDay day = 2`, always populated (empty strings when
  the day has neither field).
- `rpc SetPlanObjective(SetPlanObjectiveRequest) returns (SetPlanObjectiveResponse)`
- `rpc SetPlanNotes(SetPlanNotesRequest) returns (SetPlanNotesResponse)`

Both setters return the updated `PlanDay`.

**Rationale**: The read path piggybacks on `ListPlanEntries` because every consumer that
wants the objective or notes is already asking for that day's entries — the TUI on every
day change and on every background refresh (`listPlanCmd`, `internal/tui/plan_update.go:52`),
and the CLI on `twig plan`. A separate `GetPlanDay` would double the round trips on the
TUI's hot path for no gain. Adding a field to an existing response message is wire-compatible.

The write path stays split because the two fields are edited by two independent user
actions (FR-008, FR-014), each validated differently (length limit vs none). One
`UpdatePlanDay` carrying two optional strings would need presence tracking to distinguish
"leave alone" from "clear" — exactly the complexity Principle I says to avoid.

**Alternatives considered**:

- *`GetPlanDay` as a separate read RPC* — rejected: a second round trip per day view and
  per auto-refresh tick, for a field the client already fetches alongside.
- *One `UpdatePlanDay` with `optional string` fields* — rejected: presence-vs-empty
  ambiguity, and no caller wants to set both at once.

**Contract-first**: per Principle II the proto changes and
`specs/074-plan-objective-notes/contracts/` land and are reviewed before handler, CLI or
TUI work begins.

---

## D3: Objective pane and its editor

**Decision**: A full-width pane above the existing two-column row in `viewPlanning`
(`internal/tui/view.go:65`), rendered only when the day's objective is non-empty. Inner
height is the rendered content's height, capped at 3 lines. A new `planObjectiveEdit` mode
puts a `textinput.Model` in that same slot — and the slot is shown while editing even on a
day that has no objective yet. Enter saves, Esc cancels, no buttons.

**Rationale**: `viewPlanning` computes `innerH` from `m.height` and hands it to two
`paneBox` calls joined with `lipgloss.JoinHorizontal`. Reserving the objective pane's
height before that computation is a local change: subtract its height from `innerH`, prepend
its rendered string. When there is no objective the height is zero and the layout is
byte-identical to today's (FR-007, SC-003).

The 3-line cap keeps a pathologically long objective from eating the planner (FR-021)
while leaving room for a wrapped two-line objective on a narrow terminal.

Enter-to-save is a deliberate departure from the tab-through-fields-and-buttons pattern
of `renderPlanFormView` (`internal/tui/view.go` / `plan_view.go:167`). The user asked for
it explicitly, and it is defensible under Principle III because the field is single-line —
the same reason `filterInput` and the date prompt already submit on Enter.

**Alternatives considered**:

- *Reuse the existing right-pane form machinery* — rejected: the spec requires the editor
  to appear where the objective is displayed (FR-009).
- *Always render the pane, empty when unset* — rejected: FR-007 requires omission.

---

## D4: Notes pane and its editor

**Decision**: The right column splits vertically — entry details on top, a `Notes` pane
below. The split is even (`details = ceil(h/2)`, notes gets the remainder), with details
floored at 3 inner lines so it stays usable on short terminals. The notes pane is always
present, empty when the day has no notes. A new `planNotesEdit` mode replaces the **whole**
right column with a `textarea.Model` titled `Notes`; `ctrl+s` saves, `esc` cancels, `ctrl+g`
opens the external editor, Enter inserts a newline.

**Rationale**: The right column already swaps between details and takeover content — 
`viewPlanning` picks `renderPlanDetail` when `m.plan.mode == planList` and
`renderPlanRightPane` otherwise. Notes editing joins that existing branch, so the editor
gets the full column height rather than being crammed into a half-pane, and no new layout
concept is introduced.

An even split is the simplest rule that keeps both panes predictable and testable. Details
content is typically short, so notes get usable space in practice.

`ctrl+s`-to-save (not Enter) is required because notes are multi-line (FR-015), and it is
what `keys.Save` already means everywhere in the program (`internal/tui/keymap.go:223`).
`ctrl+g` reuses `openEditorCmd` (`internal/tui/editor.go:34`) unchanged; only the
`editorFinishedMsg` routing at `internal/tui/update.go:1040` needs a new branch, which
today already dispatches on `m.goal.compose.active` before falling through to the task
edit form. The new branch checks `m.plan.mode == planNotesEdit` first.

**Alternatives considered**:

- *Notes editor confined to the notes pane* — rejected: too cramped for the "work through
  planning the day in writing" use the spec describes.
- *Details keeps its natural content height, notes take the rest* — rejected: makes the
  notes pane's size jump as the selection moves between entries.

---

## D5: Markdown rendering

**Decision**: Both fields render through `m.md.Render(text, markdown.Options{Width, Styled})`
— the same call `renderTaskDescription` uses (`internal/tui/details.go:17`). Editors hold
raw source.

**Rationale**: One code path for both fields, and both inherit the shared theme and the
unstyled fallback automatically (FR-018, FR-019, Principle III). `RenderInline` was
considered for the single-line objective, but block `Render` on a one-line string yields
one line anyway, so the extra branch would buy nothing.

---

## D6: Key bindings

**Decision**: Add `PlanObjective` (`o`) and `PlanNotes` (`n`) to `KeyMap`, matched in the
planning tab's key switch only when `m.plan.mode == planList`, and surfaced in
`FullHelp`/`ShortHelp` under the existing `k.PlanningMode` gate
(`internal/tui/keymap.go:330`).

**Rationale**: `o` is bound nowhere in the TUI today (verified: no `"o"` binding in
`internal/tui/*.go`). `n` is `NewSub` on the Tasks tab; the planning tab dispatches through
its own switch, so no collision — but a test must pin that `n` on Tasks still creates a
subtask (FR-023). Gating on `planList` satisfies the edge case that `o`/`n` must not open a
second editor over an in-progress form.

---

## D7: Keeping the panes fresh

**Decision**: `planEntriesMsg` (`internal/tui/plan_update.go:25`) gains a `day *planv1.PlanDay`
field. `handlePlanEntriesMsg` applies it under the existing rules — stale-day responses are
already discarded, and background (`bg`) responses must not overwrite the buffer of an open
objective or notes editor.

**Rationale**: The TUI auto-refreshes the plan tab (feature 069) and already carries
day-identity checks in `handlePlanEntriesMsg` (`internal/tui/plan_update.go:518`) and tests
in `autorefresh_apply_test.go` (`TestBackgroundLoad_PlanWrongDayDiscarded`). Threading the
new fields through the message they already ride on means FR-022 falls out of machinery
that exists, and the "don't clobber a draft" rule mirrors how a background load already
preserves `mode` (`TestLoad_PreservesMode`).

---

## D8: CLI surface

**Decision**: `objective` joins the subcommand switch in `runPlan`
(`internal/cli/plan.go:57`), after the existing `--date` parsing so it inherits the flag and
the today-default for free (FR-027). No argument prints `resp.Msg.Day.Objective` and nothing
at all when empty; one argument calls `SetPlanObjective`. Usage text in `printPlanUsage`
gains a line. Notes get no CLI surface (FR-029).

**Rationale**: `--date` is parsed before dispatch, so the subcommand needs no flag handling
of its own — Principle III's "learn one command, predict the others" holds by construction.
Printing bare text with no label keeps the read usable in a shell prompt or script
(spec Assumptions).

**Tone (Principle IV)**: the read path prints the objective verbatim and nothing else —
decoration would break scripting. The confirmation on write and any error text carry the
warm tone, e.g. `Objective set for Thu Aug 6 — go get it.` Errors stay actionable, e.g.
`That objective is a bit long — keep it under 255 characters.`

---

## Facts confirmed in the codebase

| Fact | Location |
|---|---|
| No day-level table exists; days are implied by entries | `services/twig/db/migrations/000006_plan_entries.up.sql` |
| Latest migration is `000014` | `services/twig/db/migrations/` |
| Plan handler holds `*db.Queries` + `*pgxpool.Pool`, has `parseDay`, `validatePlanName` | `services/twig/internal/handler/plan.go:22-64` |
| Planning tab is a two-column `JoinHorizontal` of `paneBox` calls | `internal/tui/view.go:65-155` |
| Right column already swaps details ↔ takeover content by `plan.mode` | `internal/tui/view.go:97-111` |
| Plan modes enum to extend | `internal/tui/model.go:81-88` |
| `keys.Save` = `ctrl+s`, `keys.Cancel` = `esc`, `keys.Editor` = `ctrl+g` | `internal/tui/keymap.go:223-239` |
| External editor helper is mode-agnostic and reusable | `internal/tui/editor.go:34` |
| `editorFinishedMsg` routing already branches by surface | `internal/tui/update.go:1040` |
| `--date` parsed before subcommand dispatch | `internal/cli/plan.go:40-65` |
| Markdown renderer + theme shared by TUI surfaces | `internal/markdown/markdown.go:55`, `internal/tui/details.go:17` |
| Handler tests are integration tests needing `DATABASE_URL` | `AGENTS.md`, `services/twig/internal/handler/*_test.go` |
