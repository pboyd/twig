# Implementation Plan: Rename App to Twig

**Branch**: `024-rename-to-twig` | **Date**: 2026-05-30 | **Spec**: [spec.md](spec.md)

**Input**: Feature specification from `specs/024-rename-to-twig/spec.md`

## Summary

Rename all user-facing occurrences of "todo" to "twig" across the CLI, TUI, and web frontend. Changes affect environment variable names, config file path, CLI usage strings, error messages, build targets, and web branding. No API, database, or module path changes.

## Technical Context

**Language/Version**: Go 1.23 (CLI/server), TypeScript/React 19 (web)

**Primary Dependencies**: ConnectRPC, Bubble Tea (TUI), Vite (web)

**Storage**: N/A (no schema changes)

**Testing**: `go test ./...` (Go), `npm test` (web)

**Target Platform**: Linux CLI + browser

**Project Type**: CLI tool + web SPA

**Performance Goals**: N/A

**Constraints**: No breaking changes to the RPC API or database schema

**Scale/Scope**: ~9 files changed, ~30 string substitutions

## Constitution Check

| Principle | Status | Notes |
|-----------|--------|-------|
| I. Simplicity / YAGNI | ✅ | Pure rename — no new abstractions, no speculative changes |
| II. API-First Design | ✅ | No API changes; RPC contracts unchanged |
| III. UI/UX Consistency | ✅ | New name applied consistently across all surfaces |
| IV. Playful User Messages | ✅ | Error messages updated to warmer tone per research.md |

## Project Structure

### Documentation (this feature)

```text
specs/024-rename-to-twig/
├── plan.md         ← this file
├── research.md     ← Phase 0 output (touch point catalogue)
└── tasks.md        ← Phase 2 output (/speckit-tasks)
```

### Source Files Modified

```text
services/todo/internal/config/config.go       # env vars + config path
services/todo/internal/cli/cli.go             # usage strings + error message
services/todo/internal/cli/task.go            # usage strings
services/todo/internal/cli/pom.go             # usage strings + error message
services/todo/internal/cli/plan.go            # usage strings + error message
services/todo/internal/tui/tui.go             # error message
services/todo/internal/cli/cli_test.go        # env var names in tests
services/todo/internal/config/config_test.go  # env var names in tests
Makefile                                      # binary build targets
services/todo-web/index.html                  # page title
services/todo-web/src/components/AppHeader.tsx  # app header text
services/todo-web/src/pages/TaskTreePage.tsx    # localStorage key
```

**Structure Decision**: Single-feature modifications spread across existing files. No new files or directories are created.

## Implementation Steps

### Step 1 — Config package: env vars and config path

**File**: `services/todo/internal/config/config.go`

| Before | After |
|--------|-------|
| `os.Getenv("TODO_ADDR")` | `os.Getenv("TWIG_ADDR")` |
| `os.Getenv("TODO_API_KEY")` | `os.Getenv("TWIG_API_KEY")` |
| `dir + "/todo/config.toml"` | `dir + "/twig/config.toml"` |
| Comment: `// TODO_ADDR overrides...` | `// TWIG_ADDR overrides...` |

### Step 2 — Config tests: env var names

**File**: `services/todo/internal/config/config_test.go`

Update any `t.Setenv("TODO_ADDR", ...)` and `t.Setenv("TODO_API_KEY", ...)` calls to use the new names. Update any string assertions that check for `"TODO_API_KEY"` in error output.

### Step 3 — CLI root: usage strings and error message

**File**: `services/todo/internal/cli/cli.go`

- Replace all `todo` references in usage strings with `twig`
- Config path hint: `~/.config/todo/config.toml` → `~/.config/twig/config.toml`
- Error message (tone update): `"error: API key not set; set TODO_API_KEY env var or api_key in %s\n"` → `"no API key found — set TWIG_API_KEY or add api_key to %s\n"`

### Step 4 — CLI task: usage strings

**File**: `services/todo/internal/cli/task.go`

Replace all `todo task` usage string prefixes with `twig task`.

### Step 5 — CLI pom: usage strings and error message

**File**: `services/todo/internal/cli/pom.go`

- Replace `todo pom` usage string prefix with `twig pom`
- Error message (tone update, same pattern as Step 3): `TODO_API_KEY` → `TWIG_API_KEY`

### Step 6 — CLI plan: usage strings and error message

**File**: `services/todo/internal/cli/plan.go`

- Replace all `todo plan` usage string prefixes with `twig plan`
- Error message (same tone update): `TODO_API_KEY` → `TWIG_API_KEY`

### Step 7 — TUI: error message

**File**: `services/todo/internal/tui/tui.go`

- Error message (same tone update): `TODO_API_KEY` → `TWIG_API_KEY`

### Step 8 — CLI tests: env var names

**File**: `services/todo/internal/cli/cli_test.go`

- `t.Setenv("TODO_ADDR", ...)` → `t.Setenv("TWIG_ADDR", ...)`
- `t.Setenv("TODO_API_KEY", ...)` → `t.Setenv("TWIG_API_KEY", ...)`
- Any string assertions checking for `"TODO_API_KEY"` in error output → `"TWIG_API_KEY"`

### Step 9 — Makefile: build targets

**File**: `Makefile`

| Before | After |
|--------|-------|
| `go build -o todo ./cmd/todo` | `go build -o twig ./cmd/todo` |
| `podman build -t todo-server services/todo` | `podman build -t twig-server services/todo` |

### Step 10 — Web: page title

**File**: `services/todo-web/index.html`

- `<title>Todo</title>` → `<title>Twig</title>`

### Step 11 — Web: app header

**File**: `services/todo-web/src/components/AppHeader.tsx`

- Display text `Todo` → `Twig`

### Step 12 — Web: localStorage key

**File**: `services/todo-web/src/pages/TaskTreePage.tsx`

- `"todo-expanded-tasks"` → `"twig-expanded-tasks"`

  > Note: Existing browser state under the old key will be silently dropped (no expanded state persisted). This is acceptable — the user just needs to re-expand their tree once.

### Step 13 — Verification

1. `cd services/todo && go test ./...` — all tests pass
2. `cd services/todo && go build -o twig ./cmd/todo` — binary builds
3. `cd services/todo-web && npm test` — all tests pass
4. Run `./twig --help` — output references `twig`, no `todo` product references
5. Check web app: page title and header show "Twig"

## CLAUDE.md Update

Update the binary name reference in `CLAUDE.md`:
- `go build -o todo ./cmd/todo` → `go build -o twig ./cmd/todo`
- Any `TODO_API_KEY` / `TODO_ADDR` references in the dev setup section → `TWIG_API_KEY` / `TWIG_ADDR`
