# Phase 0 Research: Task CRUD API

**Feature**: 001-task-crud-api | **Date**: 2026-05-16

The technology stack is fixed by the existing `services/todo` backend (Go + Connect-Go + pgx + sqlc + golang-migrate + Buf). No stack selection was needed. The decisions below resolve the design questions specific to this feature. No `NEEDS CLARIFICATION` items remain.

## R1 — Identifier generation (FR-004)

**Decision**: Define `id` as `BIGINT GENERATED ALWAYS AS IDENTITY PRIMARY KEY`.

**Rationale**: An identity column produces positive, monotonically increasing integers and draws from a sequence that never reuses values — even after a row is deleted — satisfying FR-004's "MUST NOT reuse the identifier of a deleted task". `GENERATED ALWAYS` prevents callers from supplying an `id`, keeping it server-assigned. sqlc maps `BIGINT` to Go `int64`.

**Alternatives considered**: `SERIAL` — legacy, equivalent behavior but discouraged in modern PostgreSQL. Application-generated IDs — rejected; reinvents what the database does correctly and risks gaps/collisions.

## R2 — Parent reference, existence enforcement, and cascade delete (FR-013, FR-014, FR-017)

**Decision**: Add a nullable self-referencing column `parent_id BIGINT REFERENCES tasks(id) ON DELETE CASCADE`.

**Rationale**: One column carries three requirements at once:
- `NULL` parent ⇒ top-level task (FR-013).
- The foreign-key constraint rejects any insert/update whose `parent_id` does not match an existing row, satisfying FR-014 at the database level. The handler pre-checks existence to return a clean `InvalidArgument` error rather than surfacing a raw constraint violation.
- `ON DELETE CASCADE` makes deleting a task automatically delete its children, which recurse — deleting an entire subtree — satisfying FR-017 (cascade delete) without any application-side traversal.

**Alternatives considered**: Application-side recursive delete — rejected; the database already does this atomically and correctly. A nullable parent with no FK — rejected; would push existence checks and orphan cleanup into application code, violating Principle I (reuse over invention).

## R3 — Cycle prevention (FR-016)

**Decision**: Before applying an `UpdateTask` that sets `parent_id`, run a recursive CTE that walks the ancestor chain of the proposed parent and rejects the update if the task being updated appears in that chain (which also covers a task naming itself as parent). `CreateTask` needs no cycle check — a brand-new task has no descendants, so it cannot become its own ancestor.

**Rationale**: A cycle is created exactly when the task being updated is the proposed parent itself or an ancestor of it. A `WITH RECURSIVE` query starting at the proposed parent and following `parent_id` upward enumerates `{parent} ∪ ancestors(parent)`; testing whether the updated task's `id` is in that set is a single, depth-unbounded check — correct for the "no depth restriction" requirement (FR-015). The data is always acyclic (this very check guarantees it), so the recursion always terminates.

**Alternatives considered**: A `BEFORE UPDATE` trigger — rejected; adds a database object and hidden control flow, heavier than one query. An application-side parent-pointer loop — rejected; equivalent logic but an extra round trip per hop instead of one query. PostgreSQL `ltree` materialized paths — rejected; a whole extension and denormalized path column for what one recursive query handles (Principle I).

## R4 — Optional fields in the proto contract (FR-003)

**Decision**:
- `due` — `google.protobuf.Timestamp`. Message-typed fields have natural presence: a `nil` value means "no due date".
- `parent_id` — `optional int64`. The explicit-presence `optional` keyword distinguishes "no parent" (field absent) from any concrete id value.
- `description` — plain `string`; an empty string represents "no description". Stored as `NOT NULL DEFAULT ''`.

**Rationale**: This mirrors how the request and response messages must behave under the full-replace update model (clarified in the spec): an `UpdateTask` request that omits `due` or `parent_id` clears them. Using message presence for `due` and explicit `optional` presence for `parent_id` lets the handler tell "clear this field" apart from "set it to a value". `description` has no meaningful "cleared vs empty" distinction, so a plain string suffices.

**Alternatives considered**: `FieldMask` for partial updates — rejected; the spec clarified full-replace semantics, so a mask is unused complexity. A sentinel `parent_id = 0` for "no parent" — rejected; conflates a valid-looking value with absence and is fragile.

## R5 — Name validation: placement and rules (FR-002, FR-010)

**Decision**: Validate the name in the handler — trim surrounding whitespace, reject empty/whitespace-only with `InvalidArgument`, reject length > 255 (measured on the trimmed value) with `InvalidArgument`. Keep a `CHECK (btrim(name) <> '')` constraint and `VARCHAR(255)` as database backstops.

**Rationale**: Handler-level validation produces precise, caller-friendly error messages and the correct Connect error code, satisfying FR-012's "clear, distinguishable error". The database constraints are defense in depth: they guarantee the invariant even if a future code path forgets to validate, but they are not the primary error surface (a raw `VARCHAR` overflow or `CHECK` violation is an opaque message).

**Alternatives considered**: Database-only validation — rejected; surfaces opaque SQLSTATE errors, hard to map to a helpful response. Storing the name untrimmed — rejected; the spec's edge cases treat a whitespace-only name as missing, which implies trimming before the empty check.

## R6 — Error mapping to Connect codes (FR-012)

**Decision**: Map failure types to Connect error codes:
- Validation failure (missing/blank name, over-length name, unknown `parent_id`, cycle) → `CodeInvalidArgument`.
- Task not found (Get/Update/Delete on an unknown `id`) → `CodeNotFound`.
- Unexpected database error → `CodeInternal`.

`pgx` returns `pgx.ErrNoRows` from `:one` queries when nothing matches; the handler treats that as `NotFound` for Get/Update/Delete.

**Rationale**: Distinct codes make the two failure classes the spec calls out (validation vs. not-found) programmatically distinguishable, satisfying FR-012. This matches the existing `health.go` handler, which already maps failures to `connect.NewError(connect.CodeInternal, err)`.

**Alternatives considered**: A single generic error code — rejected; callers could not distinguish a bad request from a missing resource.

## R7 — List ordering and shape (FR-006)

**Decision**: `ListTasks` returns a flat `repeated Task` ordered by `id` ascending (`ORDER BY id`). The hierarchy is conveyed through each task's `parent_id`; the response is not nested.

**Rationale**: The spec clarified list order as identifier-ascending and declared sorting/filtering and nested representations out of scope. A flat ordered list is the simplest query and makes acceptance tests deterministic. Clients can reconstruct the tree from `parent_id` values if they need to.

**Alternatives considered**: A nested/tree-shaped response — rejected; out of scope per the spec and more complex to build and test.

## R8 — Testing approach

**Decision**: Table-driven unit tests for handler validation logic (name trimming, length, presence). Integration tests that run the handlers against a real PostgreSQL instance (the one from `compose.yaml`) to cover create/get/list/update/delete, cascade delete of a subtree, and cycle rejection.

**Rationale**: Validation rules are pure and cheap to unit-test. The interesting behavior — cascade delete, cycle detection, identifier non-reuse, persistence — lives in SQL and only a real database exercises it faithfully. The project already runs PostgreSQL via `compose.yaml`, so no new infrastructure is needed.

**Alternatives considered**: Mocking the `db.Queries` layer — rejected; would not exercise the FK cascade or the recursive CTE, which are the parts most worth testing.
