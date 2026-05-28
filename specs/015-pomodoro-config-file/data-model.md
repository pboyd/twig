# Data Model: Pomodoro Config File

This feature introduces no database state. The only "data" is an in-memory `Config` struct loaded once at CLI startup from a TOML file on disk.

## `Config` (Go struct, package `internal/config`)

```go
type Config struct {
    APIURL   string         // resolved API server URL ("" => use default)
    APIKey   string         // resolved API auth token ("" => no key available)
    Pomodoro PomodoroConfig // pomodoro lifecycle hooks
}

type PomodoroConfig struct {
    OnStart    string // shell command run after a pomodoro starts (empty => no-op)
    OnCancel   string // shell command run after a pomodoro is canceled (empty => no-op)
    OnComplete string // shell command run after a pomodoro completes (empty => no-op)
}
```

### TOML representation

```toml
api_url = "http://localhost:8080"
api_key = "..."

[pomodoro]
on_start    = "do-not-disturb on"
on_cancel   = "do-not-disturb off"
on_complete = "do-not-disturb off && notify-send 'Pomodoro done'"
```

All keys are optional. An absent key resolves to the empty string.

## Precedence rules

For each of `APIURL` and `APIKey`, resolution order is:

1. Corresponding environment variable, if non-empty (`TODO_ADDR` → `APIURL`; `TODO_API_KEY` → `APIKey`).
2. Config-file value, if non-empty.
3. For `APIURL` only: built-in default `http://localhost:8080`.
4. For `APIKey`: if still empty, treat as a hard error at the call site (CLI exits with the same `error: TODO_API_KEY is not set` message, updated to mention the config file too — see FR-010).

Hook fields have no env-var equivalent and no default — empty means "do nothing."

## Validation rules

- The file must parse as valid TOML. Parse errors propagate as an actionable error from `Load()` mentioning the file path and the underlying TOML error (FR-011).
- Unknown top-level keys are *ignored* for forward compatibility (so a future v2 key doesn't break v1 binaries). Unknown keys inside `[pomodoro]` are likewise ignored.
- A missing file is not an error: `Load()` returns a zero-value `Config{}` and `os.IsNotExist(err) == true` is treated as a clean "no file."

## Lifecycle

```text
CLI/TUI process start
        │
        ▼
config.Load(path)  ─── file missing? ───► return Config{}
        │ file present, parses OK
        ▼
Config{...}         ◄── env-var overrides applied here
        │
        ▼
Threaded into runTask / runPom / runPlan / TUI client builder
        │
        ▼
runCountdownAndComplete fires hooks via cfg.Pomodoro.OnXxx
        │
        ▼
Process exit (no persistence)
```

## State transitions (pomodoro hook firing)

| From state    | Event                       | Hook fired             |
|---------------|-----------------------------|------------------------|
| (none)        | `pom start` → RPC OK        | `Pomodoro.OnStart`     |
| (none)        | `pom start` → RPC error     | *(none)*               |
| running       | `pom resume` reattach       | *(none — already started)* |
| running       | timer reaches 0 → complete  | `Pomodoro.OnComplete`  |
| running       | user `pom cancel`           | `Pomodoro.OnCancel`    |
| running       | SIGKILL / terminal closed   | *(none — ungraceful)*  |

`resume` does not fire `OnStart` because the pomodoro already started in a previous invocation; firing it again would double-toggle DND.
