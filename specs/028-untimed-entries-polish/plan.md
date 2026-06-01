# Implementation Plan: Untimed Plan Entries — Polish & Bugfixes

**Branch**: `028-untimed-entries-polish` | **Date**: 2026-06-01 | **Spec**: [spec.md](./spec.md)

**Input**: Feature specification from `/specs/028-untimed-entries-polish/spec.md`

## Summary

Four follow-up fixes to the untimed plan entries shipped in feature 027, none of which touch the proto contract or the database schema:

1. **Status-bar feedback** when a task is sent to a plan from the Tasks tab (`p` / `ctrl+p`). The TUI has no info/notice channel today — `renderStatus` shows only `m.err` (errors) or the help line. Add a transient notice field surfaced on the active tab, set on a successful send and on a duplicate-rejection.
2. **Prevent duplicate untimed entries.** Enforce "at most one untimed entry per (day, task)" **server-side in the handler**, so every path (Tasks-tab `p`/`ctrl+p`, CLI `plan task`, and clearing a start via `plan mv`/TUI edit) inherits it. The check piggybacks on the existing `LockPlanEntriesForDay` scan inside the serializable transaction — no new SQL query.
3. **Fix untimed-entry rendering.** `cli.RenderUntimed` draws its own simplified box geometry (label line `┣label┫` as the *first* row, no `┏┓` top border, independent `┗┛` bottoms) which is why titles overlap the top border, adjacent boxes don't join, and the last line's colors are off. Re-render untimed entries with the **same box geometry the grid uses** (`┏┓` top, `┃┃` interior, shared `┣┫` boundaries between adjacent entries, `┗┛` bottom) so they are pixel-identical to gridded entries.
4. **Visual separation** between the untimed pane and the day planner grid, present only when the untimed pane is non-empty.

No web frontend changes — `twig-web` consumes only `task.v1`.

## Technical Context

**Language/Version**: Go 1.x (single module at `services/twig/`)

**Primary Dependencies**: ConnectRPC, pgx/v5, sqlc (generated DB layer), Bubble Tea + Lipgloss (TUI)

**Storage**: PostgreSQL — `plan_entries` table (composite PK `user_id, day, id`). **No schema change.**

**Testing**: `go test ./...`; handler/CLI/TUI tests use `export_test.go` shims; no running DB required

**Target Platform**: Linux server (`cmd/server`) + terminal CLI/TUI (`cmd/twig`)

**Project Type**: Single Go module, two binaries (server + CLI). React SPA at `services/twig-web/` is out of scope (does not consume `plan.v1`).

**Performance Goals**: N/A — interactive single-user CLI/TUI, low-volume per-day plan reads

**Constraints**: No proto or DB schema change; all existing timed-entry behavior preserved; generated code (`gen/`, `internal/db/`) only via `make proto` / `sqlc generate` (neither expected this feature)

**Scale/Scope**: A handful of plan entries per day per user

## Constitution Check

*GATE: Must pass before Phase 0 research. Re-check after Phase 1 design.*

| Principle | Status | Notes |
|-----------|--------|-------|
| I. Simplicity / YAGNI | ✅ | Duplicate check reuses the existing `LockPlanEntriesForDay` scan (no new query); rendering fix **deletes** the bespoke `RenderUntimed` geometry in favor of the grid's existing box code; the notice field is one model string. No new abstractions. |
| II. API-First Design | ✅ | No proto change. The one behavioral contract delta (a new duplicate-rejection error on `AddPlanTask`/`MovePlanEntry`) is documented in `contracts/` before implementation. |
| III. UI/UX Consistency | ✅ | The whole point of fix #3 is to make untimed entries render through the **same** grid box code/theme tokens; the notice reuses the existing status-bar area; the separator uses existing border glyphs/theme colors. |
| IV. Playful User Messages | ✅ | New copy (send confirmation, duplicate-rejection) authored in the warm/playful house tone, reviewed against Principle IV. |

No violations → Complexity Tracking table omitted.

## Project Structure

### Documentation (this feature)

```text
specs/028-untimed-entries-polish/
├── plan.md              # This file
├── research.md          # Phase 0 — decisions & rationale
├── data-model.md        # Phase 1 — entity invariant (no schema change)
├── quickstart.md        # Phase 1 — manual verification script
├── contracts/
│   └── plan-service.md   # Phase 1 — behavioral delta (duplicate-untimed rejection)
└── checklists/
    └── requirements.md   # (from /speckit-specify)
```

### Source Code (repository root)

```text
services/twig/
├── internal/
│   ├── handler/
│   │   └── plan.go            # FR-003/004/006: reject 2nd untimed entry for a (day,task)
│   │                          #   in AddPlanTask (untimed branch) and MovePlanEntry (clear branch)
│   ├── cli/
│   │   └── plan_grid.go       # FR-007/008/009/010: re-render untimed entries with grid box geometry
│   │                          #   (shared ┏┓/┃┃/┣┫/┗┛); FR-011 separator above the grid
│   └── tui/
│       ├── model.go           # notice field (info channel distinct from err)
│       ├── update.go          # p/ctrl+p: pass notice context into the send cmd
│       ├── plan_update.go     # addPlanTaskCmd carries success notice; route send errors to active tab
│       └── view.go            # renderStatus: show notice when present; plan_view separator wiring
└── (no proto/, gen/, db/ changes expected)
```

**Structure Decision**: Single Go module. Three independent change sites — handler (duplicate rule), cli renderer (visual fixes + separator), tui (notice channel) — matching the spec's independently-testable user stories. No regeneration of `gen/` or `internal/db/`.

## Phase 0 — Research

See [research.md](./research.md). Key decisions:

1. **Duplicate rule lives in the handler**, enforced inside the existing serializable transaction by scanning the already-locked day entries for an untimed entry with the same `task_id` (excluding self on Move). One rule, all client paths (CLI + TUI) covered, no new SQL.
2. **Rendering fix = reuse the grid's box geometry**, not patch `RenderUntimed`'s bespoke drawing. The bugs are structural (missing `┏┓` top, no shared `┣┫` joins, divergent last-line styling); matching the grid path eliminates all three at once and guarantees parity (FR-010).
3. **Notice is a new model field** (`notice string`) separate from `m.err`; `renderStatus` shows it when set and no error is active. Send feedback (success or duplicate-rejection) must surface on the **Tasks** tab, so the result is routed to the tab-agnostic notice/err rather than the planning-only `m.plan.err`.
4. **No auto-dismiss** (explicitly out of scope): the notice persists until the next action replaces/clears it, matching existing error behavior.
5. **Separator placement**: a single divider line emitted between the untimed pane and the grid, only when untimed entries exist; absent otherwise so the no-untimed layout is byte-for-byte unchanged (FR-012).

## Phase 1 — Design & Contracts

- **Data model**: [data-model.md](./data-model.md) — no schema change; documents the new `(day, task)` at-most-one-untimed invariant and where it is enforced.
- **Contracts**: [contracts/plan-service.md](./contracts/plan-service.md) — behavioral delta only: `AddPlanTask` (untimed) and `MovePlanEntry` (clear-start) now return a `FailedPrecondition` duplicate-untimed error; request/response messages unchanged.
- **Quickstart**: [quickstart.md](./quickstart.md) — manual verification across CLI and TUI for all four fixes.
- **Agent context**: CLAUDE.md SPECKIT marker updated to point at this plan.

## Post-Design Constitution Re-Check

Re-evaluated after Phase 1: still ✅ on all four principles. No proto/schema surface added; the rendering change consolidates onto existing grid code (a net simplification); duplicate enforcement reuses existing locking; new copy is queued for tone review.
