# Implementation Plan: TUI Tree State Persistence

**Branch**: `036-tui-tree-state-persistence` | **Date**: 2026-06-04 | **Spec**: [spec.md](./spec.md)

**Input**: Feature specification from `/specs/036-tui-tree-state-persistence/spec.md`

## Summary

Persist the TUI task tree's expand/collapse state across application restarts. State is stored **locally** on the user's machine, **scoped per profile**, and covers **only** which tasks are expanded. The TUI currently holds expansion state in an in-memory `map[int64]bool` (`Model.expanded`) keyed by task ID, where the absence of an entry means "collapsed" (the current launch default). The feature adds: (1) a small local JSON file recording, per profile, the set of currently-expanded task IDs; (2) loading that set at launch to seed `Model.expanded`; and (3) saving it (atomically, asynchronously, and pruned of deleted tasks) whenever the user changes expansion and on quit.

## Technical Context

**Language/Version**: Go 1.x (root CLI/TUI module `github.com/pboyd/twig`)

**Primary Dependencies**: `charmbracelet/bubbletea` (TUI runtime), `encoding/json` (stdlib, for state file), `BurntSushi/toml` (existing, for config — not changed here)

**Storage**: Local JSON file in the user's XDG state directory (`$XDG_STATE_HOME/twig/tree-state.json`, falling back to `~/.local/state/twig/tree-state.json`). No database, no server involvement.

**Testing**: `go test ./...` from repo root; table-driven unit tests following existing `internal/tui` conventions (`export_test.go` shims for unexported helpers). Tests inject a temp file path — no real filesystem dependence on the user's home dir.

**Target Platform**: Linux/macOS terminal (TTY) — same as the existing TUI.

**Project Type**: Single project (Go CLI/TUI), repo-root module. No web/mobile tiers touched.

**Performance Goals**: State load adds no perceptible startup delay (SC-005); the file is tiny (a list of integers per profile). Saves are asynchronous via a `tea.Cmd` and never block the UI.

**Constraints**: Must not surface errors to the user on missing/corrupt state (FR-005, FR-006) — degrade silently to default (all-collapsed). Stored state must not grow without bound (FR-007) — prune to live task IDs on save. Writes must be atomic (temp file + rename) so concurrent sessions / abnormal exit cannot corrupt the file.

**Scale/Scope**: Personal task trees (tens to low-hundreds of tasks). A handful of profiles. Single source file plus tests; wiring into `tui.Run` and two/three handler sites.

## Constitution Check

*GATE: Must pass before Phase 0 research. Re-check after Phase 1 design.*

| Principle | Status | Notes |
|-----------|--------|-------|
| I. Simplicity / YAGNI | ✅ | One small helper file (load/save/prune) + a JSON file. Stores the minimal representation (the expanded-ID set), reusing the existing `map[int64]bool`. No new abstraction layer, no DB, no server changes. |
| II. API-First Design | ✅ | No server API change. The relevant contract is the on-disk state-file schema, defined in `contracts/tree-state-file.md` before implementation. |
| III. UI/UX Consistency | ✅ | No new visible surface, colors, or key bindings — the tree renders exactly as today; only its initial expansion is seeded from disk. |
| IV. Playful User Messages | ✅ | Persistence is silent by design (FR-005/FR-006 require no error surfaced). No new user-facing text is introduced. |

**Post-Design Re-check**: ✅ Unchanged — the design adds no new abstractions, surfaces, or user-facing copy.

## Project Structure

### Documentation (this feature)

```text
specs/036-tui-tree-state-persistence/
├── plan.md              # This file
├── research.md          # Phase 0 output
├── data-model.md        # Phase 1 output
├── quickstart.md        # Phase 1 output
├── contracts/
│   └── tree-state-file.md   # On-disk state-file schema (the "contract")
└── tasks.md             # Phase 2 output (/speckit-tasks — NOT created here)
```

### Source Code (repository root)

```text
internal/
├── tui/
│   ├── treestate.go         # NEW: resolve XDG state path, load, save (atomic), and prune the per-profile expanded-ID set
│   ├── treestate_test.go    # NEW: unit tests (load missing/corrupt, round-trip, prune, per-profile isolation)
│   ├── export_test.go       # MODIFIED: expose treestate helpers if needed by tests
│   ├── model.go             # MODIFIED: seed Model.expanded from loaded state; hold state-file path + profile key
│   ├── update.go            # MODIFIED: emit async save command on Expand/Collapse and on quit
│   └── tui.go               # MODIFIED: in Run(), resolve state path, load set for the active profile, pass into newModel
└── config/
    └── config.go            # (reference only) shows the os.UserConfigDir env-respecting path pattern; not modified
```

**Structure Decision**: Single Go project, repo-root module. All changes live in `internal/tui` (the TUI owns its view state). Because Go's stdlib has `os.UserConfigDir()`/`os.UserCacheDir()` but **no `os.UserStateDir()`**, `treestate.go` resolves the XDG state directory itself: `$XDG_STATE_HOME` if set, else `$HOME/.local/state`, then `/twig/tree-state.json`. The new `treestate.go` isolates filesystem concerns from the Bubble Tea model so it can be unit-tested with an injected path.

## Complexity Tracking

> No Constitution Check violations. This section intentionally left empty.
