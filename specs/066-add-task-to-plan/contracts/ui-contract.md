# UI Contract: Plan Control on the Create-Task Form

**Feature**: 066-add-task-to-plan | **Date**: 2026-07-17

Per Constitution Principle II, this is the contract implementation must conform to. No **network**
contract changes (see `rpc-usage.md`), so the contract for this feature is the UI surface itself —
the interface it exposes to users, and the component interface it exposes to call sites.

## 1. Shared behavior contract (both surfaces)

| ID | Contract |
|---|---|
| UC-01 | The control offers exactly four choices, in this order: **No plan**, **Today**, **Tomorrow**, **a specific date**. |
| UC-02 | A freshly opened create form shows **No plan**. Always. No persistence, no memory of last use. |
| UC-03 | Choosing a day and saving creates the task, then adds it to that day's plan as an **untimed** entry. |
| UC-04 | Today/Tomorrow resolve against the user's **local date at save time**, not at form-open time. |
| UC-05 | The control appears **only** when creating a task: not when editing a task, not on any goal form. |
| UC-06 | Touching the control makes the form dirty (so discard-confirmation behaves as it already does). |
| UC-07 | The control changes no other field's value. |
| UC-08 | The whole control is operable by keyboard alone. |

## 2. TUI contract (`internal/tui/edit.go`)

### Field placement and rendering

The Plan field sits between Snooze and Goal, rendered by the same `fieldLabel` helper as every other
field (Principle III — no ad-hoc styling):

```
  Estimate: 2
  Snooze until:
► Plan: ‹ Today ›     ←/→ cycle · ctrl+g: summon the calendar
  Goal: ‹ Ship v2 ›
```

| State | Renders as |
|---|---|
| none | `‹ No plan ›` |
| today | `‹ Today ›` |
| tomorrow | `‹ Tomorrow ›` |
| date | `‹ 2026-07-24 ›` |

The hint line follows the existing focused-field convention: shown only when `focusIndex ==
focusPlan`, and replaced by the calendar's own hint line (`enter: pick  esc: never mind  …`) while
the calendar is open — mirroring `writeDateField` exactly.

### Key bindings — all existing, none new

| Key | Behavior on the Plan field | Existing binding |
|---|---|---|
| `←` / `→` | cycle `None → Today → Tomorrow → None` (wrapping); from `date`, re-enter the loop and clear the date | same as the Goal selector |
| `ctrl+g` | open the calendar seeded from the picked date, else today | `keys.Calendar`, extended to `focusPlan` |
| `enter` | advance focus (the field is not a button) | same as other single-line fields |
| `tab` / `shift+tab` | move focus; the field is skipped when `showPlanField` is false | `cycleFocus` |

**This feature introduces no new key binding.** (Principle III.)

### Component interface

```go
// NewChildForm / newBlankForm — creating a task
f.showPlanField = true   // create-task forms only
f.planIdx       = planChoiceNone

// NewEditForm — editing a task, and every goal form
f.showPlanField = false
```

`editSavedMsg.planDay` is the only output: an ISO `YYYY-MM-DD`, or `""` for no plan.

### Save-path contract (`createTaskCmd`)

```go
func createTaskCmd(
    client taskv1connect.TaskServiceClient,
    planClient planv1connect.PlanServiceClient,  // NEW param
    msg editSavedMsg,
) tea.Cmd
```

Ordering guarantee: `CreateTask` → (`SetTaskGoal` if any) → `AddPlanTask` if `msg.planDay != ""`.
The plan call is last, so its failure cannot cost the user the task or the goal association.

## 3. Web contract (`services/twig-web/src/components/TaskForm.tsx`)

### Component interface

```ts
interface TaskFormProps {
  onSubmit: (name: string, description: string, planDay?: string) => Promise<void>;
  onCancel?: () => void;
  loading?: boolean;
  initialName?: string;
  initialDescription?: string;
  submitLabel?: string;
  showPlanControl?: boolean;   // NEW — default false
}
```

`showPlanControl` defaults to **false**, so the edit call site
(`pages/TaskDetailPage.tsx:257`) requires no change and cannot accidentally acquire the control.
`planDay` is `undefined` unless a day is selected.

### Call-site contract

| Call site | `showPlanControl` | `onSubmit` must |
|---|---|---|
| `pages/TaskTreePage.tsx:165` (create root) | `true` | plan after create; invalidate `listTasks` + that day's `listPlanEntries` |
| `components/TreeRow.tsx:194` (create sub-task) | `true` | same |
| `pages/TaskDetailPage.tsx:257` (edit) | omit | unchanged; ignores the third arg |

### Markup and accessibility contract

```
  Add to plan
  ┌─────────┬───────┬──────────┬────┐
  │ No plan │ Today │ Tomorrow │ 📅 │
  └─────────┴───────┴──────────┴────┘
              ^^^^^ selected

  (📅 pressed) →  [ 2026-07-24 ▾ ]
```

| Requirement | Contract |
|---|---|
| Grouping | the segment row is a `radiogroup` with an accessible name ("Add to plan") |
| Selection | exactly one segment carries `aria-checked` / `aria-pressed`; selection is conveyed by more than color alone |
| Keyboard | every segment is tab-reachable and activates on `Enter`/`Space` (UC-08) |
| Date input | `<input type="date">` with an accessible label, matching `AddToPlanControl`'s existing usage |
| Touch targets | min-height 44px, matching the app's existing controls |
| Styling | design tokens / existing `Button` component only — no per-page one-off styles (Principle III) |
| Theming | light and dark, like every existing control |

## 4. User-facing copy contract (Principle IV)

Existing strings are reused verbatim — they already carry the house tone:

| Situation | String | Source |
|---|---|---|
| planned for today | `"On today's plan! Go get it. 🎉"` | `messages.addedToToday` |
| planned for another day | `` `Added to ${label} — marked and ready.` `` | `messages.addedToDay` |
| already on that plan | `"It's already waiting there — no need to add it twice."` | `messages.alreadyOnPlan` |
| server unreachable | `"Couldn't reach the server. Want to try again?"` | `messages.connectivityError` |

**One new string** is required — the partial-failure case (FR-007), which has no existing
equivalent. It must be accurate about the split outcome, actionable, and warm without making light
of the failure:

> Suggested: `"Task saved, but it didn't make it onto the plan. Want to add it from the list?"`

It must state both halves (task kept, plan missed) and point at the recovery path. The TUI needs the
same information in its error line; it may share the wording.

**Tone constraint on labels**: the segment/selector labels stay literal — "No plan", "Today",
"Tomorrow". Principle IV asks for playfulness in messages, not in control labels, where whimsy would
cost clarity ("playfulness MUST NOT obscure meaning").

## 5. Contract tests

Each row is a test that must exist and pass. `[T]` = TUI (Go), `[W]` = web (Vitest).

| # | Contract | Test |
|---|---|---|
| CT-01 | UC-02 | `[T]` new create form has `planIdx == planChoiceNone`; `[W]` a fresh `TaskForm` renders "No plan" selected |
| CT-02 | UC-01 | `[T]` cycling right from none yields Today, Tomorrow, then none; `[W]` all four choices render |
| CT-03 | UC-03 | `[T]` save with Today ⇒ `editSavedMsg.planDay == today`; `[W]` submit with Today ⇒ `onSubmit` 3rd arg is today's ISO |
| CT-04 | UC-03 | `[T]` `AddPlanTask` is called with no `start_minute` and `duration_minute: 0` |
| CT-05 | UC-02 | `[T]`/`[W]` save untouched ⇒ no plan call, `planDay` empty/undefined |
| CT-06 | UC-04 | `[T]` with `nowFunc` pinned across a midnight boundary, Today resolves to the save-time day |
| CT-07 | UC-05 | `[T]` edit form and goal form have `showPlanField == false`; `[W]` `TaskDetailPage`'s form renders no plan control |
| CT-08 | FR-006 | `[T]`/`[W]` `CreateTask` fails ⇒ `AddPlanTask` never called |
| CT-09 | FR-007 | `[T]`/`[W]` `AddPlanTask` fails ⇒ task stays created, partial-failure message shown, form input not discarded |
| CT-10 | UC-06 | `[T]` touching the plan control makes `isDirty()` true |
| CT-11 | FR-008 | `[W]` success invalidates both `listTasks` and that day's `listPlanEntries` |
| CT-12 | UC-01 | `[T]` calendar pick sets `planDate` and renders the ISO date; cycling away clears it |
| CT-13 | UC-07 | `[T]` changing the plan choice leaves due/snooze/estimate/goal untouched in `editSavedMsg` |
| CT-14 | UC-08 | `[W]` the control is reachable and operable by keyboard; segments expose radio semantics |
