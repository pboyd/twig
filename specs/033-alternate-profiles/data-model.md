# Data Model: Alternate Profiles

This feature is client-side configuration only; the "data" is the parsed config file. No database entities, no proto messages.

## Entities

### Config (existing — extended)

The top-level parsed config. The root-level credentials act as the implicit `default` profile.

| Field | TOML key | Type | Notes |
|-------|----------|------|-------|
| APIURL | `api_url` | string | Root/default account API URL. Existing field. |
| APIKey | `api_key` | string | Root/default account API key. Existing field. |
| Pomodoro | `pomodoro` | PomodoroConfig | Existing. **Always** sourced from the root, regardless of selected profile (FR-008a). Not part of any profile. |
| **Profiles** | `profile` | **map[string]Profile** | **NEW.** Named alternate profiles keyed by profile name (`[profile.<name>]`). |

### Profile (NEW)

A named set of account credentials only.

| Field | TOML key | Type | Notes |
|-------|----------|------|-------|
| APIURL | `api_url` | string | Profile's API URL. No fallback to root if empty (falls back only to built-in default URL during Resolve). |
| APIKey | `api_key` | string | Profile's API key. No fallback to root if empty (→ "no API key found" error if also unset by env). |

A `Profile` carries **only** credentials. Non-credential settings placed inside a `[profile.<name>]` table are out of scope and ignored (forward-compatible; consistent with the existing "unknown keys ignored" contract).

### PomodoroConfig (existing — unchanged)

`on_start`, `on_cancel`, `on_complete`. Root-level only.

## Selection & Resolution Logic

Two pure steps on `Config`, ordered Select → Resolve:

### `Profile(name string) (Config, bool)` (NEW)

Returns the effective config for the chosen profile and whether it exists.

- `name == ""` or `name == "default"` → return the root config unchanged; `ok = true` always (the default always exists).
- otherwise → look up `Profiles[name]`:
  - found → return a copy of the config whose `APIURL`/`APIKey` are **replaced** by the profile's values (no merge with root credentials), with `Pomodoro` left as the root's; `ok = true`.
  - not found → `ok = false` (caller raises the unknown-profile error with the config path).

The returned `Config` need not retain `Profiles` (downstream consumers only read credentials + pomodoro).

### `Resolve() Config` (existing — unchanged behavior)

Applied **after** `Profile(...)`:
1. `TWIG_ADDR` overrides `APIURL` when set.
2. `TWIG_API_KEY` overrides `APIKey` when set.
3. `APIURL` defaults to `http://localhost:8080` when still empty.

Because Resolve runs after selection, env vars override the credentials of whichever profile was selected (FR-009), and the default/no-flag path is byte-for-byte identical to today (FR-010/FR-013).

## Profile name resolution (selection input)

Computed once at launch, outside `Config`:

```
profileName = flagValue            if --profile present (non-empty)
            = os.Getenv("TWIG_PROFILE")   else
            = ""                   (→ default) if neither set / env empty
```

- `--profile` present without a value → usage error (FR-012).
- `TWIG_PROFILE=""` → treated as unset → default (edge case in spec).

## Validation Rules

| Rule | Source | Behavior |
|------|--------|----------|
| Selected profile must exist | FR-011 | `Profile()` returns `ok=false`; caller exits non-zero, names profile + config path, contacts no account. |
| `--profile` requires a value | FR-012 | Usage error from the flag extractor in `main.go`. |
| API key must be present after resolution | existing | Existing "no API key found" error; now names the resolved source. |
| Unknown keys in `[profile.<name>]` | existing config contract | Ignored (forward compatibility). |
| Empty config / no `[profile.*]` tables | FR-013 | `Profiles` is nil; default path behaves exactly as today. |

## State Transitions

None. Profile selection is a one-time, launch-time read; no in-session switching (spec assumption).
