# Contract: Planning Tab Entry Detail Rendering

**Feature**: 071-plan-task-descriptions | **Date**: 2026-07-31

## Scope of this contract

This feature adds no RPC, no proto message, no HTTP endpoint, and no database column. It reads `task.v1.Task.description`, a field that already exists and is already delivered to the TUI by an unchanged `task.v1.TaskService/ListTasks` call.

| Surface | Change |
|---|---|
| `api/proto/**` | none |
| `services/twig/**` | none |
| `api/gen/**` | none — no regeneration needed |
| Database schema / migrations | none |
| Web frontend | none |
| CLI subcommands | none |

The server is unaware of this feature.

The contract below is therefore an **internal rendering contract** for the TUI, expressed at the level this feature operates: given a selected plan entry and its linked task, what the detail pane must and must not put on screen. It is the reviewable artifact Principle II requires.

---

## C1: Description presence

Let `entry` be the selected `*planv1.PlanEntry`, `task` the linked `*taskv1.Task` (already resolved by the caller), and `desc = task.GetDescription()`.

**C1.1** When `entry.TaskId != 0` **and** `task != nil` **and** `strings.TrimSpace(desc) != ""`, the detail pane MUST render `desc`.

**C1.2** When `entry.TaskId == 0` (an event), the pane MUST NOT render any description block. Events have no description and none may be synthesized.

**C1.3** When `task == nil` — the entry links a task that is absent from the in-memory tree — the pane MUST NOT render a description block, MUST NOT render an error or placeholder, and MUST render every other field exactly as it does today.

**C1.4** When `strings.TrimSpace(desc) == ""`, the pane MUST render no description block, no label, no heading, and no blank separator line. Output MUST be byte-identical to the pre-feature output for the same entry.

**C1.5** The pane MUST render the description of the entry currently selected. Changing the selection MUST change the rendered description in the same frame.

**C1.6** The description MUST be rendered only in the read-only detail view (plan mode `planList`). When a picker or form occupies the right pane, no description is rendered.

---

## C2: Rendering fidelity

**C2.1** With a non-nil `*markdown.Renderer`, the description MUST be rendered by `Render` (block-level), not `RenderInline`, with `markdown.Options{Width: <pane inner width>, Styled: <caller's styled flag>}`.

**C2.2** The renderer instance MUST be the `Model`'s existing `m.md`. No second renderer, no alternative theme, no feature-local options struct.

**C2.3** Given the same description string and the same `Width` and `Styled` values, the output of the Planning tab description block MUST equal the output of the Tasks tab description block. This is the normative form of SC-003 and is directly testable.

**C2.4** With a nil renderer, the description MUST fall back to `wrapDescription(desc, width)` — the same fallback `renderDetails` uses.

**C2.5** When `styled` is false, the rendered description MUST contain no ANSI escape sequences.

---

## C3: Layout position

**C3.1** The description MUST appear after all existing detail fields: entry name, `Window`, `Duration`, `Task` (when present), and the pomodoro row (when present).

**C3.2** Exactly one blank line MUST separate the description from the field above it.

**C3.3** No existing field's text, label, order, alignment, or styling may change.

**C3.4** The description MUST be rendered at the pane's inner width so it wraps within the pane and never bleeds into the calendar grid.

---

## C4: Height containment

**C4.1** In the styled two-pane path, the right pane's content MUST be clamped to the pane's inner height before being passed to `paneBox`.

This is required, not incidental: `paneBox` applies `lipgloss.Height`, which pads short content but does **not** truncate tall content. Without an explicit clamp, a long description grows the right pane past the left, misaligns the `JoinHorizontal`, and pushes the status line off screen.

**C4.2** After clamping, the total rendered height of the Planning view MUST NOT exceed the terminal height, for a description of any length.

**C4.3** The plain-text path already clamps via `splitLines(rightContent, maxLines)` and MUST continue to do so. No change is required there.

**C4.4** Clamping MUST truncate — show the leading portion that fits. It MUST NOT scroll, paginate, ellipsize, or reflow.

**C4.5** The calendar grid pane's own height behaviour MUST be unchanged.

---

## C5: Non-regression

**C5.1** Entry selection, scheduling, entry editing, event creation, the task picker, untimed entries, and the calendar grid MUST behave exactly as before.

**C5.2** The Tasks tab, Goals tab, and Report tab MUST be unaffected.

**C5.3** The description MUST be read-only on the Planning tab. No key binding may edit, clear, or expand it.

**C5.4** All nine existing `renderPlanDetail` test functions MUST continue to pass unmodified.

---

## C6: Constitution compliance

**C6.1** (Principle III) The description MUST use the existing shared markdown theme via `m.md`. No ad-hoc colors or styles.

**C6.2** (Principle IV) This feature introduces **no new user-facing strings**. The existing `(no entry selected)` placeholder is unchanged, and an absent description is signalled by absence rather than by a message. There is consequently no new copy to review for tone — and adding one ("This task has no description yet!") would violate C1.4's requirement of unchanged output.
