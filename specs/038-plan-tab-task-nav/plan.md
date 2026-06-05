# Implementation Plan: Plan Tab Task Navigation

**Branch**: `038-plan-tab-task-nav` | **Date**: 2026-06-05 | **Spec**: [spec.md](spec.md)

## Summary

Add a `ctrl+t` shortcut to the Planning tab that switches to the Tasks tab and positions the cursor on the linked task, expanding any collapsed ancestor nodes to make it visible. This is a pure TUI change — no server-side or API changes required.

## Technical Context

**Language/Version**: Go 1.23

**Primary Dependencies**: Bubble Tea (TUI framework), `charmbracelet/bubbles/key`

**Storage**: N/A (read-only navigation; no persistence)

**Testing**: `go test ./...` from repo root; TUI package tests in `internal/tui/`

**Target Platform**: Terminal (Linux/macOS)

**Project Type**: CLI/TUI

**Performance Goals**: Instantaneous — all data is already in memory

**Constraints**: Must not break existing `ctrl+t` usage elsewhere (none identified — verify during implementation)

**Scale/Scope**: Single keypress handler; ~20 lines of new code

## Constitution Check

| Principle | Status | Notes |
|-----------|--------|-------|
| I. Simplicity / YAGNI | ✅ | Uses `ensureVisible` + `buildVisible` + `findCursor` which already exist for this exact pattern. No new abstractions needed. |
| II. API-First Design | ✅ | No API changes. TUI-only; contracts/ui-bindings.md documents the key binding contract. |
| III. UI/UX Consistency | ✅ | New key binding follows existing `key.NewBinding` pattern and is added to `FullHelp()` for the planning mode. |
| IV. Playful User Messages | ✅ | No new user-facing messages. The no-op case for event entries is silent (same pattern as other no-ops). |

## Project Structure

### Documentation (this feature)

```text
specs/038-plan-tab-task-nav/
├── plan.md              # This file
├── research.md          # Phase 0 output
├── contracts/
│   └── ui-bindings.md   # Key binding contract (UI contract)
└── tasks.md             # Phase 2 output (/speckit-tasks)
```

### Source Code (repository root)

```text
internal/tui/
├── keymap.go            # Add PlanGoToTask key.Binding + DefaultKeyMap entry + FullHelp entry
└── update.go            # Add case in handlePlanningKey (planList guard is already in place)
```

No new files. No changes to server, proto, or db.

## Phase 0: Research

See [research.md](research.md).

Key findings:
- `ensureVisible(id int64)` already exists at `update.go:1081` — walks the tree and expands all ancestors of the target task
- `buildVisible` + `findCursor` pattern for repositioning the cursor already used in `refreshedMsg` handler (`update.go:296-303`)
- `ctrl+t` is not bound to any existing action in `keymap.go` — safe to use
- `findCursor` returns `0` (not `-1`) when a task is not found — acceptable for the deleted-task edge case (cursor lands at top of list)
- The `planList` guard at `update.go:501-503` already prevents action keys from firing while sub-forms are open; the new handler inherits this guard automatically

## Phase 1: Design & Contracts

See [contracts/ui-bindings.md](contracts/ui-bindings.md).

### Implementation steps

1. **`internal/tui/keymap.go`** — three small changes:
   - Add `PlanGoToTask key.Binding` field to `KeyMap` struct
   - Add binding in `DefaultKeyMap()`: `key.WithKeys("ctrl+t")`, help text `"ctrl+t", "go to task"`
   - Add `k.PlanGoToTask` to the `FullHelp()` planning-mode slice (second row alongside `PlanAddTask`, `PlanAddEvent`, `PlanEdit`, `PlanRemove`)

2. **`internal/tui/update.go`** — one new case in the `handlePlanningKey` switch (after the `PlanRemove` case, before the `Complete` case):
   ```go
   case key.Matches(msg, m.keys.PlanGoToTask):
       if len(m.plan.entries) == 0 {
           return m, nil
       }
       entry := m.plan.entries[m.plan.cursor]
       if entry.TaskId == 0 {
           return m, nil
       }
       m.activeTab = tabTasks
       m.keys.PlanningMode = false
       m.plan.err = nil
       m.ensureVisible(entry.TaskId)
       m.visible = buildVisible(m.tree, m.expanded, m.showCompleted, m.pendingComplete)
       m.cursor = findCursor(m.visible, entry.TaskId)
       return m, nil
   ```

No helper functions, no new types, no new files beyond the contract document.
