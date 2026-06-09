# Implementation Plan: Hide Completed Unscheduled Plan Entries

**Branch**: `044-hide-completed-unscheduled` | **Date**: 2026-06-09 | **Spec**: [spec.md](./spec.md)

**Input**: Feature specification from `/specs/044-hide-completed-unscheduled/spec.md`

## Summary

When an unscheduled (untimed) plan entry's linked task is completed, the entry should disappear from the plan view in both the TUI and the web app. Timed entries and events are unaffected. The completion data is **already on the wire** — `plan.v1.PlanEntry.completed` is populated server-side by `ListPlanEntries` — so this is a pure client-side display filter; no proto/server/DB changes.

Technical approach:
- **Web**: exclude completed untimed entries in the unit-tested `groupPlan` helper (`services/twig-web/src/lib/planView.ts`). Completion in the web app originates outside the planner (task list/detail), so the planner simply filters on its next render/refetch.
- **TUI**: reuse the existing tasks-tab deferred-hide pattern. A new `plan.pendingComplete` (entry id) keeps a just-completed untimed entry visible-and-crossed-out while it is the active highlight; the entry is filtered out on the next render where it is no longer highlighted (cursor move, day change, tab switch). Crossed-out styling already exists (`applyCompletion` → `DimStrike`).

## Technical Context

**Language/Version**: Go 1.x (CLI/TUI + server), TypeScript / React 19 (web)

**Primary Dependencies**: bubbletea v2 + lipgloss (TUI), ConnectRPC (transport), Vite + Vitest (web). No new dependencies.

**Storage**: PostgreSQL via the server — **not touched** by this feature.

**Testing**: `go test ./...` (root TUI module), `npm test` (Vitest) in `services/twig-web/`.

**Target Platform**: Terminal (TUI), modern browser (web SPA).

**Project Type**: Existing multi-module full-stack app (Go CLI/TUI, Go server, React web). Brownfield change to two client surfaces only.

**Performance Goals**: No change — filtering a day's worth of plan entries (tens of items) is negligible.

**Constraints**: Behavior must match across TUI and web for the same data (FR-003); must not delete or mutate plan entries (FR-007, view-time filter only).

**Scale/Scope**: Small. ~1 web helper change + tests; TUI filter + `pendingComplete` plumbing in `internal/tui` + tests. No API surface change.

## Constitution Check

*GATE: Must pass before Phase 0 research. Re-check after Phase 1 design.*

| Principle | Status | Notes |
|-----------|--------|-------|
| I. Simplicity / YAGNI | ✅ | Reuses the existing `pendingComplete` deferred-hide pattern and existing crossed-out styling; web change is one predicate in `groupPlan`. No new abstractions. |
| II. API-First Design | ✅ | No contract change. The contract already exposes `PlanEntry.completed` (populated by `ListPlanEntries`); `contracts/` documents the existing field this feature consumes. |
| III. UI/UX Consistency | ✅ | TUI mirrors the tasks-tab completion interaction (crossed-out + highlighted, hide on navigate-away) and reuses the shared theme/strikethrough; web reuses the existing `PlanEntryRow` styling. |
| IV. Playful User Messages | ✅ | No new user-facing copy. Existing completion notices ("done and dusted") are unchanged; hiding introduces no new text. |

**Result**: PASS (initial and post-design). No entries in Complexity Tracking.

## Project Structure

### Documentation (this feature)

```text
specs/044-hide-completed-unscheduled/
├── plan.md              # This file
├── research.md          # Phase 0 output
├── data-model.md        # Phase 1 output
├── quickstart.md        # Phase 1 output
├── contracts/           # Phase 1 output (documents the existing PlanEntry.completed contract)
└── tasks.md             # Phase 2 output (/speckit-tasks — NOT created here)
```

### Source Code (repository root)

```text
internal/tui/                  # TUI (root module)
├── model.go                   # planState — add pendingComplete *int32
├── update.go                  # plan key handling: set pendingComplete on complete;
│                              #   clear it on cursor move / day nav / tab switch / go-to-task
├── plan_update.go             # handlePlanEntriesMsg — apply the untimed-completed filter
├── plan_view.go               # renderPlanning* — renders via splitPlanEntries (no logic change expected)
└── *_test.go                  # TUI table/golden tests for the new behavior

internal/cli/
└── plan_grid.go               # RenderUntimed + applyCompletion/DimStrike (already crosses out — reused as-is)

services/twig-web/src/
├── lib/planView.ts            # groupPlan — exclude completed untimed entries
├── lib/planView.test.ts       # unit tests for the filtered grouping
└── pages/PlanPage.tsx         # consumes grouped.untimed (no change expected)

api/proto/plan/v1/plan.proto   # UNCHANGED — PlanEntry.completed already exists
services/twig/                 # UNCHANGED — server already populates completed
```

**Structure Decision**: Brownfield edit to the two existing client surfaces. The TUI change lives in `internal/tui` (state + filter + navigation reset), reusing rendering in `internal/cli/plan_grid.go`. The web change lives in the existing unit-tested `services/twig-web/src/lib/planView.ts`. The server module and `api/` proto are intentionally untouched.

## Complexity Tracking

> No Constitution Check violations — table intentionally empty.
