# Implementation Plan: External Commands for Plan Entry Boundaries

**Branch**: `070-plan-entry-hooks` | **Date**: 2026-07-28 | **Spec**: [spec.md](spec.md)

**Input**: Feature specification from `/specs/070-plan-entry-hooks/spec.md`

## Summary

Add a `[plan]` section to the config file with four optional command strings — `on_task_start`, `on_task_end`, `on_event_start`, `on_event_end` — that the TUI executes at the start and end of today's timed plan entries, so the user is told when to switch instead of having to watch the clock. Commands support `%s` (name), `%q` (quoted, shell-escaped name), `%t` (`HH:MM`), and `%%`.

The approach mirrors the existing pomodoro hooks: same config file, same `sh -c` execution with stdio detached from the alt-screen, same non-fatal failure handling. Two pieces are new. A dedicated cache of *today's* entries, refreshed in the background, because the Plan tab's visible day is user-navigable and stays unloaded until first visited. And a monotonic **watermark** — a boundary fires when `watermark < boundaryTime <= now` — which collapses "fire once", "nothing retroactive", "re-derive on edit", and "skip boundaries a clock jump flew over" into a single comparison, with no fired-set to maintain or evict.

Client-only: no protobuf change, no server change, no database change.

## Technical Context

**Language/Version**: Go 1.24 (root CLI/TUI module `github.com/pboyd/twig`)

**Primary Dependencies**: `charm.land/bubbletea/v2` (TUI runtime, tickers, async commands), `BurntSushi/toml` (config parsing), `os/exec` (hook execution), `api/gen/plan/v1` (existing `PlanEntry` type, consumed unchanged)

**Storage**: None. All feature state is in-memory and dies with the process. Config is read from the existing `~/.config/twig/config.toml`.

**Testing**: `go test ./...` from repo root. Pure-logic unit tests in `internal/tui` and `internal/config`, using the existing `m.nowFunc` clock-injection seam and the `export_test.go` shim convention.

**Target Platform**: Linux/macOS terminal. Hooks require a POSIX `sh` on `PATH`.

**Project Type**: CLI/TUI client in a three-module Go workspace. This feature touches the root module only.

**Performance Goals**: Boundary lateness ≤15s (spec allows 60s). One extra `ListPlanEntries` per 60s **only when hooks are configured**; zero additional requests otherwise.

**Constraints**: Must not block the render loop (FR-015). Hook stdout/stderr must never reach the terminal while the alt-screen is active. With no `[plan]` section, behaviour must be indistinguishable from today (SC-006).

**Scale/Scope**: ~10-40 plan entries per day → ≤80 boundaries to derive per 15s tick. Trivial; no indexing or caching beyond the entry slice itself.

## Constitution Check

*GATE: Must pass before Phase 0 research. Re-checked after Phase 1 design.*

| Principle | Status | Notes |
|-----------|--------|-------|
| I. Simplicity / YAGNI | ✅ | No new package (logic lives in `internal/tui` beside its only caller — research D7). No fired-set: one watermark comparison replaces a keyed map and its eviction rules (D4). No daemon, no timer cancellation, no new abstraction over the existing pomodoro hook runner. Complexity Tracking table is empty. |
| II. API-First Design | ✅ | The user-facing contract is the config schema, written to `contracts/config-schema.md` before implementation. No wire contract changes — `PlanEntry` is consumed exactly as it exists today. |
| III. UI/UX Consistency | ✅ | No new visual surface. Config keys mirror `[pomodoro].on_*` naming; execution semantics, failure handling, and stdio detachment are identical to the pomodoro hooks, so a user who has configured one already knows how the other behaves. Failures use the existing `m.notice` status-bar channel. |
| IV. Playful User Messages | ✅ | One new user-facing string: the hook-failure notice. It names the offending config key so it stays actionable, and carries the warm tone — e.g. `on_task_start went off the rails (exit 127) — check that command in your config.` |

**Post-Phase-1 re-check**: unchanged. The design added no abstractions, no packages, and one status-bar string.

## Project Structure

### Documentation (this feature)

```text
specs/070-plan-entry-hooks/
├── plan.md                      # This file
├── spec.md                      # Feature specification
├── research.md                  # Phase 0 — 7 decisions with rejected alternatives
├── data-model.md                # Phase 1 — in-memory entities and state transitions
├── quickstart.md                # Phase 1 — user setup + implementer notes
├── contracts/
│   └── config-schema.md         # Phase 1 — the [plan] section contract
├── checklists/
│   └── requirements.md          # Spec quality checklist (passing)
└── tasks.md                     # Phase 2 — NOT created by /speckit-plan
```

### Source Code (repository root)

```text
internal/config/
├── config.go                    # MODIFY: PlanConfig struct, Config.Plan field, enabled()
└── config_test.go               # MODIFY: parsing / absent-section / profile-isolation cases

internal/tui/
├── plan_hook.go                 # NEW: boundaries, watermark, expansion, ticker, fetch, runner
├── plan_hook_test.go            # NEW: unit tests for the pure logic
├── model.go                     # MODIFY: planHookState field; newModel takes config.PlanConfig
├── tui.go                       # MODIFY: pass cfg.Plan into newModel
├── update.go                    # MODIFY: Init starts the ticker; handle 3 new messages
└── export_test.go               # MODIFY: shims for unexported helpers under test

CLAUDE.md                        # MODIFY: SPECKIT plan pointer → this feature
```

**Structure Decision**: Root module only (`github.com/pboyd/twig`). The `api/` and `services/twig/` modules are untouched — boundaries are derived entirely from plan entries the client already fetches, so there is nothing for the server to learn about. Within the root module the work splits cleanly in two: config parsing in `internal/config`, and everything else in a single new `internal/tui/plan_hook.go` sitting beside `pomodoro.go`, whose hook-runner shape it copies.

## Implementation Phases

**Phase A — Config (independent, unblocks everything)**
`PlanConfig` struct, `Config.Plan` field, `enabled()` predicate, tests. Confirm `Config.Profile()` keeps `Plan` on the root, matching `Pomodoro`.

**Phase B — Pure logic (independent of A, fully unit-testable)**
`expandHookCmd` (single forward pass; `%s`/`%q`/`%t`/`%%`/passthrough; escape `"` `\` `$` `` ` ``), boundary derivation from `[]*planv1.PlanEntry` (skip untimed, end = start + duration, task-vs-event routing, name fallback to the linked task), and the watermark predicate with `(at, edge)` sorting.

**Phase C — Wiring (depends on A + B)**
`planHookState` on `Model`, `cfg.Plan` threaded through `tui.go` → `newModel`, watermark seeded at init, 15s ticker started in `Init` only when `enabled()`, and the three message handlers (`planHookTickMsg`, `planHookEntriesMsg`, `planHookErrMsg`).

**Phase D — Verification**
`go test ./...`, then a manual run confirming both that hooks fire (US1/US2/US3 acceptance scenarios) and that an unconfigured install issues no extra requests and starts no ticker (SC-006).

Phases A and B can proceed in parallel; C depends on both.

## Complexity Tracking

> No Constitution Check violations. Table intentionally empty.
