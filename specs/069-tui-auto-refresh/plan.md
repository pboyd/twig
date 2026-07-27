# Implementation Plan: TUI Auto-Refresh

**Branch**: `069-tui-auto-refresh` | **Date**: 2026-07-27 | **Spec**: [spec.md](./spec.md)

**Input**: Feature specification from `/specs/069-tui-auto-refresh/spec.md`

## Summary

Replace the TUI's fully-manual refresh with two automatic triggers. First, entering any tab reloads that tab, making the rule uniform across all four instead of the current mix of always, once-ever, and never. Second, a one-minute heartbeat reloads the active tab when its data is over ten minutes old, so a session left running catches up with changes made on another device.

The technical approach is deliberately small: every tab already owns a working fetch command, so the feature adds a trigger rather than a loader. A `bg bool` on the four existing result messages distinguishes clock-initiated loads from user-initiated ones, and only three behaviors branch on it — error suppression, error preservation, and the Report tab's scroll reset. Along the way the Goals and Plan tabs adopt the ID-based cursor anchoring the Tasks tab already has, which removes an inconsistency rather than adding a special case.

## Technical Context

**Language/Version**: Go 1.24 (root module `github.com/pboyd/twig`)

**Primary Dependencies**: `charm.land/bubbletea/v2` for the event loop and `tea.Tick`; `charm.land/bubbles/v2` for widgets. No new dependency.

**Storage**: N/A — all new state is in-memory `Model` state, discarded on exit. Nothing is persisted, not even the last-load marks.

**Testing**: `go test ./internal/tui/...`, driving `Model.Update` with synthetic messages through the existing `export_test.go` shims and the `fakeTaskClient` / `fakePlanClient` stubs. Time is injected through the existing `m.nowFunc`.

**Target Platform**: Terminal, any platform the CLI already supports.

**Project Type**: Terminal UI within a multi-module Go repo. This feature touches one package.

**Performance Goals**: An idle session issues at most 6 background requests per hour per SC-005. The heartbeat itself does no work beyond two comparisons per minute.

**Constraints**: The heartbeat must never fire while the user is mid-interaction (FR-012). Exactly one tick chain may exist (FR-010). No new user-visible element (FR-024).

**Scale/Scope**: Roughly 150 lines across three files in `internal/tui`, plus tests. No server, proto, database, or web changes.

## Constitution Check

*GATE: Must pass before Phase 0 research. Re-checked after Phase 1 design.*

| Principle | Status | Notes |
|-----------|--------|-------|
| I. Simplicity / YAGNI | ✅ | One boolean on four existing messages instead of four new message types; one `time.Time` per tab instead of a load-tracking abstraction; no configurability, no backoff, no pending-refresh queue. The one place the design *removes* a branch — uniform ID-based cursor anchoring — was chosen over gating the fix behind the background flag. Complexity Tracking table is empty. |
| II. API-First Design | ✅ | No proto, RPC, or endpoint changes; the five RPCs called are already in use with unchanged requests. The reviewable contract is `contracts/refresh-contract.md`, which fixes the trigger, staleness, and apply rules before implementation begins. |
| III. UI/UX Consistency | ✅ | The feature's substance *is* consistency: four tabs that currently reload on entry three different ways converge on one rule, and three tabs converge on the Tasks tab's cursor-anchoring behavior. No new colors, styles, or key bindings — `ctrl+r` keeps working unchanged (C4.1). |
| IV. Playful User Messages | ✅ | No new user-facing text. FR-024 and C4.2 forbid a new indicator or status line, and background failures are silent by design (FR-021), so no message exists to review for tone. |

## Project Structure

### Documentation (this feature)

```text
specs/069-tui-auto-refresh/
├── plan.md                        # This file
├── spec.md                        # Feature specification
├── research.md                    # Phase 0 output — 7 decisions
├── data-model.md                  # Phase 1 output — new Model state
├── quickstart.md                  # Phase 1 output — build, verify, review
├── contracts/
│   └── refresh-contract.md        # Phase 1 output — internal behavioral contract
├── checklists/
│   └── requirements.md            # Spec quality checklist
└── tasks.md                       # Phase 2 output — NOT created by /speckit-plan
```

### Source Code (repository root)

```text
internal/tui/
├── model.go              # + tasksLastLoad on Model; + lastLoad on goalState,
│                         #   planState, reportState
├── update.go             # + autoRefreshTickMsg, autoRefreshTickCmd, constants
│                         # + autoRefreshTickMsg handler (the heartbeat)
│                         # + autoRefreshEligible() predicate
│                         # + bg on listTasksResultMsg, listGoalsResultMsg,
│                         #   reportResultMsg
│                         # ~ Init: start the tick chain once
│                         # ~ tab-entry paths: always reload (4 sites)
│                         # ~ listTasksResultMsg handler: bg guards, lastLoad
│                         # ~ listGoalsResultMsg handler: bg guards, ID anchoring,
│                         #   lastLoad
│                         # ~ reportResultMsg handler: bg guards, scroll, lastLoad
├── plan_update.go        # + bg, bgDay on planEntriesMsg
│                         # ~ handlePlanEntriesMsg: bg guards, entry-ID anchoring,
│                         #   day check, lastLoad
└── export_test.go        # + shims for lastLoad fields and autoRefreshEligible
```

**Structure Decision**: Single package, `internal/tui` in the root module. The other two modules (`api/`, `services/twig/`) and the web SPA are untouched — this is a client-side trigger change against RPCs that already exist. No new files are created; the feature is small enough that a separate `autorefresh.go` would separate the heartbeat from the four handlers it coordinates with, which is where the real logic lives.

## Implementation Phases

Phased to match the spec's story priorities, so each phase is independently shippable.

### Phase A — Tab-switch refresh (User Story 1, P1)

Delivers the completed-task bug fix with no timer and no new state.

1. Add `listTasksCmd` (batched with `listScheduledDaysCmd`) to the Goals→Tasks path at `update.go:1430` and the Plan→Tasks path at `update.go:1890`.
2. Remove the `if !m.goal.loaded` guard from the two Goals-entry paths (`update.go:1309`, `update.go:2253`) so Goals always reloads.
3. Confirm the Plan-entry and Report-entry paths already reload unconditionally; no change expected.

Ships alone. Satisfies FR-001 through FR-004 and acceptance A1, A2.

### Phase B — Cursor anchoring and apply-side safety (User Story 3, P3, partial)

Done before the heartbeat exists, so the heartbeat lands on handlers that are already safe to call unprompted.

4. Add `lastLoad` fields per `data-model.md`; stamp them on successful load in all four handlers.
5. Change the Goals handler (`update.go:1084`) to anchor the cursor by goal ID, mirroring the Tasks handler at `update.go:848`.
6. Change `handlePlanEntriesMsg` (`plan_update.go:531`) to anchor by entry ID.
7. Verify by test that no load handler writes `m.mode`, `m.goal.mode`, or `m.plan.mode` (C3.1).

Satisfies FR-014 through FR-017 and acceptance A9, A10.

### Phase C — The heartbeat (User Story 2, P2)

8. Add `autoRefreshInterval` (10 min) and `autoRefreshTickRate` (1 min) constants, `autoRefreshTickMsg`, and `autoRefreshTickCmd`.
9. Start the chain in `Init` (`update.go:816`) — the only start point besides the handler's own reschedule.
10. Add `autoRefreshEligible()` per `data-model.md`.
11. Add the `autoRefreshTickMsg` handler: reschedule always; dispatch only when eligible and stale; stamp `lastLoad` at dispatch.
12. Add the `bg` field to the four messages and background-flavored dispatch for each tab's fetch command.

Satisfies FR-005 through FR-013 and acceptance A3, A4, A5, A6.

### Phase D — Background apply semantics

13. Add the `msg.bg && msg.err != nil` early return to all four handlers (FR-020, FR-021).
14. Stop clearing an existing error on a successful background load (FR-018).
15. Add the active-tab guard to all four handlers, plus the `bgDay` check in the Plan handler (FR-019).
16. Skip the Report scroll reset when `msg.bg` (FR-017, C3.8).

Satisfies acceptance A7, A8, A11, A12, A13.

### Phase E — Tests

Unit tests in `internal/tui`, one per acceptance row in `contracts/refresh-contract.md` §C5, driving `Model.Update` directly with injected time. Plus a guard test asserting `autoRefreshTickCmd` has exactly two non-test call sites, since that invariant is invisible at runtime until the beat rate silently doubles.

## Risks

| Risk | Mitigation |
|---|---|
| A second `autoRefreshTickCmd` call site is added later, doubling the beat rate permanently — invisible in testing, visible only as extra server load | Documented as an invariant in `data-model.md` and `quickstart.md`; covered by a source-level guard test in Phase E |
| A future load path forgets to stamp `lastLoad`, making a tab refresh every minute | All four stamps land in the four result handlers, which are the only places load results are applied |
| Suppression misses a mode added later, letting a refresh fire under a new form | `autoRefreshEligible` keys off `m.mode == modeList` rather than enumerating modes, so a new non-list mode suppresses automatically |
| Ten minutes proves too slow or too aggressive in practice | Single constant, trivially changed; deliberately not exposed as config per FR-009 |

## Complexity Tracking

> No Constitution Check violations. Table intentionally empty.
