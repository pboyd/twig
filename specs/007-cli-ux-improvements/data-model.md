# Data Model: CLI UX Improvements

No new entities. No schema changes. This document captures which existing fields drive new rendering behavior.

## Task (existing, from `proto/task/v1/task.proto`)

| Field | Type | Used by this feature for |
|-------|------|--------------------------|
| `id` | `int64` | Already displayed as `[id]` prefix; unchanged. |
| `name` | `string` | Body of the rendered line. Stays unchanged when `todo task mod <id> --parent <p>` is run without a name. |
| `parent_id` | `int64?` | Updated by `todo task mod <id> --parent <p>` without touching `name`. |
| `completed_at` | `Timestamp?` | Presence drives the dim+strikethrough rendering. Replaces the `[x]` / `[ ]` checkbox marker. |
| `estimate` | `int32` | When `> 0`, rendered as ` (N)` after the task name. `0` means "no estimate" (per proto comment) and renders nothing. |
| `due` | `Timestamp?` | Existing `(due …)` suffix; unchanged by this feature. |

## State transitions

None added. Existing transitions (incomplete → complete via `runComplete`) already drive the `completed_at` field that this feature reads.
