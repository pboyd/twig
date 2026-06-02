# Contract: `twig` Config File Schema (profiles addition)

**Location**: `$XDG_CONFIG_HOME/twig/config.toml` (falls back to `~/.config/twig/config.toml`)

**Format**: TOML 1.0

**Status**: Extends the schema in `specs/015-pomodoro-config-file/contracts/config-schema.md`. This document adds alternate profiles. All prior keys and semantics remain unchanged. Unknown keys MUST continue to be ignored for forward compatibility.

## Added schema

```toml
# --- Root / default account (unchanged) -----------------------------------
# The root-level api_url/api_key form the implicit "default" profile.
api_url = "https://work.example.com"
api_key = "work-token"

# --- Pomodoro hooks (unchanged) -------------------------------------------
# Always sourced from the root, regardless of which profile is selected.
[pomodoro]
on_complete = "notify-send 'Pomodoro done'"

# --- Alternate profiles (NEW) ---------------------------------------------
# Each [profile.<name>] table holds an alternate account's credentials.
# A profile carries ONLY credentials (api_url, api_key). Any other keys are ignored.
[profile.home]
api_url = "https://home.example.com"
api_key = "home-token"

[profile.laptop]
api_key = "laptop-token"   # api_url omitted → built-in default URL (NOT the root api_url)
```

> Note: the table separator is a dot (`[profile.home]`), per TOML. A colon form (`[profile:home]`) is not valid TOML and will fail to parse.

## Profile selection

A profile is chosen at launch (see `cli-interface.md`):

- `--profile <name>` flag, or
- `TWIG_PROFILE` environment variable (flag wins when both are set), or
- neither set → the root/default profile.

The name `default` (and the empty value) always refers to the root-level credentials, even if a `[profile.default]` table is also present (the root is authoritative for `default`).

## Resolution semantics (hybrid)

| Setting | Source when a named profile is selected |
|---------|------------------------------------------|
| `api_url` | The profile's `api_url`; if the profile omits it, the **built-in default** (`http://localhost:8080`) — **never** the root `api_url`. Then `TWIG_ADDR` overrides if set. |
| `api_key` | The profile's `api_key`; if the profile omits it, **none** (→ "no API key found" error) — **never** the root `api_key`. Then `TWIG_API_KEY` overrides if set. |
| `[pomodoro].*` | Always the root values. Profiles cannot override pomodoro hooks. |

For the default profile (no selection), behavior is identical to the pre-feature schema: root credentials, with `TWIG_ADDR`/`TWIG_API_KEY` overriding.

## Precedence summary

**Credential value** (per selected profile):

| Setting | Env var (wins if non-empty) | Selected profile's key | Built-in default |
|---------|-----------------------------|------------------------|------------------|
| API URL | `TWIG_ADDR` | profile `api_url` (root for default) | `http://localhost:8080` |
| API key | `TWIG_API_KEY` | profile `api_key` (root for default) | *(none — error if unset)* |

**Profile selection** (independent axis):

| Wins | Source |
|------|--------|
| 1 | `--profile <name>` flag |
| 2 | `TWIG_PROFILE` env var |
| 3 | root/default profile |

## Error cases (reader contract)

| Condition | Reader behavior |
|-----------|-----------------|
| File missing / empty | Treat as empty config. No error. No profiles. |
| File not valid TOML | `config: failed to parse <path>: <toml error>`. Exits non-zero. |
| Selected profile name not present | Names the missing profile + config path; exits non-zero; contacts no account. (See `cli-interface.md` for wording.) |
| API key missing from profile + env | `no API key found` error (existing). |
| Unknown key inside `[profile.<name>]` | Ignored (forward compatibility). |

## Compatibility

- Configs with no `[profile.*]` tables behave exactly as before (zero breaking changes).
- `TWIG_ADDR`/`TWIG_API_KEY` continue to override credentials, now for whichever profile is selected.
- The new `TWIG_PROFILE` variable and `--profile` flag are the only additions.
