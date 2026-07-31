# Implementation Plan: Task Descriptions on the Planning Tab

**Branch**: `071-plan-task-descriptions` | **Date**: 2026-07-31 | **Spec**: [spec.md](./spec.md)

**Input**: Feature specification from `/specs/071-plan-task-descriptions/spec.md`

## Summary

Show the linked task's markdown description in the Planning tab's read-only Details pane, rendered exactly as it is on the Tasks tab.

The approach is deliberately small. `renderPlanDetail` (`internal/tui/plan_view.go:304`) already receives both the linked `*taskv1.Task` and the shared `*markdown.Renderer` — it just never reads `Description`. Appending a description block there, using the same four-line render-or-fall-back pattern `renderDetails` uses, satisfies the whole of user stories 1 and 2 without a signature change.

User story 3 needs one genuine addition. `paneBox` applies `lipgloss.Height`, which pads short content but does **not** truncate tall content (verified empirically — see research R4). The detail pane has never overflowed only because it emits about five short lines today; a multi-paragraph description would grow the right pane past the left, misalign the horizontal join, and push the status line off screen. So the styled Planning path clamps its right-pane content to the pane's inner height before boxing. The plain-text path already clamps via `splitLines` and is left alone.

## Technical Context

**Language/Version**: Go 1.24 (root module `github.com/pboyd/twig`)

**Primary Dependencies**: `charm.land/lipgloss/v2` (layout, styling), `github.com/yuin/goldmark` via `internal/markdown` (GFM rendering), Bubble Tea (TUI runtime). No new dependency.

**Storage**: N/A — no schema change, no migration. `task.description` already exists and is already fetched.

**Testing**: `go test ./internal/tui/` — pure-function unit tests, no database, no TTY, no running server.

**Target Platform**: Terminal (Linux/macOS), styled and plain-text paths both supported.

**Project Type**: TUI feature in the root CLI module. Server, web frontend, and `api/` are untouched.

**Performance Goals**: No perceptible lag when moving the plan cursor (SC-005). Free by construction: `markdown.Renderer` caches on `{text, width, styled, inline}` (`markdown.go:56`), so repeat renders of the same description are a map lookup.

**Constraints**: Rendered output must be identical to the Tasks tab at equal width (SC-003). Total view height must never exceed the terminal height (SC-004). Unstyled path must emit no ANSI codes.

**Scale/Scope**: Three files touched, roughly 20 lines of production code plus tests. No new model field, no new user-facing string.

## Constitution Check

*GATE: Must pass before Phase 0 research. Re-checked after Phase 1 design — still passing, unchanged.*

| Principle | Status | Notes |
|-----------|--------|-------|
| I. Simplicity / YAGNI | ✅ | Reuses `renderPlanDetail`, `m.md`, and `wrapDescription` as they stand. No new pane, toggle, model field, abstraction, or dependency. The one new helper, `clampLines`, is three lines and exists only because `lipgloss.Height` does not truncate. Extracting a shared description-block helper across `details.go` and `plan_view.go` was considered and **rejected** — three similar lines beat a premature abstraction. Complexity Tracking table is empty. |
| II. API-First Design | ✅ | No RPC, proto, endpoint, or column changes; the server is unaware of this feature. `contracts/plan-detail-contract.md` is committed before implementation as the reviewable internal rendering contract, covering presence (C1), fidelity (C2), position (C3), containment (C4), and non-regression (C5). |
| III. UI/UX Consistency | ✅ | The description uses the Model's existing shared `m.md` renderer and theme, with the same `Options` as the Tasks tab — C2.3 makes byte-identical output a testable requirement rather than an aspiration. Placement mirrors `renderDetails`: short labelled fields above, free-form block last. No ad-hoc colors or styles. Truncate-don't-scroll matches the sibling Tasks tab detail pane. |
| IV. Playful User Messages | ✅ | **No new user-facing text.** The `(no entry selected)` placeholder is unchanged, and an absent description is signalled by absence. Adding cheerful copy for the empty case was considered and rejected: it would violate FR-003 and contract C1.4, which require the no-description pane to be byte-identical to today's. Nothing to review for tone. |

## Project Structure

### Documentation (this feature)

```text
specs/071-plan-task-descriptions/
├── plan.md                              # This file
├── spec.md                              # Feature specification
├── research.md                          # Phase 0 — R1..R6, including the lipgloss height finding
├── data-model.md                        # Phase 1 — entities (all pre-existing) + display rules D1..D5
├── quickstart.md                        # Phase 1 — code shape and manual verification walkthrough
├── contracts/
│   └── plan-detail-contract.md          # Phase 1 — internal rendering contract C1..C6
├── checklists/
│   └── requirements.md                  # Spec quality checklist (16/16 passing)
└── tasks.md                             # Phase 2 — created by /speckit-tasks, not by this command
```

### Source Code (repository root)

```text
internal/tui/
├── plan_view.go          # MODIFY — renderPlanDetail: append the description block
│                         #          (both plain and styled paths); drop the `_ = width` marker
├── view.go               # MODIFY — clamp the styled Planning right-pane content to innerH
│                         #          before paneBox; add the clampLines helper
├── plan_view_test.go     # MODIFY — tests for presence, absence, fidelity, containment
├── details.go            # REFERENCE ONLY — renderDesc pattern and wrapDescription reused as-is
└── markdown/             # (internal/markdown) REFERENCE ONLY — Render, Options, cache
```

Unchanged and explicitly out of scope: `api/`, `services/twig/`, `services/twig-web/`, all CLI subcommands, and every other TUI tab.

**Structure Decision**: Root CLI module only, confined to `internal/tui`. This is a rendering change over state already resident in the `Model`, so the three-module layout described in `CLAUDE.md` is otherwise untouched — no `make proto`, no `sqlc generate`.

## Phase breakdown

**Phase 0 — Research** (complete, `research.md`): placement (R1), markdown reuse (R2), presence rules (R3), the lipgloss height gap (R4), data freshness (R5), test approach (R6). No open questions.

**Phase 1 — Design** (complete): `data-model.md` fixes display rules D1-D5 and the field ordering; `contracts/plan-detail-contract.md` states C1-C6; `quickstart.md` gives the code shape and a manual verification table.

**Phase 2 — Tasks**: generated by `/speckit-tasks`. Expected shape, in dependency order:

1. Description block in the plain path of `renderPlanDetail` + tests (US1, US2).
2. Description block in the styled path + tests (US1, US2).
3. Fidelity test asserting Planning-tab output equals Tasks-tab output at equal width (SC-003, C2.3).
4. `clampLines` helper + styled-path clamp in `view.go` + overflow regression test (US3, C4).
5. Full `go test ./internal/tui/` and the `quickstart.md` manual walkthrough.

Steps 1-3 deliver the P1 stories and are independently shippable; step 4 delivers P2.

## Risks

| Risk | Mitigation |
|---|---|
| Clamping the right pane changes how forms and pickers render in that pane | The clamp applies to whatever occupies the pane, and forms are already short. C5.1 requires no behaviour change; the existing form and picker tests guard it. |
| A description's rendered width exceeds the pane and bleeds into the grid | `Render` is given the pane's inner width, matching how the grid and untimed panes are already sized. Covered by the narrow-terminal check in `quickstart.md`. |
| The nine existing `renderPlanDetail` test functions pass `md == nil` | The nil fallback to `wrapDescription` is preserved deliberately (R2). C5.4 requires all nine to pass unmodified. |
| The identical latent overflow on the Tasks tab detail pane goes unfixed | Deliberate scope call, recorded in research R4 and quickstart "Out of scope" so it reads as an omission by decision, not by oversight. Worth a follow-up feature. |

## Complexity Tracking

> No Constitution Check violations. Table intentionally empty.
