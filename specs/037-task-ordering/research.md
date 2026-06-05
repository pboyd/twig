# Phase 0 Research: User-Configurable Task Order

All spec `[NEEDS CLARIFICATION]` items were resolved during `/speckit-clarify` (web drag is reorder-only; TUI `{`/`}` skips hidden siblings). The remaining open questions were design/technology decisions, resolved below.

## Decision 1: How to represent sibling order

**Decision**: Add an integer `position` column to `tasks`, meaningful **only within a `(user_id, parent_id)` sibling group**. Siblings are ordered by `position` ascending, `id` ascending as a stable tiebreaker. Reordering renumbers the affected group to a contiguous `0..n-1` sequence inside a single transaction.

**Rationale**:
- Simplest representation that satisfies the requirement (Principle I). An integer column + `ORDER BY position, id` is trivial to read and reason about.
- Sibling groups are personal-scale (tens of items at most), so renumbering a whole group on each reorder is cheap and keeps positions clean and gap-free — no rebalancing logic, no precision drift.
- `id` tiebreaker guarantees a total order even before any backfill/edge race, so the tree is always deterministic (supports SC-005: every task appears exactly once).

**Alternatives considered**:
- *Fractional/float positions* (insert-between via averaging): avoids renumbering but introduces precision exhaustion and messy values; unjustified complexity at this scale.
- *Linked list (`prev_id`/`next_id`)*: O(1) splice but fragile to maintain, harder to query/order, and easy to corrupt. Rejected (Principle I).
- *Client-only ordering preference*: violates FR-002 (must persist centrally and be shared across clients).

## Decision 2: The reordering API contract

**Decision**: One new RPC, `ReorderTask`, that moves a task to sit immediately **before** or **after** a sibling **anchor**:

```
ReorderTask(task_id, oneof { before_task_id | after_task_id })
```

The anchor must be a sibling of `task_id` (same `parent_id`, same user) and not the task itself. The RPC never changes `parent_id`. The server renumbers the group and returns the updated sibling list.

**Rationale**:
- A single anchor-based operation serves **both** write clients cleanly and symmetrically:
  - TUI `{` → `before_task_id` = previous *visible* sibling; TUI `}` → `after_task_id` = next *visible* sibling.
  - Web drag → drop target maps to a before/after anchor (see `reorderAnchor.ts`).
- Anchor-based moves are **concurrency-friendly** (FR-015): the operation expresses *relative* intent, so it converges without requiring the client to submit the whole group or to know about siblings another client just added. No task is dropped or duplicated because the server reads the live group, removes the task, and reinserts it relative to the still-present anchor.
- Choosing **both** before and after anchors (a `oneof`) keeps every client mapping a one-liner; "move to front" = before the first sibling, "move to end" = after the last sibling, so no separate "to front/end" mode is needed.

**Alternatives considered**:
- *`ReorderSiblings(parent_id, ordered_ids[])` full-list rewrite*: easy for web drag but requires clients to send (and keep in sync) the full group including hidden tasks, and needs set-equality handling against concurrent inserts. More data, more failure modes. Rejected.
- *Reuse `UpdateTask` with a position field*: `UpdateTask` is full-replace over editable fields and would conflate parent moves with ordering and invite lost updates. A dedicated, narrow RPC is clearer (Principle II).
- *Direction-only `Reorder(task_id, UP|DOWN)`*: fine for the TUI but can't express the web's arbitrary drop target. Rejected.

## Decision 3: Default position on create and on re-parent

**Decision**: A newly created task is appended to the **end** of its sibling group (`position = COALESCE(MAX(position) in group, -1) + 1`). When `UpdateTask` changes a task's `parent_id` (the existing "move" capability), the moved task is assigned the **end** position of its destination group; the source group keeps its relative order.

**Rationale**: Matches the spec assumptions and FR-012/FR-013; end-of-list is the least surprising default and avoids disturbing existing siblings. Reparenting placement stays the responsibility of `UpdateTask` (re-parenting via web drag is explicitly out of scope per the clarification).

**Alternatives considered**: Insert-at-top — rejected as more surprising for a backlog-style list.

## Decision 4: TUI behavior with hidden (completed) tasks

**Decision**: `{`/`}` operate over the **visible** sibling sequence. The client computes the visible previous/next sibling as the anchor, so the move swaps past any hidden siblings in between and always produces a visible change unless the task is at a visible boundary (first sibling for `{`, last for `}`), where it is a silent no-op.

**Rationale**: Directly implements the `/speckit-clarify` answer ("move past hidden ones") and FR-005/FR-007. The TUI already has the full task set in memory (`ListTasks` returns all tasks), so it can derive the visible-sibling anchor without extra round-trips.

**Alternatives considered**: Operating on the raw stored order (swap with immediate, possibly-hidden neighbor) — rejected because a keypress could produce no visible change and feel unresponsive.

## Decision 5: Web drag-and-drop library

**Decision**: Use `@dnd-kit/core` + `@dnd-kit/sortable`.

**Rationale**:
- React 19-compatible, actively maintained, and the de-facto standard for accessible sortable lists; provides keyboard-operable drag and sensible pointer/touch sensors out of the box (supports Principle III consistency and accessible UX).
- Sortable context maps cleanly onto a single sibling group, which is exactly our reorder scope (reorder-only, no re-parenting), keeping the integration small.

**Alternatives considered**:
- *Native HTML5 drag-and-drop*: zero dependencies but poor accessibility, awkward nested-tree behavior, and significant hand-rolled drop-target math. The extra dependency is justified by avoiding that bespoke complexity.
- *react-dnd*: heavier API and less momentum than dnd-kit for simple sortable lists.

## Decision 6: Where sorting is applied across clients

**Decision**: Centralize sibling sorting in the tree builders. Change `internal/cli/render.go` `SortNodes` to order by `position` (then `id`), which the TUI inherits via `cli.BuildTree`. In the web app, sort each sibling list by `position` in `src/lib/tree.ts`. Server `ListTasks` may keep returning all tasks; ordering is authoritative in the builders, so the flat-list order from the DB is not relied upon.

**Rationale**: One change point per consumer; both Go clients share `cli.BuildTree`, so the CLI automatically "respects the order defined elsewhere" (FR-010/FR-011) with no new CLI code. Keeping sort in the builder (rather than depending on SQL row order) is robust to how the flat list is assembled.

**Alternatives considered**: Relying solely on `ORDER BY position` in `ListTasks` — fragile because tree builders re-group by parent and currently re-sort; explicit builder sorting is clearer and unit-testable.
