# Phase 1 Data Model: TUI Tree State Persistence

This feature introduces no server-side or database entities. The only persisted data is a local, client-side view-state file. Entities below are conceptual and map directly to in-memory Go types and the on-disk JSON described in `contracts/tree-state-file.md`.

## Entity: Tree View State (file)

The complete persisted artifact — one JSON file per machine, shared across profiles via keys.

| Field | Type | Description |
|-------|------|-------------|
| profiles | map<string, ExpandedSet> | Maps a profile key to that profile's expanded-task set. |

- **Identity**: The file itself, located at `$XDG_STATE_HOME/twig/tree-state.json` (fallback `~/.local/state/twig/tree-state.json`).
- **Lifecycle**: Created lazily on first save. Read at TUI launch. Overwritten atomically on each save. Never deleted by the app.
- **Validation / robustness**: A missing or unparseable file is treated as an empty Tree View State (no profiles). No error is surfaced (FR-005, FR-006).

## Entity: Profile Key

| Field | Type | Description |
|-------|------|-------------|
| key | string | `"default"` for the root/unnamed account; otherwise the profile name verbatim. |

- **Derivation**: From the `profile` argument to `tui.Run`. Empty or `"default"` → `"default"`.
- **Relationship**: One Profile Key → one ExpandedSet within the file (FR-009: profiles are isolated).

## Entity: Expanded Set

The remembered expansion choices for a single profile.

| Field | Type | Description |
|-------|------|-------------|
| expandedTaskIDs | list<int64> | Task IDs whose subtree is currently expanded. Order is not significant; duplicates are not meaningful. |

- **Semantics**: Presence of an ID = expanded. Absence = collapsed (the launch default). This mirrors `Model.expanded map[int64]bool` where `true` entries are the set members.
- **Validation rules**:
  - IDs are stable task identifiers (server-assigned, `int64`).
  - On **save**, the set is pruned to IDs present in the live task tree (FR-007) — deleted-task IDs are dropped, bounding growth.
  - On **load**, all stored IDs seed `Model.expanded[id] = true`; any that no longer exist are harmless (they match no node in `buildVisible`) and are pruned on the next save.

## Relationship to existing in-memory state

```
tree-state.json
└── profiles["default"].expandedTaskIDs = [1, 4, 9]
        │  load at Run() → seed
        ▼
Model.expanded = {1:true, 4:true, 9:true}   (internal/tui/model.go)
        │  toggled by Expand/Collapse keys (internal/tui/update.go)
        ▼  save (async tea.Cmd), pruned to live tree IDs
profiles["default"].expandedTaskIDs = [...]
```

No change to task data, the `cli.TreeNode` structure, or any server message — restoring state only seeds local view flags (FR-008).
