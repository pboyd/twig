# Research: TUI Auto-Refresh

**Feature**: 069-tui-auto-refresh | **Date**: 2026-07-27

All findings come from reading the existing TUI package. No external research was required: this feature adds no dependency, no server capability, and no new protocol.

## Existing state of the four load paths

| Tab | Fetch command | Result message | Handler |
|---|---|---|---|
| Tasks | `listTasksCmd` (+ `listScheduledDaysCmd`) | `listTasksResultMsg` | `update.go:843` |
| Goals | `listGoalsCmd` | `listGoalsResultMsg` | `update.go:1069` |
| Plan | `listPlanCmd` / `listPlanHighlightCmd` | `planEntriesMsg` | `plan_update.go:514` |
| Report | `fetchReportCmd` | `reportResultMsg` | `update.go:1056` |

Every tab already has a working fetch command that a background refresh can reuse verbatim. This is the single most important finding: the feature adds a *trigger*, not a *loader*.

## Decision 1: Reuse the existing fetch commands, tag the result message

**Decision**: Add a `bg bool` field to `listTasksResultMsg`, `listGoalsResultMsg`, `planEntriesMsg`, and `reportResultMsg`. Each fetch command gains a background-flavored constructor (or a bool parameter) that sets it. The four existing handlers branch on that flag where background behavior must differ.

**Rationale**: The alternative — separate message types per tab for background loads — doubles the message surface and forces the handler logic to be duplicated or extracted into shared helpers. A single boolean on an existing struct keeps every load path in one handler, so the two flavors cannot silently diverge. Principle I favors the boolean.

**Alternatives considered**:
- *Separate `bgTasksMsg`, `bgGoalsMsg`, … types*: rejected, four new types plus four new handler arms that must mirror the originals.
- *A generic wrapper `backgroundMsg{inner tea.Msg}`*: rejected, the handler would have to unwrap and re-dispatch, and Bubble Tea's type switch stops being a straightforward map from message to behavior.

## Decision 2: A one-minute heartbeat, not a ten-minute ticker

**Decision**: `autoRefreshTickCmd()` returns `tea.Tick(time.Minute, …)` producing `autoRefreshTickMsg{}`. It is started exactly once, from `Init` (`update.go:816`), and the `autoRefreshTickMsg` handler reschedules it unconditionally as its only other start point. The tick itself performs no fetch; it evaluates staleness.

**Rationale**: A plain ten-minute ticker fires on a fixed drum that ignores what the user just did, so a manual refresh at minute 9 is followed by an automatic one at minute 10, and a tab loaded at minute 11 can sit stale until minute 20. Evaluating every minute against a per-tab timestamp gives a true ten-minute debounce for a cost of one timer and no network traffic.

**Rationale for the single start point**: `pomTickCmd` is started from both `Init`-adjacent paths and `pomStartedMsg` (`update.go:914`), which is safe there because the pomodoro tick self-terminates when `m.pom == nil`. The auto-refresh tick never terminates, so a second start point would double the beat rate permanently. FR-010 exists for this reason.

**Alternatives considered**:
- *Ten-minute `tea.Tick` with no staleness check*: rejected per above.
- *A tick rescheduled with a computed remaining duration*: rejected as more state for no user-visible gain.

## Decision 3: Stamp `lastLoad` on background dispatch, not only on arrival

**Decision**: `lastLoad` is set when data successfully lands from any trigger, and additionally at the moment a background refresh is dispatched.

**Rationale**: If `lastLoad` moved only on success, a background refresh against an unreachable server would leave the tab stale, so the next beat one minute later would retry, and so on — sixty requests an hour against a dead server, violating SC-005 and contradicting the "silent, unobtrusive" intent. Stamping on dispatch means a failed background attempt simply waits another ten minutes. It also doubles as the in-flight guard: a dispatched refresh makes the tab non-stale, so the next beat cannot dispatch a second one.

**Note on FR-006**: the spec says the mark updates "whenever data lands". Dispatch-stamping is a superset of that rule, not a contradiction — it adds one more moment at which the mark moves. The observable consequence is exactly what FR-022 and SC-005 jointly require.

**Alternatives considered**:
- *Separate `lastLoad` and `lastAttempt` fields*: rejected, two fields where one suffices; nothing in the feature reads the two apart.
- *Exponential backoff on failure*: rejected as unjustified complexity for a ten-minute cadence.

## Decision 4: Discard a stale background result by checking the active tab in the handler

**Decision**: Each handler opens with a guard of the form `if msg.bg && m.activeTab != <that handler's tab> { return m, nil }`. The Plan handler additionally checks `msg.bgDay == m.plan.day`.

**Rationale**: Because each tab owns its own state struct, a background result landing after a tab switch writes into invisible state rather than corrupting the visible tab, so the risk is smaller than it first appears. The guard still matters for two reasons: it prevents a background error path from touching state the user is now looking at, and it keeps the "background refresh only ever touches the active tab" invariant true by construction rather than by argument. The Plan day check is the one case where a late result would genuinely clobber visible data, because the Plan tab's identity includes which day it shows.

**Precedent**: `filterGen` (`model.go`) already establishes generation-guarding of stale async responses in this package. The tab check is the same idea with the tab itself as the generation token, needing no counter.

## Decision 5: Fix cursor anchoring for all triggers, not just background ones

**Decision**: Change the Goals handler and the Plan handler to re-anchor the cursor by goal ID and entry ID respectively when no explicit highlight is requested, for every trigger. Do not gate this on `msg.bg`.

**Rationale**: The Tasks handler already does exactly this (`update.go:848-856`, "Preserve cursor by task id"), so the other tabs are the inconsistent ones. Index clamping is wrong for a manual refresh too — it is just less likely to bite, because the user pressed the key and expects motion. Making all four tabs agree removes a branch instead of adding one, and satisfies Principle III's demand that a user who learns one surface can predict the others.

**Alternatives considered**:
- *Gate ID-anchoring behind `msg.bg`*: rejected, it preserves a known-wrong behavior on three tabs and adds a conditional to do it.

## Decision 6: Report scroll preservation is background-only

**Decision**: `reportResultMsg` keeps `m.reportData.scroll = 0` for user-initiated loads and skips the reset when `msg.bg` is set.

**Rationale**: Unlike cursor anchoring, the scroll reset is *correct* for a user-initiated load: switching report presets should return the reader to the top. It is only wrong when the user did not ask. This is the one place where the two flavors genuinely differ in intent, so it is the one place that keeps a `msg.bg` branch on the apply side.

## Decision 7: Eligibility is a single predicate over existing mode fields

**Decision**: One method, `func (m Model) autoRefreshEligible() bool`, returning true only when `m.mode == modeList`, neither confirmation flag is set, and the active tab's own mode field is at rest (`m.goal.mode == goalList`, `m.plan.mode == planList`).

**Rationale**: The suppression set in FR-012 — edit forms, new-item forms, pickers, move mode, delete confirmations, help, filter bar — maps exactly onto states already represented by `m.mode`, `m.goal.mode`, and `m.plan.mode`. `modeHelp`, `modeMove`, `modeFilter`, `modeEdit`, `modeNewSubtask`, `modeNewRoot`, and `modeDatePrompt` are all non-`modeList` values of `m.mode`, so a single equality check covers most of the list without enumerating it. No new state is introduced to track "is the user busy".

## Non-findings

- **No server change**: no proto, no RPC, no migration, no handler work in `services/twig`.
- **No new user-facing text**: FR-024 forbids a new indicator, so Principle IV has no new surface to review.
- **No dependency change**: `tea.Tick` is already used by `pomTickCmd` (`pomodoro.go:48`) and `planTickCmd` (`plan_update.go:214`).
- **Testing infrastructure exists**: `export_test.go` shims plus `fakeTaskClient`, `fakePlanClient`, and `fakePomClient` already support driving `Model.Update` with synthetic messages. `m.nowFunc` (`model.go`) already provides injectable time, which the staleness comparison will use via `m.nowOrDefault()`.
