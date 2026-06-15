# Data Model: Reorder Untimed Plan Entries

## Modified entity: Plan Entry (`plan_entries`)

A plan entry is one scheduled (or untimed) segment of a single day for a user. This feature adds an ordering attribute.

### Existing columns (unchanged)

| Column | Type | Notes |
|---|---|---|
| `user_id` | BIGINT | PK part; FK → `users(id)` ON DELETE CASCADE |
| `day` | DATE | PK part |
| `id` | INT | PK part; per-(user, day) sequence (`NextPlanEntryId`) |
| `task_id` | BIGINT (nullable) | FK → `tasks(id)`; null for events |
| `name` | VARCHAR(255) (nullable) | required when `task_id` is null |
| `start_minute` | SMALLINT (nullable) | **null ⇒ untimed**; 0–1439 when set |
| `duration_minute` | SMALLINT | > 0, ≤ 1440 |

Primary key: `(user_id, day, id)`.

### New column

| Column | Type | Notes |
|---|---|---|
| `position` | SMALLINT NOT NULL DEFAULT 0 | Relative order within the `(user_id, day)` group. Only semantically significant for untimed entries; timed entries are ordered by `start_minute` first, so their `position` is inert. |

### New index

```sql
CREATE INDEX plan_entries_user_day_position_idx ON plan_entries (user_id, day, position);
```

Supports the ordered listing and the `FOR UPDATE` group read during reorder.

### Backfill (migration up)

```sql
WITH ordered AS (
    SELECT user_id, day, id,
           ROW_NUMBER() OVER (
               PARTITION BY user_id, day
               ORDER BY id
           ) - 1 AS pos
    FROM plan_entries
)
UPDATE plan_entries p SET position = ordered.pos
FROM ordered
WHERE ordered.user_id = p.user_id AND ordered.day = p.day AND ordered.id = p.id;
```

This preserves today's effective untimed order (creation order = `id ASC`) so no entry visibly moves on upgrade (spec Assumption: default order before manual reorder = creation order).

## Validation rules

- **Reorder is untimed-only**: both the moved entry and the anchor MUST have `start_minute IS NULL`. A reorder request naming a timed entry (as mover or anchor) is rejected (`InvalidArgument`). (FR-004, FR-007)
- **Same day**: the moved entry and anchor MUST share the same `(user_id, day)`. (FR-001 scope; spec Assumption: order is per-day)
- **Distinct anchor**: the anchor `id` MUST differ from the moved `id`.
- **Exactly one anchor**: precisely one of `before_id` / `after_id` is set.
- **No-op at boundary**: ranking the first untimed entry higher or the last lower leaves positions unchanged and returns success. (FR-005)
- **Set integrity**: after a reorder, the day's untimed entries are exactly the same set, renumbered `0..n-1` with no gaps, losses, or duplicates. (SC-003)

## Ordering semantics

`ListPlanEntriesForDay` returns entries ordered by:

```sql
ORDER BY start_minute ASC NULLS FIRST, position ASC, id ASC
```

- Untimed entries (`start_minute IS NULL`) sort first, among themselves by `position` then `id`.
- Timed entries sort by `start_minute`; `position` does not affect their relative order.

## State transitions affecting position

| Transition | Effect on `position` |
|---|---|
| New untimed entry inserted | `position = COALESCE(MAX(position)+1, 0)` within `(user_id, day)` → lands at the end of the group (spec Assumption). |
| Untimed entry gains a start time (scheduled) | Leaves the untimed group visually; its stored `position` is retained but no longer ordering-significant. Remaining untimed entries keep relative order. |
| Timed entry's start time cleared (returns to untimed) | Re-enters the untimed group; default placement at the end of the group per the insert/end rule. |
| Untimed entry reordered via `ReorderPlanEntry` | Day's untimed entries renumbered `0..n-1` in the new order. |
| Entry deleted | Remaining entries keep their stored positions (gaps are harmless; ordering only compares values). |
