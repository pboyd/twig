# Research: Task Descriptions on the Planning Tab

**Feature**: 071-plan-task-descriptions | **Date**: 2026-07-31

## Summary

Every piece the feature needs already exists. `renderPlanDetail` (`internal/tui/plan_view.go:304`) already receives both the linked `*taskv1.Task` and the `*markdown.Renderer`; it simply never reads `task.Description`. `renderDetails` (`internal/tui/details.go:20`) already establishes exactly how a description should be presented. The only genuinely new engineering is height clamping, which turns out to be a real gap rather than an assumption — see R4.

---

## R1: Where the description goes

**Decision**: Append the description inside the existing `renderPlanDetail`, after the pomodoro row, separated by one blank line. No new pane, no toggle, no change to the calendar grid.

**Rationale**: `renderPlanDetail` is already the Planning tab's read-only per-entry detail view, rendered into the right pane by `view.go:104` (styled) and `view.go:136` (plain). It already receives the linked task — `view.go:100-103` looks it up with `findTask(m.tree, sel.TaskId)` and passes `nil` for events. The signature needs no change at all. Placing the description last mirrors `renderDetails`, where the short labelled fields sit above and the free-form block sits below.

**Alternatives considered**:
- *Inline in the calendar grid rows* — rejected. Grid rows are fixed-height 15-minute slots rendered by `internal/cli`; variable-length text has nowhere to live there, and it would wreck the grid geometry.
- *A separate description pane or popup* — rejected under Principle I. A third pane in a two-pane layout for one field is unjustifiable complexity.
- *Above the labelled fields* — rejected. A long description would push `Window`/`Duration` off the visible area, hiding the fields the tab exists to show.

---

## R2: Markdown rendering

**Decision**: Reuse the exact `renderDesc` pattern from `details.go:25-30`:

```go
if md != nil {
    return md.Render(task.GetDescription(), markdown.Options{Width: width, Styled: styled})
}
return wrapDescription(task.GetDescription(), width)
```

**Rationale**: This is the same renderer instance (`m.md`, constructed once at `model.go:240`) with the same `Options`, so the Planning tab output is identical to the Tasks tab at equal width — satisfying SC-003 by construction rather than by parallel implementation. `Render` is block-level (headings, lists, tables, code) as opposed to `RenderInline`, which `renderPlanDetail` already uses for the entry *name*. The renderer caches on `{text, width, styled, inline}` (`markdown.go:56`), so re-rendering the same description while moving the cursor is a map lookup — SC-005 comes free.

The `md == nil` fallback matters: `md` is never nil in production, but every existing `renderPlanDetail` test passes `nil` (`plan_view_test.go:345, 361, 809, …`). Keeping the fallback preserves those tests and lets new plain-text tests avoid constructing a renderer.

**Alternatives considered**:
- *A new description-specific renderer or options struct* — rejected under Principle I and Principle III. Divergent rendering between tabs is precisely what Principle III forbids.
- *Extracting a shared `renderDescriptionBlock` helper used by both `details.go` and `plan_view.go`* — rejected for now. The body is four lines; the constitution prefers three similar lines to a premature abstraction. Noted as a future cleanup if a third caller appears.

---

## R3: Which entries get a description

**Decision**: Show the description only when `entry.TaskId != 0`, `task != nil`, and `strings.TrimSpace(task.GetDescription()) != ""`. Otherwise render nothing — no label, no blank line.

**Rationale**: Descriptions belong to tasks; events (`TaskId == 0`) have none. `task` is already `nil` for events, and `findTask` returns `nil` when the task is absent from `m.tree`, so a single `task != nil` guard covers both the event case and the missing-task case. The existing pomodoro row at `plan_view.go:343` and `:368` already guards on `task != nil` the same way, so the new code matches its neighbour.

Trimming whitespace before the emptiness test is a deliberate tightening over `details.go:65`, which tests `!= ""` only. A whitespace-only description would otherwise emit a blank separator plus blank rendered output — visible dead space in a pane the spec (FR-003) says must look untouched.

---

## R4: Height overflow — the one real gap

**Decision**: Clamp the Planning tab's right-pane content to the pane's inner height before handing it to `paneBox`, in the styled path in `view.go`.

**Rationale**: `paneBox` (`view.go:706`) sets `lipgloss.Height(innerHeight + 2)`, and **lipgloss `Height` is a minimum, not a maximum** — it pads short content but does not truncate tall content. Verified empirically:

```go
lipgloss.NewStyle().Border(lipgloss.RoundedBorder()).Width(20).Height(5).Render("a\nb\nc\nd\ne\nf\ng\nh")
// → 10 lines: the box grows to fit all 8 content rows.
```

Today this never bites on the Planning tab because `renderPlanDetail` emits at most about five short lines. A multi-paragraph description changes that: the right pane would grow taller than the left, `JoinHorizontal` would align them at the top and leave a ragged bottom, and the status line would be pushed off-screen. That directly violates FR-006 and SC-004.

Note the asymmetry that made this easy to miss: the *left* pane is already safe, because `renderPlanGrid` (`plan_view.go:253-257`) trims its own output to `height` lines before returning. The right pane has no equivalent because it never needed one. The fix restores the symmetry.

The plain-text path is already safe — `splitLines(rightContent, maxLines)` (`view.go:142`) truncates to exactly `maxLines`. No change needed there.

**Scope note**: The Tasks tab has the identical latent overflow (`view.go:245-248` passes `renderDetailPane` output straight to `paneBox`), and a long enough task description breaks that layout today. That is a **pre-existing bug outside this feature's scope**; this feature must not leave the Planning tab in the same state, but fixing the Tasks tab is a separate change. Flagged here so it is a deliberate omission rather than an oversight.

**Alternatives considered**:
- *Make `paneBox` clamp its own content* — tempting and arguably more correct, but it silently changes every one of the nine `paneBox` call sites, including panes on other tabs that this feature has no mandate to touch. Rejected as scope creep with real regression risk.
- *Add scrolling to the detail pane* — explicitly out of scope per the spec's Assumptions. The Tasks tab detail pane does not scroll either; adding it here would violate Principle III by making two sibling panes behave differently.
- *Truncate inside `renderPlanDetail`* — rejected. The function does not know the pane's height, and threading a height parameter through would change a signature that nine existing test functions depend on.

---

## R5: Data freshness

**Decision**: No change. The description comes from `m.tree`, refreshed by the same `ListTasks` load that already feeds the pomodoro estimate shown in this pane.

**Rationale**: Feature 069 (`tui-auto-refresh`) already governs when `m.tree` reloads — tab entry, `ctrl+r`, mutations, and the ten-minute heartbeat. The description inherits that behaviour exactly, which is what the spec's edge case asks for. `m.tree` holds the full task tree; the Tasks tab's display filters apply to `m.visible`, not `m.tree`, so a description stays visible on the Planning tab even when its task is filtered out of the task list.

---

## R6: Testing approach

**Decision**: Table-free unit tests in `internal/tui/plan_view_test.go` alongside the ten existing `renderPlanDetail` tests, plus one clamping test for the view layer.

Coverage to add:
1. Task entry with a description → description text present in output.
2. Task entry with an empty description → output byte-identical to today's.
3. Whitespace-only description → treated as empty.
4. Event entry (`TaskId == 0`, `task == nil`) → no description.
5. Entry whose task is missing from the tree (`task == nil`, `TaskId != 0`) → no description, other fields intact.
6. Markdown constructs render the same as `renderDetails` at equal width, with a real `markdown.Renderer` — the direct SC-003 assertion.
7. Unstyled path emits no ANSI codes (extends the existing `plan_view_test.go:423` guarantee to the new block).
8. A long description does not make the styled Planning view exceed the terminal height — the FR-006/SC-004 regression test.

**Rationale**: `renderPlanDetail` is a pure function of its arguments, so tests 1-7 need no model, no server, and no TTY. Test 8 needs a `Model` with `width`/`height` set and asserts on the line count of the full rendered view; `plan_view_test.go` and `view_test.go` already contain models built this way.

---

## Open questions

None. No NEEDS CLARIFICATION markers were carried in from the spec, and nothing surfaced during research.
