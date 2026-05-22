# CLAUDE.md

This file provides guidance to Claude Code (claude.ai/code) when working with code in this repository.

## Commands

All Go commands run from `services/todo/`:

```bash
# Run all tests
cd services/todo && go test ./...

# Run tests for a specific package
cd services/todo && go test ./internal/cli/...

# Build the CLI binary
cd services/todo && go build -o todo ./cmd/todo

# Build and run the server (via podman-compose)
make dev

# Regenerate protobuf + ConnectRPC stubs (requires buf CLI)
make proto

# Regenerate sqlc DB query code (requires sqlc CLI)
cd services/todo && sqlc generate

# Run DB migrations manually
make migrate-up   # requires DATABASE_URL env var
make migrate-down
```

## Dev Environment

The full stack (PostgreSQL + server) runs via `podman-compose` (not Docker):

```bash
make dev   # builds and (re)starts the server container; postgres starts automatically
```

This is equivalent to `podman-compose up -d --build --force-recreate server`.

**PostgreSQL** is exposed on `localhost:5432`:
- User: `todo`, Password: `todo`, Database: `todo`
- Connection string: `postgres://todo:todo@localhost:5432/todo?sslmode=disable`
- Connect directly: `psql postgres://todo:todo@localhost:5432/todo`

The server container connects to postgres via the internal hostname `postgres` (not `localhost`). The `DATABASE_URL` for the server is `postgres://todo:todo@postgres:5432/todo?sslmode=disable`.

The server auto-runs migrations on startup, so no separate migration step is needed when using `make dev`.

To provision a user (generates an API key):
```bash
# Inside the running server container, or against a local binary with DATABASE_URL set:
./todo-server --provision-user name:password
```

The CLI requires:
- `TODO_API_KEY` — API key from provisioning
- `TODO_ADDR` — server address (defaults to `http://localhost:8080`)

## Architecture

Single Go module at `services/todo/` with two binaries:

- **`cmd/server`** — HTTP/2 server (port 8080) with ConnectRPC (gRPC-compatible) handlers and a plain HTTP auth layer (`/auth/login`, `/auth/logout`). Runs migrations on startup.
- **`cmd/todo`** — CLI client that talks to the server via ConnectRPC. Entry point is `internal/cli.Run()`. Commands: `task`, `pom` (Pomodoro timer), `plan` (daily planning).

### Key internal packages

| Package | Role |
|---|---|
| `internal/db` | sqlc-generated query layer (PostgreSQL via pgx/v5). **Do not edit by hand** — regenerate with `sqlc generate`. |
| `internal/handler` | ConnectRPC service implementations (`Task`, `Plan`, `Health`, `Pomodoro`). Each handler holds a `*db.Queries`. |
| `internal/auth` | Session/API-key management and HTTP middleware. |
| `internal/cli` | All CLI rendering and command dispatch. TTY detection gates ANSI styling. |
| `internal/plan` | Daily plan business logic (separate from CLI rendering). |
| `internal/pomodoro` | Pomodoro timer logic. |
| `gen/` | Protobuf + ConnectRPC generated code. **Do not edit by hand** — regenerate with `make proto`. |

### Data flow

```
CLI (cmd/todo) → ConnectRPC over HTTP/2 → handler/ → db/ → PostgreSQL
```

Auth sits outside ConnectRPC: `/auth/login` and `/auth/logout` are plain HTTP endpoints; all ConnectRPC routes are wrapped by `auth.Middleware`.

### Adding a new API endpoint

1. Define the message/service in `proto/<domain>/v1/`.
2. `make proto` to regenerate `gen/`.
3. Implement the service interface in `internal/handler/`.
4. Register the handler in `cmd/server/main.go`.
5. Add SQL queries in `db/queries/`, run `sqlc generate` to update `internal/db/`.

### Testing conventions

- Handler tests and CLI tests use `export_test.go` shims to access unexported helpers.
- No integration test infrastructure — tests do not require a running database.

<!-- SPECKIT START -->
For additional context about technologies to be used, project structure,
shell commands, and other important information, read the current plan at
`specs/009-plan-span-markers/plan.md`.
<!-- SPECKIT END -->
