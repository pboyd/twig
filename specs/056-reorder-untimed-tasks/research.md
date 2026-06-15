# Research: Reorder Untimed Plan Entries in the TUI

All technical context was resolvable from the existing codebase; there were no open NEEDS CLARIFICATION items. The findings below record the decisions and the precedent each one follows.

## Decision 1: Persist order via a `position` column on `plan_entries`

- **Decision**: Add `position SMALLINT NOT NULL DEFAULT 0` to `plan_entries`, scoped to a `(user_id, day)` group, and use it to order untimed entries. Backfill existing rows by `ROW_NUMBER() OVER (PARTITION BY user_id, day ORDER BY id)`.
- **Rationale**: This is exactly how feature 037 added user-configurable order to `tasks` (migration `000008_task_position.up.sql`: `ALTER TABLE ... ADD COLUMN position INTEGER NOT NULL DEFAULT 0`, backfill by `ROW_NUMBER()`, add a covering index). Reusing the pattern satisfies Principle I (no new mechanism) and guarantees persistence across restarts (FR-006). A `SMALLINT` matches the existing `start_minute`/`duration_minute` sizing on this table and is ample for per-day entry counts.
- **Alternatives considered**:
  - *TUI-local order (e.g., reuse the tree-state persistence from 036)*: Rejected — would not survive across machines, would diverge from the task/goal ordering model, and the spec's persistence assumption calls for order to be a property of the data.
  - *Float/fractional positions to avoid renumbering*: Rejected as premature optimization (Principle I); per-day groups are tiny, so full transactional renumber is cheap and matches `ReorderTask`.

## Decision 2: Fold `position` into the existing day-listing order

- **Decision**: Change `ListPlanEntriesForDay` ordering from `start_minute ASC NULLS FIRST, id ASC` to `start_minute ASC NULLS FIRST, position ASC, id ASC`.
- **Rationale**: For timed entries `start_minute` dominates, so `position` is inert for them (Edge case: timed entries untouched, FR-004). For untimed entries (`start_minute IS NULL`) the secondary `position` key becomes the effective order, with `id` as a stable tiebreaker. This single shared query means the CLI and web app reflect the user-defined untimed order automatically (spec Assumption: order is a property of the data) without any new code on those surfaces.
- **Alternatives considered**: A separate untimed-only query — rejected as redundant; one ordered query already serves every reader.

## Decision 3: Anchor-based `ReorderPlanEntry` RPC on the existing `PlanService`

- **Decision**: Add `rpc ReorderPlanEntry(ReorderPlanEntryRequest) returns (ReorderPlanEntryResponse)`. The request carries `day`, the entry `id`, and a `oneof anchor { int32 before_id; int32 after_id; }`. The response returns the day's untimed entries in their new order with updated positions.
- **Rationale**: Mirrors `ReorderTask` (`api/proto/task/v1/task.proto`), which uses a `oneof anchor { before_task_id; after_task_id }` and returns the reordered sibling group. The `{`/`}` TUI gesture maps cleanly: rank-higher = "place before the previous untimed sibling", rank-lower = "place after the next untimed sibling". Returning the reordered group lets the client update without guessing. Identity on this table is `(user_id, day, id)`, so the request needs `day` + `id` (unlike tasks, which key on a global `task_id`).
- **Alternatives considered**:
  - *Reusing `MovePlanEntry`*: Rejected — that RPC changes `start_minute`/`duration` and would conflate scheduling with ordering; an untimed entry must stay untimed (FR-004).
  - *Index-based target instead of an anchor*: Rejected for consistency with `ReorderTask`'s anchor model and to avoid races on stale indices.

## Decision 4: Transactional read-lock + renumber in the handler

- **Decision**: In `ReorderPlanEntry`, begin a tx, validate that both the moved entry and the anchor exist, are on the same `day`, and are both untimed (`start_minute IS NULL`); lock the day's untimed entries `FOR UPDATE`, remove the moved entry from the ordered slice, reinsert at the anchor position, and rewrite positions `0..n-1`.
- **Rationale**: Directly mirrors the `ReorderTask` handler (`services/twig/internal/handler/task.go:448`), which reads `ListSiblingGroup ... FOR UPDATE`, rebuilds the slice in memory, and writes `UpdateTaskPosition`. This guarantees SC-003 (no loss/duplication) and convergence under concurrent edits. Rejecting a timed anchor or moved entry enforces FR-004 with a clear `InvalidArgument`.
- **Alternatives considered**: Single-statement swap of two positions — rejected; the anchor-insert model generalizes to any move and matches the existing handler, and the swap-only model complicates the "skip hidden completed entries" edge case.

## Decision 5: Wire `{`/`}` in `handlePlanningKey`, untimed-only

- **Decision**: Add `RankUp`/`RankDown` cases to `handlePlanningKey` (`internal/tui/update.go:1552`). On keypress, if the highlighted entry (`m.plan.entries[m.plan.cursor]`) is untimed, find its previous/next *untimed* neighbor and call a new `reorderPlanEntryCmd`; otherwise no-op. Reload the day highlighting the moved entry's id.
- **Rationale**: The `RankUp`/`RankDown` bindings (`{`/`}`) already exist in the keymap and are surfaced in the plan help row (`keymap.go:361`); they are currently handled only in the Tasks-tab handler, not in `handlePlanningKey`. Adding the cases here reuses the same gesture and help text (Principle III). Restricting to untimed entries satisfies FR-007 and the "highlight on a timed entry" edge case. Skipping hidden completed entries when finding the neighbor satisfies that edge case, matching the 037 "move past hidden ones" rule. Highlight-after-reload reuses the existing `listPlanHighlightCmd` / `planMutatedMsg.highlightID` mechanism so the moved entry stays selected (FR-003).
- **Alternatives considered**: A dedicated plan reorder key — rejected; would violate Principle III and force users to learn a new key (counter to SC-005).
