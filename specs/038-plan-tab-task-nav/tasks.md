# Tasks: Plan Tab Task Navigation

**Input**: Design documents from `specs/038-plan-tab-task-nav/`

**Prerequisites**: plan.md ✅, spec.md ✅, research.md ✅, contracts/ui-bindings.md ✅

**Organization**: Tasks grouped by user story. No setup or foundational phases needed — this feature touches two existing files with no new infrastructure.

## Format: `[ID] [P?] [Story] Description`

- **[P]**: Can run in parallel (different files, no dependencies)
- **[Story]**: Which user story this task belongs to

---

## Phase 3: User Story 1 — Jump to Task from Plan Entry (Priority: P1) 🎯 MVP

**Goal**: Pressing `ctrl+t` on a linked plan entry switches to the Tasks tab with the cursor on the linked task, tree expanded as needed.

**Independent Test**: Open the Planning tab with at least one plan entry linked to a task. Highlight the entry and press `ctrl+t`. Verify the Tasks tab becomes active and the cursor is on the linked task. Verify that if the task is a subtask under a collapsed parent, the parent is expanded and the task is visible.

### Implementation for User Story 1

- [ ] T001 [US1] Add `PlanGoToTask key.Binding` field to `KeyMap` struct in `internal/tui/keymap.go`
- [ ] T002 [US1] Add `ctrl+t` binding in `DefaultKeyMap()` with help text `"ctrl+t", "go to task"` in `internal/tui/keymap.go`
- [ ] T003 [US1] Add `k.PlanGoToTask` to the `FullHelp()` planning-mode second row (alongside `PlanAddTask`, `PlanAddEvent`, `PlanEdit`, `PlanRemove`) in `internal/tui/keymap.go`
- [ ] T004 [US1] Add `PlanGoToTask` handler case in `handlePlanningKey` in `internal/tui/update.go`: switch to `tabTasks`, call `m.ensureVisible(entry.TaskId)`, rebuild `m.visible` via `buildVisible`, set `m.cursor` via `findCursor`

**Checkpoint**: `ctrl+t` on a linked plan entry navigates to the task in the Tasks tab with the tree expanded.

---

## Phase 4: User Story 2 — No Action for Unlinked Entries (Priority: P2)

**Goal**: Pressing `ctrl+t` on an event entry (no linked task) produces no visible change.

**Independent Test**: Press `ctrl+t` on a plan event entry. Confirm the active tab and cursor position are unchanged before and after.

### Implementation for User Story 2

- [ ] T005 [US2] Add guard in the T004 handler: if `entry.TaskId == 0`, return early with no state change (this guard is part of the same case added in T004 — verify it is present and correct in `internal/tui/update.go`)

> **Note**: US2 behavior is implemented as a guard within the same handler case as US1 (T004). T005 is a verification/review step to confirm the guard is correct.

**Checkpoint**: `ctrl+t` on an event entry is a silent no-op; tab and cursor are unchanged.

---

## Phase N: Polish & Cross-Cutting Concerns

- [ ] T006 [P] Verify `ctrl+t` does not conflict with any existing key binding across both tab modes by reviewing all `key.WithKeys` calls in `internal/tui/keymap.go`
- [ ] T007 Run `go test ./...` from repo root and confirm all existing tests pass

---

## Dependencies & Execution Order

### Phase Dependencies

- **User Story 1 (Phase 3)**: No dependencies — start immediately
- **User Story 2 (Phase 4)**: Depends on T004 (handler case must exist before the guard can be verified)
- **Polish (Phase N)**: Depends on T001–T005 complete

### Within User Story 1

- T001 before T002 and T003 (field must exist before it can be referenced)
- T002 and T003 can run in parallel after T001 (both edit `keymap.go` but different functions — do sequentially in a single pass)
- T004 after T001 (field must exist to reference `m.keys.PlanGoToTask`)

### Parallel Opportunities

- T004 (update.go) and T002/T003 (keymap.go) edit different files — can be written in a single session reading both files once, but should be applied in dependency order (keymap first, then update)

---

## Parallel Example: User Story 1

```bash
# Sequential within same file:
# keymap.go: T001 → T002 → T003 (add field, then binding, then help entry)

# Then independent:
# update.go: T004 (add handler case — references m.keys.PlanGoToTask added above)
```

---

## Implementation Strategy

### MVP (User Story 1 Only)

1. Complete T001–T004 in `keymap.go` and `update.go`
2. **Validate**: Manually test `ctrl+t` on a linked plan entry
3. US2 (T005) is a guard already embedded in T004 — verify and ship

### Full Feature

1. T001 → T002 → T003 → T004 → T005 → T006 → T007
2. Total: ~25 lines of code across two files

---

## Notes

- `ensureVisible` already exists at `internal/tui/update.go:1081` — do not rewrite it
- `findCursor` returns `0` (not `-1`) when task not found — this is acceptable (cursor lands at top)
- The `planList` guard at `update.go:501-503` already prevents the new case from firing during sub-forms — no additional mode check needed
- See `contracts/ui-bindings.md` for the full key binding contract
