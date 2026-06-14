# Phase 1 Data Model: Self-Serve Account Management

No schema migration is required. The existing `000003_auth` migration already provides every
table and column this feature needs. This document records the entities as they participate
in the feature and the new **sqlc queries** to be added.

## Entities (existing tables)

### User account — `users`

| Column | Type | Notes |
|---|---|---|
| `id` | `BIGINT` identity PK | The authenticated identity carried in `auth.UserID(ctx)`. |
| `username` | `TEXT UNIQUE NOT NULL` | Set at provisioning; **not editable** by this feature. Used as the rate-limit key for password-change abuse protection. |
| `password_hash` | `TEXT NOT NULL` | bcrypt (cost 12). Verified on `ChangePassword`, replaced on success. The only column this feature writes. |

Relationships: owns 0..N `api_keys`, owns 0..N `sessions` (both `ON DELETE CASCADE`).

### API key — `api_keys`

| Column | Type | Notes |
|---|---|---|
| `id` | `BIGINT` identity PK | Stable, non-secret identifier used for listing and revocation. |
| `user_id` | `BIGINT FK → users(id)` | Ownership; every query in this feature filters on it. |
| `key_hash` | `TEXT UNIQUE NOT NULL` | SHA-256 of the raw key. **Never** returned to clients. |
| `label` | `TEXT` (nullable) | Optional, descriptive, non-unique. Default applied when client omits it. |
| `created_at` | `TIMESTAMPTZ NOT NULL` | Shown as non-secret metadata. |

The raw secret key value is **not stored** — only its hash. It exists transiently in the
`CreateApiKey` response and is shown to the user exactly once (FR-009, FR-011, SC-005).

### Session — `sessions`

Read-only for this feature. Listed here only to state the invariant: `ChangePassword` does
**not** create or delete any `sessions` row (FR-006). No query touches this table.

## Validation rules

- **New password length** ≥ 8 characters; no character-composition rules (FR-005). Enforced
  server-side in the handler (authoritative) and mirrored client-side for fast feedback.
- **New password confirmation** must equal the new password (FR-004). Enforced client-side
  before submission; the server takes a single new-password field and does not see the
  confirmation.
- **Current password** must verify against `password_hash` via bcrypt before any write
  (FR-003). Failures are rate-limited (FR-019).
- **Label** is optional; empty/absent → stored default label (FR-010). No uniqueness check.
- **Ownership**: list/create/revoke operate only on rows where `user_id = auth.UserID(ctx)`
  (FR-017). The client never supplies a `user_id`.

## State transitions

- **API key**: `created` → `revoked(deleted)`. One-way, immediate, permanent (hard delete).
  Revoking an already-deleted key is an idempotent no-op (success).
- **Password**: `old hash` → `new hash` on a verified change. No effect on session or key
  state.

## New / reused sqlc queries (`services/twig/db/queries/auth.sql`)

> After editing the `.sql` file, run `sqlc generate` (from `services/twig/`) to regenerate
> `internal/db`. **Do not hand-edit generated code.**

| Query | Kind | Purpose | Status |
|---|---|---|---|
| `GetUserByID` | `:one` | `SELECT * FROM users WHERE id = $1` — fetch `password_hash` + `username` for the authenticated user. | **NEW** |
| `UpdateUserPasswordByID` | `:exec` | `UPDATE users SET password_hash = $2 WHERE id = $1` — write the new hash keyed by the trusted id. | **NEW** |
| `ListApiKeysByUser` | `:many` | `SELECT id, label, created_at FROM api_keys WHERE user_id = $1 ORDER BY created_at, id` — non-secret metadata only (no `key_hash`). | **NEW** |
| `CreateApiKey` | `:one` | Insert a new hashed key for the user. | **EXISTS** (reused) |
| `DeleteApiKeyForUser` | `:execrows` | `DELETE FROM api_keys WHERE id = $1 AND user_id = $2` — ownership-scoped, returns affected-row count for idempotency. | **NEW** |

Note: `ListApiKeysByUser` deliberately projects only non-secret columns so `key_hash` cannot
leak through the handler even by mistake (defense in depth for FR-009 / SC-005).
