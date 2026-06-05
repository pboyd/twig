---
description: "Task list for TUI Tree State Persistence"
---

# Tasks: TUI Tree State Persistence

**Input**: Design documents from `/specs/036-tui-tree-state-persistence/`

**Prerequisites**: plan.md, spec.md, research.md, data-model.md, contracts/tree-state-file.md

**Tests**: Included. The `internal/tui` package is test-heavy by convention and the plan's testing approach calls for `treestate_test.go`. Tests use injected temp paths — no DB or server required.

**Organization**: Tasks are grouped by user story (from spec.md) so each story can be implemented and verified independently.

## Format: `[ID] [P?] [Story] Description`

- **[P]**: Can run in parallel (different file, no dependency on an incomplete task)
- **[Story]**: US1 / US2 / US3 — maps to the spec's user stories
- All paths are relative to repo root `/home/user/dev/twig/`

## Path Conventions

Single Go project, repo-root module `github.com/pboyd/twig`. All feature code lives in `internal/tui/`. No `src/` tree.

---

## Phase 1: Setup

**Purpose**: Create the new persistence module file.

- [X] T001 Create `internal/tui/treestate.go` with the package declaration and the `treeStateFile` type (`map[string][]int64`, profile key → expanded task IDs), documented to match `specs/036-tui-tree-state-persistence/contracts/tree-state-file.md`

---

## Phase 2: Foundational (Blocking Prerequisites)

**Purpose**: Path resolution, the read side, and model plumbing that every user story depends on.

**⚠️ CRITICAL**: No user story work can begin until this phase is complete.

- [X] T002 Implement `treeStatePath()` and `profileKey()` in `internal/tui/treestate.go` — resolve `$XDG_STATE_HOME` (fallback `$HOME/.local/state`) then `/twig/tree-state.json`; normalize `""` and `"default"` to the key `"default"`
- [X] T003 Implement `loadTreeState(path)` (read + JSON-decode the file; a **missing** file returns an empty `treeStateFile`, no error) and `expandedSet(key) map[int64]bool` (for seeding the model) in `internal/tui/treestate.go`
- [X] T004 [P] Add `statePath` and `profileKey` fields to `Model` and seed `expanded` from a passed-in `map[int64]bool` in `newModel` (`internal/tui/model.go`)

**Checkpoint**: The state file can be located and read; the model can be constructed with a pre-seeded expansion map.

---

## Phase 3: User Story 1 - Resume the tree exactly as I left it (Priority: P1) 🎯 MVP

**Goal**: Expansion choices survive quit/relaunch — the tree reopens exactly as the user left it.

**Independent Test**: Expand/collapse a known set of tasks, quit, relaunch, and confirm the same tasks are expanded/collapsed.

### Implementation for User Story 1

- [X] T005 [US1] Implement `saveTreeState` in `internal/tui/treestate.go`: atomically write the active profile's expanded task IDs to `tree-state.json` via a temp file in the same directory + `os.Rename`, creating the directory if absent
- [X] T006 [US1] Add `persistTreeStateCmd` to `internal/tui/update.go`: a `tea.Cmd` that snapshots the current expanded set for the active profile and calls `saveTreeState` (returns an ignored/no-op message; errors are swallowed per FR-005)
- [X] T007 [US1] In `tui.Run` (`internal/tui/tui.go`), resolve the state path and profile key, call `loadTreeState`, and seed `newModel`'s expanded map from `expandedSet(key)`
- [X] T008 [US1] Emit `persistTreeStateCmd` from the `Expand` and `Collapse` key handlers in `internal/tui/update.go` after each toggle
- [X] T009 [US1] Persist on quit: in the `tea.Quit` paths of `handleListKey` (`internal/tui/update.go`), save before exiting (synchronous save, or `tea.Sequence(persistCmd, tea.Quit)`) so a normal quit always records the latest state
- [X] T010 [P] [US1] Add tests in `internal/tui/treestate_test.go`: `treeStatePath` honors `$XDG_STATE_HOME` and falls back to `~/.local/state`; and a save→load round-trip asserts the expanded set is restored
- [X] T011 [US1] Add any needed unexported-helper shims to `internal/tui/export_test.go` so `treestate_test.go` can exercise the package internals

**Checkpoint**: MVP complete — curate the tree, quit, relaunch, and the same branches are expanded. SC-001 and SC-002 verifiable.

---

## Phase 4: User Story 2 - Sensible behavior for unseen tasks & robustness (Priority: P2)

**Goal**: New/unknown tasks use the default (collapsed) without corrupting saved state; deleted tasks don't accumulate; missing/corrupt state degrades silently.

**Independent Test**: With saved state present, add a new task (defaults to collapsed while others keep their state); corrupt the file and confirm the TUI still opens; delete a remembered task and confirm its ID stops being persisted.

### Implementation for User Story 2

- [X] T012 [US2] Make `loadTreeState` tolerate a corrupt/unparseable file by returning an empty `treeStateFile` with no error (FR-006) in `internal/tui/treestate.go`
- [X] T013 [US2] Add pruning to the save path: compute the live task-ID set from `Model.tree` in `persistTreeStateCmd` (`internal/tui/update.go`) and have `saveTreeState` write only IDs present in that live set (FR-007), in `internal/tui/treestate.go`
- [X] T014 [P] [US2] Tests in `internal/tui/treestate_test.go`: missing file → empty set; corrupt file → empty set with no error; pruning drops IDs absent from the supplied live set
- [X] T015 [P] [US2] Test in `internal/tui/tree_test.go` that a task with no stored/seeded entry renders collapsed via `buildVisible` (unknown/new tasks default correctly)

**Checkpoint**: Robust against missing/corrupt files and bounded against deleted-task growth; SC-003 and SC-004 verifiable.

---

## Phase 5: User Story 3 - State stays separate per profile (Priority: P3)

**Goal**: Each profile/account remembers its own tree state; one profile's choices never appear under another.

**Independent Test**: Curate distinct trees under two profiles, relaunch each with `--profile`, and confirm each shows only its own state.

### Implementation for User Story 3

- [X] T016 [US3] Make `saveTreeState` preserve other profiles' entries — load the existing file, replace only the active profile's key, then atomically write — in `internal/tui/treestate.go` (FR-009)
- [X] T017 [US3] Confirm `tui.Run` (`internal/tui/tui.go`) passes the resolved profile key (from `--profile` / `TWIG_PROFILE` selection) into both load and `newModel`, so load/seed/save all use the same key
- [X] T018 [P] [US3] Tests in `internal/tui/treestate_test.go`: saving profile `"work"` leaves a pre-existing `"default"` entry untouched; `profileKey("")` and `profileKey("default")` resolve to the same key

**Checkpoint**: All three stories independently functional; per-profile isolation verifiable.

---

## Phase 6: Polish & Cross-Cutting Concerns

**Purpose**: Verification and constitution compliance.

- [X] T019 [P] Run `gofmt -w` and `go vet ./...` on the changed files; resolve any findings
- [X] T020 Run the full suite from repo root: `go test ./...`, and confirm a clean build with `go build -o twig ./cmd/twig`
- [X] T021 Execute `specs/036-tui-tree-state-persistence/quickstart.md` manual validation (resume, corrupt-file fallback, deleted-task prune, per-profile isolation)
- [X] T022 [P] Verify persistence introduces no new user-facing text (silent per Constitution IV); if any error path must surface text, give it a playful, actionable tone

---

## Dependencies & Execution Order

### Phase Dependencies

- **Setup (Phase 1)**: No dependencies — start immediately.
- **Foundational (Phase 2)**: Depends on Setup. **Blocks all user stories.**
- **User Stories (Phase 3–5)**: All depend on Foundational. US1 is the MVP.
- **Polish (Phase 6)**: Depends on the desired stories being complete.

### User Story Dependencies

- **US1 (P1)**: Depends only on Foundational. Delivers the MVP.
- **US2 (P2)**: Depends on Foundational; T013 extends `saveTreeState` created in US1 (T005), so US2 follows US1.
- **US3 (P3)**: Depends on Foundational; T016 also extends `saveTreeState` (T005). US3 follows US1.
- **US2 ↔ US3**: Both modify `saveTreeState` in `internal/tui/treestate.go`, so do them sequentially (P2 then P3), not in parallel.

### Within Each Story

- Implementation before its tests can pass; tests marked [P] live in separate files and can be written alongside.
- Save (T005) before pruning (T013) and before preserve-other-keys (T016).

### Parallel Opportunities

- **Phase 2**: T004 (`model.go`) is [P] vs T002/T003 (`treestate.go`) — different files.
- **US1**: T010 (`treestate_test.go`) is [P] vs implementation in `treestate.go`/`update.go`/`tui.go`.
- **US2**: T014 (`treestate_test.go`) and T015 (`tree_test.go`) are [P] — different test files.
- **Polish**: T019 and T022 are [P].
- Note: T002 and T003 share `treestate.go` → **not** parallel with each other.

---

## Implementation Strategy

### MVP First

Complete **Phase 1 → Phase 2 → Phase 3 (US1)**. That alone delivers the feature's core promise: the tree reopens as the user left it. Ship/validate before layering robustness.

### Incremental Delivery

1. **US1** — resume on restart (MVP). Verify SC-001, SC-002, SC-005.
2. **US2** — robustness + bounded growth. Verify SC-003, SC-004.
3. **US3** — per-profile isolation. Verify FR-009.
4. **Polish** — fmt/vet, full suite, quickstart, tone check.

### Suggested MVP Scope

T001–T011 (Setup + Foundational + US1).
