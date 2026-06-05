# Phase 0 Research: TUI Tree State Persistence

All clarifications from the spec's `## Clarifications` session are resolved (local-only storage, per-profile scope, expand/collapse only). No `NEEDS CLARIFICATION` markers remain. This document records the design decisions grounded in the existing codebase.

## Decision 1: Representation — store the set of expanded task IDs

- **Decision**: Persist, per profile, the list of task IDs whose subtree is currently **expanded**. Collapsed/unknown tasks are simply absent.
- **Rationale**: The TUI already models expansion as `Model.expanded map[int64]bool` (`internal/tui/model.go:90`), and `buildVisible` (`internal/tui/tree.go:48`) treats a missing/false entry as collapsed. The current launch default is "all collapsed" because `newModel` initializes an empty map (`internal/tui/model.go:121`). Storing only the expanded set therefore (a) reproduces saved state exactly, and (b) makes unknown/new tasks default to collapsed automatically — satisfying FR-004 with zero extra logic.
- **Alternatives considered**:
  - *Store every task's explicit true/false*: larger, redundant, and still needs a default for unknown tasks. Rejected (violates Simplicity).
  - *Store collapsed set instead*: would invert the default and require knowing the full task set to be meaningful. Rejected.

## Decision 2: File location & format

- **Decision**: A single JSON file at `$XDG_STATE_HOME/twig/tree-state.json` (falling back to `~/.local/state/twig/tree-state.json` when `$XDG_STATE_HOME` is unset), a JSON object mapping profile key → array of expanded task IDs (integers). The default/root account uses the key `"default"`.
- **Rationale**: The XDG Base Directory Specification designates `$XDG_STATE_HOME` for "state data that should persist between restarts but is not important/portable enough for config," giving "current state of the application that can be reused on restart (view, layout, open files, undo history…)" as the canonical example — an exact match for tree expansion state. Go's stdlib lacks an `os.UserStateDir()`, so `treestate.go` resolves it directly: read `$XDG_STATE_HOME`; if empty, use `$HOME/.local/state`. JSON via `encoding/json` (stdlib) is the simplest serializer for an integer-list payload and needs no new dependency. A single file with a profile-keyed map keeps per-profile isolation (FR-009) trivial and makes pruning a per-key operation.
- **Alternatives considered**:
  - *`$XDG_CONFIG_HOME` (`~/.config`, via `os.UserConfigDir()`)*: that directory is for hand-authored, user-editable config; this state is machine-managed. Wrong semantics. Rejected.
  - *`$XDG_CACHE_HOME` (`~/.cache`, via `os.UserCacheDir()`)*: stdlib-supported and env-respecting, but cache semantics are "regenerable, safe to delete anytime." Expansion state is the user's curation and cannot be regenerated — a cache wipe would silently discard it. Acceptable degradation but looser than state. Rejected in favor of `$XDG_STATE_HOME`.
  - *Embed in `config.toml`*: config is user-editable, hand-authored, and round-tripping TOML while preserving comments is fragile. View state is machine-managed — keep it separate. Rejected.
  - *One file per profile* (`tree-state.<profile>.json`): more files, more path logic. Rejected (no benefit at this scale).

## Decision 3: When to save

- **Decision**: Emit an asynchronous save (a `tea.Cmd`) whenever the user toggles expansion (the `Expand`/`Collapse` key handlers in `internal/tui/update.go:693,711`) and also on quit (the `tea.Quit` paths in `handleListKey`). The command writes the current pruned expanded set for the active profile.
- **Rationale**: Saving on each deliberate change keeps the on-disk state current and satisfies the abnormal-exit edge case "to the extent reasonably achievable" without relying on a shutdown hook (Bubble Tea offers no guaranteed exit callback). Routing the write through a `tea.Cmd` keeps `Update` side-effect-free and never blocks rendering. The file is tiny, so per-toggle writes are negligible.
- **Alternatives considered**:
  - *Save only on quit*: simpler (2 sites) but loses changes on abnormal termination and on incidental expansions. The async command is cheap enough that change-driven saving is worth it.
  - *Debounce/batch writes*: unnecessary at this file size and task count (YAGNI).

## Decision 4: Pruning & robustness

- **Decision**: On every save, write only IDs present in the current `Model.tree`; this bounds file growth (FR-007). On load, a missing file yields an empty set (no error); an unreadable/corrupt file is ignored and treated as empty, and the next save overwrites it with valid content (FR-005, FR-006). All writes are atomic: write to a temp file in the same directory, then `os.Rename` over the target.
- **Rationale**: Pruning at save time is the natural choke point — the live tree is known there, whereas at load time the tree hasn't been fetched yet. Atomic rename prevents corruption from concurrent sessions ("last writer wins", FR-007 edge case) or interrupted writes. Silent fallback matches the spec's no-error requirement and the constitution's expectation that persistence stays invisible.
- **Alternatives considered**:
  - *Prune at load*: not possible before tasks are fetched; would require a second pass. Rejected.
  - *File locking across sessions*: overkill for a single-user local tool; last-writer-wins is acceptable. Rejected (YAGNI).

## Decision 5: Profile key resolution

- **Decision**: Derive the profile key in `tui.Run` from its existing `profile` parameter (`internal/tui/tui.go:15`): `""` and `"default"` map to `"default"`; any named profile uses its name verbatim.
- **Rationale**: `Run` already resolves the profile for credentials (`config.Profile`), so the name is in scope at the exact point where state must be loaded and the model constructed. This mirrors the precedence already documented for profiles and needs no new plumbing.
- **Alternatives considered**: Passing the whole `Config` down to the state layer — unnecessary; only the resolved key is needed.

## Testing approach

- Unit-test `treestate.go` with an injected file path (temp dir), table-driven per `internal/tui` conventions: missing file → empty set; corrupt file → empty set, no error; round-trip save→load; prune drops IDs absent from a given live set; two profiles in one file stay isolated; atomic overwrite leaves a valid file.
- No running database or server is required, consistent with the project's testing conventions.
