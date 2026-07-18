# Implementation Plan: Add New Tasks to a Plan

**Branch**: `066-add-task-to-plan` | **Date**: 2026-07-17 | **Spec**: [spec.md](./spec.md)

**Input**: Feature specification from `/specs/066-add-task-to-plan/spec.md`

## Summary

Put a plan control on the create-task form of both the TUI and the web app, so a task that needs
doing today reaches today's plan in the same save that creates it — instead of requiring the user to
find the task again and schedule it as a second act.

The control offers **No plan** (the default), **Today**, **Tomorrow**, and any specific date. In the
TUI it is a left/right cycling field copying the Goal selector, with the existing `ctrl+g` calendar
for arbitrary dates. On the web it is a segmented radio row with a revealed date input.

**The technical shape of this work is: no server, no proto, no schema — two existing RPCs called in
sequence from the client.** `AddPlanTask` already creates untimed entries on any day. Everything
below is form state and one chained call, following patterns each surface already contains.

## Technical Context

**Language/Version**: Go 1.x (CLI/TUI); TypeScript 5 / React 19 (web)

**Primary Dependencies**: Bubble Tea v2 + Bubbles (TUI); ConnectRPC (transport); TanStack Query +
connect-query (web data); Tailwind (web styling). **No new dependency on either surface.**

**Storage**: PostgreSQL — **untouched**. No migration, no `sqlc generate`, no query change.

**Testing**: `go test ./...` from repo root (TUI, table-driven over `Update`, `export_test.go`
shims); `npm test` in `services/twig-web/` (Vitest + Testing Library). No integration-test
infrastructure exists and this feature does not need any.

**Target Platform**: Linux/macOS terminal (TUI); evergreen browsers (web)

**Project Type**: Multi-module full-stack — TUI/CLI client + React SPA against a shared Go server.
This feature touches **client tiers only**.

**Performance Goals**: No new perceptible cost. Saving with a day selected adds exactly one RPC to
an action that already makes one; saving without touching the control adds none.

**Constraints**: Task creation must remain unchanged for anyone who ignores the control (SC-003).
The plan write may never cost the user the task (FR-006/FR-007).

**Scale/Scope**: Two form surfaces; ~4 source files changed per surface plus tests. No new
component on the TUI side, one modified component on the web side.

## Constitution Check

*GATE: Must pass before Phase 0 research. Re-check after Phase 1 design.*

### Initial check (pre-research)

| Principle | Status | Notes |
|-----------|--------|-------|
| I. Simplicity / YAGNI | ✅ | Reuses `AddPlanTask` rather than adding a combined RPC; extends the existing form rather than adding a component. No abstraction introduced — see Complexity Tracking (empty). |
| II. API-First Design | ✅ | No wire contract changes. Contracts (`contracts/ui-contract.md`, `contracts/rpc-usage.md`) are written and committed before implementation. |
| III. UI/UX Consistency | ✅ | TUI copies the existing Goal selector and reuses `keys.Calendar` — **no new key binding**, no ad-hoc styling. Web uses existing tokens/`Button` and `lib/planDays.ts`. Both surfaces present the same four choices in the same order. |
| IV. Playful User Messages | ✅ | Existing tone-compliant plan copy is reused verbatim. Exactly one new string (partial failure); tone contract set in `contracts/ui-contract.md` §4. |

### Post-design re-check (after Phase 1)

| Principle | Status | Notes |
|-----------|--------|-------|
| I. Simplicity / YAGNI | ✅ | Design added no layer. The one thing considered and rejected — a hidden `textinput` backing the Plan field to reuse `writeDateField` (research R3) — was rejected *because* it was indirection. `planChoiceDate` as a non-cycling destination is the simplest thing that renders a picked date in a cycling field. |
| II. API-First Design | ✅ | `contracts/rpc-usage.md` pins the exact `AddPlanTask` field values (`start_minute` absent, `duration_minute: 0`) and the `FAILED_PRECONDITION` handling, before any code. |
| III. UI/UX Consistency | ✅ | Confirmed against the code: `fieldLabel` for rendering, `cycleFocus`'s existing hidden-field skipping, `keys.Calendar` extended rather than duplicated. Web control mirrors the TUI's choices and order (spec US3). |
| IV. Playful User Messages | ✅ | Copy contract distinguishes messages (playful) from control labels (literal — "playfulness MUST NOT obscure meaning"). |

**Gate result: PASS.** No violations; Complexity Tracking table is empty.

## Project Structure

### Documentation (this feature)

```text
specs/066-add-task-to-plan/
├── plan.md                    # This file
├── spec.md                    # Feature specification
├── research.md                # Phase 0 — reuse decisions, code anchors
├── data-model.md              # Phase 1 — transient UI state (no persisted change)
├── quickstart.md              # Phase 1 — how to build, run, and see it work
├── contracts/
│   ├── ui-contract.md         # Phase 1 — user-facing + component contract, CT-01..CT-14
│   └── rpc-usage.md           # Phase 1 — how the existing RPCs must be called
├── checklists/
│   └── requirements.md        # Spec quality checklist (16/16)
└── tasks.md                   # Phase 2 — NOT created by /speckit-plan
```

### Source Code (repository root)

Only client code changes. The `services/twig` server module and the `api` module are **not touched**.

```text
internal/tui/                          # root module — TUI
├── edit.go                            # MODIFY: focusPlan, planIdx/planDate/showPlanField,
│                                      #   cycle + calendar handling, render, dirty tracking,
│                                      #   planDay on editSavedMsg + buildSaveMsg resolution
├── update.go                          # MODIFY: createTaskCmd takes planClient, chains AddPlanTask
│                                      #   after CreateTask; partial-failure message
├── export_test.go                     # MODIFY: shims for the new form state
├── edit_test.go                       # MODIFY/ADD: CT-01,02,03,06,10,12,13
└── plan_from_create_test.go           # ADD: CT-04,05,08,09 (save-path/RPC ordering)

services/twig-web/src/                 # web SPA
├── components/
│   ├── TaskForm.tsx                   # MODIFY: showPlanControl prop, segmented control,
│   │                                  #   widened onSubmit
│   ├── TaskForm.test.tsx              # MODIFY/ADD: CT-01,02,03,05,14
│   └── TreeRow.tsx                    # MODIFY: showPlanControl on the sub-task form; plan after create
├── pages/
│   ├── TaskTreePage.tsx               # MODIFY: showPlanControl on the root form; plan after create
│   └── TaskTreePage.test.tsx          # MODIFY/ADD: CT-08,09,11
└── theme/messages.ts                  # MODIFY: one new string (partial failure)

# Unchanged, listed to be explicit about the blast radius:
#   api/proto/**, api/gen/**           — no proto change, no `make proto`
#   services/twig/**                   — no handler, query, or migration change
#   services/twig-web/src/pages/TaskDetailPage.tsx  — edit form keeps no plan control (FR-011)
#   services/twig-web/src/components/AddToPlanControl.tsx — existing task-row path untouched
```

**Structure Decision**: The repo's existing three-module layout (root CLI/TUI, `api/`,
`services/twig/`) plus the `services/twig-web/` SPA is unchanged. This feature adds no directory and
no module. Work lands in `internal/tui/` for the terminal and `services/twig-web/src/` for the
browser, which is exactly where each surface's create form already lives.

## Implementation Sequence

Ordered so that each step is independently verifiable, and so the P1 slice (spec US1) is complete
and shippable before US2/US3 work begins.

1. **TUI form state** — `focusPlan` + renumbering, `planIdx`, `showPlanField`, cycling, rendering,
   dirty tracking. Verifiable via `edit_test.go` with no RPC involved.
2. **TUI save path** — `planDay` on `editSavedMsg`, `createTaskCmd` gains `planClient` and chains
   `AddPlanTask`, partial-failure handling. Completes US1 for the TUI.
3. **TUI calendar** — extend `keys.Calendar` to `focusPlan`, seed from `planDate`, own render path.
   Completes US2 for the TUI.
4. **Web control** — `showPlanControl` prop, segmented radio row, date input, widened `onSubmit`.
   Verifiable in `TaskForm.test.tsx` alone.
5. **Web call sites** — `TaskTreePage` and `TreeRow` plan-after-create + dual invalidation; new
   message string. Completes US1/US2 for the web.
6. **Parity pass** — US3: confirm both surfaces' choices, order, and default match; run both suites.

**Ordering constraint**: step 2 depends on step 1 (needs `planDay` to exist); step 5 depends on step
4. TUI (1–3) and web (4–5) are otherwise independent and could proceed in parallel.

## Risks

| Risk | Mitigation |
|---|---|
| `focusPlan` renumbers `focusGoal`/`focusSave`/`focusCancel`/`focusCount` | The constants are used symbolically throughout `edit.go`; grep for numeric literals against these before landing. Existing form tests cover focus traversal and will catch a mistake. |
| `openCalendar` seeds from a focused *text input*; the Plan field has none | Called out in research R3 as the one place existing plumbing does not transfer. Plan field seeds from `planDate` (or today) and gets its own render path instead of `writeDateField`. |
| Partial failure (task created, plan not) leaves the user unsure what happened | FR-007 + explicit message contract (`ui-contract.md` §4). The task is never rolled back; CT-09 covers both surfaces. |
| Widening `onSubmit` could disturb the edit call site | `showPlanControl` defaults to false and TS allows a 2-arg handler to satisfy a 3-arg type — `TaskDetailPage` needs no change. CT-07 asserts the control is absent there. |

## Complexity Tracking

> **Fill ONLY if Constitution Check has violations that must be justified**

No violations. No abstraction, layer, module, or dependency is introduced by this feature.
