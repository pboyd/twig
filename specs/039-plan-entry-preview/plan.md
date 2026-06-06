# Implementation Plan: Live Plan Entry Preview with Overlap Indication

**Branch**: `039-plan-entry-preview` | **Date**: 2026-06-05 | **Spec**: [spec.md](./spec.md)

**Input**: Feature specification from `/specs/039-plan-entry-preview/spec.md`

## Summary

While any of the three timed planning forms is open (Edit entry, Schedule task, Add event), render a live, visually-distinct **preview** of the entry-being-edited inside the day-planner grid pane, positioned at the time window described by the form's current Start/Duration values. The preview uses **dashed box-drawing runes** so it is never mistaken for a saved entry, and the portion that **overlaps** another timed entry on the same day is marked in the theme's warning (red) color. The two-pane layout (grid + form) that makes this possible already exists; the work is to construct a transient preview entry from in-progress form values and teach the grid renderer to draw it distinctly and flag conflicts. No backend, proto, or DB changes.

## Technical Context

**Language/Version**: Go 1.x (root CLI/TUI module `github.com/pboyd/twig`)

**Primary Dependencies**: Bubble Tea / Bubbles (`textinput`), Lipgloss (styling), existing `internal/cli` grid renderer, `internal/cli/timeparse` (Start/Duration parsing)

**Storage**: N/A (transient, view-only; nothing persisted)

**Testing**: `go test ./...` — table-driven renderer tests in `internal/cli` and reducer/view tests in `internal/tui` (no DB required)

**Target Platform**: Terminal (TTY) — styled (ANSI) and plain fallback modes

**Project Type**: CLI/TUI client (single root module); no server/web changes

**Performance Goals**: Preview recomputes on each render (per keystroke) with no perceptible lag — the grid already re-renders every frame; the added work is O(entries) interval math

**Constraints**: Must work in both styled and non-styled rendering paths; must reuse the shared theme palette (Principle III); preview must never mutate saved entries

**Scale/Scope**: One day's entries (tens, not thousands); a single preview entry at a time

## Constitution Check

*GATE: Must pass before Phase 0 research. Re-check after Phase 1 design.*

| Principle | Status | Notes |
|-----------|--------|-------|
| I. Simplicity / YAGNI | ✅ | Extends existing `GridOptions`/`RenderGrid` with a few fields; reuses the existing two-pane layout and `timeparse`. Duplicates the one-line interval-overlap check in the root module (the server's `plan.Overlap` lives in a separate module and can't be imported) — three lines beat a shared-module dependency. No new abstractions or packages. |
| II. API-First Design | ✅ | The only contract is the internal Go API surface (`GridOptions` additions + `RenderGrid` preview behavior). Documented in `contracts/grid-preview.md` before implementation. No proto/HTTP contract changes. |
| III. UI/UX Consistency | ✅ | Preview color = shared `dim` (faint+italic in styled mode); conflict color = shared `errorColor` (faint+italic); both from `internal/tui/theme.go`. Heavy double-dash runes (`╍`/`╏`) are a structural cue reused across styled + plain modes. No ad-hoc colors. |
| IV. Playful User Messages | ✅ | Feature is visual; introduces no new prose. If a one-line hint is added, it follows the existing warm planning-tab tone. |

## Project Structure

### Documentation (this feature)

```text
specs/039-plan-entry-preview/
├── plan.md              # This file
├── research.md          # Phase 0 output
├── data-model.md        # Phase 1 output
├── quickstart.md        # Phase 1 output
├── contracts/
│   └── grid-preview.md  # Internal Go API contract (GridOptions + RenderGrid preview)
├── checklists/
│   └── requirements.md  # From /speckit-specify
└── tasks.md             # Phase 2 output (/speckit-tasks — NOT created here)
```

### Source Code (repository root)

```text
internal/
├── cli/
│   ├── plan_grid.go        # RenderGrid / RenderUntimed — extend GridOptions + preview rune/style + conflict marking
│   └── plan_grid_test.go   # Add preview + overlap rendering tests
└── tui/
    ├── theme.go            # (reuse) dim = preview, errorColor = conflict; add style helpers if needed
    ├── plan_preview.go      # NEW: build transient preview entry from form fields + compute conflict rows
    ├── plan_view.go         # renderPlanGrid / renderPlanGridContent — inject preview into timed slice + set GridOptions
    ├── plan_grid.go         # planGridOptions — wire preview fields
    └── plan_preview_test.go # NEW: preview-construction + self-exclusion + conflict-detection tests
```

**Structure Decision**: Single root module. Rendering primitives stay in `internal/cli` (where `RenderGrid` lives and is unit-tested without a TTY); preview construction and form-state reading stay in `internal/tui` (where the form/model live). The boundary mirrors the existing split: `cli` renders given data + options; `tui` decides what data/options to pass.

## Phase 0: Outline & Research

See [research.md](./research.md). Key decisions resolved:

1. **Preview entry representation** — a synthetic `*planv1.PlanEntry` with a sentinel `Id` (negative, e.g. `-1`) built at render time from the form's current Start/Duration (and Name/picked-task) values. Only built when Start parses to a valid time (untimed → no timed preview, per FR-009).
2. **Distinct preview styling** — swap the entry's solid box runes (`━ ┃`) for **heavy double-dash** equivalents (`╍` horizontal, `╏` vertical; corners `┏┓┗┛` and single-row end-caps `┣┫` retained at heavy weight). When the preview box abuts another timed entry at a shared divider, the divider uses light tees (`├╍…╍┤`) instead of heavy (`┣╍…╍┫`) to reinforce the tentative feel. In styled mode the preview also takes the `dim` foreground with `Faint` + `Italic` modifiers so the label text visibly recedes. This structural cue works in both styled and plain modes (FR-004).
3. **Overlap detection location** — computed in the TUI layer against the real timed entries (reusing a local one-line interval check), producing the set of 15-min slot-times where the preview overlaps another entry; passed to `RenderGrid` so it can color those preview rows. Self-overlap excluded by removing the edit target from the comparison set (FR-006).
4. **Conflict marking** — styled mode colors the overlapping preview rows with `errorColor` (red), satisfying the preferred "overlapping portion in red." Plain mode uses an unambiguous fallback marker in the gutter marker-column for conflicting rows (FR-005 simpler-indicator clause).
5. **Window inclusion** — the preview entry is included when computing the grid window so the window expands to keep the preview visible; it must never render at a wrong position (FR-009 / edge case).
6. **Lifecycle** — preview exists only while `m.plan.mode` is a timed form; cancel/save returns to `planList` and the preview disappears, replaced on reload by the saved entry (FR-010, FR-011). Saved entries are read-only throughout (FR-012).

## Phase 1: Design & Contracts

- **data-model.md** — the transient Preview Entry, the `GridOptions` additions, and the derived Conflict-slot set.
- **contracts/grid-preview.md** — the internal Go API contract: new `GridOptions` fields, their semantics, the dashed rune set, conflict-styling hooks, and the invariants `RenderGrid` must uphold (preview never persisted; saved entries unstyled by conflict; touching ≠ overlap).
- **quickstart.md** — manual verification script (open each form, type Start/Duration, observe preview move/resize, force an overlap, confirm red marking, cancel/save).
- **Agent context** — update the plan reference inside the `<!-- SPECKIT START -->` / `<!-- SPECKIT END -->` markers in `CLAUDE.md` to point to this plan.

### Post-Design Constitution Re-Check

| Principle | Status | Notes |
|-----------|--------|-------|
| I. Simplicity / YAGNI | ✅ | No new packages; ~3 new `GridOptions` fields; one small new TUI file for preview construction. |
| II. API-First Design | ✅ | `contracts/grid-preview.md` defines the Go surface before code. |
| III. UI/UX Consistency | ✅ | Colors sourced from `theme.go`; dashed runes consistent across modes. |
| IV. Playful User Messages | ✅ | No new prose (visual feature). |

## Complexity Tracking

> No Constitution violations. Table intentionally empty.
