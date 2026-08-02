# Research: Task Parity for the Web App

No NEEDS CLARIFICATION markers existed in the Technical Context; research resolved the design choices below by reading the existing frontend, the task proto, and the specs/072 goal-rules contract.

## R1. Delete confirmation UX

**Decision**: Inline confirmation panel on `TaskDetailPage` — pressing "Delete" swaps the action row for a warm warning (naming that subtasks go too, when they exist) with confirm/cancel `Button`s. On success: invalidate `listTasks`, navigate to `/tasks`, show a success toast. On `NotFound` (already gone elsewhere): show the friendly "already gone" message and navigate/refresh anyway.

**Rationale**: The SPA has no modal component and Principle I forbids introducing one for a single use; `window.confirm` can't carry the playful tone Principle IV requires and looks foreign to the UI (Principle III). An inline panel reuses existing `Button` + card styling. The existing status-update delete (GoalDetailPage) has no confirm step, but task deletion cascades to descendants, which the spec (FR-002) says must be confirmed.

**Alternatives considered**: `window.confirm` (rejected: tone/consistency); a new shared `ConfirmDialog` component (rejected: YAGNI — one call site).

## R2. Due date and snooze editing surface

**Decision**: Add two optional date fields ("Due" and "Snooze until") to `TaskForm`, shown only when a new `showScheduleFields` prop is set (used by TaskDetailPage's edit mode). `onSubmit` gains the two values as ISO day strings (`""` = clear). `buildUpdatePayload` gains parameters for the edited due/snooze Timestamps while continuing to carry `parentId` untouched.

**Rationale**: Editing already happens through `TaskForm`; adding fields there keeps one form component (Principle III) instead of a second edit form. Gating by prop keeps the create-task flow unchanged (spec scopes due/snooze to editing). `UpdateTask` is full-replace, so the payload builder is the single safe place to merge edited fields with preserved ones — the existing stale-parent refetch in `handleSaveEdit` already guards the move/goal hazard from specs/072.

**Alternatives considered**: Separate quick-set controls on the detail view (rejected: more surface area, duplicate validation); editing due/snooze on create too (rejected: out of spec scope, easy to add later).

## R3. Date ⇄ Timestamp conversion semantics

**Decision**: New `src/lib/dateFields.ts` with pure functions converting an `<input type="date">` ISO day string to a protobuf `Timestamp` at **midnight UTC** of that calendar day, and back. Used for both due and snooze. Past dates are allowed (no client validation).

**Rationale**: The proto pins `snooze_until` to midnight UTC of the chosen day and clients compare calendar days; `TreeRow`'s existing snooze check already reads UTC calendar fields. Due dates render via `formatDueDate` (date-only), so midnight UTC keeps round-trips stable. Matching TUI semantics means no new validation (spec assumption: past dates allowed — a past snooze simply isn't snoozed).

**Alternatives considered**: Local-midnight timestamps (rejected: breaks the documented UTC-day snooze contract and makes round-trip display shift by timezone).

## R4. Goal link/unlink surface and rules

**Decision**: A "Goal" section on `TaskDetailPage` (view mode). It resolves the task's effective goal with a new pure helper `effectiveGoal.ts` (walk `parentId` ancestors in the `listTasks` result; nearest ancestor's `goalId` wins — mirrors the proto's "descendants inherit client-side" note):

- **Direct goal**: show goal name (linked to the goal page) + controls to switch or unlink.
- **Inherited goal**: show goal name with an explanation that it's inherited from a parent task; no controls (the server would reject with `FailedPrecondition` anyway — better to prevent than apologize).
- **No goal**: a native `<select>` of goals from `listGoals` (grouped/ordered like `goalGroups`; hidden states excluded); choosing one calls `setTaskGoal`. Empty goal list → friendly empty message instead of a bare control.

Unlink calls `setTaskGoal` with `goalId` unset. `FailedPrecondition` (e.g. a descendant has its own goal, or a race made an ancestor goal appear) maps to specific playful messages; `NotFound` and other errors map to existing patterns. After any change: invalidate `getTask` + `listTasks` (and `listGoals` is untouched — goal objects don't change).

**Rationale**: `SetTaskGoal` is the only legal write path for goal links (`goal_id` is ignored on `UpdateTask` writes), and its two `FailedPrecondition` causes are documented in the proto — FR-008 requires translating both into plain language. Pre-empting the ancestor case in the UI (inherited display, no controls) satisfies scenario 4 with a better experience than a post-hoc error, while the error mapping still covers races. A native select matches Principle I; no goals→ message satisfies scenario 6.

**Alternatives considered**: Goal picker inside the edit form (rejected: goal changes go through a different RPC with different failure modes than the full-replace update; mixing them in one submit conflates two transactions); a searchable combobox component (rejected: YAGNI at current goal counts).

## R5. Where due/snooze become visible

**Decision**: Show due date (via existing `formatDueDate`) and snooze state ("Snoozed until …") on the `TaskDetailPage` view card. Task tree rows already mark snoozed tasks with 💤 (`TreeRow`) — reused as-is, satisfying FR-006 with zero new list-view work.

**Rationale**: FR-006 requires snoozed tasks be distinguishable; the tree already does this, and the detail page is where the newly editable values must be verifiable (US2/US3 acceptance scenarios reload-and-see).

**Alternatives considered**: Adding due-date badges to tree rows (rejected: not required by spec; separate visual-design question).

## R6. Cache invalidation / no-reload updates

**Decision**: Follow the existing pattern: `queryClient.invalidateQueries` with `createConnectQueryKey` for `getTask` and `listTasks` after every mutation; deletion also navigates to `/tasks`. Errors surface via `ErrorBanner`/toast per existing conventions.

**Rationale**: This is the established mechanism in `TaskDetailPage`/`TreeRow` and satisfies FR-010/SC-004 without new machinery.
