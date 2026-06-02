# Research: Alternate Profiles

Phase 0 decisions. All spec clarifications were resolved in the `/speckit-clarify` session (hybrid resolution, env-wins-for-credentials, `TWIG_PROFILE` selection). The remaining open item was the concrete config-file syntax for profiles; that and the supporting implementation choices are settled below.

## Decision 1 — Profile table syntax: `[profile.<name>]` (dotted)

**Decision**: Represent profiles as TOML dotted tables under a `profile` key, e.g. `[profile.home]`, mapped to a `Profiles map[string]Profile` field with the struct tag `toml:"profile"`.

**Rationale**:
- The request's literal `[profile:home]` (a bare key containing a colon) is **invalid TOML**. Verified against the project's TOML library (`github.com/BurntSushi/toml`): it fails to parse with `expected '.' or ']' to end table name, but got ':' instead`.
- `[profile.home]` is idiomatic TOML and maps directly onto `map[string]Profile` with zero custom parsing — fully decoded by `toml.Decode` with no leftover `Undecoded` keys (verified).
- The user explicitly confirmed `profile.home` is acceptable.

**Alternatives considered**:
- `["profile:home"]` (quoted colon key): parses, but produces a flat top-level key `profile:home` mixed in with `api_url`/`api_key`, requiring manual prefix-stripping and a custom map. More code, less idiomatic — rejected (violates Principle I).
- `[[profile]]` array-of-tables with a `name` field: works but forces a linear scan and a `name` attribute per entry; a keyed map is simpler and gives O(1) lookup and natural uniqueness — rejected.

## Decision 2 — Profile selection precedence and where it is parsed

**Decision**: The effective profile name is resolved as: `--profile <name>` flag value if the flag is present, else the `TWIG_PROFILE` environment variable, else empty (the root/default profile). An empty string and the literal `default` both mean the root. Extraction of the leading `--profile` flag happens in `cmd/twig/main.go` **before** subcommand dispatch and before the TUI-launch decision.

**Rationale**:
- Matches FR-002a/FR-002b and the clarified precedence (flag wins over env).
- `--profile` is a *global* launch option, not a per-subcommand flag, so it must be stripped from `os.Args` before `cli.Run`/`tui.Run` see the remaining args. Parsing it once in `main.go` keeps a single source of truth and lets `twig --profile home` (no subcommand) still fall through to the TUI when remaining args are empty and stdout is a TTY.
- Both `--profile home` and `--profile=home` forms are accepted; `--profile` with no following value is a usage error (FR-012).

**Alternatives considered**:
- Parsing `--profile` inside each subcommand: duplicated logic across `task`/`pom`/`plan`, and wouldn't cover the no-subcommand TUI launch. Rejected.
- A `flag.FlagSet` for global flags: heavier than needed for a single recognized leading flag; the codebase currently hand-dispatches args. A small hand-written extractor keeps it consistent and simple.

## Decision 3 — Resolution order: select profile, then apply env overrides, then defaults

**Decision**: Compute the effective config in this order:
1. **Select** — pick credentials (`api_url`, `api_key`) from the chosen profile (or root for default). No fallback from a named profile's missing credential to the root credential (hybrid rule). Non-credential settings (`[pomodoro]`) are always taken from the root regardless of profile.
2. **Resolve** — apply `TWIG_ADDR`/`TWIG_API_KEY` env overrides on top of the selected credentials, then apply the built-in default API URL (`http://localhost:8080`) if still empty.

**Rationale**:
- Selecting first then letting env override implements both the hybrid rule (profile supplies the baseline) and the "env always wins for credentials" rule (FR-009) with no special-casing of named vs default.
- Keeps the existing `Resolve()` behavior intact for the default/no-flag path (FR-010, FR-013): with no profile selected, the root credentials flow into `Resolve()` exactly as today.
- A new `Profile(name) (Config, bool)` method performs the selection and reports whether a named profile exists, so the caller can raise the unknown-profile error (FR-011) with the config path in the message.

**Alternatives considered**:
- Folding selection into `Resolve()`: would couple env handling with profile lookup and obscure the unknown-profile error path. Keeping `Profile` and `Resolve` as two small steps is clearer and independently testable.

## Decision 4 — `Config` struct gains a map field; update equality in existing tests

**Decision**: Add `Profiles map[string]Profile` to `config.Config`. Update `config_test.go` to compare with `reflect.DeepEqual` instead of `==`.

**Rationale**:
- A struct containing a map is not comparable with `==`; the existing `TestLoad`/`TestResolve` use `got != tc.wantCfg`, which would no longer compile. Switching those comparisons to `reflect.DeepEqual` is a mechanical, low-risk change and is the standard Go idiom for structs with maps/slices.

**Alternatives considered**:
- Keeping profiles in a parallel structure outside `Config`: avoids the equality change but fragments the parsed config and complicates loading. Rejected — a single decoded struct is simpler.

## Decision 5 — Unknown-profile error tone and content

**Decision**: When a named profile is absent, exit non-zero without contacting any account, with a warm, actionable message naming the profile and config path, e.g.:

`twig: hmm, I couldn't find a profile named "work" — check the spelling, or add a [profile.work] section to <path>`

**Rationale**: Satisfies FR-011/SC-004 (names profile + config location, no silent fallback) and Principle IV (playful but still actionable). Exact wording is fixed in `contracts/cli-interface.md`.
