# Implementation Plan: Plan Span Markers

**Branch**: `009-plan-span-markers` | **Date**: 2026-05-22 | **Spec**: [spec.md](./spec.md)

**Input**: Feature specification from `/specs/009-plan-span-markers/spec.md`

## Summary

Replace the per-slot repetition of the task name in the `todo plan` grid view with box-drawing span markers. The first slot of each contiguous block shows `┌ <id> <name>`; intermediate slots show `│`; the last slot shows `└`; single-slot tasks show `─ <id> <name>`. Free slots are unchanged. Implementation is a localized rewrite of the rendering loop in `services/todo/internal/cli/plan_grid.go` — no API, no protobuf, no data-model changes.

## Technical Context

**Language/Version**: Go 1.22 (existing module at `services/todo/`)

**Primary Dependencies**: standard library only for this feature (`fmt`, `strings`); existing generated `planv1` package for the `PlanEntry` input type

**Storage**: N/A (pure rendering)

**Testing**: `go test ./internal/cli/...` — table-driven tests already exist in `plan_grid_test.go`

**Target Platform**: CLI binary (`cmd/todo`) on Linux/macOS terminals with UTF-8 locales

**Project Type**: CLI (single Go module, multi-binary)

**Performance Goals**: N/A — render runs once per `todo plan` invocation on at most a single day's entries (~100 rows)

**Constraints**: Output must remain stable for the snapshot tests in `plan_grid_test.go`; updated snapshots will need to reflect the new visual.

**Scale/Scope**: One function edit in one file plus its test file. No new files in `services/todo/` source tree.

## Constitution Check

*GATE: Must pass before Phase 0 research. Re-check after Phase 1 design.*

### Principle I — Simplicity / YAGNI

PASS. The change is a localized rewrite of one rendering function. No new types, no new packages, no abstractions. The block-grouping logic is computed inline during the existing single-pass loop — no separate "block" model is introduced because the rendering pass already iterates time slots and looks up the owning entry per slot. The only added state is "is this the entry's last slot?" which is derived from the entry's existing `EndMinute`.

### Principle II — API-First Design

PASS (vacuously). This feature has no API surface: no protobuf change, no handler change, no contract between frontend and backend. The change is confined to CLI rendering. The output format is a user-facing visual contract documented in the spec's Functional Requirements and acceptance scenarios — that *is* the contract for this feature.

No violations. Complexity Tracking table not needed.

## Project Structure

### Documentation (this feature)

```text
specs/009-plan-span-markers/
├── plan.md              # This file
├── spec.md              # Feature specification (done)
├── research.md          # Phase 0 — see below
├── data-model.md        # Phase 1 — N/A (no new entities; documented as such)
├── quickstart.md        # Phase 1 — manual verification script
├── contracts/           # N/A for this feature (CLI rendering only)
└── tasks.md             # Phase 2 output of /speckit-tasks (not created here)
```

### Source Code (repository root)

```text
services/todo/
└── internal/
    └── cli/
        ├── plan_grid.go         # MODIFY: rewrite RenderGrid's per-slot cell logic
        └── plan_grid_test.go    # MODIFY: update existing snapshots; add new cases
                                 #   (single-slot, two-slot block, adjacent blocks)
```

**Structure Decision**: Modify the two existing files in `services/todo/internal/cli/`. No new files. The existing `RenderGrid` function in `plan_grid.go` is the only production-code site that needs to change. Its test file gains additional table-driven cases for the new markers.

## Complexity Tracking

> No constitution violations. Table omitted.
