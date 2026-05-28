# Contract: `todo` Config File Schema

**Location**: `$XDG_CONFIG_HOME/todo/config.toml` (falls back to `~/.config/todo/config.toml`)

**Format**: TOML 1.0

**Status**: Stable for v1. New keys may be added in future versions; unknown keys MUST be ignored by readers for forward compatibility.

## Full schema

```toml
# All keys are OPTIONAL. An absent or empty key means "use the default" or "do nothing."

# --- Server connection ----------------------------------------------------

# Base URL of the todo server.
# Default: "http://localhost:8080"
# Overridden by env var TODO_ADDR when set.
api_url = "http://localhost:8080"

# Bearer token issued by `todo-server --provision-user`.
# No default. If unset here AND TODO_API_KEY env var is unset, the CLI exits
# with an error directing the user to set one of the two sources.
# Overridden by env var TODO_API_KEY when set.
api_key = "abc123..."

# --- Pomodoro lifecycle hooks ---------------------------------------------

[pomodoro]

# Shell command executed AFTER a pomodoro successfully starts.
# Not executed when `pom resume` reattaches to an already-running pomodoro.
# Not executed if the start RPC fails.
on_start = "do-not-disturb on"

# Shell command executed AFTER the user explicitly cancels a running pomodoro.
# Not executed on ungraceful termination (SIGKILL, terminal close).
# Mutually exclusive with on_complete for a given pomodoro instance.
on_cancel = "do-not-disturb off"

# Shell command executed AFTER a pomodoro completes naturally (timer hits 0).
# Mutually exclusive with on_cancel for a given pomodoro instance.
on_complete = "do-not-disturb off && notify-send 'Pomodoro done'"
```

## Execution semantics

- Hook commands are executed via `sh -c <command>`. Standard shell syntax (pipes, redirects, `&&`, env expansion) is supported.
- Hooks run synchronously in the foreground of the CLI/TUI process. Output is inherited (stdout/stderr of the hook appear in the user's terminal).
- A non-zero exit status or "command not found" results in a non-fatal warning on stderr; the pomodoro itself is unaffected.

## Precedence rules

| Setting   | Env var (wins if non-empty) | Config file key       | Built-in default          |
|-----------|-----------------------------|-----------------------|---------------------------|
| API URL   | `TODO_ADDR`                 | `api_url`             | `http://localhost:8080`   |
| API key   | `TODO_API_KEY`              | `api_key`             | *(none — error if unset)* |
| Hooks     | *(no env var)*              | `[pomodoro].on_*`     | empty (no-op)             |

## Error cases (reader contract)

| Condition                                  | Reader behavior                                                                            |
|--------------------------------------------|--------------------------------------------------------------------------------------------|
| File does not exist                        | Treat as empty config. No error.                                                           |
| File exists but is not readable            | Error: `config: failed to read <path>: <os error>`. CLI exits non-zero.                    |
| File exists but is not valid TOML          | Error: `config: failed to parse <path>: <toml error>`. CLI exits non-zero.                 |
| API key missing from both env and config   | Error: `error: API key not set; set TODO_API_KEY env var or api_key in <path>`. Exits non-zero. |
| Unknown top-level or `[pomodoro]` key      | Ignored. No warning. (Forward compatibility.)                                              |

## Compatibility

- **`--exec` flag**: REMOVED. `todo pom start --exec ...` and `todo pom resume --exec ...` now return an unknown-flag error. Migration path: move the command string into `[pomodoro].on_complete` in the config file.
- **Env vars**: `TODO_API_KEY` and `TODO_ADDR` continue to work and take precedence over config values.
