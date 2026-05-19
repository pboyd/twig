# Authentication Design

**Date**: 2026-05-19
**Status**: Approved

## Overview

The todo service authenticates two kinds of clients: browser frontends and CLI tools. Both resolve to a specific user, and each user sees only their own tasks.

Browser clients log in with a username and password; the server issues a session stored in PostgreSQL and identifies subsequent requests via an `HttpOnly` cookie. CLI clients carry a long-lived API key as a `Bearer` token in the `Authorization` header. A single middleware layer checks both mechanisms and injects the resolved `user_id` into the request context. All task handlers remain unaware of which credential type authenticated the request.

User accounts are provisioned at deploy time — no in-app user management is needed. The service targets a LAN deployment over HTTPS with one or two users per instance.

## Data Model

### `users`

| Column          | Type   | Constraints              |
|-----------------|--------|--------------------------|
| `id`            | serial | primary key              |
| `username`      | text   | unique, not null         |
| `password_hash` | text   | not null (bcrypt)        |

### `sessions`

| Column       | Type        | Constraints               |
|--------------|-------------|---------------------------|
| `id`         | text        | primary key (random token)|
| `user_id`    | integer     | FK → users.id, not null   |
| `created_at` | timestamptz | not null                  |
| `expires_at` | timestamptz | not null                  |

Sessions expire after a configurable duration (default 30 days). The `id` is a cryptographically random 32-byte value, hex-encoded, and used directly as the cookie value.

### `api_keys`

| Column       | Type        | Constraints               |
|--------------|-------------|---------------------------|
| `id`         | serial      | primary key               |
| `user_id`    | integer     | FK → users.id, not null   |
| `key_hash`   | text        | not null (SHA-256)        |
| `label`      | text        |                           |
| `created_at` | timestamptz | not null                  |

The raw key is a cryptographically random 32-byte value, hex-encoded. It is printed once at provisioning time and never stored in plaintext. Revoke a key by deleting its row.

### `tasks` (modified)

Add `user_id integer NOT NULL REFERENCES users(id)` to the existing tasks table. Every query that reads or writes tasks gains a `WHERE user_id = $n` filter.

## Endpoints

Two plain HTTP endpoints handle session lifecycle. They are not Connect-Go RPCs.

| Method | Path           | Description                                         |
|--------|----------------|-----------------------------------------------------|
| POST   | `/auth/login`  | Validate credentials, create session, set cookie    |
| POST   | `/auth/logout` | Delete session, clear cookie                        |

`/auth/login` accepts JSON `{"username": "...", "password": "..."}` and responds with `200 OK` on success or `401 Unauthorized` on failure. It sets a `Set-Cookie` header with `HttpOnly; Secure; SameSite=Strict`.

## Auth Middleware

The middleware runs before every Connect-Go handler. It checks credentials in this order:

1. **`Authorization: Bearer <token>`** — hash the token (SHA-256), look up in `api_keys`, resolve to `user_id`.
2. **Session cookie** — look up the cookie value in `sessions`, check `expires_at`, resolve to `user_id`.

On success, the middleware stores the `user_id` in the request context and calls the next handler. On failure, it returns `401 Unauthorized` immediately. The `/auth/login` and `/auth/logout` paths bypass the middleware.

## Provisioning

User accounts and API keys are created at deploy time via a server flag:

```
./server --provision-user=alice:s3cr3t --provision-user=bob:hunter2
```

On startup with that flag, the server:
1. Inserts each user (bcrypt-hashing the password).
2. Generates one API key per user.
3. Prints each raw key to stdout exactly once.
4. Exits.

Operators store the printed keys in their CLI config files. To rotate a key, delete its row and re-run provisioning.

## Security Properties

| Property             | Mechanism                                              |
|----------------------|--------------------------------------------------------|
| Password storage     | bcrypt (cost ≥ 12)                                     |
| Session token        | 32 random bytes; `HttpOnly; Secure; SameSite=Strict`   |
| API key storage      | SHA-256 hash; raw key printed once, never stored       |
| Transit security     | HTTPS (LAN deployment requirement)                     |
| CSRF                 | `SameSite=Strict` cookie attribute                     |
| Per-user isolation   | `user_id` column on tasks, enforced in every query     |

## Out of Scope

- In-app user management UI
- Password reset flow
- Multi-factor authentication
- OAuth / OIDC / external identity providers
- API key rotation UI (delete row and re-provision)
