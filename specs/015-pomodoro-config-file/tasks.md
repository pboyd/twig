---

description: "Task list for feature 015-pomodoro-config-file"
---

# Tasks: Pomodoro Config File

**Input**: Design documents from `/specs/015-pomodoro-config-file/`

**Prerequisites**: plan.md, spec.md, research.md, data-model.md, contracts/config-schema.md, quickstart.md

**Tests**: Tests are included where the existing package already has test coverage that would otherwise drift (notably `internal/cli/pom_test.go` and `internal/cli/cli_test.go`). The new `internal/config` package gets table-driven tests because it owns precedence logic — incorrect resolution is a silent correctness bug.

**Organization**: Grouped by user story (US1: P1 MVP, US2: P2 lifecycle hooks, US3: P3 config-driven server settings).

## Path Conventions

All paths are relative to the repository root and refer to the Go module under `services/todo/`.

---

## Phase 1: Setup (Shared Infrastructure)

**Purpose**: Add the TOML dependency and create the `internal/config` package skeleton.

- [X] T001 Add `github.com/BurntSushi/toml` to `services/todo/go.mod` via `cd services/todo && go get github.com/BurntSushi/toml`, then run `go mod tidy`. Verify `services/todo/go.sum` is updated.
- [X] T002 Create directory `services/todo/internal/config/` and an empty package file `services/todo/internal/config/config.go` declaring `package config`.

---

## Phase 2: Foundational (Blocking Prerequisites)

**Purpose**: Implement the `config.Config` struct, `Load()` function, and env-var precedence resolution. Every user story consumes this package.

**CRITICAL**: No user story work can begin until this phase is complete.

- [X] T003 In `services/todo/internal/config/config.go`, define the `Config` and `PomodoroConfig` structs exactly as specified in `specs/015-pomodoro-config-file/data-model.md` (fields: `APIURL`, `APIKey`, `Pomodoro.OnStart`, `Pomodoro.OnCancel`, `Pomodoro.OnComplete`). Use TOML tags `api_url`, `api_key`, `[pomodoro]` with `on_start`, `on_cancel`, `on_complete`.
- [X] T004 In `services/todo/internal/config/config.go`, implement `DefaultPath() (string, error)` returning `<UserConfigDir>/todo/config.toml` via `os.UserConfigDir()`.
- [X] T005 In `services/todo/internal/config/config.go`, implement `Load(path string) (Config, error)` that: (a) returns zero `Config{}` with nil error if the file does not exist (`errors.Is(err, fs.ErrNotExist)`), (b) returns a wrapped error mentioning the path on read or parse failure, (c) decodes the TOML into a `Config`, (d) ignores unknown keys (use `toml.DecodeFile` defaults). Per FR-011.
- [X] T006 In `services/todo/internal/config/config.go`, implement `(c Config) Resolve() Config` that applies env-var precedence: `TODO_ADDR` overrides `APIURL`; `TODO_API_KEY` overrides `APIKey`. Apply the built-in default `http://localhost:8080` for `APIURL` when both env and config are empty. Hooks pass through unchanged.
- [X] T007 [P] Create `services/todo/internal/config/config_test.go` with table-driven tests covering: (a) missing file → zero config, (b) empty file → zero config, (c) all keys set → all fields populated, (d) malformed TOML → error mentions the file path, (e) `Resolve()` env-var precedence over config, (f) `Resolve()` config used when env unset, (g) `Resolve()` default URL when both empty, (h) `Resolve()` empty `APIKey` left empty (callers handle the error).

**Checkpoint**: `go test ./internal/config/...` passes. The CLI does not yet read the config — that comes in the user-story phases.

---

## Phase 3: User Story 1 - Persistent pomodoro completion hook via config (Priority: P1) 🎯 MVP

**Goal**: Replace `--exec` with `[pomodoro].on_complete` from the config file. Works identically from CLI and TUI.

**Independent Test**: With a config file specifying `on_complete = "touch /tmp/pom-done"` and no `--exec` flag, start a pomodoro from `todo pom start <id>` and from the TUI; in both cases `/tmp/pom-done` exists after completion. Passing `--exec` produces an unknown-flag error.

### Implementation for User Story 1

- [X] T008 [US1] In `services/todo/internal/cli/cli.go`, add a package-level helper `loadConfig() (config.Config, error)` that calls `config.DefaultPath()` then `config.Load()` then `Config.Resolve()`. Return any error verbatim so callers can print and exit non-zero. Import `github.com/pboyd/todo/services/todo/internal/config`.
- [X] T009 [US1] In `services/todo/internal/cli/pom.go`, delete all `--exec` flag parsing: remove `parseExecFlag`, the `--exec` branches in `parseStartArgs`, `buildResumeArgs`, and the help text lines for `--exec` in `printPomUsage`. Any unknown flag (including `--exec`) must result in an error — confirm `parseStartArgs` rejects unknown args. Per FR-003.
- [X] T010 [US1] In `services/todo/internal/cli/pom.go::runCountdownAndComplete`, replace the `execCmd string` parameter with a `hooks config.PomodoroConfig` parameter (or pass the resolved `Config`). On natural completion, call `execHook(hooks.OnComplete)` only when non-empty. Keep the existing "warn but do not fail" behavior (FR-008).
- [X] T011 [US1] In `services/todo/internal/cli/pom.go::runPomTop` and its `start`/`resume` dispatchers, load config via `loadConfig()` at entry, surface load errors to stderr with non-zero exit, and pass the resolved `PomodoroConfig` down to `runCountdownAndComplete`. Update all call sites that previously passed `execCmd`.
- [X] T012 [US1] In `services/todo/internal/tui/pomodoro.go`, ensure the TUI's `execPomodoroStart` / `execPomodoroResume` paths flow through the same CLI runners (`pom.go::runCountdownAndComplete`) that now consume the config-driven hook. Confirm no separate hook plumbing is needed in the TUI; if the TUI uses an internal helper that bypasses CLI runners, route it back through them so FR-012 holds.
- [X] T013 [US1] Update `services/todo/internal/cli/pom_test.go`: delete tests that exercised `--exec` parsing; add a test that builds a temporary config file pointing at a marker-file shell command, invokes the pomodoro completion path (synthetic short duration via existing test hooks), and asserts the marker file exists.
- [X] T014 [US1] Update `services/todo/internal/cli/pom.go::printPomUsage` to document the new behavior: remove the `--exec` line, add a one-line pointer to the config file (`Configured via [pomodoro].on_complete in <path>; see 'todo help config' or specs/015-pomodoro-config-file/quickstart.md`).

**Checkpoint**: A user with `[pomodoro].on_complete` set runs `todo pom start <id>` (or the TUI equivalent) and sees the configured command run after completion, with no flag passed. `todo pom start <id> --exec foo` returns a non-zero exit and an unknown-flag message.

---

## Phase 4: User Story 2 - Start and cancel hooks for DND automation (Priority: P2)

**Goal**: Add `on_start` (fires after a successful `pom start`) and `on_cancel` (fires only on explicit cancel — not Ctrl-C, per the Q1 clarification) hooks.

**Independent Test**: Configure all three hooks with unique marker-file touches. Run `todo pom start <id>` → marker for start exists, no others. `todo pom cancel` → marker for cancel exists, marker for complete does NOT. Let another start run to completion → marker for complete exists, marker for cancel does NOT. Press Ctrl-C during a pomodoro → no marker file is created (Ctrl-C is detach, not cancel).

### Implementation for User Story 2

- [X] T015 [US2] In `services/todo/internal/cli/pom.go::runStart` (or wherever the `StartPomodoro` RPC is invoked), fire `execHook(hooks.OnStart)` immediately after the RPC returns success. Do NOT fire on RPC error. Per FR-004 and R5.
- [X] T016 [US2] In `services/todo/internal/cli/pom.go`, locate the explicit cancel pathway (`todo pom cancel` subcommand handler). Fire `execHook(hooks.OnCancel)` after the cancel RPC returns success. Do NOT fire on RPC error. Per FR-005 (clarified by Q1).
- [X] T017 [US2] In `services/todo/internal/cli/pom.go::runResume`, confirm that resume does NOT fire `on_start` and that resume errors out (without firing any hook) when no pomodoro is active server-side. Per Q3 clarification and FR-004 update.
- [X] T018 [US2] In `services/todo/internal/cli/pom.go`, ensure the countdown loop's SIGINT/Ctrl-C handling does NOT fire `on_cancel`. The signal handler should detach from the countdown UI cleanly while leaving the server-side pomodoro running; no hook fires. Per the Q1 clarification (Ctrl-C is detach, not cancel).
- [X] T019 [US2] In `services/todo/internal/tui/pomodoro.go` (and `update.go` if the cancel key lives there), ensure the TUI's cancel action fires `OnCancel` via the same shared runner used by the CLI cancel subcommand. The TUI's pomodoro-start action must already fire `OnStart` by virtue of routing through `runStart` (T012 enforced this); verify and add a test if needed.
- [X] T020 [P] [US2] Add tests in `services/todo/internal/cli/pom_test.go` for: (a) successful start fires `OnStart` once, (b) failed start RPC fires no hooks, (c) explicit cancel fires `OnCancel` and not `OnComplete`, (d) natural completion fires `OnComplete` and not `OnCancel`, (e) resume against an inactive pomodoro errors without firing any hook, (f) hook command that exits non-zero produces a warning but does not abort the pomodoro path (FR-008).

**Checkpoint**: All three lifecycle hooks behave per the spec, end-to-end from both CLI and TUI. Ctrl-C confirmed non-firing.

---

## Phase 5: User Story 3 - API URL and key in config file (Priority: P3)

**Goal**: `api_url` and `api_key` can be set in the config file; env vars override; missing-key error names both sources.

**Independent Test**: Unset `TODO_API_KEY` and `TODO_ADDR`. Set them in `~/.config/todo/config.toml`. Run `todo task list` — succeeds against the configured server. Then `export TODO_ADDR=http://wrong-host:9999` and run again — the command attempts the env-var value (proving precedence). With both unset and not in config, `todo task list` exits with an error message that names both `TODO_API_KEY` and the config-file path.

### Implementation for User Story 3

- [X] T021 [US3] In `services/todo/internal/cli/cli.go::runTask`, replace the direct `os.Getenv("TODO_API_KEY")` / `os.Getenv("TODO_ADDR")` reads with a call to `loadConfig()` (from T008) and use `cfg.APIKey` / `cfg.APIURL`. Update the missing-key error message to: `error: API key not set; set TODO_API_KEY env var or api_key in <config-path>` per FR-010 and the contract.
- [X] T022 [US3] In `services/todo/internal/cli/task.go` (line ~269), replace the direct `os.Getenv("TODO_ADDR")` read with the resolved `cfg.APIURL` from `loadConfig()` — refactor the function signature to accept a `Config` or have the caller pass `addr`/`apiKey` so we have a single source of truth.
- [X] T023 [US3] In `services/todo/internal/cli/plan.go`, do the same replacement for `TODO_API_KEY` / `TODO_ADDR`. Use `loadConfig()` once at the top of `runPlan` (and any sub-runners that previously read env directly).
- [X] T024 [US3] In `services/todo/internal/cli/pom.go::runPomTop`, the config is already loaded for hooks (T011); reuse it for `apiKey` and `addr` instead of reading env separately. Update the missing-key error.
- [X] T025 [US3] In `services/todo/internal/tui/client.go::NewClient`, accept a `config.Config` (or `addr`/`apiKey` strings already resolved) instead of reading env vars directly. Update the single caller in the TUI bootstrap to pass the resolved values.
- [X] T026 [P] [US3] Update `services/todo/internal/cli/cli_test.go` to add cases for: (a) config-file value used when env unset, (b) env wins over config when both set, (c) missing-key error message mentions both `TODO_API_KEY` and the config path. Use `t.Setenv` plus a temporary config file via `t.TempDir()` and an injectable `DefaultPath` (add a package-level `defaultPath` var or accept the path via `loadConfig` parameter — simplest: have tests set `XDG_CONFIG_HOME` to a temp dir).

**Checkpoint**: Running CLI commands in a fresh shell with no `TODO_*` env vars works against a config-only setup. Existing env-var-driven tests still pass.

---

## Phase 6: Polish & Cross-Cutting Concerns

**Purpose**: Documentation and final verification.

- [X] T027 [P] Update `CLAUDE.md` "Dev Environment" section to mention the optional config file at `~/.config/todo/config.toml` and link to `specs/015-pomodoro-config-file/contracts/config-schema.md`. Keep the existing env-var instructions; add a one-sentence note that env vars override config.
- [X] T028 [P] Update `services/todo/internal/cli/cli.go::printRootUsage` (or add a new `todo help config` path) so users discover where the config lives. Minimal change: append one line: `Config file: ~/.config/todo/config.toml (see specs/015-pomodoro-config-file/contracts/config-schema.md)`. Per FR-013.
- [ ] T029 Run the full quickstart end-to-end against a live server (`make dev`): create the config, run a real pomodoro from CLI and from TUI, cancel one, let one complete, observe marker files. Capture any deviation as a follow-up task.
- [X] T030 Final `cd services/todo && go test ./...` plus `go build ./cmd/todo` — both must succeed cleanly with no skipped tests related to this feature.

---

## Dependencies

```text
Phase 1 (Setup: T001, T002)
        │
        ▼
Phase 2 (Foundational: T003 → T004 → T005 → T006 → T007)
        │
        ├──▶ Phase 3 (US1 / P1 MVP)  ─── ships independently
        │
        ├──▶ Phase 4 (US2 / P2)      ─── depends on Phase 3's hook plumbing
        │                                  (T015–T020 reuse the hook path from T010/T011)
        │
        └──▶ Phase 5 (US3 / P3)      ─── independent of Phases 3 & 4;
                                          only depends on Phase 2 config loader
                ▼
        Phase 6 (Polish: T027–T030)
```

**Cross-story note**: US3 (config-driven server settings) is technically *independent* of US1/US2 but the loader (T008) is shared. If US1 ships first, US3 can be implemented later in any order; if US3 ships first, US1/US2 just consume the same `Config`. The dependency graph above shows the recommended *implementation* order, not a hard prerequisite chain.

## Parallel Execution Opportunities

- T002 can run alongside T001 once `go.mod` is in place.
- T007 (config tests) can be written in parallel with T003–T006 implementation since they live in separate files of the same package.
- T020 (US2 tests) and T026 (US3 tests) live in separate test files and can be written in parallel.
- T027 (CLAUDE.md edit) and T028 (CLI help text) touch disjoint files — fully parallel.

## Implementation Strategy

**MVP (ship after Phase 3)**: A user can configure `on_complete` once and never pass `--exec` again. This alone is the feature's primary motivation. Phase 4 and Phase 5 are additive improvements.

**Incremental delivery**: Each user-story phase ends at a checkpoint where the feature is *demonstrably useful* — Phase 3 alone gives users the `--exec` replacement; Phase 4 adds DND automation; Phase 5 lets them drop the env vars. The phases can be merged separately if desired.

**Task count**: 30 tasks total — Setup: 2, Foundational: 5, US1: 7, US2: 6, US3: 6, Polish: 4.
