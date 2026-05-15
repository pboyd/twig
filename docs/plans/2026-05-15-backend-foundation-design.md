# Backend Foundation Design

**Date**: 2026-05-15
**Status**: Approved

## Overview

Initialize a Go backend server at `services/todo` with Connect-Go, PostgreSQL (sqlc + golang-migrate), Buf-generated protos, and a Podman-based dev environment. No business logic. The goal is a wired-up foundation with one verifiable end-to-end path: an HTTP request reaches a Connect handler, the handler runs `SELECT 1` against Postgres, and returns `"ok"`.

## Directory Layout

```
services/todo/
├── cmd/server/main.go          # Entry point; wires server + handler
├── internal/
│   └── handler/
│       └── health.go           # HealthService Connect handler
├── internal/db/                # sqlc-generated Go code (output dir)
├── db/
│   ├── migrations/
│   │   ├── 000001_init.up.sql
│   │   └── 000001_init.down.sql
│   └── queries/
│       └── health.sql          # SELECT 1 named Ping
├── proto/
│   └── health/v1/
│       └── health.proto
├── gen/                        # buf-generated Go (committed)
│   └── health/v1/
├── buf.yaml
├── buf.gen.yaml
├── sqlc.yaml
├── Dockerfile
├── compose.yaml
└── Makefile
```

## Proto Contract

Service: `HealthService` in package `health.v1`.

```protobuf
service HealthService {
  rpc Check(CheckRequest) returns (CheckResponse);
}

message CheckRequest {}
message CheckResponse {
  string status = 1; // "ok" on success
}
```

Buf generates Go code into `gen/`. The `make proto` target runs `buf generate`.

## Database Layer

- **Migrations**: golang-migrate. `000001_init.up.sql` is intentionally empty — it establishes the migration framework is wired without creating premature schema.
- **Queries**: `db/queries/health.sql` defines `-- name: Ping :one` running `SELECT 1`. sqlc generates a typed `Ping(ctx)` function.
- **sqlc output**: `internal/db/` package. The handler imports `*db.Queries` directly — no intermediate service layer.

## Server

`cmd/server/main.go`:
- Opens a Postgres connection (`pgx` driver, DSN from `DATABASE_URL` env var).
- Runs pending migrations on startup via golang-migrate.
- Creates `*db.Queries`.
- Registers `HealthServiceHandler` with Connect-Go.
- Listens on `:8080`.

`internal/handler/health.go`:
- Holds `*db.Queries`.
- `Check` calls `queries.Ping(ctx)` and returns `CheckResponse{Status: "ok"}`.

## Container & Dev Environment

**Dockerfile**: Two-stage build.
- Builder: `golang:1.24-alpine` — compiles the binary.
- Final: `gcr.io/distroless/static-debian12` — minimal, no shell. Binary copied from builder. Exposes `8080`.

**compose.yaml**: Two services.
- `postgres`: `postgres:17-alpine`, healthcheck via `pg_isready`. Credentials in env vars.
- `server`: built from `Dockerfile`, depends on `postgres` being healthy. `DATABASE_URL` injected.

**Makefile targets**:

| Target | Action |
|---|---|
| `proto` | `buf generate` |
| `build` | `podman build -t todo-server .` |
| `dev` | `podman-compose up --build` |
| `migrate-up` | Run all pending migrations |
| `migrate-down` | Roll back one migration |

## Constitution Compliance

- **Simplicity/YAGNI**: No service layer, no domain models, no abstractions beyond what Connect-Go and sqlc require. One handler, one query.
- **API-First**: Proto contract is defined and generated before any handler code is written.
