# Phase 1 Data Model: Add New Tasks to a Plan

**Feature**: 066-add-task-to-plan | **Date**: 2026-07-17

## Persisted data: unchanged

This feature adds **no** database table, column, migration, or proto message. It writes one row to
the existing plan-entry store through the existing `AddPlanTask` RPC, identical to what the
task-list scheduling path already writes.

| Entity | Change |
|---|---|
| Task | none — the plan choice is never stored on the task (spec: Key Entities) |
| Plan entry | none — this feature creates an ordinary untimed task entry |

## Transient UI state (the whole model)

The plan choice lives only in the open create form and dies with it. Nothing persists between form
openings (spec Assumptions: "No new stored state").

### The selection

One value with four possible states, shared conceptually by both surfaces:

| State | Meaning | Resolves to (at save) |
|---|---|---|
| `None` | default; do not plan | no plan call |
| `Today` | plan for the local current day | `nowFunc()` → `YYYY-MM-DD` |
| `Tomorrow` | plan for the local next day | `nowFunc()+1d` → `YYYY-MM-DD` |
| `Date(d)` | plan for an explicitly picked day | `d` verbatim |

**Invariants**:
- Exactly one state is active at a time.
- `None` is the state of every freshly opened create-task form (FR-003).
- `Today`/`Tomorrow` are *relative* until save; they are not frozen at open time (FR-005, and the
  midnight edge case).
- The selection never affects any other field (FR-014).

### TUI: `editFormModel` additions (`internal/tui/edit.go`)

```go
const (
    focusName        = 0
    focusDescription = 1
    focusDue         = 2
    focusEstimate    = 3
    focusSnooze      = 4
    focusState       = 5
    focusPlan        = 6  // NEW — before Goal, matching the spec mockup
    focusGoal        = 7  // renumbered
    focusSave        = 8  // renumbered
    focusCancel      = 9  // renumbered
    focusCount       = 10 // renumbered
)

// Plan selector (task create forms only; hidden when showPlanField is false).
planIdx       int    // index into planChoices; 0 = none
planDate      string // ISO day, set only when a calendar pick made planIdx == planChoiceDate
showPlanField bool   // true only when creating a task (not editing, not a goal)
```

The cycle list is fixed and ordered to match the spec mockup:

```go
const (
    planChoiceNone     = 0
    planChoiceToday    = 1
    planChoiceTomorrow = 2
    planChoiceDate     = 3 // only reachable via the calendar; skipped when cycling
)
```

`planChoiceDate` is a *destination*, not a cycle stop: left/right moves through
`None → Today → Tomorrow → None`. A calendar pick jumps the selector to `planChoiceDate`; cycling
from there re-enters the three-stop loop and clears `planDate`.

**Dirty tracking**: `origPlanIdx` snapshots the opened state so `isDirty()` reports a touched plan
control, keeping the discard-confirmation behavior correct (spec Assumptions).

### TUI: `editSavedMsg` addition (`internal/tui/edit.go`)

```go
// planDay is the resolved ISO day (YYYY-MM-DD) to add the new task to.
// Empty means the user chose no plan. Populated only when showPlanField was true.
planDay string
```

Resolution from `planIdx` to `planDay` happens in `buildSaveMsg`, using `f.now()` — so the value on
the wire is always an absolute day, and every consumer downstream is time-independent.

### Web: `TaskForm` state (`services/twig-web/src/components/TaskForm.tsx`)

```ts
type PlanChoice = "none" | "today" | "tomorrow" | "date";

const [planChoice, setPlanChoice] = useState<PlanChoice>("none");
const [planDate, setPlanDate] = useState("");   // ISO day; only meaningful when planChoice === "date"
const [showDateInput, setShowDateInput] = useState(false);
```

Resolved at submit into the optional third argument of `onSubmit`:

```ts
onSubmit: (name: string, description: string, planDay?: string) => Promise<void>;
```

`planDay` is `undefined` for "none" and for a `"date"` choice with no date entered yet; otherwise an
ISO day from `todayIso()` / `tomorrowIso()` / `planDate`.

## State transitions

```
                    ┌──────────────────────────────────┐
                    │                                  │
   (form opens) ──► None ──►  Today ──► Tomorrow ──────┘
                     ▲                       (cycle: TUI ←/→, web: click)
                     │
                     └──── Date(d) ◄──── calendar pick / date input
                        (cycle away clears d)
```

- **Any → Date(d)**: TUI `ctrl+g` pick; web 📅 affordance + date input.
- **Date(d) → None**: cycling away, or clearing the date input. `planDate` is cleared with it, so a
  stale date can never be submitted after the user has moved off it.
- **Save**: the active state resolves to an ISO day or to nothing. No other transition.

## Validation

| Rule | Where | On violation |
|---|---|---|
| A `Date` choice must carry a parseable ISO day | both surfaces, at save | control keeps its previous value; no plan call (spec edge case) |
| Past days are allowed | both | none — consistent with existing scheduling (US2 scenario 4) |
| Plan choice never blocks task creation | both | task is created regardless (FR-006/FR-007) |

There is deliberately no cross-field validation against Snooze or Due: the spec states the settings
are independent and the user is neither blocked nor warned.

## Write sequence and failure modes

```
  save
   │
   ├─► CreateTask ──── fails ──► no plan call at all           (FR-006)
   │                             report: task not created
   │
   └─► ok (task id)
        │
        ├─ planDay == ""  ──► done                             (FR-005 default path)
        │
        └─► AddPlanTask(day, id, no start_minute, dur 0)
             ├─ ok    ──► report success; refresh tree + that day's plan  (FR-008)
             └─ fails ──► task STAYS created                   (FR-007)
                          report: created but not planned
```

The partial-failure branch is the only genuinely new behavior in the model, and it is deliberately
not a rollback: deleting a task the user successfully created, because a *secondary* convenience
failed, would destroy their work to preserve a tidy invariant. The task is kept and the shortfall is
reported.
