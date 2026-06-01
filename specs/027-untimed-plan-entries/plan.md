# Implementation Plan: Untimed Plan Entries

**Branch**: `027-untimed-plan-entries` | **Date**: 2026-06-01 | **Spec**: [spec.md](./spec.md)

**Input**: Feature specification from `/specs/027-untimed-plan-entries/spec.md`

## Summary

Let a plan **task** entry live on a day without a start time ("untimed"), while events stay always-timed. The faithful representation is a **nullable `start_minute`**: the proto fields that carry a start become `optional int32`, the DB column drops `NOT NULL`, and a new CHECK keeps events timed. The CLI's `plan task` / `plan mv` start argument becomes optional and accepts a case-insensitive `null` sentinel; the TUI grows a top-left **untimed pane** (shown only when non-empty), folds untimed entries into the existing unified Up/Down selection, makes the start field optional in the add/edit forms, and adds Tasks-tab shortcuts `p` (send to today, untimed) and `ctrl+p` (date prompt defaulting to tomorrow, untimed). No web frontend changes — `twig-web` consumes only `task.v1`.

## Technical Context

**Language/Version**: Go 1.x (single module at `services/twig/`)

**Primary Dependencies**: ConnectRPC, pgx/v5, sqlc (generated DB layer), Bubble Tea + Lipgloss (TUI), buf (proto codegen)

**Storage**: PostgreSQL — `plan_entries` table (composite PK `user_id, day, id`)

**Testing**: `go test ./...`; handler/CLI/TUI tests use `export_test.go` shims; no running DB required

**Target Platform**: Linux server (`cmd/server`) + terminal CLI/TUI (`cmd/twig`)

**Project Type**: Single Go module, two binaries (server + CLI). Sibling React SPA is **out of scope** (does not use the plan service).

**Performance Goals**: N/A — interactive single-user CLI/TUI and low-volume per-day plan reads

**Constraints**: Must preserve all existing timed-entry behavior; DB change must be a forward migration; generated code (`gen/`, `internal/db/`) only changes via `make proto` / `sqlc generate`

**Scale/Scope**: A handful of plan entries per day per user

## Constitution Check

*GATE: Must pass before Phase 0 research. Re-check after Phase 1 design.*

| Principle | Status | Notes |
|-----------|--------|-------|
| I. Simplicity / YAGNI | ✅ | Nullable `start_minute` is the minimal faithful model. Reuses the existing add/edit forms (start already optional in duration), the existing unified cursor, and the existing grid styling for the untimed pane. No new abstraction layers. |
| II. API-First Design | ✅ | Proto contract delta is written first under `contracts/` (start fields → `optional`), regenerated via `make proto` before handler/CLI/TUI work. |
| III. UI/UX Consistency | ✅ | Untimed pane reuses the grid's dark-box styling and theme tokens (one line per 15 min); `p`/`ctrl+p` follow existing keybinding conventions and appear in help; CLI `null` sentinel mirrors the existing optional-arg style. |
| IV. Playful User Messages | ✅ | New copy (help text, send-to-plan confirmations, invalid-date and empty-selection errors) authored in the warm/playful house tone. |

No violations → Complexity Tracking table omitted.

## Project Structure

### Documentation (this feature)

```text
specs/027-untimed-plan-entries/
├── plan.md              # This file
├── research.md          # Phase 0 — decisions & rationale
├── data-model.md        # Phase 1 — entity + schema delta
├── quickstart.md        # Phase 1 — manual verification script
├── contracts/
│   └── plan-service.md   # Phase 1 — proto contract delta + behavior
└── checklists/
    └── requirements.md   # (from /speckit-specify)
```

### Source Code (repository root)

```text
services/twig/
├── proto/plan/v1/plan.proto                 # start fields → optional int32
├── gen/plan/v1/...                           # regenerated (make proto) — do not hand-edit
├── db/
│   ├── migrations/000007_plan_untimed.up.sql # drop NOT NULL; add event-timed CHECK
│   ├── migrations/000007_plan_untimed.down.sql
│   └── queries/plan.sql                       # nullable start in insert/update; ORDER BY NULLS FIRST
├── internal/
│   ├── db/...                                 # regenerated (sqlc generate) — do not hand-edit
│   ├── handler/plan.go                        # untimed add/move; nil-start mapping; skip overlap when untimed
│   ├── cli/
│   │   ├── plan.go                            # optional/`null` start for task & mv; usage text
│   │   ├── plan_grid.go                       # grid ignores untimed; shared untimed-box renderer
│   │   ├── render.go / plan show              # print untimed list above the grid
│   │   └── timeparse/                         # ParseStartOrNull helper
│   └── tui/
│       ├── keymap.go                          # PlanSendToday (p), PlanSendPickDay (ctrl+p)
│       ├── model.go                            # tasks-tab date-prompt mode; entry split helpers
│       ├── update.go                           # tasks-tab p/ctrl+p handlers; date-prompt reducer
│       ├── plan_update.go                      # optional-start cmds; edit `null`=unschedule
│       └── plan_view.go                        # render untimed pane; layout above grid
```

**Structure Decision**: Single Go module. Changes flow through the existing layered path (proto → handler → db; cli; tui). The React SPA at `services/twig-web/` is untouched because it does not consume `plan.v1`.

## Phase 0 — Research

See [research.md](./research.md). Key decisions:

1. **Untimed representation** = nullable `start_minute` (proto `optional int32`, DB column nullable), not a `-1` sentinel. Presence semantics are explicit and survive the wire.
2. **Events stay timed** via a DB CHECK (`task_id IS NOT NULL OR start_minute IS NOT NULL`) and an unchanged required start in `AddPlanEventRequest`.
3. **Ordering**: `ORDER BY start_minute ASC NULLS FIRST, id ASC` — untimed first, deterministic creation order within the untimed group.
4. **CLI `null` sentinel** parsed in the start position (case-insensitive) so a duration can still be supplied for an untimed entry.
5. **TUI unified cursor** is preserved by keeping all entries in one `m.plan.entries` slice ordered untimed-first; the grid renders only timed entries, the pane only untimed ones, and `SelectedID` highlights whichever pane owns the selection.
6. **`p`/`ctrl+p`** add untimed task entries; `p` always targets *today* (independent of the Planning tab's in-view day); `ctrl+p` opens a `YYYY-MM-DD` prompt defaulting to tomorrow.
7. **Clear** (`DeletePlanEntriesFromMinute`) naturally excludes untimed entries (NULL fails `start_minute >= cutoff`) — desired and requires no change.

## Phase 1 — Design & Contracts

- **Data model**: [data-model.md](./data-model.md) — `plan_entries.start_minute` nullable + event-timed CHECK; `PlanEntry` gains optional start.
- **Contracts**: [contracts/plan-service.md](./contracts/plan-service.md) — `optional int32 start_minute` on `PlanEntry`, `AddPlanTaskRequest`, `MovePlanEntryRequest`; behavioral rules for untimed add / schedule / unschedule.
- **Quickstart**: [quickstart.md](./quickstart.md) — end-to-end manual verification across CLI and TUI.
- **Agent context**: CLAUDE.md SPECKIT marker updated to point at this plan.

## Post-Design Constitution Re-Check

Re-evaluated after Phase 1: still ✅ on all four principles. The contract-first proto delta is captured before implementation; no new abstractions were introduced; the untimed pane and CLI sentinel reuse existing styling/argument conventions; all new copy is slated for tone review.
