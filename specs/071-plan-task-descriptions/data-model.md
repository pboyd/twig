# Data Model: Task Descriptions on the Planning Tab

**Feature**: 071-plan-task-descriptions | **Date**: 2026-07-31

## Scope

This feature introduces **no new entities, no schema changes, no migrations, and no proto changes**. It reads one field that already exists, travels over an RPC that is already called, and is already held in memory by the TUI. This document records the entities involved and the rules governing how the field is displayed.

---

## Entities involved (all pre-existing)

### `task.v1.Task`

| Field | Type | Role in this feature |
|---|---|---|
| `id` | `int64` | Target of `findTask(m.tree, entry.TaskId)` |
| `name` | `string` | Already displayed; unchanged |
| `description` | `string` | **The field this feature surfaces.** Markdown source, may be empty, may be multi-paragraph |
| `estimate` | `int32` | Already displayed via the pomodoro row; unchanged |
| `completed_pomodoro_count` | `int32` | Already displayed; unchanged |

No field is added, widened, or given new validation. `description` is already written by the Tasks tab editor (feature 030) and already read by `renderDetails`.

### `plan.v1.PlanEntry`

| Field | Type | Role in this feature |
|---|---|---|
| `id` | `int32` | Selection identity; unchanged |
| `task_id` | `int64` | `0` means event → no description. Non-zero → look up the task |
| `name` | `string` | Already displayed via `RenderInline`; unchanged |
| `start_minute` | `*int32` | Already displayed; unchanged |
| `duration_minute` | `int32` | Already displayed; unchanged |
| `completed` | `bool` | Already displayed; unchanged |

A `PlanEntry` carries **no description of its own** and gains none. The description is always the linked task's.

---

## Relationship

```text
PlanEntry ──task_id──▶ Task ──description──▶ rendered markdown block
     │                  ▲
     │                  │
     └── task_id == 0 ──┴── event: no Task, no description
```

Resolution is in-memory only: `findTask(m.tree, entry.TaskId)` walks the task tree the TUI already holds. There is no additional fetch, and a cache miss is not an error — it degrades to "no description shown".

---

## In-memory state

| State | Location | Change |
|---|---|---|
| `m.tree` | `Model.tree` (`model.go:101`) | none — already populated by `ListTasks` |
| `m.md` | `Model.md` (`model.go:240`) | none — already constructed; its cache absorbs repeat renders |
| `m.plan.entries` | `planState.entries` | none |
| `m.plan.cursor` | `planState.cursor` | none |

**No new model field is added.** This is worth stating plainly: the feature is a rendering change over state that is already resident.

---

## Display rules

The rules below are the normative statement of FR-001 through FR-007. `desc` is `task.GetDescription()`.

| # | Condition | Result |
|---|---|---|
| D1 | `entry == nil` | Existing placeholder; rule set does not apply |
| D2 | `entry.TaskId == 0` (event) | No description block |
| D3 | `task == nil` (missing from tree) | No description block; all other fields render normally |
| D4 | `strings.TrimSpace(desc) == ""` | No description block, no separator, no reserved space |
| D5 | Otherwise | One blank line, then `desc` rendered as block markdown at pane width |

### Rendering derivation

| Input | Output |
|---|---|
| `md != nil` | `md.Render(desc, markdown.Options{Width: width, Styled: styled})` |
| `md == nil` | `wrapDescription(desc, width)` |

`styled` follows the caller's `styled` argument, so the unstyled path emits no ANSI codes — preserving the guarantee asserted at `plan_view_test.go:423`.

### Ordering within the pane

```text
  <entry name>          (RenderInline, bold accent when styled)
  Window:   HH:MM–HH:MM
  Duration: N min
  Task:     #ID  status        (only when TaskId != 0)
  <pomodoro row>                (only when task != nil and the row is non-empty)
                                ← blank separator (only under D5)
  <description>                 (only under D5)
```

The description is last because it is the only variable-length field; every rule above it keeps its current position and formatting.

---

## Height constraint

Not a data rule, but a display invariant the model must uphold (FR-006):

> The right pane's content, after the description is appended, MUST be clamped to the pane's inner height before rendering, in the styled path. The plain-text path is already clamped by `splitLines`.

See `contracts/plan-detail-contract.md` §C4 for the normative form.

---

## Validation rules

None. The description is display-only here (FR-008): the Planning tab neither writes nor validates it. All authoring and validation remain on the Tasks tab.

---

## State transitions

None. No entity in this feature has a lifecycle.
