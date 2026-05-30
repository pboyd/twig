# Research: Rename App to Twig

## Scope of Changes

All user-facing name references are catalogued below. Internal names (Go module path, directory names, database name) are out of scope per the spec.

### Decision: User-facing touch points only
- **Rationale**: The spec explicitly excludes the Go module path (`github.com/pboyd/todo/...`), service directories (`services/todo/`, `services/todo-web/`), and database names. Only names a user sees or types change.
- **Alternatives considered**: A full directory rename would require updating every Go import path — significant mechanical work with no user-visible benefit.

---

## Catalogued Touch Points

### 1. Environment Variable Names
**File**: `services/todo/internal/config/config.go`
- `os.Getenv("TODO_ADDR")` → `os.Getenv("TWIG_ADDR")`
- `os.Getenv("TODO_API_KEY")` → `os.Getenv("TWIG_API_KEY")`

### 2. Config File Path
**File**: `services/todo/internal/config/config.go`
- `dir + "/todo/config.toml"` → `dir + "/twig/config.toml"`

### 3. CLI Usage Strings and Error Messages
**File**: `services/todo/internal/cli/cli.go`
- `"Run 'todo help' for usage."` × 2
- `"Usage: todo <command> [arguments]"`
- `"todo help [command]"`
- `"Run 'todo help <command>' for command-specific help."`
- `"Config file: ~/.config/todo/config.toml ..."` 
- `"Run 'todo help task' for usage."`
- `"Usage: todo task [<subcommand>] [arguments]"`
- `"todo task [--completed | --all]"` (and other task examples)
- `"set TODO_API_KEY env var or api_key in %s"` (error message)

**File**: `services/todo/internal/cli/task.go`
- `"usage: todo task complete <id>"`
- `"usage: todo task add [--parent <id>] [--due <timestamp>] <name>"`
- `"usage: todo task rm <id>"`
- `"usage: todo task mod <id> [<name>] [--parent <id>] [--due <timestamp>]"` × 2

**File**: `services/todo/internal/cli/pom.go`
- `"set TODO_API_KEY env var or api_key in %s"` (error message)
- `"Run 'todo help pom' for usage."`
- `"Usage: todo pom <subcommand> [arguments]"`

**File**: `services/todo/internal/cli/plan.go`
- `"set TODO_API_KEY env var or api_key in %s"` (error message)
- `"Run 'todo help plan' for usage."`
- `"Usage: todo plan [--date YYYY-MM-DD] [subcommand]"`
- `"Usage: todo plan task <task_id> <start> [duration|end]"`
- `"Usage: todo plan event <name> <start> [duration|end]"`
- `"Usage: todo plan rm <n>"`
- `"Usage: todo plan rename <n> <name>"`
- `"Usage: todo plan mv <n> <start> [duration|end]"`

### 4. TUI Error Message
**File**: `services/todo/internal/tui/tui.go`
- `"set TODO_API_KEY env var or api_key in %s"` (error message)

### 5. Build Targets
**File**: `Makefile`
- `go build -o todo ./cmd/todo` → `go build -o twig ./cmd/todo`
- `podman build -t todo-server services/todo` → `podman build -t twig-server services/todo`

### 6. Web App — Page Title
**File**: `services/todo-web/index.html`
- `<title>Todo</title>` → `<title>Twig</title>`

### 7. Web App — App Header
**File**: `services/todo-web/src/components/AppHeader.tsx`
- `Todo` (display text) → `Twig`

### 8. Web App — LocalStorage Key
**File**: `services/todo-web/src/pages/TaskTreePage.tsx`
- `"todo-expanded-tasks"` → `"twig-expanded-tasks"`

### 9. Test Files (env var names)
**File**: `services/todo/internal/cli/cli_test.go`
- `t.Setenv("TODO_ADDR", ...)` × 2 → `TWIG_ADDR`
- `t.Setenv("TODO_API_KEY", ...)` × 2 → `TWIG_API_KEY`
- String assertions referencing `"TODO_API_KEY"` in error output

**File**: `services/todo/internal/config/config_test.go`
- Any `TODO_ADDR` / `TODO_API_KEY` env var references

---

## Principle IV Review (Playful User Messages)

The existing error message pattern `"error: API key not set; set TWIG_API_KEY env var or api_key in %s"` is accurate but dry. Since we are touching these strings anyway, the rename is an opportunity to align with Principle IV. Recommended updated tone:
> `"no API key found — set TWIG_API_KEY or add api_key to %s"`

This softens the delivery while remaining fully actionable.

---

## No Data Model or Contract Changes

This feature makes no changes to the ConnectRPC API, database schema, or shared type definitions. `data-model.md` and `contracts/` are not generated for this feature.
