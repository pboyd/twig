# Data Model: Goal Task Filter Shortcut

**Feature**: 065-goal-task-filter | **Date**: 2026-07-16

## Persisted entities

**None.** This feature adds no database table, column, migration, proto message, or sqlc query. It reads
existing in-memory state and composes a query string against the existing filter engine.

The spec's Key Entities (Goal, Task, Filter expression) all exist today. For completeness, their bearing
on this feature:

| Entity | Where it lives | Role here | Changed? |
|--------|----------------|-----------|----------|
| Goal | `goalv1.Goal` (`api/gen/goal/v1`) | Only `Id` is read, to build the expression. Never displayed. | No |
| Task | `taskv1.Task` / `cli.TreeNode` | The filter result set. Membership in a goal's scope is decided server-side. | No |
| Filter expression | `string` in `model.filterInput` / `model.filterExpr` | Composed by the shortcut instead of typed. Otherwise identical. | No |

## In-memory state touched

All fields already exist on `tui.Model` (`internal/tui/model.go`). The feature adds **one** field, and
that field is a key binding rather than data.

### Added

| Field | Type | Location | Purpose |
|-------|------|----------|---------|
| `GoalGoToTasks` | `key.Binding` | `KeyMap` (`internal/tui/keymap.go`) | Declares `ctrl+t` on the Goals tab |

### Read (never mutated by this feature)

| Field | Type | Role |
|-------|------|------|
| `goal.goals` | `[]*goalv1.Goal` | Source list; filtered through `visibleGoals()` |
| `goal.cursor` | `int` | Selects the goal to scope by |
| `goal.showAll` | `bool` | Input to `visibleGoals()`, so the jump targets the goal the user sees |
| `goal.mode` | `goalViewMode` | Gates the shortcut via existing early returns in `handleGoalsKey` |
| `showAll` | `bool` | Tasks-tab visibility; passed through to `FilterTasks` **unmodified** |
| `client` | `taskv1connect.TaskServiceClient` | Dispatches `FilterTasks` |

### Written

| Field | From → To | Why |
|-------|-----------|-----|
| `activeTab` | `tabGoals` → `tabTasks` | FR-001 |
| `keys.GoalMode` | `true` → `false` | Help listing follows the active tab |
| `filterInput` (value) | *(prior text)* → `"^goal_id=<id>"` | FR-003/FR-004; **must precede dispatch** |
| `filterInput` (focus) | *(any)* → blurred | Focus belongs to the task list |
| `mode` | *(any)* → `modeList` | Land on the list, not the filter prompt |
| `filterGen` | `n` → `n+1` | Discard in-flight filter responses |
| `filterInvalid` | *(any)* → `false` | The composed expression is always well-formed |
| `goal.err`, `err` | *(any)* → `nil` | Clear the departing tab's transient error |

### Written indirectly, by the existing `handleFilterResult`

The shortcut deliberately does **not** set these. Letting the existing handler own them is what keeps this
feature on one filter code path.

| Field | Set by | Note |
|-------|--------|------|
| `filterExpr` | `handleFilterResult` | Read from `filterInput.Value()` — the reason input-first ordering matters |
| `filterMatches`, `filteredIDs` | `handleFilterResult` | Result set and O(1) lookup |
| `visible` | `handleFilterResult` | Rebuilt via `buildVisibleFiltered` |
| `cursor` | `handleFilterResult` | Settles via `clampCursor`; no explicit positioning |
| `scrollOffset` | `reconcileScroll` | Re-clamped after the list rebuilds |

## State transitions

### Happy path

```text
Goals tab, goalList mode, cursor on goal G
  │
  │  ctrl+t
  ▼
activeTab := tabTasks; keys.GoalMode := false; errors cleared
filterInput := "^goal_id=<G.Id>"; blurred; mode := modeList; filterGen++
  │
  │  filterCmd(client, "^goal_id=<G.Id>", showAll, now, gen)
  ▼
Tasks tab, previous list still rendered, filter in flight
  │
  │  filterResultMsg{gen, ids}
  ▼
handleFilterResult: gen matches → filterExpr := input value
                                  filteredIDs := ids
                                  visible := buildVisibleFiltered(...)
                                  cursor := clampCursor(...)
  ▼
Tasks tab showing goal G's tree (or the "no tasks match" state)
```

### Guarded paths

| Trigger | Transition |
|---------|-----------|
| `ctrl+t`, goals list empty or cursor out of range | *(no transition)* — FR-006 |
| `ctrl+t` in any goal sub-mode | *(not this shortcut)* — early return owns the key, FR-007 |
| `filterResultMsg` with stale `gen` | Discarded by the existing guard |
| `filterResultMsg` with `err` | Existing path: `filterInvalid := true`, last valid results retained |

### After arrival — no new states

These all run on existing code with no shortcut-specific branches (FR-003, FR-009):

| User action | Result |
|-------------|--------|
| `/` | Filter input reopens pre-filled with `^goal_id=<id>` |
| `esc` in list | `clearFilter()` — full task list restored |
| `c` | Toggles show-all; filter re-fires with the same expression, new `showAll` |
| Edit/complete/move a task | Existing re-fire paths (`update.go:804`, `:2360`, `:2546`) re-dispatch `filterExpr` |
| `tab` / `shift+tab` | Normal tab navigation; the filter persists on the Tasks tab as usual |

## Validation rules

| Rule | Enforced where |
|------|----------------|
| Goal must exist under the cursor | Shortcut precondition (FR-006) |
| Goal ID is a base-10 integer | `fmt.Sprintf("^goal_id=%d", g.Id)` — type-enforced |
| Expression must parse | Guaranteed by construction; `filterInvalid := false` is safe |
| Goal deleted after jumping | No client-side rule — the server returns an empty set, not an error |
| Completion/snooze visibility | **Not** encoded in the expression; owned by `showAll` (FR-009) |
