---
description: "Task list for 070-plan-entry-hooks"
---

# Tasks: External Commands for Plan Entry Boundaries

**Input**: Design documents from `/specs/070-plan-entry-hooks/`

**Prerequisites**: [plan.md](plan.md), [spec.md](spec.md), [research.md](research.md), [data-model.md](data-model.md), [contracts/config-schema.md](contracts/config-schema.md), [quickstart.md](quickstart.md)

**Tests**: Included. The repository treats unit tests as standard practice (`CLAUDE.md` § Testing conventions), every entity in this feature is pure logic, and `quickstart.md` enumerates the required cases. Tests are written alongside each slice rather than strictly before it, matching the existing `internal/tui` convention.

**Organization**: Grouped by user story. The slicing follows the spec's own priority rationale — US3 notes that hooks "still fire and can still alert" without substitution, so Foundational + US1 delivers working task alerts with literal command strings, and placeholder expansion lands as its own increment.

## Format: `[ID] [P?] [Story] Description`

- **[P]**: Can run in parallel (different files, no dependencies on incomplete tasks)
- **[Story]**: Which user story this task belongs to (US1, US2, US3)
- All paths are relative to the repository root (`/home/user/dev/twig`)

## Path Conventions

Root Go module `github.com/pboyd/twig` only. `api/` and `services/twig/` are untouched.

- Config: `internal/config/`
- TUI: `internal/tui/`

---

## Phase 1: Setup

**Purpose**: Establish a known-good baseline so later failures are attributable to this feature.

- [X] T001 Run `go test ./...` from the repository root and confirm a green baseline before making changes

---

## Phase 2: Foundational (Blocking Prerequisites)

**Purpose**: Config parsing, model state, and the firing machinery that every user story depends on. At the end of this phase the machinery is in place but no hook keys are consulted yet, so nothing executes.

**⚠️ CRITICAL**: No user story work can begin until this phase is complete.

- [X] T002 Add `PlanConfig` struct (`on_event_start`, `on_event_end`, `on_task_start`, `on_task_end` with `toml` tags), the `Plan PlanConfig \`toml:"plan"\`` field on `Config`, and the `enabled()` predicate in `internal/config/config.go`, per data-model.md §1
- [X] T003 Add config tests in `internal/config/config_test.go`: `[plan]` parses all four keys; absent section yields zero-value `PlanConfig` with `enabled()` false; partial section leaves other keys empty; unknown key inside `[plan]` is ignored; `Config.Profile()` keeps `Plan` from the root table (mirror the existing `Pomodoro` assertions in `TestProfile`)
- [X] T004 Add `planHookState` struct (`cfg`, `day`, `entries`, `loaded`, `lastFetch`, `watermark`) and the `planHooks planHookState` field on `Model` in `internal/tui/model.go`; extend `newModel` to take a `config.PlanConfig` parameter and seed `planHooks.cfg` plus `planHooks.watermark` = now (FR-006), per data-model.md §4
- [X] T005 Update the `newModel` call in `internal/tui/tui.go:50` to pass `cfg.Plan`
- [X] T006 Update `ExportNewModel` (and any other `newModel` callers) in `internal/tui/export_test.go` for the new signature, passing `config.PlanConfig{}`
- [X] T007 Create `internal/tui/plan_hook.go` with the three message types (`planHookTickMsg`, `planHookEntriesMsg`, `planHookErrMsg`) and `planHookTickCmd()` firing every 15 seconds, per data-model.md §5 and research.md D3
- [X] T008 Add `listPlanHooksCmd(client, day)` to `internal/tui/plan_hook.go` — calls `ListPlanEntries` for the given day and returns `planHookEntriesMsg`. Keep it separate from `listPlanCmd`; do not touch `handlePlanEntriesMsg` (research.md D2)
- [X] T009 Add `runPlanHook(cmd, key string) tea.Cmd` to `internal/tui/plan_hook.go` — `exec.Command("sh", "-c", cmd)` with stdio deliberately left unset so output cannot corrupt the alt-screen, returning `planHookErrMsg{key, err}` on failure and nil for an empty command. Mirror `runPomHook` at `internal/tui/pomodoro.go:58-70` (research.md D6)
- [X] T010 Add `boundaryEdge` (with `edgeEnd` before `edgeStart` so it sorts first) and the `planBoundary` struct to `internal/tui/plan_hook.go`, per data-model.md §2-3
- [X] T011 Add `dueBoundaries(bs []planBoundary, watermark, now time.Time) []planBoundary` to `internal/tui/plan_hook.go` — filters on `watermark.Before(b.at) && !b.at.After(now) && now.Sub(b.at) <= 2*time.Minute`, then sorts by `(at, edge)` so ends precede starts (FR-016, research.md D4)
- [X] T012 Add the tick reducer to `internal/tui/plan_hook.go`: on `planHookTickMsg`, roll `day` over when the local date changed (clearing `entries` and `loaded`, FR-008); dispatch `listPlanHooksCmd` when `lastFetch` is older than 60s; when `loaded`, derive boundaries, dispatch the due ones via `tea.Batch`, and advance `watermark` to now — **and when not `loaded`, leave `watermark` untouched** (data-model.md §4 transitions). Use `m.nowOrDefault()` for every time read
- [X] T013 Add the `planHookEntriesMsg` reducer to `internal/tui/plan_hook.go`: on success replace `entries`, set `loaded`, set `lastFetch`; on error set only `lastFetch`, leaving `entries`/`loaded` intact so a transient RPC failure neither disarms the hooks nor surfaces noise
- [X] T014 Wire into `internal/tui/update.go`: append `planHookTickCmd()` to the `Init` batch at line 847 **only when `m.planHooks.cfg.enabled()`** (SC-006), and add `case planHookTickMsg`, `case planHookEntriesMsg`, and `case planHookErrMsg` to `Update`
- [X] T015 Add `internal/tui/plan_hook_test.go` covering `dueBoundaries`: fires exactly once across consecutive evaluations; a boundary already past at startup never fires; a boundary more than 2 minutes late is skipped (clock jump); an end and a start on the same instant come back end-first

**Checkpoint**: Ticker runs and today's entries are fetched when hooks are configured; no commands execute yet because no hook key is selected. `go test ./...` passes.

---

## Phase 3: User Story 1 - Be told when it is time to start and stop a planned task (Priority: P1) 🎯 MVP

**Goal**: Timed plan entries linked to a task run `on_task_start` at their start and `on_task_end` at their end, with the command string used literally.

**Independent Test**: Set `on_task_start = 'echo "START" >> /tmp/twig-hooks.log'`, put a timed task entry on today's plan a minute out, leave the TUI open, and confirm one line appears at (not before) the start minute.

### Implementation for User Story 1

- [X] T016 [US1] Add `planBoundaries(entries []*planv1.PlanEntry, day string, tree []*cli.TreeNode) []planBoundary` to `internal/tui/plan_hook.go` — skip entries with a nil `StartMinute` (FR-007); start = local midnight of `day` + `StartMinute`; end = start + `DurationMinute`; set `task: entry.TaskId != 0`; resolve `name` from `entry.Name` falling back to `findTaskName(tree, entry.TaskId)` (`internal/tui/pomodoro.go:194`), per data-model.md §3
- [X] T017 [US1] Add `hookFor(cfg config.PlanConfig, b planBoundary) (cmd, key string)` to `internal/tui/plan_hook.go` returning `OnTaskStart`/`OnTaskEnd` for task boundaries (FR-002); return `"", ""` for event boundaries until US2
- [X] T018 [US1] Call `planBoundaries` and `hookFor` from the tick reducer in `internal/tui/plan_hook.go`, dispatching `runPlanHook` for each due boundary with a non-empty command
- [X] T019 [US1] Extend `internal/tui/plan_hook_test.go`: untimed entry produces no boundaries; a timed task entry produces exactly a start and an end at the right instants; end = start + duration; zero-duration entry yields coincident start and end; name falls back to the linked task's name when `entry.Name` is empty; `hookFor` picks `on_task_start`/`on_task_end` and returns empty for events

**Checkpoint**: US1 is fully functional — task entries fire literal commands at both boundaries.

---

## Phase 4: User Story 2 - Be told when a planned event begins and ends (Priority: P2)

**Goal**: Timed entries not linked to a task run `on_event_start` / `on_event_end`, independently of the task keys.

**Independent Test**: Configure only `on_event_start` / `on_event_end`, add a timed event to today's plan, and confirm those fire while the task keys stay silent.

### Implementation for User Story 2

- [ ] T020 [US2] Extend `hookFor` in `internal/tui/plan_hook.go` to return `OnEventStart`/`OnEventEnd` for boundaries where `task` is false (FR-003)
- [ ] T021 [US2] Extend `internal/tui/plan_hook_test.go`: an event entry routes to the event keys; a task entry never fires an event key and vice versa; with only event keys configured, a task boundary dispatches nothing

**Checkpoint**: US1 and US2 both work, independently of each other.

---

## Phase 5: User Story 3 - Put the entry name and time into the command (Priority: P3)

**Goal**: `%s`, `%q`, `%t`, and `%%` are substituted into the command immediately before execution, with `%q` safe against any entry name.

**Independent Test**: Configure `on_task_start = 'echo "%t: start %q" >> /tmp/twig-hooks.log'` with an entry named `Fix "auth" $bug`, and confirm the logged line reads `14:00: start "Fix "auth" $bug"`.

### Implementation for User Story 3

- [ ] T022 [US3] Add `shellEscapeDoubleQuoted(name string) string` to `internal/tui/plan_hook.go` — backslash-escapes `"`, `\`, `$`, and `` ` ``, the exact set the POSIX shell reinterprets inside double quotes (FR-010)
- [ ] T023 [US3] Add `expandHookCmd(cmd, name string, at time.Time) string` to `internal/tui/plan_hook.go` — a **single left-to-right pass** into a `strings.Builder`: `%s` → name verbatim, `%q` → `"` + escaped name + `"`, `%t` → `at.Format("15:04")`, `%%` → `%`, any other `%X` → both characters emitted unchanged, trailing bare `%` emitted as-is. Do not use `strings.ReplaceAll` chains — they re-scan substituted text and reopen the injection hole FR-013 closes (research.md D5)
- [ ] T024 [US3] Pass each due boundary's command through `expandHookCmd(cmd, b.name, b.at)` before handing it to `runPlanHook` in the tick reducer in `internal/tui/plan_hook.go` (FR-011: `%t` is the scheduled boundary time, not the wall-clock time of execution)
- [ ] T025 [US3] Extend `internal/tui/plan_hook_test.go`: each of `%s`, `%q`, `%t`, `%%`; unknown `%z` passes through; trailing bare `%`; non-recursion (a name of `100%s done` must not expand again); `%q` escapes `"`, `\`, `$`, and `` ` ``; `%t` zero-pads (`09:05`); `%t` reports the scheduled time when `now` is later; empty name yields `""` for `%q`
- [ ] T026 [US3] Verify the worked example in `contracts/config-schema.md` — `notify-send "%t: start %q"` with a task named `Fix "auth" $bug` at 14:00 — as an explicit test case in `internal/tui/plan_hook_test.go`

**Checkpoint**: All three user stories are independently functional.

---

## Phase 6: Polish & Cross-Cutting Concerns

- [ ] T027 Add the hook-failure notice in the `planHookErrMsg` handler in `internal/tui/update.go` — set `m.notice` (not `m.err`; this is the user's config misbehaving, not twig), naming the offending key and staying actionable in the warm tone Principle IV requires, e.g. `on_task_start went off the rails (exit 127) — check that command in your config.`
- [ ] T028 [P] Document the `[plan]` section in `AGENTS.md` alongside the existing pomodoro-hooks pointer, referencing `specs/070-plan-entry-hooks/contracts/config-schema.md`. **Edit `AGENTS.md` directly — `CLAUDE.md` is a symlink to it** and tooling refuses to write through the link
- [ ] T029 Run `go test ./...` from the repository root and confirm green
- [ ] T030 Manual validation per `quickstart.md`: build with `go build -o twig ./cmd/twig`, run through the US1/US2/US3 acceptance scenarios with a file-writing hook, and confirm back-to-back entries fire end-before-start (FR-016)
- [ ] T031 Verify SC-006: with **no** `[plan]` section in the config, confirm no hook ticker starts, no extra `ListPlanEntries` traffic is issued, and nothing is executed
- [ ] T032 Verify FR-008 day scope: with the Plan tab navigated to tomorrow, confirm today's boundaries still fire (this is the failure mode that ruled out reusing `m.plan.entries` — research.md D2)

---

## Dependencies & Execution Order

### Phase Dependencies

- **Setup (Phase 1)**: No dependencies
- **Foundational (Phase 2)**: Depends on Setup — **blocks all user stories**
- **User Stories (Phases 3-5)**: All depend on Foundational
  - US1 (P1) and US2 (P2) touch the same `hookFor` function, so they are sequential in practice
  - US3 (P3) is independent of both and could be built in parallel with them by a second developer
- **Polish (Phase 6)**: Depends on all desired stories

### Within Foundational

```text
T002 ──> T003                    (config, then its tests)
T004 ──> T005, T006              (newModel signature, then its callers)
T007 ──> T008, T009, T010
T010 ──> T011 ──> T015
T011, T009 ──> T012 ──> T013 ──> T014
```

T002/T003 (config) and T004-T015 (TUI) are two independent tracks — see plan.md Phase A / Phase B.

### Within Each User Story

- US1: T016 → T017 → T018 → T019
- US2: T020 → T021
- US3: T022 → T023 → T024 → T025 → T026

### Parallel Opportunities

Genuinely parallel work is limited because most of the feature lives in one new file (`internal/tui/plan_hook.go`), and two tasks editing the same file cannot run concurrently.

- **T002/T003** (`internal/config/`) run in parallel with **T004-T015** (`internal/tui/`) — different packages, no shared symbols
- **T005** and **T006** — different files, both unblocked by T004
- **T028** (docs) runs in parallel with any code task

---

## Parallel Example: Foundational Phase

```bash
# Two independent tracks after T001:
Track A (internal/config/): T002 → T003
Track B (internal/tui/):    T004 → T005, T006 → T007 → ... → T015
```

---

## Implementation Strategy

### MVP First (Foundational + User Story 1)

1. Phase 1: Setup — green baseline
2. Phase 2: Foundational — machinery in place, nothing firing
3. Phase 3: US1 — task boundaries fire literal commands
4. **STOP and VALIDATE**: a fixed `notify-send 'Time to switch!'` already solves the stated problem
5. Ship or continue

### Incremental Delivery

1. Foundational → nothing user-visible, no behaviour change
2. + US1 → task alerts work (MVP)
3. + US2 → event alerts work
4. + US3 → alerts say *which* task and *what time*

Each increment leaves the tree green and the feature usable.

---

## Notes

- The whole firing rule is one predicate: `watermark < b.at <= now && now-b.at <= 2min`. There is deliberately **no** fired-set — see research.md D4 before adding one.
- The single most likely bug: advancing `watermark` on a tick where `loaded` is false. An in-flight first fetch then lets the watermark sail past a boundary, silently swallowing it. T012 calls this out.
- `m.nowOrDefault()` (`internal/tui/update.go:2887`) is the clock seam; tests inject via `m.nowFunc`.
- Commit after each task or logical group. Stop at any checkpoint to validate a story independently.
