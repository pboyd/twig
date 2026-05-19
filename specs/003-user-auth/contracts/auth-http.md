# Contract: Authentication HTTP Surface

This is the API contract for feature `003-user-auth`. It governs the two plain
HTTP endpoints and the credential transport scheme. It is committed before
implementation per Constitution Principle II.

The `TaskService` RPC contract (`task.proto`, committed under
`specs/001-task-crud-api/contracts/`) is **unchanged** by this feature: per-user
scoping is enforced server-side and is invisible on the wire.

## 1. Credential transport

A request to any task RPC MUST carry exactly one of the following. The
middleware checks them in this order and uses the first present:

1. **API key** — `Authorization: Bearer <raw-key>` header. `<raw-key>` is the
   64-character hex key issued at provisioning.
2. **Session cookie** — `Cookie: todo_session=<session-id>`. `<session-id>` is
   the 64-character hex value issued by `POST /auth/login`.

Precedence (FR-016): if both are present, the `Authorization` header wins; the
cookie is not consulted. If neither is present, or the presented credential is
invalid/expired/revoked, the request is rejected (see §4).

The paths `/auth/login` and `/auth/logout` are exempt from this check and
require no credential (FR-017).

## 2. `POST /auth/login`

Validates a username/password and starts a session.

### Request

```
POST /auth/login HTTP/1.1
Content-Type: application/json

{ "username": "alice", "password": "s3cr3t" }
```

| Field | Type | Required | Notes |
|-------|------|----------|-------|
| `username` | string | yes | Exact match against `users.username`. |
| `password` | string | yes | Verified against the stored bcrypt hash. |

### Response — success

```
HTTP/1.1 200 OK
Set-Cookie: todo_session=<64-hex>; Path=/; HttpOnly; Secure; SameSite=Strict; Max-Age=2592000
```

- Status `200 OK`, empty body.
- The cookie value is a freshly created session ID.
- `Max-Age` equals the configured session lifetime in seconds (default
  2592000 = 30 days). The `sessions.expires_at` row value matches.
- Attributes `HttpOnly; Secure; SameSite=Strict` are mandatory (FR-006, FR-007).

### Response — failure

```
HTTP/1.1 401 Unauthorized
Content-Type: application/json

{ "error": "invalid username or password" }
```

- Returned for an unknown username **or** a wrong password — the message MUST
  NOT distinguish the two (FR-003).
- No `Set-Cookie` header is sent.

### Other responses

| Status | When |
|--------|------|
| `400 Bad Request` | Body is not valid JSON, or `username`/`password` missing. |
| `405 Method Not Allowed` | Method is not `POST`. |

## 3. `POST /auth/logout`

Ends the session named by the request cookie.

### Request

```
POST /auth/logout HTTP/1.1
Cookie: todo_session=<64-hex>
```

### Response

```
HTTP/1.1 200 OK
Set-Cookie: todo_session=; Path=/; HttpOnly; Secure; SameSite=Strict; Max-Age=0
```

- Status `200 OK`, empty body.
- The session row is deleted; the `Max-Age=0` cookie clears the browser cookie.
- Idempotent: a missing, unknown, or already-expired cookie still returns
  `200 OK` with the clearing cookie (no error — there is nothing to protect).
- `405 Method Not Allowed` if the method is not `POST`.

## 4. Middleware rejection

For any non-`/auth/*` path with a missing or invalid credential, the middleware
responds and does **not** call the task handler:

```
HTTP/1.1 401 Unauthorized
```

This applies uniformly regardless of credential type (FR-014). "Invalid" covers:
unknown API key hash, revoked (deleted) API key, unknown session ID, and a
session whose `expires_at` is in the past.

On success the middleware resolves a `user_id`, places it in the request
context, and the task RPC proceeds — operating only on that user's tasks
(FR-013).

## 5. Authentication event logging

The server logs, via the standard logger (FR-018): login success (with
username and resolved `user_id`), login failure (with attempted username),
logout, and every middleware rejection. Passwords, raw API keys, and session
IDs are never logged.

## 6. Out of scope for this contract

- No rate limiting or lockout on `/auth/login` (spec clarification).
- No account-management, password-reset, or key-rotation endpoints.
- The browser frontend that consumes `/auth/login` is not part of this repo;
  the endpoints are verified with HTTP-level tests.
