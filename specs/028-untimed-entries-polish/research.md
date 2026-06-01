# Phase 0 Research: Untimed Plan Entries — Polish & Bugfixes

All four items are refinements to the 027 implementation. None requires a proto or DB schema change. The research below resolves the "how" for each, grounded in the current code.

## 1. Where to enforce "at most one untimed entry per (day, task)"

**Decision**: Enforce in the **handler** (`internal/handler/plan.go`), inside the existing serializable transaction, by scanning the day's already-locked entries for an untimed entry (`StartMinute` invalid) with the same `task_id`.

- `AddPlanTask` untimed branch (`isTimed == false`): today it skips the `LockPlanEntriesForDay` lock entirely (only the timed branch locks for overlap). Add a lock + scan; if any locked entry has `TaskID == req.TaskId` and `!StartMinute.Valid`, reject.
- `MovePlanEntry` clear branch (`isTimed == false`): the entry being moved has a `task_id` (`existing.TaskID`). Lock the day and scan for **another** untimed entry (id ≠ this one) with the same `task_id`; reject if found.

**Rationale**:
- Both the CLI (`twig plan task`, `twig plan mv`) and the TUI (`p`, `ctrl+p`, edit-clear) call the same RPCs, so a single server-side rule covers every path in the spec (FR-006) with no client duplication.
- The serializable transaction + row lock already in place makes the check race-free against a concurrent second add.
- Reuses `LockPlanEntriesForDay` (already returns the full day's rows) — **no new sqlc query**, honoring Principle I.

**Alternatives considered**:
- *Unique partial index* (`UNIQUE (user_id, day, task_id) WHERE start_minute IS NULL`): enforces the invariant at the DB and is elegant, but it is a schema change (migration + regenerated error-mapping) and surfaces as a generic constraint-violation that must be translated to a friendly message anyway. Rejected to keep this feature schema-free; can be revisited if the app ever bypasses the handler.
- *Client-side check in CLI/TUI*: would duplicate the rule in two places and still race. Rejected.

**Error shape**: `connect.CodeFailedPrecondition` with playful copy (e.g. "That one's already parked here without a time — it can only wait in one spot."). Mirrors the existing overlap rejection style.

## 2. Fixing the untimed-entry rendering

**Decision**: Replace `RenderUntimed`'s bespoke box drawing with the **grid's box geometry** so untimed entries render identically to gridded ones (FR-007–FR-010).

**Root cause** (current `RenderUntimed`, `internal/cli/plan_grid.go`):
- It treats `rows = duration/15` and makes **row 0 the label line** `│ ┣label┫ │` — there is *no* `┏━━━┓` top border, so the title visually merges with the pane's top border (the "title overlaps the top border" bug).
- Each entry emits its own `┗━━━┛` bottom and the next entry starts fresh — adjacent entries never share a `┣━━━┫` boundary the way the grid does (the "boxes don't join" bug).
- Selection/last-line styling diverges from the grid's per-line handling (the "color/background wrong on the last line" bug).

The grid (`RenderGrid`) already solves all of this with a per-line model: `┏┓` top, `┃content┃` interior, shared `┣┫` between abutting entries, `┗┛` bottom, and consistent `applySelection`/`applyCompletion` on every line.

**Approach**: Render the stacked untimed entries through the same line-drawing logic the grid uses for entry boxes — laying entries out back-to-back (each entry's bottom boundary shared with the next entry's top, exactly like adjacent grid entries) — rather than maintaining a parallel renderer. Factor the shared box-line drawing so both call it. The untimed pane keeps the grid's geometry constants (`boxWidth`, `contentWidth`, gutter) which it already mirrors.

**Rationale**: The defects are structural, not cosmetic; matching the grid path fixes all three simultaneously and makes future drift impossible (Principle III). It is also a net code reduction (delete the divergent branch).

**Alternatives considered**:
- *Patch `RenderUntimed` line-by-line* (add a `┏┓` row, special-case joins, fix last-line color): more code, still a second renderer that can drift. Rejected.

## 3. Status-bar feedback channel in the TUI

**Decision**: Add a `notice string` field to the model (info channel) distinct from `err error`. `renderStatus` (`internal/tui/view.go`) shows the notice when set and no error is active; otherwise the existing error/help behavior is unchanged.

**Why a new field**: `renderStatus` today renders only `m.err`/`m.plan.err` (as "error: …") or the help line — there is no non-error message path. A success confirmation must not be styled as an error.

**Routing for the send path**: On the Tasks tab, `p`/`ctrl+p` issue `addPlanTaskCmd`, whose result returns as `planMutatedMsg`. Currently a failure sets `m.plan.err`, which `renderStatus` only shows when `activeTab == tabPlanning` — so on the Tasks tab a duplicate rejection would be **invisible**. Fix:
- Carry a success notice from the send into `planMutatedMsg` (the keypress site knows the task name and target day), set `m.notice` on success.
- Route the send's error to the tab-agnostic `m.err` (or the notice line) so the duplicate rejection is visible on the Tasks tab (FR-005, US1 scenario 3).

**Persistence**: The notice persists until the next user action clears/replaces it (no timer). Auto-dismiss is explicitly out of scope (FR-013); revisit alongside a general status-message-timeout change.

**Copy** (Principle IV): e.g. `p` → "Tucked '<task>' into today's plan."; `ctrl+p` → "Tucked '<task>' into <date>'s plan."; duplicate → the rejection message from item 1.

## 4. Visual separation between the untimed pane and the day planner

**Decision**: Emit a single separator line between the untimed pane output and the grid, only when the untimed pane is non-empty. When there are no untimed entries the pane and separator are both absent and the layout is unchanged (FR-012).

**Form**: A horizontal divider built from the existing border glyphs/theme tokens spanning the grid width (exact glyph is a design detail; it must read as a clear boundary and stay within the established palette per Principle III). Wired where `RenderUntimed` output is concatenated with the grid (`plan_view.go`: `combined := RenderUntimed(...) + grid`).

**Rationale**: Lowest-risk, theme-consistent way to delimit the two regions; trivially conditional on pane non-emptiness so the common no-untimed case is byte-for-byte unchanged.

## Summary of cross-cutting decisions

- **No proto change, no DB migration, no `sqlc`/`buf` regeneration.**
- One handler rule (reusing existing locking) covers all duplicate-prevention paths.
- The rendering fix consolidates onto the grid's box code rather than maintaining a second renderer.
- The TUI gains a minimal info/notice channel; auto-dismiss is deferred.
