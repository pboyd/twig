# Interaction Contract: Goals-Tab `ctrl+t`

**Feature**: 065-goal-task-filter | **Date**: 2026-07-16 | **Status**: Authoritative

This project's constitution requires a contract committed before implementation (Principle II, Quality
Gate 1). This feature exposes **no network API surface** — no new proto messages, RPCs, or endpoints. The
server-side `^goal_id` filter it depends on already ships and is unchanged (see `research.md` R1).

The contract this feature introduces is therefore a **TUI interaction contract**: the keybinding, the
expression it emits, and the state guarantees that hold afterward. Implementation MUST conform to this
document; it is the source of truth for the tests in `plan.md` Change 3.

## 1. Binding

| Property | Value |
|----------|-------|
| Key | `ctrl+t` |
| Scope | Goals tab, normal list view only |
| KeyMap field | `GoalGoToTasks` |
| Help text | `ctrl+t` — "show goal's tasks" |
| Help placement | Goals `FullHelp` (`GoalMode` branch). Not in `ShortHelp`. |

### Key sharing with the Plan tab

`ctrl+t` is already bound as `PlanGoToTask` on the Plan tab. Both bindings coexist and are dispatched by
active tab; neither may be modified to accommodate the other.

| Tab | Binding | Meaning |
|-----|---------|---------|
| Plan | `PlanGoToTask` | Jump to Tasks, cursor on the selected entry's task |
| Goals | `GoalGoToTasks` | Jump to Tasks, filtered to the selected goal's tree |
| Tasks, Report | *(unbound)* | No effect |

Plan-tab behavior MUST be byte-for-byte unchanged (FR-008).

## 2. Emitted expression

```text
^goal_id=<id>
```

Where `<id>` is the `Id` of the goal under the Goals-tab cursor, resolved from
`visibleGoals(m.goal.goals, m.goal.showAll)[m.goal.cursor]` — the same list the cursor and detail pane
index into.

**The expression is exactly this and nothing more.** Implementations MUST NOT append `completed=false`,
`snoozed=false`, or any other condition. The bare relational condition is what preserves the show-all
default (§4); adding conditions would hard-code visibility and break the `c` toggle.

**Invariants**:

- No spaces, no quoting, no parentheses. The value is a base-10 integer.
- The expression is well-formed input to the existing filter parser
  (`services/twig/internal/filter/parser.go:186`) and requires no parser change.
- It is indistinguishable from the same text typed by hand at the `/` prompt. There is exactly one filter
  code path, and this shortcut uses it.

## 3. Preconditions and no-ops

| Condition | Required behavior |
|-----------|-------------------|
| Goal under cursor exists | Perform the jump |
| Goals list empty, or cursor out of range | **No-op**: no tab switch, no command, no error (FR-006) |
| Goals-tab sub-mode active — `goalEdit`, `goalNew`, `goalNewTask`, `goalPickLink`, `goalPickUnlink`, `goalStatusHistory`, `goalStatusReader`, `goalStatusConfirmDel`, `goalConfirmDelete`, or `modeHelp` | **Not treated as this shortcut**; the open view keeps its own key handling (FR-007) |

Sub-mode gating is satisfied structurally: `handleGoalsKey` early-returns for every mode above before
reaching the main action switch where this case lives. Implementations MUST NOT add a separate top-level
`ctrl+t` intercept, which would bypass those guards.

## 4. Post-conditions

After a successful jump, all of the following MUST hold:

| # | Guarantee | Requirement |
|---|-----------|-------------|
| P1 | `activeTab == tabTasks` | FR-001 |
| P2 | `filterInput.Value() == "^goal_id=<id>"` — set **before** dispatch | FR-003, FR-004 |
| P3 | `FilterTasks` is called with that expression and the **current, unmodified** `showAll` | FR-002, FR-009 |
| P4 | On result, `filterExpr == "^goal_id=<id>"` | FR-003 |
| P5 | Any prior filter — both `filterExpr` and input text — is fully replaced, never merged | FR-004 |
| P6 | The filter input is blurred; focus belongs to the task list | FR-003 |
| P7 | `filterGen` is incremented before dispatch, so in-flight responses are discarded | — |
| P8 | Goals-tab help flags follow the tab (`keys.GoalMode == false`) | FR-010 |

### P2 is load-bearing

`handleFilterResult` (`internal/tui/update.go:199-200`) assigns `m.filterExpr = m.filterInput.Value()` —
it reads the **input widget**, not the expression passed to `filterCmd`. Dispatching without first calling
`SetValue` applies the filter correctly but leaves `filterExpr` holding stale text, which then breaks P4,
P5, and every mutation re-fire path (`update.go:804`, `:2360`, `:2546`). Order is: `SetValue` →
`filterGen++` → `filterCmd`.

### Show-all interaction

`^goal_id=<id>` names neither `completed` nor `snoozed`, so per the per-attribute override semantics of
spec 062 the default visibility stays in force —
confirmed at `services/twig/internal/filter/eval.go:56`. Consequences the implementation MUST preserve:

- With show-all **off**: open, unsnoozed tasks for the goal.
- With show-all **on**: completed and snoozed tasks for the goal appear too.
- Toggling `c` after arriving re-fires the filter and updates visibility live.

## 5. User-facing text

This feature introduces **no new strings**. Two touchpoints inherit existing, tone-reviewed copy:

- **Help entry**: "show goal's tasks" — matches the plain, verb-led register of neighboring Goals help
  entries ("add task", "link task", "status history").
- **Empty result** (a goal with no tasks, FR-004): routes to the existing "no tasks match this filter"
  state. No goal-specific variant is in scope.

Should implementation conclude a goal-specific empty state is warranted, it falls under Principle IV and
MUST be warm and light rather than terse — matching the register of existing empty states such as
`goal_view.go:268` ("A blank canvas! Press 'ctrl+n' to plant your first goal.") — and it MUST remain
accurate about *why* the list is empty.

## 6. Out of scope

- Displaying goal IDs anywhere in the interface. This feature removes the need to know them.
- Any web-app equivalent (`services/twig-web/`).
- Any change to the filter expression language, parser, or evaluator.
- A reverse jump (Tasks → Goals).
