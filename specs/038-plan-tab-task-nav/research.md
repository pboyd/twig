# Research: Plan Tab Task Navigation

## Key Binding Availability

**Decision**: Use `ctrl+t` as the shortcut.

**Rationale**: `ctrl+t` is not bound in `DefaultKeyMap()`. Verified by reading `internal/tui/keymap.go` in full — no existing binding uses this key.

**Alternatives considered**: `ctrl+j` (jump), `g` (go to — but `g` is unbound and lowercase letters are reserved for modal actions like `t` = add task). `ctrl+t` matches the feature's mnemonic (task) and mirrors the user's stated preference.

---

## Tree Expansion Pattern

**Decision**: Reuse `ensureVisible` + `buildVisible` + `findCursor`.

**Rationale**: `ensureVisible(id int64)` at `update.go:1081` already walks the tree recursively and sets `m.expanded[ancestorID] = true` for every ancestor of the target. This is exactly the behavior required. The same three-step pattern (`ensureVisible` → `buildVisible` → `findCursor`) is used in the `refreshedMsg` handler at `update.go:296-303`.

**Alternatives considered**: Writing a new helper — rejected (YAGNI; the existing helper covers this case completely).

---

## Missing Task Handling

**Decision**: If `findCursor` cannot locate the task (returns 0), the tab switch still happens and the cursor lands at the top of the visible list. No error is shown.

**Rationale**: `findCursor` returns `0` on no-match (not `-1`), so the cursor lands gracefully at the top. This is acceptable for the edge case of a deleted task. A notice message would add complexity for an unlikely scenario. The spec explicitly allows silent no-op for missing tasks.

**Alternatives considered**: Show a notice "Task not found" — rejected as unnecessary complexity for a rare edge case.

---

## planList Guard

**Decision**: No explicit mode check is needed inside the new handler.

**Rationale**: The `handlePlanningKey` function already short-circuits to `handlePlanModalKey` at line 501 when `m.plan.mode != planList`. The new case is in the main switch which is only reached when `m.plan.mode == planList`.

---

## showCompleted Interaction

**Decision**: If the linked task is completed and `showCompleted` is false, `ensureVisible` will expand ancestors, but `buildVisible` will still exclude completed tasks from the visible list. `findCursor` will return 0 (top of list). Tab switch still happens.

**Rationale**: The spec explicitly deems this acceptable degradation. Adding a forced `showCompleted = true` toggle on navigation would be unexpected side-effect behavior.
