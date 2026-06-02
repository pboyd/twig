# Contract: `--profile` CLI / TUI Interface

## Global flag

```
twig [--profile <name>] <command> [args...]
twig [--profile <name>]                 # launches the TUI on a TTY
```

- `--profile` is a **global** option recognized only **before** the subcommand (or alone, to launch the TUI).
- Accepted forms: `--profile home` and `--profile=home`.
- `--profile` with no value → usage error, exit non-zero:
  `twig: --profile needs a profile name (try: twig --profile home …)`
- `--profile default` selects the root/default account (equivalent to omitting the flag, aside from being explicit).

## Environment variable

- `TWIG_PROFILE=<name>` selects the profile when `--profile` is absent.
- `TWIG_PROFILE=""` (empty) is treated as unset → default profile.
- When both `--profile` and `TWIG_PROFILE` are set, `--profile` wins.
- This is separate from `TWIG_ADDR`/`TWIG_API_KEY`, which override the credentials of whichever profile is selected.

## Applies to all surfaces

The selected profile applies uniformly to every subcommand (`task`, `pom`, `plan`) and to the interactive TUI. `twig --profile home` with no subcommand launches the TUI against the `home` account when stdout is a TTY.

## Messages (Principle IV — warm + actionable)

| Situation | Message (stderr) | Exit |
|-----------|------------------|------|
| Unknown profile | `twig: hmm, I couldn't find a profile named "<name>" — check the spelling, or add a [profile.<name>] section to <config-path>` | non-zero |
| `--profile` missing value | `twig: --profile needs a profile name (try: twig --profile home …)` | non-zero |
| API key missing after selection | *(existing)* `no API key found — set TWIG_API_KEY or add api_key to <config-path>` | non-zero |

- No account is contacted when profile selection fails.
- `<config-path>` is the resolved config file location (`config.DefaultPath()`).

## Behavioral contract (maps to FRs)

| Behavior | FR |
|----------|----|
| `--profile <name>` selects a configured profile for CLI + TUI | FR-002, FR-003, FR-004 |
| `TWIG_PROFILE` selects when `--profile` absent; flag wins | FR-002a, FR-002b |
| `default`/no selection → root credentials | FR-005, FR-006 |
| Credentials profile-scoped, no root fallback | FR-007, FR-008 |
| Pomodoro hooks always from root | FR-008a |
| Env credential vars override selected profile | FR-009, FR-010 |
| Unknown profile → named error, no account contact | FR-011 |
| `--profile` without value → usage error | FR-012 |
| No-profile configs unchanged | FR-013 |
| No provisioning/server change | FR-014 |
