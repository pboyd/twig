# CLAUDE.md

This file provides guidance to Claude Code (claude.ai/code) when working with code in this repository.

Running `twig` with no arguments launches the interactive TUI on a TTY.

## Commands

All Go commands run from `services/twig/`:

```bash
# Run all tests
cd services/twig && go test ./...

# Run tests for a specific package
cd services/twig && go test ./internal/cli/...

# Build the CLI binary
cd services/twig && go build -o twig ./cmd/twig

# Build and run the server (via podman-compose)
make dev

# Regenerate protobuf + ConnectRPC stubs (requires buf CLI)
make proto

# Regenerate sqlc DB query code (requires sqlc CLI)
cd services/twig && sqlc generate

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
- User: `twig`, Password: `twig`, Database: `twig`
- Connection string: `postgres://twig:twig@localhost:5432/twig?sslmode=disable`
- Connect directly: `psql postgres://twig:twig@localhost:5432/twig`

The server container connects to postgres via the internal hostname `postgres` (not `localhost`). The `DATABASE_URL` for the server is `postgres://twig:twig@postgres:5432/twig?sslmode=disable`.

The server auto-runs migrations on startup, so no separate migration step is needed when using `make dev`.

To provision a user (generates an API key):
```bash
# Inside the running server container, or against a local binary with DATABASE_URL set:
./twig-server --provision-user name:password
```

The CLI requires:
- `TWIG_API_KEY` — API key from provisioning
- `TWIG_ADDR` — server address (defaults to `http://localhost:8080`)

Both can also be set in the optional config file at `~/.config/twig/config.toml`; env vars take precedence. See `specs/015-pomodoro-config-file/contracts/config-schema.md` for the full schema including pomodoro lifecycle hooks (`on_start`, `on_cancel`, `on_complete`).

## Architecture

Single Go module at `services/twig/` with two binaries:

- **`cmd/server`** — HTTP/2 server (port 8080) with ConnectRPC (gRPC-compatible) handlers and a plain HTTP auth layer (`/auth/login`, `/auth/logout`). Runs migrations on startup.
- **`cmd/twig`** — CLI client that talks to the server via ConnectRPC. Entry point is `internal/cli.Run()`. Commands: `task`, `pom` (Pomodoro timer), `plan` (daily planning).

### Key internal packages

| Package | Role |
|---|---|
| `internal/db` | sqlc-generated query layer (PostgreSQL via pgx/v5). **Do not edit by hand** — regenerate with `sqlc generate`. |
| `internal/handler` | ConnectRPC service implementations (`Task`, `Plan`, `Health`, `Pomodoro`). Each handler holds a `*db.Queries`. |
| `internal/auth` | Session/API-key management and HTTP middleware. |
| `internal/cli` | All CLI rendering and command dispatch. TTY detection gates ANSI styling. |
| `internal/config` | Config file loading (`~/.config/twig/config.toml`), env-var precedence resolution. |
| `internal/plan` | Daily plan business logic (separate from CLI rendering). |
| `internal/pomodoro` | Pomodoro timer logic. |
| `gen/` | Protobuf + ConnectRPC generated code. **Do not edit by hand** — regenerate with `make proto`. |

### Data flow

```
CLI (cmd/twig) → ConnectRPC over HTTP/2 → handler/ → db/ → PostgreSQL
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

## Frontend (services/twig-web/)

A React 19 + TypeScript SPA at `services/twig-web/` — sibling to the Go module. It consumes the existing `task.v1.TaskService` ConnectRPC endpoints and `/auth/*` HTTP endpoints. **The backend is not modified.**

All frontend commands run from `services/twig-web/`:

```bash
# Install dependencies
npm install

# Generate TypeScript from proto (re-run when services/twig/proto changes)
npm run gen

# Run the Vite dev server (proxies /auth, /task.v1, /health.v1 → :8080)
npm run dev        # → http://localhost:5173

# Run unit/interaction tests
npm test

# Production build (outputs to dist/)
npm run build
```

The dev proxy (`vite.config.ts`) keeps the SPA and API on one origin so the `SameSite=Strict` session cookie works. Never call `:8080` cross-origin.

Generated TS lives in `src/gen/` — do not edit by hand; regenerate with `npm run gen`.

Key frontend source paths:
- `src/lib/tree.ts` — flat `ListTasks` → nested tree builder (unit-tested)
- `src/lib/updatePayload.ts` — `UpdateTask` full-replace payload builder (unit-tested)
- `src/theme/messages.ts` — all user-facing copy (warm/playful tone)
- `src/theme/tokens.ts` — color/spacing design tokens
- `src/components/` — Button, Field, Spinner, ErrorBanner, AppHeader, TaskForm, TreeRow, EmptyState
- `src/pages/` — LoginPage, TaskTreePage, TaskDetailPage

<!-- SPECKIT START -->
For additional context about technologies to be used, project structure,
shell commands, and other important information, read the current plan at
`specs/024-rename-to-twig/plan.md`.
<!-- SPECKIT END -->
