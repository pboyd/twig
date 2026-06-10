# CLAUDE.md

This file provides guidance to Claude Code (claude.ai/code) when working with code in this repository.

Running `twig` with no arguments launches the interactive TUI on a TTY.

## Commands

```bash
# Run CLI tests (from repo root)
go test ./...

# Run server tests
cd services/twig && go test ./...

# Build the CLI binary (from repo root)
go build -o twig ./cmd/twig

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

**Alternate profiles**: The CLI supports multiple accounts in one config file via `[profile.<name>]` TOML tables. Select a profile at launch with `--profile <name>` (flag wins) or `TWIG_PROFILE=<name>` (env var). Omitting the flag/env uses the root/default account. See `specs/033-alternate-profiles/contracts/config-schema.md` for the full profile schema and precedence rules.

## Architecture

Three Go modules:

| Module | Path | Purpose |
|---|---|---|
| `github.com/pboyd/twig` | repo root | CLI/TUI (`cmd/twig`, `internal/{cli,tui,config,pomodoro}`) |
| `github.com/pboyd/twig/api` | `api/` | Shared protobuf types + ConnectRPC stubs (`api/gen/`) |
| `github.com/pboyd/twig/services/twig` | `services/twig/` | HTTP server (`cmd/server`, `internal/{handler,db,auth,plan}`) |

Both the root CLI module and the server module depend on `api/` via local `replace` directives. This lets them share generated code without the CLI pulling in server-only deps (pgx, golang-migrate) or vice-versa.

### Binaries

- **`cmd/server`** — HTTP/2 server (port 8080) with ConnectRPC (gRPC-compatible) handlers and a plain HTTP auth layer (`/auth/login`, `/auth/logout`). Runs migrations on startup.
- **`cmd/twig`** — CLI client that talks to the server via ConnectRPC. Entry point is `internal/cli.Run()`. Commands: `task`, `pom` (Pomodoro timer), `plan` (daily planning).

### Key internal packages

| Package | Module | Role |
|---|---|---|
| `internal/db` | `services/twig` | sqlc-generated query layer (PostgreSQL via pgx/v5). **Do not edit by hand** — regenerate with `sqlc generate`. |
| `internal/handler` | `services/twig` | ConnectRPC service implementations (`Task`, `Plan`, `Health`, `Pomodoro`). Each handler holds a `*db.Queries`. |
| `internal/auth` | `services/twig` | Session/API-key management and HTTP middleware. |
| `internal/cli` | root | All CLI rendering and command dispatch. TTY detection gates ANSI styling. |
| `internal/config` | root | Config file loading (`~/.config/twig/config.toml`), env-var precedence resolution. |
| `internal/plan` | `services/twig` | Daily plan business logic (separate from CLI rendering). |
| `internal/pomodoro` | root | Pomodoro timer logic. |
| `api/gen/` | `api` | Protobuf + ConnectRPC generated code. **Do not edit by hand** — regenerate with `make proto`. |

### Data flow

```
CLI (cmd/twig) → ConnectRPC over HTTP/2 → handler/ → db/ → PostgreSQL
```

Auth sits outside ConnectRPC: `/auth/login` and `/auth/logout` are plain HTTP endpoints; all ConnectRPC routes are wrapped by `auth.Middleware`.

### Adding a new API endpoint

1. Define the message/service in `api/proto/<domain>/v1/`.
2. `make proto` to regenerate `api/gen/`.
3. Implement the service interface in `services/twig/internal/handler/`.
4. Register the handler in `services/twig/cmd/server/main.go`.
5. Add SQL queries in `services/twig/db/queries/`, run `sqlc generate` to update `internal/db/`.

### Testing conventions

- Handler tests and CLI tests use `export_test.go` shims to access unexported helpers.
- No integration test infrastructure — tests do not require a running database.

## Frontend (services/twig-web/)

A React 19 + TypeScript SPA at `services/twig-web/` — sibling to the server module. It consumes the existing `task.v1.TaskService` ConnectRPC endpoints and `/auth/*` HTTP endpoints. **The backend is not modified.**

All frontend commands run from `services/twig-web/`:

```bash
# Install dependencies
npm install

# Generate TypeScript from proto (re-run when api/proto changes)
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
`specs/046-tui-date-picker/plan.md`.
<!-- SPECKIT END -->
