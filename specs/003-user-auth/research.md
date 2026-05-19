# Phase 0 Research: User Authentication

All decisions below are resolved. No `NEEDS CLARIFICATION` items remain. The
approved design `docs/plans/2026-05-19-auth-design.md` is the source of truth;
this file records the implementation-level choices it left open.

## Decision 1 — Auth enforcement: `net/http` middleware, not a Connect interceptor

**Decision**: Implement authentication as an ordinary `http.Handler` middleware
that wraps the `http.ServeMux` carrying the Connect task handlers. The
middleware reads the `Authorization` header and the session `Cookie` directly
from `*http.Request`, resolves a `user_id`, stores it in the request context,
and calls the next handler — or writes `401` and stops.

**Rationale**: The design calls for "a single middleware layer" that checks both
credential types. Session credentials arrive as an HTTP cookie; reading cookies
is natural with `*http.Request.Cookie` in a standard middleware. The two
`/auth/*` endpoints are plain HTTP, not RPCs, and must bypass the same layer —
trivial to express with path matching in an `http.Handler`. A Connect
interceptor would only see RPC traffic, would not cover the plain endpoints, and
would add a Connect-specific abstraction for no gain.

**Alternatives considered**:
- *Connect interceptor* — rejected: does not wrap the non-RPC `/auth/*`
  endpoints, and interceptor context plumbing is more indirection than a plain
  `context.WithValue` in middleware.
- *Per-handler checks* — rejected: violates FR-014 (uniform enforcement) and
  Principle I (repetition across five RPCs).

## Decision 2 — `user_id` propagation via request context

**Decision**: The middleware stores the resolved `user_id` with
`context.WithValue` under an unexported package key in `internal/auth`. Task
handlers call an exported `auth.UserID(ctx)` accessor. The Connect handler
methods already receive `ctx`, so no signature changes are needed.

**Rationale**: This is the standard Go request-scoped-value pattern and keeps
the task handlers unaware of *how* the user was authenticated (design goal:
"task handlers remain unaware of which credential type authenticated").

**Alternatives considered**:
- *Passing `user_id` as an RPC field* — rejected: would change `task.proto`,
  let a client spoof another user, and break the design's isolation model.

## Decision 3 — Password hashing: `golang.org/x/crypto/bcrypt`

**Decision**: Add `golang.org/x/crypto/bcrypt` as the one new dependency. Hash
passwords with cost factor 12. Store only the bcrypt string in
`users.password_hash`.

**Rationale**: FR-008 requires non-reversible password storage; the design
specifies bcrypt cost ≥ 12. `x/crypto/bcrypt` is the de-facto standard Go
implementation, is maintained by the Go team, and pulls in no transitive
third-party dependencies. Rolling a hash by hand would violate "reuse over
invention." This is a required dependency, not a speculative one, so it raises
no Constitution Principle I concern.

**Alternatives considered**:
- *`argon2` (`x/crypto/argon2`)* — rejected: also fine cryptographically, but
  the design explicitly specifies bcrypt; matching the approved design avoids
  needless divergence.
- *Standard library only* — rejected: the stdlib has no password-hashing KDF.

## Decision 4 — API keys and session tokens: stdlib crypto

**Decision**: Generate both API keys and session IDs as 32 bytes from
`crypto/rand`, hex-encoded (64 hex chars). Store API keys as their `crypto/sha256`
hash in `api_keys.key_hash`; store the session ID **as-is** in `sessions.id`
(it is itself the random token and the cookie value). Compare API-key hashes
with `crypto/subtle.ConstantTimeCompare`.

**Rationale**: Matches the design's "32 random bytes, hex-encoded" for both
credentials. A session ID is a random opaque token used directly as the cookie
value, so it is stored verbatim and looked up by primary key — the design's
`sessions.id` *is* the token. An API key is presented on every CLI request, so
it is stored only as a SHA-256 hash (fast, sufficient for a high-entropy random
secret — bcrypt's slowness is unnecessary and undesirable for per-request
lookups). Constant-time comparison avoids a timing side channel on key lookup.

**Alternatives considered**:
- *bcrypt for API keys* — rejected: bcrypt on every request adds latency for no
  security benefit when the key is already 256 bits of randomness.
- *Hashing the session ID* — rejected: it is already a random token; hashing
  adds a step without changing the threat model for a primary-key lookup.

## Decision 5 — `tasks.user_id` column and ID typing

**Decision**: Migration `000003_auth` creates `users`, `sessions`, `api_keys`,
then adds `user_id BIGINT NOT NULL REFERENCES users(id)` to `tasks` plus a
supporting index. `users.id` and `api_keys.id` use
`BIGINT GENERATED ALWAYS AS IDENTITY`, matching the existing `tasks` table
convention rather than the design's illustrative `serial`.

**Rationale**: The project's existing `tasks` table uses
`BIGINT GENERATED ALWAYS AS IDENTITY`, and `sqlc` maps that to Go `int64`.
Using the same type for `users.id` keeps `Task.UserID`, `User.ID`, and the
generated query parameters all `int64`, avoiding `int32`/`int64` conversions.
The todo service has no production deployment yet (features 001–003 are being
built in sequence), so the `tasks` table carries no pre-auth rows that would
need a backfill; the column can be added `NOT NULL` directly. The `000003_auth`
down migration drops the column and the three tables.

**Assumption**: No environment running this migration holds tasks created
before auth exists. If that ever changes, the migration would need a nullable
column, a backfill to an owner, then a `SET NOT NULL` — out of scope here.

**Alternatives considered**:
- *`serial` / `integer` for `users.id`* — rejected: introduces an `int32` type
  into otherwise-`int64` code for no benefit.
- *Nullable `user_id` with backfill* — rejected: unnecessary with no legacy
  rows; a nullable owner column would also weaken the FR-012 guarantee.

## Decision 6 — Per-user task scoping in queries

**Decision**: Every query in `db/queries/task.sql` gains a `user_id` parameter
and a `WHERE user_id = $n` (or `AND user_id = $n`) clause. `CreateTask` inserts
`user_id`. `GetTask`, `UpdateTask`, `DeleteTask`, and `TaskExists` match on both
`id` and `user_id`, so a task owned by another user is indistinguishable from a
missing task (returns `pgx.ErrNoRows` → Connect `NotFound`). The
`parentChainContains` cycle check and parent-existence check operate within the
caller's `user_id`.

**Rationale**: Enforcing isolation in the SQL layer (FR-013) means no task
handler can accidentally leak across users, and a direct-ID probe for another
user's task behaves as "not found" — exactly the spec edge case.

**Alternatives considered**:
- *Filtering in Go after a broad `SELECT`* — rejected: easy to forget on one
  code path, and fetches rows the caller may not see.

## Decision 7 — CLI credential delivery: `TODO_API_KEY` environment variable

**Decision**: The CLI reads its API key from the `TODO_API_KEY` environment
variable and attaches it as `Authorization: Bearer <key>` on every request via
a small Connect interceptor (or an `http.RoundTripper` wrapper) built in
`cli.go`. If `TODO_API_KEY` is unset, task commands fail with a clear message.

**Rationale**: The CLI already resolves its backend address from the `TODO_ADDR`
environment variable; reading the key the same way is consistent and is the
simplest mechanism (Principle I). Operators "store the printed keys in their CLI
config files" (design) by exporting `TODO_API_KEY` from their shell profile — no
bespoke config-file parser is needed.

**Alternatives considered**:
- *Dedicated config file (e.g. `~/.todo/config`)* — rejected for now: adds a
  file format and parser the feature does not need; an env var satisfies the
  requirement.
- *`--api-key` flag* — rejected: would expose the secret in shell history and
  process listings.

## Decision 8 — `/auth/login` and `/auth/logout` wiring and middleware bypass

**Decision**: Register `/auth/login` and `/auth/logout` as plain
`http.HandlerFunc`s on the same `http.ServeMux` as the Connect handlers. The
auth middleware checks the request path first: requests to `/auth/login` and
`/auth/logout` skip credential checks and pass straight through; all other
paths require a valid credential.

**Rationale**: Matches the design ("`/auth/login` and `/auth/logout` bypass the
middleware"). Login must be reachable with no credential (FR-017); logout
deletes whatever session the cookie names. JSON request/response bodies use
`encoding/json`. `401 Unauthorized` is returned for bad credentials with a
generic message so the response does not reveal whether the username or the
password was wrong (FR-003).

**Alternatives considered**:
- *Separate mux / port for `/auth`* — rejected: more wiring than a path check;
  the design keeps everything on one server.

## Decision 9 — Provisioning via repeatable `--provision-user` flag

**Decision**: `cmd/server` accepts a repeatable `--provision-user=name:password`
flag (a custom `flag.Value` collecting a slice). When any are present, the
server runs migrations, then for each entry creates-or-updates the user
(bcrypt-hashing the password) and issues a fresh API key, prints each raw key
once to stdout, and exits without starting the listener. An existing username is
updated (password reset, new key issued) per the clarification, not duplicated.

**Rationale**: Matches the design's provisioning flow and FR-015/FR-019.
Re-running provisioning is therefore idempotent in effect and safe.

**Alternatives considered**:
- *Separate `provision` subcommand binary* — rejected: a flag on the existing
  server binary is what the design specifies and needs no new `cmd/` entry.

## Decision 10 — Authentication event logging

**Decision**: Log sign-in success, sign-in failure, sign-out, and rejected
credentials with the standard library `log` package (the same logger the server
already uses). Log the username for login attempts and the `user_id` once
resolved; never log passwords, raw API keys, or session tokens.

**Rationale**: Satisfies FR-018 and the clarification with no new dependency
and no log-retention machinery (a full audit trail was explicitly out of scope).

**Alternatives considered**:
- *Structured logging library* — rejected: the codebase uses stdlib `log`;
  introducing `slog` or a third-party logger is unrelated scope.
