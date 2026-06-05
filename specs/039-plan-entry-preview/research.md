# Phase 0 Research: Live Plan Entry Preview with Overlap Indication

All unknowns from Technical Context are resolved below. No `NEEDS CLARIFICATION` remain.

## Existing code survey (grounding)

- **Two-pane layout already exists.** `internal/tui/view.go` (the planning branch, ~lines 49–128) renders the day-planner grid in the left pane and, whenever `m.plan.mode != planList`, the active form in the right pane (`renderPlanRightPane` + `planFormPaneTitle`). Both a styled path (Lipgloss `paneBox` + `JoinHorizontal`) and a plain row-join fallback exist. **FR-001 (side-by-side) is therefore already satisfied** — no layout work is required.
- **Grid renderer.** `internal/cli/plan_grid.go` `RenderGrid(entries, day, now, width, isTTY, opts)` lays out timed entries on 15-minute rows, snapping each entry to `snapDown15`/`snapUp15`, and draws boxes with solid runes `┏ ┓ ┗ ┛ ┃ ━` (heavy) plus `┣ ┫` for shared borders. `GridOptions` already carries `SelectedID`, `Styled`, `SelectionStyle`, and fixed-window overrides `WindowStartMin`/`WindowEndMin`.
- **Window sizing.** `internal/tui/plan_view.go` `renderPlanGrid` / `renderPlanGridContent` call `cli.GridWindow(timed, …)` then pass the result as fixed window bounds. Any entry in `timed` expands the window to contain it.
- **Form values.** All three forms store `[]textinput.Model` in `m.plan.form.fields`; values are read live (`.Value()`) at submit time (`submitTaskTimeForm`, `submitEventForm`, `submitEditForm`). Field order: TaskTime `[Start, Duration]`; Event `[Name, Start, Duration]`; Edit `[Name, Start, Duration]`. `m.plan.form.entryID` identifies the edit target; `m.plan.form.taskID` identifies the task being scheduled.
- **Parsing.** `internal/cli/timeparse` exposes `ParseStart`, `ParseDuration`, `ParseDurationOrEnd`, `FormatDuration` — the same helpers the submit handlers use.
- **Overlap.** `services/twig/internal/plan.Overlap(aStart,aDur,bStart,bDur)` implements half-open-interval overlap (touching ≠ overlap), but lives in the **server module** and cannot be imported by the root module.
- **Theme.** `internal/tui/theme.go` provides `dim` and `errorColor` (red) as adaptive colors.

## Decision 1 — Preview entry representation

**Decision**: At render time, when `m.plan.mode` is `planTaskTime`, `planEventForm`, or `planEdit`, parse the form's current Start and Duration values with `timeparse`. If Start parses to a valid minute, build a synthetic `*planv1.PlanEntry{ Id: previewID, Name: <derived>, StartMinute: &start, DurationMinute: dur }`. `previewID` is a negative sentinel constant (real entry IDs are positive).

**Name source**: Edit/Event → the Name field's current value (may be empty → render blank, still a valid box). TaskTime → the picked task's name via `findTask(m.tree, taskID)`, falling back to a neutral label if not found.

**Duration default**: if Duration is blank/unparseable, use a minimal slot (the grid already enforces a ≥15-min visual minimum). Per the spec edge case, a zero/blank duration renders with the planner's normal minimal-entry handling.

**Rationale**: Reusing `*planv1.PlanEntry` means the preview flows through the exact same layout code as real entries — no parallel rendering path. The sentinel ID lets the renderer recognize it without a new type.

**Alternatives rejected**: (a) A dedicated `PreviewEntry` struct — would require duplicating layout logic in `RenderGrid`. (b) Rendering the preview as an overlay drawn after the grid — fragile alignment, and wouldn't expand the window.

## Decision 2 — Distinct preview styling (FR-004)

**Decision**: When `RenderGrid` draws the entry whose `Id == opts.PreviewID`, substitute **dashed** box runes for the solid ones: `┅` (heavy horizontal dashed) for `━`, `┇` (heavy vertical dashed) for `┃`, keeping corner runes `┏ ┓ ┗ ┛` / `┣ ┫`. In styled mode additionally wrap the preview's runes with the `dim` foreground via an `opts.PreviewStyle` hook.

**Rationale**: Dashed borders are a universally-read "draft/unsaved" cue and work in **both** styled and plain modes (the rune change alone is visible without color), satisfying FR-004 in the non-TTY path too. The runes are valid box-drawing characters that align to one cell each, preserving the grid's column math.

**Alternatives rejected**: Color-only differentiation (fails in plain mode); a `[preview]` text label (consumes scarce content width, less immediate).

## Decision 3 — Overlap detection location & granularity

**Decision**: Compute overlap in the **TUI layer** (`internal/tui/plan_preview.go`), not the server. Add a local one-line interval helper (mirroring `plan.Overlap`) — duplication is endorsed by Constitution Principle I over a cross-module dependency. Granularity is per **15-minute slot**: a preview slot starting at minute `t` is "in conflict" if any *other* timed entry's `[start, start+dur)` interval intersects `[t, t+15)`. Produce a `map[int]bool` (or sorted slice) of conflicting slot-start minutes and pass it to `RenderGrid`.

**Self-exclusion (FR-006)**: When `mode == planEdit`, exclude the entry whose `Id == form.entryID` from both the rendered timed slice (replaced by the preview) and the overlap comparison set.

**Rationale**: The grid is row-quantized to 15 minutes, so slot-level conflict marking aligns exactly with how the preview box is drawn — the "overlapping portion" becomes the overlapping rows. Computing in the TUI keeps `RenderGrid` a pure renderer that receives a precomputed conflict set.

**Alternatives rejected**: Computing overlap inside `RenderGrid` (couples rendering with domain logic, and the renderer would need to re-derive which entry is "self"). Minute-exact sub-row shading (the grid has no sub-row resolution; 15-min slots are the natural unit).

## Decision 4 — Conflict marking (FR-005)

**Decision**: For preview rows whose slot is in the conflict set:
- **Styled mode**: render those rows' runes in `errorColor` (red) via an `opts.ConflictStyle` hook — the preferred "overlapping portion in red."
- **Plain mode**: place an unambiguous marker (`!`) in the gutter marker-column for those rows (the column already used for the `▶` now-marker). This is the spec's accepted "simpler indicator."

The conflict styling applies to the **preview only**; saved/existing entries are never restyled (clarified decision, FR-005).

**Rationale**: Red-on-the-overlapping-rows is exactly what the user asked for and is feasible because the renderer already styles per-row. The plain-mode gutter marker reuses an existing column and needs no width.

**Alternatives rejected**: Highlighting the conflicting saved entry too (explicitly rejected in clarification); a modal/inline warning string (less spatial, doesn't show *where*).

## Decision 5 — Window inclusion & out-of-window behavior

**Decision**: Include the preview entry in the slice handed to `cli.GridWindow` and `RenderGrid` so the visible window expands (rounded to the hour, existing behavior) to contain the preview. No special scrolling logic is added.

**Rationale**: Treating the preview like any other entry for window sizing keeps behavior consistent and guarantees the preview is visible without bespoke code. Matches the edge case "MUST NOT render at a wrong position."

**Alternatives rejected**: Forcing the viewport to scroll to the preview (new logic, and `GridWindow` already grows to fit entries); clamping the preview to the visible window (would misrepresent its real time).

## Decision 6 — Lifecycle & purity

**Decision**: The preview is derived purely from form state during rendering; nothing is stored on `planState`. Entering a timed form shows it; `Esc`/Cancel and Save both transition to `planList`, after which a normal reload renders only saved entries. No mutation of `m.plan.entries` occurs while a form is open (the edit-target exclusion happens on a copy used for rendering).

**Rationale**: Pure-derivation guarantees FR-010/FR-011/FR-012 by construction — there is no preview state to leak or forget to clear.

**Alternatives rejected**: Storing a `previewEntry` field on `planState` (introduces a clear-on-exit obligation and a mutation risk for no benefit, since the form fields are already the source of truth).

## Open questions

None. All Technical Context items resolved.
