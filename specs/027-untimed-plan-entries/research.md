# Phase 0 Research: Untimed Plan Entries

All Technical Context items were known from the existing codebase; no NEEDS CLARIFICATION remained after `/speckit-clarify`. This document records the design decisions that shape Phase 1.

## D1 — How to represent "no start time"

**Decision**: Make `start_minute` nullable end-to-end — DB column `DROP NOT NULL`, proto fields become `optional int32 start_minute` (proto3 presence → `*int32` in Go).

**Rationale**: `start_minute = 0` is a valid time (midnight), so a sentinel value cannot mean "unset" without colliding with real data. Proto3 `optional` gives explicit presence (`HasStartMinute()` / nil pointer), and a nullable SQL column is the natural storage. This is the simplest faithful model (Principle I) and keeps the wire contract self-describing (Principle II).

**Alternatives considered**:
- *`-1` / `9999` sentinel on a NOT NULL column*: avoids a migration but leaks an in-band magic value into every reader, query, and validation. Rejected as error-prone.
- *Separate `untimed_entries` table*: doubles the query/handler surface for what is one optional field. Over-engineered; rejected.

## D2 — Keeping events always-timed

**Decision**: Enforce at the DB with `CHECK (task_id IS NOT NULL OR start_minute IS NOT NULL)`, and leave `AddPlanEventRequest.start_minute` a required (non-optional) field so the event path is unchanged.

**Rationale**: The "events need a time" rule (FR-003) is a data invariant; expressing it as a CHECK makes it impossible to violate regardless of caller. Because only `AddPlanTask` and `MovePlanEntry` need an optional start, `AddPlanEvent` keeps its current validation untouched (Principle I — minimal change).

**Alternatives considered**: Handler-only validation — weaker (no DB guarantee) and easy to miss in a future caller. Rejected as the sole mechanism (handler still returns a friendly error before hitting the constraint).

## D3 — Ordering of entries

**Decision**: `ListPlanEntriesForDay` → `ORDER BY start_minute ASC NULLS FIRST, id ASC`.

**Rationale**: Puts untimed entries first (matching their visual position above the grid) and orders them by `id` = creation order (FR-005). Timed entries keep their `start_minute` ordering. The client still partitions by presence, so this is primarily about determinism.

## D4 — CLI optional start + `null` sentinel

**Decision**: `twig plan task <id> [start|null] [dur|end]` and `twig plan mv <n> [start|null] [dur|end]`.
- Start omitted → untimed, server default duration.
- `null` (case-insensitive) in the start slot → untimed, with the optional trailing arg parsed as a **duration** (an end time is meaningless without a start, per the spec's Assumptions).
- A real time in the start slot → timed (existing behavior).

A small `timeparse.ParseStartOrNull(s) (minute int, timed bool, err error)` helper centralizes the `null` check so `plan task` and `plan mv` stay consistent (Principle III — predictable across subcommands).

**Rationale**: Matches FR-006/FR-007/FR-008 exactly while reusing the existing positional-argument shape. `null` is needed only to disambiguate "I want to set a duration but no start."

## D5 — TUI selection model with two panes

**Decision**: Keep a single `m.plan.entries` slice (ordered untimed-first by D3) and a single `m.plan.cursor`. The untimed pane renders the leading untimed entries; the grid renders the timed ones; `SelectedID = entries[cursor].Id` is passed to both renderers, so only the pane containing the selection highlights it.

**Rationale**: This delivers the unified Up/Down cycle (clarified) for free — Up/Down already walks `m.plan.entries`, and edit/remove/move already key off `entries[cursor]`. No separate focus state or focus-switch key (matches the clarification). Minimal change (Principle I).

**Alternatives considered**: Two separate slices + a focused-pane enum + a Tab binding. More state, more keys, contradicts the chosen "unified cycle" clarification. Rejected.

## D6 — Rendering untimed entries like grid boxes

**Decision**: Add a shared `RenderUntimed(entries, width, isTTY, opts)` in `internal/cli` that stacks each untimed entry as a dark box one line tall per 15 minutes of duration, reusing the grid's existing style constants/theme tokens and `SelectedID` highlight semantics. Used by both the TUI untimed pane and the CLI `plan show` output (printed above the grid).

**Rationale**: One renderer for both surfaces keeps the visual language identical (Principle III) and avoids duplicating box styling. Sharing the renderer (rather than refactoring the grid's time-slot internals) is the lighter touch (Principle I).

**Alternatives considered**: Reuse `RenderGrid` with a synthetic time — pollutes the grid window and misrepresents the entry as scheduled. Rejected.

## D7 — Tasks-tab `p` / `ctrl+p`

**Decision**:
- `p`: add an untimed task entry to **today's** plan for the highlighted task (today = current local date, independent of the Planning tab's in-view day). Show a playful confirmation.
- `ctrl+p`: open an inline `YYYY-MM-DD` text prompt pre-filled with **tomorrow**; on submit, validate the date and add an untimed entry to that day. Cancel or invalid date → no-op with a friendly message.

New keymap bindings `PlanSendToday` (`p`) and `PlanSendPickDay` (`ctrl+p`); a new tasks-tab view mode hosts the date prompt (mirrors existing edit-form modes). `p` and `ctrl+p` are currently unused on the Tasks tab.

**Rationale**: Implements FR-016/FR-017/FR-018 and the clarified date-prompt behavior using the existing text-input + mode pattern already used by task edit forms (Principle III consistency).

## D8 — Interactions that need no change

- **Clear** (`DeletePlanEntriesFromMinute`, `start_minute >= cutoff`): NULL comparisons are unknown/false, so untimed entries are never cleared. Desired and free.
- **Existing CHECK constraints** (`start_minute BETWEEN 0 AND 1439`, `start_minute + duration_minute <= 1440`): pass when `start_minute` is NULL (unknown), so they continue to guard timed rows only.
- **`twig-web`**: consumes `task.v1` only; no changes.
