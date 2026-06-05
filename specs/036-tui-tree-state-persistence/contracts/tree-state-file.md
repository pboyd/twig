# Contract: Tree State File

This feature adds no network/RPC contract. The authoritative contract is the **on-disk JSON file** that the TUI reads at launch and writes on change. It is defined here so the read/write implementation conforms to a stable shape (Constitution Principle II, adapted for a local-file interface).

## Location

```
$XDG_STATE_HOME/twig/tree-state.json      # if $XDG_STATE_HOME is set
~/.local/state/twig/tree-state.json       # fallback when $XDG_STATE_HOME is unset/empty
```

- This is the XDG state directory — the spec's home for "current state of the application that can be reused on restart (view, layout…)." It is distinct from `config.toml` (which lives under `$XDG_CONFIG_HOME`/`~/.config`).
- Go's stdlib has no `os.UserStateDir()`, so the path is resolved manually: prefer `$XDG_STATE_HOME`, else `$HOME/.local/state`.
- The file is machine-managed; users are not expected to edit it.

## Format

A single JSON object mapping **profile key** → array of **expanded task IDs**.

```json
{
  "default": [1, 4, 9],
  "work": [12, 17]
}
```

### Schema

| Path | Type | Notes |
|------|------|-------|
| `<profileKey>` | array of integers | Required value for each present key. Each integer is a task ID whose subtree is expanded. |
| (object root) | object | May be empty (`{}`) — a valid "nothing remembered" state. |

- **Profile key**: `"default"` for the root/unnamed account; otherwise the exact profile name.
- **Task IDs**: 64-bit integers (`int64`), matching server-assigned task IDs. Order is insignificant.
- **Encoding**: UTF-8 JSON, written compactly (pretty-printing is optional and not required by readers).

## Read behavior (load)

- **File absent** → treat as `{}` (empty). No error surfaced. (FR-005)
- **File present but unparseable / wrong shape** → treat as `{}`. No error surfaced; the next write repairs it. (FR-006)
- **Lookup**: read the array at the active profile key; a missing key → empty set.
- **Effect**: each ID seeds `Model.expanded[id] = true`. IDs not matching any current task are ignored (pruned on next write).

## Write behavior (save)

- **Trigger**: after the user expands/collapses a task, and on quit.
- **Pruning**: before writing, the active profile's array is reduced to IDs present in the live task tree. (FR-007 — bounds growth, drops deleted tasks.)
- **Isolation**: only the active profile's key is replaced; other profiles' arrays are preserved. (FR-009)
- **Atomicity**: write to a temporary file in the same directory, then `os.Rename` over `tree-state.json`. This guarantees readers never observe a partial file and makes concurrent-session writes "last writer wins" without corruption. (FR-007 edge case)
- **Directory creation**: the `twig` config directory is created if absent before the first write.

## Non-goals

- No cursor/selection position, active tab, scroll position, filter state, or any task data is stored. (Clarification: expand/collapse only.)
- No syncing to the server or other devices. (Clarification: local only.)
