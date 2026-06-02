# Implementation Plan: Alternate Profiles

**Branch**: `033-alternate-profiles` | **Date**: 2026-06-02 | **Spec**: [spec.md](./spec.md)

**Input**: Feature specification from `/specs/033-alternate-profiles/spec.md`

## Summary

Let users keep more than one account in a single config file and choose between them at launch. The root of the config file remains the default account; named `[profile.<name>]` tables hold alternate account credentials (API URL + API key only). A profile is selected with the global `--profile <name>` flag or the `TWIG_PROFILE` environment variable (flag wins). Credentials are profile-scoped with no fallback to the root; all non-credential settings (pomodoro hooks) always come from the root. The existing `TWIG_ADDR`/`TWIG_API_KEY` env vars still override the credentials of whichever profile is selected.

Technical approach: extend `internal/config` with a `Profiles map[string]Profile` field and a `Profile(name)` selector that returns the effective (credentials-only) config; thread a profile name from `main.go` through the CLI command runners and the TUI entry point. No server, proto, or DB changes.

## Technical Context

**Language/Version**: Go 1.23 (existing module at `services/twig/`)

**Primary Dependencies**: `github.com/BurntSushi/toml` (already used by `internal/config`); no new dependencies

**Storage**: N/A — config is a single TOML file at `$XDG_CONFIG_HOME/twig/config.toml`

**Testing**: `go test ./...` (standard library testing; `internal/config` already has table-driven tests)

**Target Platform**: Linux/macOS CLI + TUI binary (`cmd/twig`)

**Project Type**: Single Go project, client-side CLI/TUI (the `cmd/twig` binary)

**Performance Goals**: N/A — one-time config parse at process start; no hot path

**Constraints**: Zero breaking changes for existing single-account configs (FR-013); `--profile` must be recognized before subcommand dispatch and before TUI launch

**Scale/Scope**: Small. ~1 new struct + 1 method in `internal/config`, global-flag extraction in `cmd/twig/main.go`, profile threading through `internal/cli` runners and `internal/tui.Run`.

## Constitution Check

*GATE: Must pass before Phase 0 research. Re-check after Phase 1 design.*

| Principle | Status | Notes |
|-----------|--------|-------|
| I. Simplicity / YAGNI | ✅ | One struct (`Profile`) + one selector method + threading a string. No new abstraction layers, no new dependencies. Reuses existing `Load`/`Resolve` flow. |
| II. API-First Design | ✅ | No RPC/proto surface changes. The relevant contract is the config-file schema + the CLI `--profile`/`TWIG_PROFILE` interface, both committed under `contracts/` before implementation (mirrors feature 015's config-schema contract). |
| III. UI/UX Consistency | ✅ | `--profile` follows existing flag conventions and applies uniformly to CLI and TUI; error output reuses the existing "no API key found" style. |
| IV. Playful User Messages | ✅ | The one new user-facing message (unknown-profile error) is written in the warm, actionable house tone; see contracts/cli-interface.md. |

**Post-Design Re-check**: ✅ Still passing. No new abstractions or surfaces introduced during design beyond those listed above.

## Project Structure

### Documentation (this feature)

```text
specs/033-alternate-profiles/
├── plan.md              # This file
├── research.md          # Phase 0 output
├── data-model.md        # Phase 1 output
├── quickstart.md        # Phase 1 output
├── contracts/           # Phase 1 output
│   ├── config-schema.md # Config file schema incl. [profile.<name>]
│   └── cli-interface.md # --profile flag / TWIG_PROFILE behavior + messages
└── tasks.md             # Phase 2 output (/speckit-tasks — NOT created here)
```

### Source Code (repository root)

```text
services/twig/
├── cmd/twig/
│   └── main.go                  # MODIFY: extract leading --profile flag; resolve flag>env; route to tui.Run/cli.Run
├── internal/
│   ├── config/
│   │   ├── config.go            # MODIFY: add Profile struct + Profiles map field + Profile(name) selector
│   │   └── config_test.go       # MODIFY: switch struct equality to reflect.DeepEqual (map field); add profile/select tests
│   ├── cli/
│   │   ├── cli.go               # MODIFY: Run + loadConfig take profile name; thread to runTask/runPomTop/runPlan
│   │   ├── pom.go               # MODIFY: runPomTop signature takes profile (passes to loadConfig)
│   │   ├── plan.go              # MODIFY: runPlan signature takes profile
│   │   └── cli_test.go / export_test.go  # MODIFY/ADD: tests for global-flag extraction + profile resolution
│   └── tui/
│       └── tui.go               # MODIFY: Run takes profile name; Load → Profile(name) → Resolve
```

**Structure Decision**: Single Go project. All changes live in the `cmd/twig` binary and its `internal/` packages. The `internal/config` package owns profile selection logic (testable without a TTY or server); `cmd/twig/main.go` owns flag extraction; `internal/cli` and `internal/tui` consume a resolved profile name. This keeps parsing, selection, and consumption cleanly separated and matches the existing data-flow layering.

## Complexity Tracking

> No Constitution Check violations. No entries required.
