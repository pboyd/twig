# Phase 0 Research: Self-Serve Account Management

All spec inputs were resolved during the clarification session (2026-06-13); there are no
open `NEEDS CLARIFICATION` markers. The research below records the design decisions taken
to fit the feature onto the existing architecture.

## Decision 1 — Transport: new ConnectRPC service, not plain HTTP

**Decision**: Add `account.v1.AccountService` (a ConnectRPC service) rather than plain HTTP
endpoints like `/auth/login`.

**Rationale**:
- Principle II (API-First) and the established pattern: `task.v1`, `plan.v1`, `goal.v1` are
  all ConnectRPC services whose handlers read the acting user from `auth.UserID(ctx)`.
- `auth.Middleware` exempts only `/auth/*` and authenticates everything else, so a new
  service is authenticated *for free* and the handler cannot be tricked into acting on
  another user — it never receives an account id from the client (satisfies FR-017, SC-006).
- The web app already has a generated connect-query client + transport with cookie
  credentials and a 401 → `/login` interceptor; a new service reuses all of it.

**Alternatives considered**:
- *Plain HTTP under `/account/*` (like `/auth/*`)*: would need bespoke auth wiring since
  `/auth/*` is middleware-exempt, duplicating credential resolution. Rejected — more code,
  diverges from the dominant pattern, weaker type-safety on the web side.
- *Extending the `/auth/*` handlers*: those are deliberately unauthenticated (login/logout);
  account management is the opposite. Rejected.

## Decision 2 — Reuse existing credential primitives

**Decision**: Reuse `auth.HashPassword`/`VerifyPassword` (bcrypt cost 12),
`auth.HashAPIKey` (SHA-256), and `auth.GenerateToken` (32 random bytes, hex). No new crypto.

**Rationale**: `ProvisionUser` already composes exactly these primitives to set a password
and mint a key; `ChangePassword`/`CreateApiKey` are subsets of that flow. Identical hashing
guarantees keys minted on the web authenticate through the same `GetApiKeyByHash` path the
CLI uses (FR-012, SC-003).

**Alternatives considered**: Introducing key prefixes / displayable key IDs derived from the
secret. Rejected as YAGNI — the `api_keys.id` surrogate key already identifies a key for
listing and revocation without touching the secret.

## Decision 3 — Current-password verification needs a user-by-id lookup

**Decision**: Add a `GetUserByID` query. The context carries only `user_id`; verifying the
current password (FR-003) requires the stored `password_hash`, and producing a default key
label or username-keyed rate-limit key may require the username.

**Rationale**: Today only `GetUserByUsername` exists, but the authenticated identity is the
numeric `user_id`. `GetUserByID` is the minimal addition. Password update is then done with
`UpdateUserPasswordByID` (keyed by the trusted id) rather than the existing
username-keyed `UpdateUserPassword`, avoiding any client-influenced key.

**Alternatives considered**: Fetch by id to get the username, then reuse the existing
username-keyed `UpdateUserPassword`. Works, but routes a write through a human-readable key
when we already hold the primary key — `UpdateUserPasswordByID` is clearer and safer.

## Decision 4 — Ownership-scoped, idempotent key revocation

**Decision**: Revoke with a single `DELETE FROM api_keys WHERE id = $1 AND user_id = $2`
(`:execrows`). Zero rows affected is treated as success (idempotent), not an error.

**Rationale**:
- The `user_id` predicate enforces FR-017 in SQL: a user cannot revoke a key they do not
  own (a foreign id simply matches nothing).
- Idempotency covers the concurrent-revocation edge case (already revoked in another tab) —
  the outcome is "the key is gone", reported cleanly rather than as a failure (FR-014 / edge
  case "Concurrent revocation").
- Hard delete matches the spec assumption: revocation is immediate and permanent, no
  soft-delete/restore.

**Alternatives considered**: A `revoked_at` soft-delete column. Rejected per spec assumption
and YAGNI; it would also require filtering every auth lookup.

## Decision 5 — Password change reuses the login rate limiter (FR-019)

**Decision**: The `AccountService` handler holds the same `*auth.LoginLimiter` instance as
`LoginHandler`. `ChangePassword` checks/records failures under the `user:<username>` key
before/after verifying the current password, mirroring login.

**Rationale**: FR-019 mandates the *same* abuse protections as sign-in. Sharing the instance
(constructed once in `main.go`) means a string of wrong current-password attempts counts
against the same per-username window as failed logins. We key by username (resolved via
`GetUserByID`) because the authenticated user is known; IP-keying is unnecessary here since
the request is already authenticated by a valid session/key.

**Alternatives considered**: A separate limiter for password changes. Rejected — FR-019 says
"the same abuse protections", and a separate bucket would let an attacker burn login *and*
password-change budgets independently.

## Decision 6 — Default API-key label (FR-010)

**Decision**: When the create request omits a label, store a sensible default rather than
rejecting. Default to a short, friendly constant (e.g. `"web key"`), consistent with the
provisioning flow's `"provisioned"` label.

**Rationale**: FR-010 + clarification: label is optional with a sensible default. A constant
keeps it predictable; duplicate labels are explicitly allowed (edge case "Duplicate labels"),
so no uniqueness handling is needed.

**Alternatives considered**: Date-stamped default (`"web key 2026-06-13"`). Reasonable, but a
plain constant is simpler and the creation date is already shown as separate metadata.

## Decision 7 — Password change does not touch sessions or keys

**Decision**: `ChangePassword` updates only `password_hash`. It does not delete sessions and
does not touch `api_keys`.

**Rationale**: Clarification chose "keep all sessions valid" (FR-006); FR-007 keeps API keys
intact. This is also the simplest implementation — a single column update — so no extra work
is required to satisfy these requirements.

## Decision 8 — Web composition and copy

**Decision**: One `/account` route renders two sections (password form, API-key list with
create + revoke), built entirely from existing components (`Button`, `Field`, `ErrorBanner`,
`Spinner`, Toast) and `theme/tokens.ts`. All strings live in `theme/messages.ts`. Add a
`/account.v1` entry to the Vite proxy and an "Account" link in `AppHeader`.

**Rationale**: Principles III and IV. The "shown once" key reveal, the last-key warning, and
the revoke confirmation are warm-but-measured destructive-action copy. Routing the new
service through the proxy preserves the same-origin `SameSite=Strict` cookie contract.

**Alternatives considered**: Separate `/account/password` and `/account/keys` routes.
Rejected as over-segmentation for a small surface; a single page with two sections is simpler
and keeps related security settings together.
