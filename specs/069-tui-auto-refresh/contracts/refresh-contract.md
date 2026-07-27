# Contract: TUI Refresh Behavior

**Feature**: 069-tui-auto-refresh | **Date**: 2026-07-27

## Scope of this contract

This feature adds no RPC, no proto message, and no HTTP endpoint. It calls four existing ConnectRPC methods that are already used by the TUI, with unchanged request shapes:

| Method | Called by | Change |
|---|---|---|
| `task.v1.TaskService/ListTasks` | Tasks tab, Report tab | none — same empty request |
| `task.v1.TaskService/CountCompletedPomodoros` | Report tab | none |
| `plan.v1.PlanService/ListPlanEntries` | Plan tab | none |
| `plan.v1.PlanService/ListScheduledDays` | Tasks tab | none |
| `goal.v1.GoalService/ListGoals` | Goals tab | none |

The server is unaware of this feature. Nothing in `api/proto/` or `services/twig/` changes.

The contract below is therefore an **internal behavioral contract** for the TUI. It is the reviewable artifact Principle II asks for, expressed at the level this feature actually operates: which triggers cause a load, and what a load may and may not do to the screen.

## C1: Trigger contract

A tab's data is loaded on exactly these occasions.

| Trigger | Condition | Flavor |
|---|---|---|
| Startup | `Init` runs | user |
| Tab entry | user switches to the tab | user |
| Manual refresh | user presses `ctrl+r` | user |
| Mutation | a change the user made completes | user |
| Heartbeat | active tab, eligible, data older than 10 minutes | background |

**C1.1** Tab entry MUST load, unconditionally, for all four tabs. No `loaded` guard, no staleness check.

**C1.2** The heartbeat MUST load at most one tab per beat, and only the active tab.

**C1.3** The heartbeat MUST NOT load when the active tab is in any mode other than its plain list mode.

**C1.4** Suppressed beats MUST NOT be queued or replayed. The next beat re-evaluates from scratch.

## C2: Staleness contract

**C2.1** The threshold is 10 minutes, fixed in code, not configurable.

**C2.2** Staleness is evaluated per tab against that tab's own last-load mark.

**C2.3** The last-load mark advances on any successful load, and on dispatch of a background load.

**C2.4** Consequently: a failed background load MUST NOT be retried before the next full interval, and a background load MUST NOT be dispatched while one is already in flight for that tab.

## C3: Apply contract

When load results arrive, the following MUST hold.

### For every flavor

**C3.1** No load handler may set `m.mode`, `m.goal.mode`, or `m.plan.mode`. Interactive modes survive any arriving result.

**C3.2** The cursor is re-anchored to the item it was on, identified by ID. When that item is gone, the cursor clamps to a valid nearby index.

**C3.3** Expansion state, active filter, and list scroll position are not modified by a load.

**C3.4** Previously loaded content stays rendered until new content replaces it. No blank or loading state is shown.

### For background loads only

**C3.5** An error MUST be discarded: no `err` field is written, no message is rendered, the previous data stays.

**C3.6** An existing error message MUST NOT be cleared by a successful background load.

**C3.7** A result MUST be discarded if the user has left the tab it was dispatched for. For the Plan tab, "the tab" includes the day being shown.

**C3.8** The Report tab's scroll offset MUST be preserved, in contrast to a user-initiated load, which resets it to the top.

## C4: Preserved behavior

**C4.1** `ctrl+r` continues to work on every tab, loading that tab's data and reporting failures exactly as it does today.

**C4.2** No new visible element is introduced: no spinner, no "last synced" text, no staleness indicator, no status-bar change.

**C4.3** The pomodoro countdown tick and the plan-tab clock tick are untouched and continue to run at their own rates.

## C5: Observable acceptance

These are the checks that verify the contract from outside the code.

| ID | Setup | Action | Expected |
|---|---|---|---|
| A1 | Task shown on Plan and Tasks tabs | Complete on Plan, switch to Tasks | Shows completed |
| A2 | Goals tab visited once, goal changed elsewhere | Return to Goals tab | Shows the change |
| A3 | Tab loaded 30 seconds ago | Beat fires | No request issued |
| A4 | Tab loaded 11 minutes ago, list mode, idle | Beat fires | One request for that tab only |
| A5 | Tab stale, edit form open | Beat fires | No request issued |
| A6 | As A5, form then closed | Next beat fires | Request issued |
| A7 | Tab stale, server unreachable | Beat fires, request fails | Same data on screen, no error text |
| A8 | As A7 | 59 more minutes pass | At most 6 requests total in the hour |
| A9 | Cursor on item X, item inserted above it elsewhere | Background load applies | Cursor still on X |
| A10 | Cursor on item X, X deleted elsewhere | Background load applies | Cursor on a valid item, no panic |
| A11 | Error displayed, background load succeeds | Result applies | Error still displayed |
| A12 | Report scrolled down, background load succeeds | Result applies | Scroll position unchanged |
| A13 | Background load dispatched for Plan day D | User navigates to day E before it lands | Day E's data remains |
