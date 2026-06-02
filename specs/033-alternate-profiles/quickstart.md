# Quickstart: Alternate Profiles

## For users

Add alternate accounts to `~/.config/twig/config.toml`. The root stays your default; each `[profile.<name>]` table is another account:

```toml
# Default (work) account — used when no profile is selected
api_url = "https://work.example.com"
api_key = "work-token"

# Pomodoro hooks live at the root and apply to every profile
[pomodoro]
on_complete = "notify-send 'Pomodoro done'"

# Home account
[profile.home]
api_url = "https://home.example.com"
api_key = "home-token"
```

Use it:

```bash
twig task                  # default (work) account
twig --profile home task   # home account
twig --profile home        # TUI against the home account (on a TTY)

# Or set it once per machine:
export TWIG_PROFILE=home
twig task                  # home account; no flag needed
twig --profile default task # flag overrides the env var → back to work
```

Notes:
- The table separator is a dot: `[profile.home]` (not `[profile:home]`).
- Profiles hold **only** `api_url` / `api_key`. Pomodoro hooks always come from the root.
- A profile that omits `api_url` uses the built-in default (`http://localhost:8080`), not the root's URL.
- `TWIG_ADDR` / `TWIG_API_KEY` still override the credentials of whichever profile you pick.
- A misspelled profile fails fast with a helpful message and contacts no server.

## For developers

Where the pieces live (all under `services/twig/`):

- `internal/config/config.go` — add `Profile` struct, `Profiles map[string]Profile` on `Config`, and `Profile(name) (Config, bool)` selector. `Resolve()` stays as-is and runs *after* selection.
- `cmd/twig/main.go` — extract a leading `--profile`/`--profile=` flag; resolve name as flag > `TWIG_PROFILE` > "". Pass the name into `tui.Run` (no remaining args + TTY) or `cli.Run`.
- `internal/cli/cli.go` — `Run`/`loadConfig` take the profile name; thread to `runTask`/`runPomTop`/`runPlan`. `loadConfig` calls `cfg.Profile(name)`, raises the unknown-profile error (with config path) on `ok=false`, then `Resolve()`.
- `internal/tui/tui.go` — `Run` takes the profile name; `Load` → `Profile(name)` → `Resolve()`.

Run the tests:

```bash
cd services/twig && go test ./...
```

Key test targets:
- `internal/config`: profile selection (default/named/missing), hybrid no-fallback for credentials, pomodoro always-from-root, env-overrides-after-select. (Switch existing `==` comparisons to `reflect.DeepEqual` for the new map field.)
- `internal/cli`: global-flag extraction (`--profile x`, `--profile=x`, missing value), flag>env>default name resolution, unknown-profile error text + exit code.

Manual smoke (no server needed for the error paths):

```bash
go build -o twig ./cmd/twig
./twig --profile nope task          # → warm unknown-profile error, exit 1
./twig --profile                    # → usage error, exit 1
```
