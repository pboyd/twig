# UI Contract: Plan Tab Task Navigation Key Binding

**Feature**: 038-plan-tab-task-nav
**Tab**: Planning

## New Binding

| Key     | Mode       | Field           | Help Text       |
|---------|------------|-----------------|-----------------|
| ctrl+t  | planList   | PlanGoToTask    | go to task      |

## Preconditions

- Active tab MUST be Planning
- `plan.mode` MUST be `planList` (not a sub-form)
- At least one plan entry MUST be present
- The highlighted entry MUST have `TaskId != 0`

## Behavior

1. Switch active tab to Tasks (`tabTasks`)
2. Set `keys.PlanningMode = false`
3. Clear `plan.err`
4. Expand all ancestor nodes of the linked task in the task tree
5. Rebuild visible row list
6. Position cursor on the linked task

## No-op Cases

- Entry has `TaskId == 0` (event): no state change
- No entries in plan: no state change
- Task not found in tree (deleted): tab switch happens, cursor lands at row 0

## Help Visibility

The binding appears in `FullHelp()` planning-mode second row alongside `PlanAddTask`, `PlanAddEvent`, `PlanEdit`, `PlanRemove`. It does NOT appear in `ShortHelp()` (keep short help concise).
