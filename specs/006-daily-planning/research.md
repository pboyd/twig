# Research: Daily Planning

All decisions below are settled. There are no `NEEDS CLARIFICATION` items.

## 1. Time storage representation

**Decision**: Store the time window as three columns: `day DATE`, `start_minute SMALLINT` (0–1439), `duration_minute SMALLINT` (1–1440), with a CHECK enforcing `start_minute + duration_minute <= 1440`.

**Rationale**:
- The spec requires "local timezone" semantics and rejects entries that cross midnight. Storing a `TIMESTAMPTZ` invites timezone conversions on every read/write and forces the server to know each user's zone.
- Integer-minute storage makes the midnight-crossing rule a static CHECK constraint instead of an application invariant. It also makes overlap arithmetic trivially integer math, so the `Overlap` helper has no time-package edge cases (DST, leap-seconds, etc. cannot occur).
- The CLI already parses human time formats; converting `"1:15pm"` to `795` is a single multiplication and is the natural boundary between human and machine representation.

**Alternatives considered**:
- `TIMESTAMPTZ` columns: rejected — requires the server to know the user's timezone to interpret "the local 1:15pm of day D", and complicates the no-midnight-crossing rule.
- `TIMESTAMP WITHOUT TIME ZONE`: rejected — Postgres stores this without zone, but every comparison must agree on which zone "1:15" means; the CLI–server contract would still need to nail this down, gaining nothing over integer minutes.
- `tstzrange` / `int4range`: rejected — adds a Postgres-specific type to the sqlc bindings and to the proto-to-Go translation for marginal benefit; two `SMALLINT`s are clearer at the schema and proto boundaries.

## 2. Overlap enforcement

**Decision**: Enforce overlap inside the handler within a `SERIALIZABLE` transaction that locks the day's existing entries (`SELECT ... FROM plan_entries WHERE user_id = $1 AND day = $2 FOR UPDATE`) before inserting or updating.

**Rationale**:
- A `(user_id, day)` lock scope is small (handful of rows in practice), so contention is negligible at the one-to-two-user scale.
- Avoids a new Postgres extension. An exclusion constraint over `(user_id, day, int4range(start_minute, start_minute + duration_minute))` would require enabling `btree_gist` and would push the half-open boundary detail into DDL — workable but a heavier dependency for zero additional safety in this concurrency profile.
- The same transaction allocates the next `id`, so id allocation and overlap checking share one round-trip.

**Alternatives considered**:
- `EXCLUDE USING gist` with `int4range`: rejected for the reason above. Worth revisiting if the user base grows or if multi-tenant write contention emerges.
- Application-level mutex: rejected — does not survive multiple server processes; the transactional approach is the canonical answer for this constraint.

## 3. Per-day id allocation

**Decision**: Inside the insert transaction, compute `id := COALESCE(MAX(id), 0) + 1` over rows matching `(user_id, day)` and insert the new row with that id.

**Rationale**:
- The spec requires monotonically increasing per-day ids that are *not* renumbered when an entry is removed. `MAX(id)+1` produces exactly that behaviour: after `1, 2, 3` and a `rm 2`, the next insert is `4`.
- Sharing the insert transaction (already taken for overlap checking) means no separate counter row, no sequence to manage per user/day, and no risk of duplicate ids under concurrent inserts.

**Alternatives considered**:
- A per-day sequence: rejected — Postgres sequences are global objects; one per `(user_id, day)` is infeasible.
- A single global `BIGSERIAL` and exposing it: rejected — the spec says the first event added to a plan has id `1`; that is per-day, not global.
- `COUNT(*) + 1`: rejected — wrong after deletions.

## 4. Task FK behaviour on task deletion

**Decision**: `task_id BIGINT REFERENCES tasks(id) ON DELETE CASCADE`.

**Rationale**:
- Mirrors the convention adopted in feature 005 (`pomodoros.task_id` cascades). Consistency over inventing a third option.
- A plan entry that references a deleted task has no obvious meaning: the user removed the work, the schedule for it should go too. Carrying a dangling entry forward (via `ON DELETE SET NULL`) would require the entry to retain a name, but per FR-005 the name can be omitted when a task is linked — those entries would become unnameable.
- The spec marks task-deletion-while-referenced as "out of scope" (FR-020). Cascade is the minimum-surprise choice that does not require changes to `DeleteTask`.

**Alternatives considered**:
- `ON DELETE SET NULL`: rejected — would require us to backfill `name` from the task's name before the FK clears, which means a trigger or application code in `DeleteTask`. That *does* change task-deletion semantics, contradicting FR-020's spirit.
- `ON DELETE NO ACTION` (default): rejected — would cause `DeleteTask` to fail with an FK error whenever a plan entry referenced the task, surprising users and definitely changing `DeleteTask` semantics.

## 5. Where time-format parsing lives

**Decision**: Parsing of `"13:15"` / `"1:15pm"` / `"01:15 PM"` and of `"90m"` / `"1h30m"` / `"2h"` lives in a new `internal/cli/timeparse` package. The wire format is integer minutes (start) and integer minutes (duration).

**Rationale**:
- The server should not be parsing human formats; that is a CLI surface detail. Future clients (e.g. a web UI) will have their own input idioms and should not be locked into matching the CLI's exact accept-list.
- Integer minutes on the wire is unambiguous and free of timezone semantics.

**Alternatives considered**:
- Sending strings on the wire and parsing server-side: rejected — couples the proto contract to the CLI's exact string acceptance, and forces a second client to either replicate the parser or send through the CLI's vocabulary.

## 6. One service vs. extending `TaskService`

**Decision**: A new `proto/plan/v1/plan.proto` defining `PlanService`. Not added to `TaskService`.

**Rationale**:
- Seven new RPCs would nearly double the size of `TaskService`, which already absorbed five RPCs in feature 005. Conceptually, planning is a distinct domain from tasks — entries can exist without a task at all (events) — so keeping them in separate proto packages tracks the actual coupling.
- The Connect mux in `cmd/server/main.go` already registers handlers per service; adding one more registration is a one-line change.

**Alternatives considered**:
- Add to `TaskService`: rejected — mixes domains and makes the proto file harder to navigate. Cross-service references aren't needed because plan entries refer to tasks by primitive `int64 task_id`, not by importing the `Task` message.

## 7. Grid rendering granularity

**Decision**: One row per 15 minutes. Each row's first column is the time label; subsequent columns are either empty (gap) or carry the entry's `id` and display name.

**Rationale**:
- Most plan entries land on quarter-hour boundaries in practice. Finer granularity (5 min) bloats the output; coarser (30 min) loses fidelity for the common 15-minute meeting.
- An entry that doesn't align to a 15-minute boundary still renders correctly: the row it begins on shows it starting "mid-row" via a `~` prefix on the time label (e.g., `~10:05`); this is a visual hint, not a re-quantization of the data. The underlying minute resolution is preserved in the database.

**Alternatives considered**:
- Pure ASCII art with per-minute resolution: rejected — terminal width and height blow up for an ordinary day.
- Variable granularity per entry: rejected — implementation complexity for negligible UX gain.
