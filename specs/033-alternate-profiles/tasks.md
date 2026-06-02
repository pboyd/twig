---
description: "Task list for Alternate Profiles"
---

# Tasks: Alternate Profiles

**Input**: Design documents from `/specs/033-alternate-profiles/`

**Prerequisites**: plan.md, spec.md, research.md, data-model.md, contracts/

**Tests**: INCLUDED. The project has established table-driven test conventions in `internal/config`, `internal/cli`, and `internal/tui`, and adding a map field to `Config` *requires* updating existing equality assertions to keep the package compiling. Test tasks are therefore integral, not optional.

**Organization**: Tasks are grouped by user story (from spec.md) for independent implementation and testing.

## Path Conventions

All paths are relative to the Go module at `services/twig/`. Run tests with `cd services/twig && go test ./...`.

This feature is client-side only — no server, proto, DB, or frontend changes.

---

## Phase 1: Setup (Shared Infrastructure)

**Purpose**: No new project scaffolding is required — this builds on the existing `cmd/twig` binary and `internal/` packages.

- [X] T001 Confirm baseline is green before starting: run `cd services/twig && go build ./... && go test ./...` and note current pass state.

---

## Phase 2: Foundational (Blocking Prerequisites)

**Purpose**: Config-file parsing of profiles + keep the `internal/config` test suite compiling. Every user story depends on this.

**⚠️ CRITICAL**: No user story work can begin until this phase is complete.

- [X] T002 Add `Profile` struct (`api_url`, `api_key` with `toml` tags) and a `Profiles map[string]Profile` field (tag `toml:"profile"`) to `Config` in `services/twig/internal/config/config.go`. Do not change `Resolve()` yet.
- [X] T003 Update `services/twig/internal/config/config_test.go` to compare `Config` values with `reflect.DeepEqual` instead of `==` (the new map field makes `Config` non-comparable). Keep all existing cases passing; add a `TestLoad` case asserting that a `[profile.home]` table decodes into `Profiles["home"]`.

**Checkpoint**: `go test ./internal/config/...` compiles and passes; `[profile.<name>]` tables now parse into `Config.Profiles`.

---

## Phase 3: User Story 1 - Switch to a named profile (Priority: P1) 🎯 MVP

**Goal**: A user can select an alternate account at launch with `--profile <name>` and have every command and the TUI use that account's credentials (hybrid: credentials from the profile, pomodoro hooks from the root).

**Independent Test**: With a config containing root credentials and a `[profile.home]` block, `twig --profile home <cmd>` targets the home account while `twig <cmd>` targets root; `twig --profile home` (no subcommand) launches the TUI against the home account.

### Tests for User Story 1 ⚠️

- [X] T004 [P] [US1] In `services/twig/internal/config/config_test.go`, add `TestProfile` covering: empty/`"default"` name → root credentials with `ok=true`; named profile → its `api_url`/`api_key` with `ok=true`; profile that omits `api_url` → empty `APIURL` (so `Resolve` later applies the built-in default), NOT the root URL; `Pomodoro` always equals the root's value regardless of selected profile.
- [X] T005 [P] [US1] In `services/twig/internal/cli/cli_test.go`, add tests for the global-flag extractor: `--profile home` and `--profile=home` yield name `home` with the flag stripped from remaining args; no flag yields empty name and unchanged args; `--profile` with no value yields a usage error.

### Implementation for User Story 1

- [X] T006 [US1] Implement `func (c Config) Profile(name string) (Config, bool)` in `services/twig/internal/config/config.go`: for `""`/`"default"` return `c` (credentials = root) with `ok=true`; for a named profile, look up `c.Profiles[name]` and on hit return a copy with `APIURL`/`APIKey` replaced by the profile's values (no merge with root creds) and `Pomodoro` left as root's, `ok=true`; on miss return `Config{}, false`. (Depends on T002.)
- [X] T007 [US1] Add a leading-global-flag extractor in `services/twig/cmd/twig/main.go` (e.g. `extractProfileFlag(args) (name string, set bool, rest []string, err error)`) that recognizes a leading `--profile <name>` / `--profile=<name>`, strips it from args, and returns a usage error (printed to stderr, exit non-zero) when the value is missing (FR-012). For US1, resolve the selected name as the flag value only (env var added in US3).
- [X] T008 [US1] Update `main()` in `services/twig/cmd/twig/main.go` to run the extractor first, then decide launch: if `rest` is empty and stdout is a TTY → `tui.Run(ctx, name)`; else → `cli.Run(name, rest)`.
- [X] T009 [US1] Thread the profile name through the CLI in `services/twig/internal/cli/cli.go`: change `Run` to `Run(profile string, args []string)` and `loadConfig` to `loadConfig(profile string)`; in `loadConfig`, after `config.Load`, call `cfg.Profile(profile)` and (for now) on `ok=false` return an error mentioning the profile and config path (warm wording finalized in US4), then `Resolve()`. Pass `profile` from `Run` into `runTask`/`runPomTop`/`runPlan`.
- [X] T010 [US1] Update `runPomTop` in `services/twig/internal/cli/pom.go` and `runPlan` in `services/twig/internal/cli/plan.go` to accept the `profile` argument and pass it to `loadConfig(profile)`. Update the `task` path (`runTask`) likewise in `services/twig/internal/cli/cli.go`. Keep `--help` short-circuits working without requiring a profile.
- [X] T011 [US1] Update `Run` in `services/twig/internal/tui/tui.go` to `Run(_ context.Context, profile string) error`, performing `Load` → `cfg.Profile(profile)` (error on `ok=false`, same as CLI) → `Resolve()`; keep the existing "no API key found" guard and client construction.
- [X] T012 [US1] Update any existing callers/tests broken by the signature changes (`cli.Run`, `tui.Run`) so `go build ./...` and `go test ./...` pass — including `internal/cli/cli_test.go` and `internal/tui/export_test.go` if they invoke these entry points.

**Checkpoint**: `twig --profile home <cmd>` uses the home account; `twig <cmd>` uses root; `twig --profile home` launches the TUI against home. `go test ./...` passes.

---

## Phase 4: User Story 2 - Explicit default and backward compatibility (Priority: P2)

**Goal**: `--profile default` selects the root account, and configs with no profile blocks behave exactly as before.

**Independent Test**: With a profile-free config, behavior is byte-for-byte identical to today; `--profile default` produces the same result as omitting the flag.

### Tests for User Story 2 ⚠️

- [X] T013 [P] [US2] In `services/twig/internal/config/config_test.go`, add cases asserting: a config with no `[profile.*]` tables yields `Profiles == nil` and `Profile("")`/`Profile("default")` return the root unchanged; a config that *does* define `[profile.default]` still has `Profile("default")` return the root credentials (the `default` name is reserved to the root, not the table).

### Implementation for User Story 2

- [X] T014 [US2] Confirm/guard the reserved `default` semantics in `Config.Profile` (`services/twig/internal/config/config.go`): selecting `"default"` (or `""`) MUST return root credentials even when a `[profile.default]` table exists. Adjust the selector only if T013 reveals a gap.

**Checkpoint**: Existing single-account configs are unaffected; `--profile default` == no flag for credential selection. `go test ./...` passes.

---

## Phase 5: User Story 3 - Precedence between flag, environment, and config (Priority: P2)

**Goal**: `TWIG_PROFILE` selects the profile when `--profile` is absent (flag wins), and `TWIG_ADDR`/`TWIG_API_KEY` override the credentials of whichever profile is selected.

**Independent Test**: With env credential vars set, `--profile home` uses the env values (overriding home's creds); `TWIG_PROFILE=home` with no flag selects home; `--profile default` overrides `TWIG_PROFILE=home`.

### Tests for User Story 3 ⚠️

- [X] T015 [P] [US3] In `services/twig/internal/config/config_test.go`, add a test asserting the Select→Resolve order: `cfg.Profile("home").Resolve()` with `TWIG_ADDR`/`TWIG_API_KEY` set returns the env values (env overrides the selected profile's credentials), and a profile with empty `api_url` and no env falls back to the built-in default URL.
- [X] T016 [P] [US3] In `services/twig/internal/cli/cli_test.go` (or `cmd/twig` if the resolver lives there), add tests for profile-name resolution precedence: flag value wins when set; `TWIG_PROFILE` used when flag absent; empty `TWIG_PROFILE` treated as unset → default; flag `default` overrides `TWIG_PROFILE=home`.

### Implementation for User Story 3

- [X] T017 [US3] Extend the name resolution from T007 in `services/twig/cmd/twig/main.go` so the selected profile name is `flag value if set, else os.Getenv("TWIG_PROFILE"), else ""`; treat an empty `TWIG_PROFILE` as unset. (No change needed to credential override order — `Resolve()` already runs after `Profile()` from US1; add a code comment noting this guarantees env-wins-for-credentials.)

**Checkpoint**: Both profile-selection inputs and credential-override env vars behave per the precedence tables in `contracts/config-schema.md`. `go test ./...` passes.

---

## Phase 6: User Story 4 - Helpful error for an unknown profile (Priority: P3)

**Goal**: Selecting a non-existent profile fails fast with a warm, actionable message naming the profile and config path, and contacts no account.

**Independent Test**: `twig --profile bogus <cmd>` (no `bogus` in config) exits non-zero, prints the named error, and makes no network call.

### Tests for User Story 4 ⚠️

- [X] T018 [P] [US4] In `services/twig/internal/cli/cli_test.go`, add a test that `loadConfig("bogus")` against a config without `bogus` returns an error whose message contains the profile name `bogus` and the config path, and does not construct a client / make a request.

### Implementation for User Story 4

- [X] T019 [US4] Finalize the unknown-profile error wording in `services/twig/internal/cli/cli.go` (`loadConfig`) and mirror it in `services/twig/internal/tui/tui.go` (`Run`) to match `contracts/cli-interface.md`: `twig: hmm, I couldn't find a profile named "<name>" — check the spelling, or add a [profile.<name>] section to <config-path>`. Ensure the error path returns/exits before any client is built or request is sent (FR-011), and verify the `--profile` missing-value message from T007 matches the contract wording.

**Checkpoint**: All four stories independently functional; error paths are warm and actionable.

---

## Phase 7: Polish & Cross-Cutting Concerns

**Purpose**: Documentation and discoverability for the new flag/env var.

- [X] T020 [P] Update `printRootUsage` in `services/twig/internal/cli/cli.go` to mention `twig [--profile <name>] <command>` and the `TWIG_PROFILE` env var, in the house tone (Principle IV).
- [X] T021 [P] Update `services/twig/CLAUDE.md` (the "CLI requires" / config section) and the repo `CLAUDE.md` config note to document `--profile` / `TWIG_PROFILE` and the `[profile.<name>]` schema, pointing to `specs/033-alternate-profiles/contracts/config-schema.md`.
- [X] T022 Run the manual smoke checks from `quickstart.md` (`./twig --profile nope task` → warm error, exit 1; `./twig --profile` → usage error, exit 1) and the full `cd services/twig && go test ./...` suite; confirm green.

---

## Dependencies & Execution Order

### Phase Dependencies

- **Setup (Phase 1)**: No dependencies.
- **Foundational (Phase 2)**: Depends on Setup. BLOCKS all user stories (parsing + compiling test suite).
- **User Stories (Phase 3–6)**: All depend on Foundational. US1 is the MVP and lays the selector + wiring the others build on.
- **Polish (Phase 7)**: Depends on the user stories being complete.

### User Story Dependencies

- **US1 (P1)**: Depends only on Foundational. Delivers the selector (`Config.Profile`) and all flag/threading wiring — the core increment.
- **US2 (P2)**: Logically depends on US1's selector (verifies/guards `default` + backward compat). Independently testable.
- **US3 (P2)**: Depends on US1's `main.go` resolver and Select→Resolve order; adds `TWIG_PROFILE` and precedence. Independently testable.
- **US4 (P3)**: Depends on US1's `loadConfig`/`tui.Run` error branch; finalizes the warm message + no-contact guarantee. Independently testable.

### Within Each User Story

- Write the listed tests first; let them fail, then implement.
- In US1: `Config.Profile` (T006) before CLI/TUI threading (T009–T011); flag extractor (T007) before `main()` routing (T008).

### Parallel Opportunities

- T004 and T005 (US1 tests) touch different files → can run in parallel.
- T013, T015, T016, T018 (story test tasks in different files) are each `[P]`.
- T020 and T021 (docs in different files) can run in parallel.
- Note: T006, T009–T012 all touch shared CLI/config files and the entry-point signatures — they are sequential, not parallel.

---

## Parallel Example: User Story 1

```bash
# Author the two failing test files together (different files):
Task: "T004 config.Profile selection tests in internal/config/config_test.go"
Task: "T005 global --profile flag extractor tests in internal/cli/cli_test.go"
# Then implement sequentially (shared files): T006 → T007 → T008 → T009 → T010 → T011 → T012
```

---

## Implementation Strategy

### MVP First (User Story 1 only)

1. Phase 1 Setup → 2. Phase 2 Foundational → 3. Phase 3 US1.
4. **STOP and VALIDATE**: switch between root and a named profile from both CLI and TUI; run `go test ./...`.
5. Ship — this alone delivers the multi-account value.

### Incremental Delivery

- Foundational + US1 → MVP (profile switching).
- + US2 → explicit default + verified backward compatibility.
- + US3 → `TWIG_PROFILE` selection + env precedence.
- + US4 → warm unknown-profile error.
- + Polish → usage text + docs.

---

## Notes

- [P] = different files, no dependencies. Most US1 implementation tasks share `cli.go`/`main.go`/`config.go` and are intentionally sequential.
- This feature has high shared-plumbing overlap; US2–US4 lean toward verifying and refining the US1 core, but each remains independently testable per its acceptance scenarios.
- No server/proto/DB/frontend changes; no new dependencies.
- Commit after each task or logical group; keep the suite green at every checkpoint.
