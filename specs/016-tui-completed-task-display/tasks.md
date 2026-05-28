# Tasks: TUI Completed Task Display

**Branch**: `016-tui-completed-task-display`
**Spec**: [spec.md](./spec.md) | **Plan**: [plan.md](./plan.md) | **Contracts**: [contracts/tui-rendering.md](./contracts/tui-rendering.md)

All paths are relative to repo root unless prefixed otherwise. Test obligations referenced as **T-A** … **T-G** come from `contracts/tui-rendering.md` § Test obligations.

## Phase 1: Setup

No new dependencies, modules, or generated code. Phase 1 is a no-op for this feature.

## Phase 2: Foundational

The single foundational change is a new helper that both user stories rely on. It MUST land before either story.

- [ ] T001 Add `Strike(s string) string` helper in `services/todo/internal/cli/render.go` next to `DimStrike`. When `isTTY` is true, wrap `s` with the ANSI strikethrough open code (`\x1b[9m`) and the reset (`\x1b[0m`); when `isTTY` is false, return `s` unchanged. Match the existing TTY-gating pattern used by `DimStrike` (do not duplicate logic — refactor to share if trivial, otherwise keep parallel).
- [ ] T002 [P] Add a test for the new `Strike` helper in `services/todo/internal/cli/render_test.go` covering both TTY-on and TTY-off paths (**T-A**).

## Phase 3: User Story 1 — Lingering completed task (Priority: P1)

**Story goal**: After pressing Complete on a row, the task stays in the tree with strikethrough styling and remains the selected row until the user moves selection.

**Independent test criteria**: With at least three incomplete tasks and selection on the middle one, pressing Complete leaves the row visible, struck through, and selected; pressing Down then removes it and selects the next task. Verifiable via the existing `tui` test harness without a running server (mock the client as `update_test.go` already does).

- [ ] T003 [US1] In `services/todo/internal/tui/update.go`, inside `handleListKey` for `key.Matches(msg, m.keys.Complete)`, set `m.pendingComplete = &id` (using the same `id` that's already extracted) before returning `completeTaskCmd`. Per contract **C-2**.
- [ ] T004 [US1] In the same file, add `m.pendingComplete = nil` at the top of the `key.Matches(msg, m.keys.Refresh)` handler so that an explicit user refresh drops the lingering row. Per contract **C-3** and FR-006. Do NOT clear it in any other handler (Expand, Collapse, Edit, Help, Filter, pomodoro, digit-key estimate, refresh-from-fetchAfterMutation).
- [ ] T005 [US1] In `services/todo/internal/tui/view.go::renderList`, when composing each row, wrap the **name segment** (not the tree prefix or marker) in `cli.Strike(...)` when `row.node.Task.GetCompletedAt() != nil`. Compose the line as `treePrefix + marker + " " + nameOrStruck`. Keep the existing truncation (use `lipgloss.Width` for visible-width measurement so ANSI codes are not cut mid-sequence) and the existing `highlightStyle` wrapping when `i == m.cursor`. Per contract **C-4**.
- [ ] T006 [P] [US1] In `services/todo/internal/tui/update_test.go`, add a test that builds a model with three incomplete tasks, positions the cursor on the middle one, simulates Complete (and the resulting `refreshedMsg` with the completed task) and asserts: (a) `m.pendingComplete != nil` and equals the completed task id, (b) the completed row is still in `m.visible`, (c) `m.cursor` still points at the completed row. Per contract **T-B**.
- [ ] T007 [P] [US1] Add a follow-on test in the same file: from the state after T006, dispatch a `Down` keypress and assert that `m.pendingComplete == nil` and the completed row is no longer in `m.visible` (assuming `showCompleted=false`). Repeat with `Up` in a sibling test. Per contract **T-C**.
- [ ] T008 [P] [US1] Add tests in the same file that, starting from the lingering state (as in T006), dispatch each of: `Expand`, `Collapse` (on a leaf), `Edit`, `Help`, `Filter`. After each, assert that `m.pendingComplete` is still non-nil and the lingering row is still in `m.visible`. Per contract **T-D**.
- [ ] T009 [P] [US1] Add a test that dispatches the user-facing `Refresh` key from the lingering state and asserts `m.pendingComplete == nil` after the handler runs (before the resulting list arrives back). Per contract **T-E**.
- [ ] T010 [P] [US1] Add a render test in `services/todo/internal/tui/view_test.go` (create if absent) that builds a small visible-row slice containing one completed row and one incomplete row, calls `renderList`, and asserts that the output contains the ANSI strikethrough open code `\x1b[9m` for the completed row's name segment and does NOT contain it on the incomplete row's name. Gate the test on `isTTY=true` via the existing `export_test.go` shim (mirror how `cli/render_test.go` does it). Per contract **T-F**.

**Checkpoint at end of Phase 3**: Story 1 is independently complete and shippable. Story 2's "completed children visible with strikethrough" already works for the *one* lingering row (because of T005), but won't yet apply universally to all completed rows shown via the show-completed filter — that's Story 2.

## Phase 4: User Story 2 — Completed tasks render with completion styling in the tree (Priority: P2)

**Story goal**: Every completed task that appears in the tree (e.g. when the user toggles the show-completed filter) renders with strikethrough on its name; the detail-pane name does not.

**Independent test criteria**: With the show-completed filter on and a parent task containing one completed and one incomplete child, the tree visibly distinguishes the two (strikethrough on the completed name) and the detail pane shows the completed child's name without strikethrough.

> Note: T005 already covers strikethrough rendering for **any** completed row, because the condition is `task.GetCompletedAt() != nil` (not "task is the pendingComplete one"). Story 2 therefore reduces to (a) removing the detail-pane strikethrough and (b) adding tests that explicitly assert the multi-completed-rows behavior.

- [ ] T011 [US2] In `services/todo/internal/tui/details.go::renderDetails`, remove the `if task.GetCompletedAt() != nil { name = cli.DimStrike(name) }` block. Always render `name` raw. The `Completed: <timestamp>` line continues to render as today. Per contract **C-5**.
- [ ] T012 [P] [US2] Add a test in `services/todo/internal/tui/details_test.go` (create if absent — model on existing tui tests using the `export_test.go` shim) that calls `renderDetails` with a completed task and asserts the rendered string does NOT contain `\x1b[9m` (strikethrough) or `\x1b[2m` (dim) around the name, but DOES contain a `Completed:` line. Per contract **T-G**.
- [ ] T013 [P] [US2] Extend the render test from T010 (or add a sibling) so it includes two completed rows at different depths and one incomplete row, and assert strikethrough is present on every completed row's name (not just one) — confirms FR-003 ("regardless of nesting depth").

## Phase 5: Polish & Cross-Cutting

- [ ] T014 Run `cd services/todo && go test ./internal/cli/... ./internal/tui/...` and confirm green. Per the quickstart's automated verification section.
- [ ] T015 Manually verify Story 1 and Story 2 against `specs/016-tui-completed-task-display/quickstart.md` on a real TTY. Spot-check the screenshot scenario from the spec (completed children of a parent task should now show strikethrough in the tree).
- [ ] T016 Audit `services/todo/internal/tui/update.go` once more to confirm `m.pendingComplete` is cleared on the right handlers (Up, Down, Refresh) and NOT on the others (Expand, Collapse, Edit, Help, Filter, pomodoro keys, digit-key estimate, post-complete `refreshedMsg`). This is a final correctness pass against contract **C-3**, not a code change.

## Dependencies

```text
T001 (Strike helper)
  └─▶ T002 (helper test, [P] with each other phase 2 work)
  └─▶ T005 (renderList uses Strike) ─▶ T010, T013 (render tests)
  └─▶ (T011 does not depend on Strike, only on dropping DimStrike)

T003 (set pendingComplete) ─▶ T006 (lingering test)
                          └─▶ T007 (cleared on Up/Down test)
                          └─▶ T008 (NOT cleared on same-row actions)

T004 (clear on Refresh) ─▶ T009 (refresh test)

T011 (drop DimStrike from details) ─▶ T012 (details test)

T014, T015, T016 run after all of Phase 3 + Phase 4 land.
```

Story 1 (Phase 3) and Story 2 (Phase 4) are independently shippable after T001 lands. Story 2 only requires T001 + T011 + T012 + T013 to be useful; Story 1 only requires T001 + T003–T005 + tests. If shipping incrementally, ship Story 1 first (P1).

## Parallel execution examples

- Within Phase 2: T002 can run in parallel with no other foundational work (it's the only test in the phase).
- Within Phase 3: T006, T007, T008, T009, T010 are all in different test functions and can be authored in parallel after T003–T005 land.
- Within Phase 4: T012 and T013 are in different test files and can be authored in parallel after T011 lands.
- Across stories: once T001 is in, T003–T005 (Story 1 impl) and T011 (Story 2 impl) touch different code paths and can land independently. Their tests can be written in parallel.

## Implementation strategy

1. **MVP scope = Story 1 (P1)**: T001 → T003 → T004 → T005 → T006–T010. Ship. This alone resolves the user's primary frustration.
2. **Increment 2 = Story 2 (P2)**: T011 → T012 → T013. Ship. This resolves the "completed children look the same as incomplete children" complaint.
3. **Polish**: T014 (green tests), T015 (manual TTY verification), T016 (audit).

Total: 16 tasks. Story 1: 8 tasks (incl. tests). Story 2: 3 tasks. Foundational: 2. Polish: 3.
