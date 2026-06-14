# API Contract: `account.v1.AccountService`

Companion to [`account.proto`](./account.proto). Documents per-RPC behavior, error mapping,
and the requirements each RPC satisfies. All RPCs are ConnectRPC, served behind
`auth.Middleware`; an unauthenticated call is rejected by the middleware with HTTP 401
before reaching the handler (FR-001, edge case "Unauthenticated access").

**Acting user**: always `auth.UserID(ctx)`. No request carries an account id; cross-account
access is structurally impossible (FR-017, SC-006).

ConnectRPC error codes below are the canonical `connect.Code` values.

---

## ChangePassword

`ChangePasswordRequest{ current_password, new_password } → ChangePasswordResponse{}`

**Behavior**
1. Resolve the caller via `GetUserByID(auth.UserID(ctx))`.
2. Rate-limit gate: if the `user:<username>` key is blocked, return `ResourceExhausted`
   (FR-019).
3. Verify `current_password` against `password_hash` (bcrypt). On failure: record a limiter
   failure and return `InvalidArgument` "current password is incorrect" (FR-003).
4. Validate `new_password` length ≥ 8; otherwise `InvalidArgument` with strength guidance
   (FR-005).
5. Hash and write via `UpdateUserPasswordByID`. Reset the limiter on success.
6. Do **not** touch sessions (FR-006) or API keys (FR-007). Reusing the current password is
   allowed (edge case "Password reuse").

**Errors**

| Condition | Code |
|---|---|
| Wrong current password | `InvalidArgument` |
| New password < 8 chars | `InvalidArgument` |
| Too many recent failures | `ResourceExhausted` |
| Lookup/update DB failure | `Internal` |

Satisfies: FR-002, FR-003, FR-004 (confirmation enforced client-side), FR-005, FR-006,
FR-007, FR-019. Acceptance: US1 scenarios 1–4.

---

## ListApiKeys

`ListApiKeysRequest{} → ListApiKeysResponse{ keys: [ApiKeyMetadata] }`

**Behavior**: `ListApiKeysByUser(auth.UserID(ctx))`, ordered by `created_at, id`. Returns
only `id`, `label`, `created_at`. The secret value and `key_hash` are never projected
(FR-008, FR-009, SC-005). An account with zero keys returns an empty list (valid state after
revoking the last key).

**Errors**: `Internal` on DB failure.

Satisfies: FR-008, FR-009. Acceptance: US3 scenario 1; US2 scenario 4 (returning to the list
shows only metadata).

---

## CreateApiKey

`CreateApiKeyRequest{ label? } → CreateApiKeyResponse{ key: ApiKeyMetadata, secret }`

**Behavior**
1. `GenerateToken()` → raw secret; `HashAPIKey(raw)` → stored hash.
2. Resolve label: use the provided label if non-empty, else the default label (FR-010).
3. `CreateApiKey` insert with `created_at = now`.
4. Return the new key's metadata plus the raw `secret` — the **only** time the secret is
   ever returned (FR-011, FR-009).
5. The key authenticates immediately via the existing `GetApiKeyByHash` path (FR-012).

**Errors**: `Internal` on token-generation or DB failure.

Satisfies: FR-010, FR-011, FR-012, FR-015 (no count check). Acceptance: US2 scenarios 1–3,
SC-003.

---

## RevokeApiKey

`RevokeApiKeyRequest{ id } → RevokeApiKeyResponse{}`

**Behavior**: `DeleteApiKeyForUser(id, auth.UserID(ctx))`. The `user_id` predicate enforces
ownership — an id the caller doesn't own matches no row. Zero rows affected is treated as
**success** (idempotent), covering concurrent/double revocation (FR-014, edge cases
"Concurrent revocation", "Last key revoked"). Deletion is immediate and permanent; the
revoked key stops authenticating on its next use while other keys keep working.

The "last key" and per-key confirmations (and the warning that revoking the in-use key cuts
CLI access) are enforced **client-side** before the call (FR-013, FR-016); the server applies
the deletion unconditionally for the owner.

**Errors**: `Internal` on DB failure. (Not-found / not-owned is **not** an error — no-op
success.)

Satisfies: FR-013 (confirmation client-side), FR-014, FR-016, FR-017. Acceptance: US3
scenarios 2–4, SC-004.

---

## Cross-cutting

- **Authorization** (FR-017, FR-018, SC-006): every handler ignores any notion of a target
  account and uses `auth.UserID(ctx)`. There is no create-account RPC; provisioning stays
  server-side only.
- **Secret exposure** (FR-009, SC-005): `secret` appears only in `CreateApiKeyResponse`;
  no other message or query can surface it.
- **Transport**: the web client reaches these RPCs same-origin; add `/account.v1` to the
  Vite dev proxy so the `SameSite=Strict` session cookie is sent.
