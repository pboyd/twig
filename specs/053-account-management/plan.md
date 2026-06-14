# Implementation Plan: Self-Serve Account Management

**Branch**: `053-account-management` | **Date**: 2026-06-13 | **Spec**: [spec.md](./spec.md)

**Input**: Feature specification from `/specs/053-account-management/spec.md`

## Summary

Give signed-in web users self-serve control over their own credentials: change their
password (verifying the current one), and list / create / revoke API keys — all scoped
strictly to the authenticated account. Account *creation* stays an admin-only
`--provision-user` action; this feature only manages an existing account.

Technical approach: introduce a new `account.v1.AccountService` ConnectRPC service
(`ChangePassword`, `ListApiKeys`, `CreateApiKey`, `RevokeApiKey`) wired through the
existing `auth.Middleware`, so every handler derives the acting user from
`auth.UserID(ctx)` and never trusts a client-supplied account identifier. The data model
already supports everything we need (`users.password_hash`, the `api_keys` table); we add
only a handful of user-scoped sqlc queries. The React SPA gains an `/account` route built
from the existing component library, reusing `auth` cookie credentials. No new
credential/storage primitives are introduced — we reuse `bcrypt` password hashing,
SHA-256 API-key hashing, raw-token generation, and the login rate limiter.

## Technical Context

**Language/Version**: Go 1.x (server + CLI), TypeScript / React 19 (web)

**Primary Dependencies**: ConnectRPC (server handlers + connect-query web client), pgx/v5
+ sqlc (DB), `golang.org/x/crypto/bcrypt`, Vite, react-router

**Storage**: PostgreSQL — existing `users` and `api_keys` tables (migration `000003_auth`).
No schema migration required.

**Testing**: `go test ./...` (handler tests via `export_test.go` shims, no live DB),
`npm test` (Vitest + Testing Library) in `services/twig-web/`

**Target Platform**: Linux server (port 8080) + same-origin SPA (Vite dev proxy → :8080)

**Project Type**: Web application (existing Go backend + React frontend monorepo)

**Performance Goals**: Interactive web latency; no throughput targets. Password change must
remain subject to the same login abuse protection (`LoginLimiter`).

**Constraints**: Same-origin SPA with `SameSite=Strict` session cookie — never call :8080
cross-origin. The new `/account.v1` path must be routed to the backend in **both** environments:
the Vite dev proxy (`vite.config.ts`) and the production Caddy reverse proxy
(`deploy/roles/app/templates/Caddyfile.j2`), whose `@backend` regex is an explicit allowlist —
unmatched paths fall through to the SPA. Backend auth model (session cookie for web,
`Authorization: Bearer` API key for CLI) is unchanged. Secret API-key value is
shown exactly once and never persisted in retrievable form.

**Scale/Scope**: Single-user-at-a-time self-service; no key-count limit (FR-015). 4 RPCs,
~4 new sqlc queries, 1 new web route.

## Constitution Check

*GATE: Must pass before Phase 0 research. Re-check after Phase 1 design.*

| Principle | Status | Notes |
|-----------|--------|-------|
| I. Simplicity / YAGNI | ✅ | Reuses existing auth primitives, tables, middleware, and component library. Adds only user-scoped queries + one service. No new abstractions, no soft-delete, no key-count tracking. |
| II. API-First Design | ✅ | `account.v1` proto contract is authored and committed under `contracts/` before any handler or web code. Frontend/backend both generate from it. |
| III. UI/UX Consistency | ✅ | `/account` page is composed from existing `Button`, `Field`, `ErrorBanner`, `Spinner`, `AppHeader`, Toast components and `theme/tokens.ts`. No one-off styles. |
| IV. Playful User Messages | ✅ | All new copy (success toasts, the "shown once" key warning, last-key and revoke confirmations, validation errors) lives in `theme/messages.ts` with a warm-but-measured tone for destructive actions. |

No violations — Complexity Tracking table is intentionally empty.

## Project Structure

### Documentation (this feature)

```text
specs/053-account-management/
├── plan.md              # This file
├── research.md          # Phase 0 output
├── data-model.md        # Phase 1 output
├── quickstart.md        # Phase 1 output
├── contracts/           # Phase 1 output
│   ├── account.proto    # account.v1.AccountService contract (source of truth)
│   └── account-api.md   # Per-RPC request/response + error semantics
└── tasks.md             # Phase 2 output (/speckit-tasks — NOT created here)
```

### Source Code (repository root)

```text
api/
└── proto/account/v1/account.proto      # NEW — copied from contracts/account.proto, then `make proto`
   api/gen/account/v1/...                # generated (do not hand-edit)

services/twig/
├── db/queries/auth.sql                  # EDIT — add GetUserByID, UpdateUserPasswordByID,
│                                         #        ListApiKeysByUser, CreateApiKey (exists),
│                                         #        DeleteApiKeyForUser
│  internal/db/...                        # regenerated by `sqlc generate` (do not hand-edit)
├── internal/handler/account.go          # NEW — AccountService implementation
├── internal/handler/account_test.go     # NEW — handler tests (export_test shims as needed)
├── internal/auth/credential.go          # REUSE — HashPassword/VerifyPassword/HashAPIKey/GenerateToken
├── internal/auth/ratelimit.go           # REUSE — LoginLimiter for ChangePassword
└── cmd/server/main.go                   # EDIT — register AccountService handler; pass limiter

deploy/
└── roles/app/templates/Caddyfile.j2     # EDIT — add account[.]v1[.] to the @backend path_regexp allowlist
                                          #        (prod reverse-proxy routing; unmatched paths fall through to the SPA)

services/twig-web/
├── vite.config.ts                       # EDIT — add "/account.v1" proxy entry (dev-only mirror of the Caddy route)
├── src/gen/account/v1/...               # generated by `npm run gen` (do not hand-edit)
├── src/App.tsx                          # EDIT — add /account route
├── src/components/AppHeader.tsx         # EDIT — add "Account" nav link
├── src/pages/AccountPage.tsx            # NEW — password + API-key management UI
├── src/pages/AccountPage.test.tsx       # NEW — interaction tests
└── src/theme/messages.ts               # EDIT — new user-facing copy
```

**Structure Decision**: Existing three-module Go monorepo (`api/`, root CLI, `services/twig`)
plus the `services/twig-web/` SPA. This feature adds a new proto domain (`account.v1`) and a
new web route, following the exact pattern already used by `task.v1` / `plan.v1` / `goal.v1`
and the `LoginPage`/`TaskTreePage` web pages. The backend auth surface (`/auth/login`,
`/auth/logout`, `auth.Middleware`) is extended only by registering one more ConnectRPC
service behind the existing middleware.

## Complexity Tracking

> No Constitution Check violations — table intentionally empty.

| Violation | Why Needed | Simpler Alternative Rejected Because |
|-----------|------------|-------------------------------------|
| — | — | — |
