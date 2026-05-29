# Phase 1 Data Model: TUI Planning Tab

This feature adds no persistent data. The "model" here is the in-memory Bubble Tea state added to `internal/tui`. Server-side entities (`PlanEntry`, `Task`) are unchanged and accessed via existing RPCs (see `contracts/plan-rpcs.md`).

## Model extensions

Added to the existing `tui.Model`:

```go
type tab int
const (
    tabTasks tab = iota
    tabPlanning
)

type Model struct {
    // ...existing task + pomodoro fields unchanged...
    activeTab  tab
    planClient planv1connect.PlanServiceClient
    plan       planState
}
```

- `activeTab` — which tab owns the screen. `Update`/`View` branch on it first.
- `planClient` — ConnectRPC client for `PlanService`, built in `client.go` like `client` (TaskService).
- `plan` — all Planning-tab state, kept separate so the Tasks tab is unaffected when inactive (FR-003).

## planState

```go
type planState struct {
    day     string            // in-view day, "YYYY-MM-DD"; defaults to today
    entries []*planv1.PlanEntry // entries for `day`, sorted by start_minute (server order)
    cursor  int               // index into entries of the highlighted entry; clamped to [0, len)
    loaded  bool              // false until the first ListPlanEntries for `day` returns
    mode    planMode          // current sub-mode (see below)
    picker  pickerState       // task picker state (valid in planPickTask)
    form    planFormState     // prompt/form state (valid in the prompt modes)
    err     error             // last planning error, surfaced in the shared status bar
}
```

### Selected entry

The highlighted entry is `entries[cursor]` when `len(entries) > 0`; otherwise there is no selection and entry actions (rename/move/remove) are no-ops. `cursor` is clamped after every list change with the existing clamp pattern.

## planMode (sub-state machine)

```go
type planMode int
const (
    planList      planMode = iota // grid shown; selection nav + action keys
    planPickTask                  // task tree picker open (step 1 of add-task)
    planTaskTime                  // start/duration form (step 2 of add-task)
    planEventForm                 // name + start + duration form (add-event)
    planRename                    // single-field name form (rename selected)
    planMove                      // start + duration form (move selected)
    planClear                     // single-field start-time form (clear; defaults to now)
)
```

Any mode other than `planList` is a **modal**: while in one, the tab-switch keys are ignored (FR-023) and `esc` cancels back to `planList` with no change (FR-020).

### Transitions

```text
planList --a--> planPickTask --(choose task)--> planTaskTime --(submit ok)--> [AddPlanTask] --> planList
planList --e--> planEventForm --(submit ok)--> [AddPlanEvent] --> planList
planList --r--> planRename     --(submit ok)--> [RenamePlanEntry] --> planList   (requires selection)
planList --m--> planMove       --(submit ok)--> [MovePlanEntry]  --> planList    (requires selection)
planList --d--> [RemovePlanEntry] --> planList                                   (requires selection)
planList --c--> planClear      --(submit ok)--> [ClearPlan] --> planList
any modal --esc--> planList (no change)
submit with invalid time/duration --> stay in modal, set err (FR-021)
RPC error --> planList, set err, entries unchanged from last good state (FR-021)
```

Day navigation and refresh happen only from `planList`:

```text
planList --[ / ]--> day = day∓1d, reload      (any day reachable)
planList --t------> day = today, reload
planList --ctrl+r-> reload current day
(tab becomes active) --> reload current day     (FR-024)
(any successful mutation) --> reload current day (FR-019, FR-024)
```

## pickerState

```go
type pickerState struct {
    tree    []*cli.TreeNode  // built via cli.BuildTree from ListTasks (incomplete tasks)
    visible []*visibleRow    // flattened, like the Tasks tab
    cursor  int
    expanded map[int64]bool
}
```

Reuses the Tasks-tab tree flattening/rendering. Selecting a node captures its task id and advances to `planTaskTime`. An empty picker can be cancelled with `esc` (edge case: no tasks).

## planFormState

```go
type planFormState struct {
    fields  []textinput.Model // 1–3 Bubbles text inputs depending on the mode
    focus   int               // focused field index (Tab/Shift-Tab cycles)
    taskID  int64             // carried from the picker for planTaskTime
    entryID int32             // target entry for planRename / planMove
}
```

Field layout per mode:

| Mode | Fields |
|------|--------|
| `planTaskTime` | start, duration (optional) |
| `planEventForm` | name, start, duration (optional) |
| `planRename` | name (prefilled with current) |
| `planMove` | start, duration (optional; blank ⇒ keep existing per RPC default) |
| `planClear` | start (prefilled with now) |

On submit, time fields parse with `timeparse.ParseStart` and duration with `timeparse.ParseDurationOrEnd(value, start)`; `duration == 0` is sent as-is so the server applies its documented default (see contract).

## Messages and commands (reducer surface)

| Message | Produced by | Effect |
|---------|-------------|--------|
| `planEntriesMsg{entries, err}` | `listPlanCmd` | replace `plan.entries`, set `loaded`, clamp cursor, or set `err` |
| `planMutatedMsg{highlightID, err}` | add/rename/move/remove/clear cmds | on success, reload then select `highlightID` (or clamp); on error set `err`, no reload |
| `planTasksMsg{tree, err}` | `listTasksForPickerCmd` | populate `picker` and enter `planPickTask`, or set `err` |
| `planTickMsg` | `planTickCmd` | re-render (advance now-marker); re-arm only if Planning tab active on today |

Command factories wrap the existing RPCs: `listPlanCmd(day)`, `addPlanTaskCmd(day, taskID, start, dur)`, `addPlanEventCmd(day, name, start, dur)`, `renamePlanCmd(day, id, name)`, `movePlanCmd(day, id, start, dur)`, `removePlanCmd(day, id)`, `clearPlanCmd(day, start)`, `listTasksForPickerCmd()`, `planTickCmd()`.

## Validation rules (client-side, before RPC)

- Start time MUST parse via `timeparse.ParseStart`; otherwise show error, stay in form.
- Duration, when non-empty, MUST parse via `timeparse.ParseDurationOrEnd`; empty ⇒ send `0` (server default).
- Event name MUST be non-empty.
- Rename/move/remove require a current selection; with no entries the keys are no-ops.
- All other validation (range, overlap/precondition, not-found) is enforced server-side and surfaced as `err` (FR-021).

## Rendering notes

- A one-line **tab bar** renders at the top in both tabs (`Tasks │ Planning`, active highlighted); on the Planning tab the in-view day label is shown (e.g. `Thu May 28`), and whether it is today.
- The tab bar consumes one line; every view's inner-height math subtracts it in addition to the existing status-bar height.
- The grid is produced by `cli.RenderGrid(entries, day, now, width, styled, GridOptions{HideID: true, SelectedID: selectedID})`; `now` drives the marker only when `day == today`.
- The shared status bar (pomodoro / help / error) and quit-guard render below both tabs, unchanged.
