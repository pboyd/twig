# Contract: Goals in the Web App

This feature introduces **no new or changed API contract**. The canonical contract is the existing, committed proto:

- `api/proto/goal/v1/goal.proto` → generated web stubs `services/twig-web/src/gen/goal/v1/`
- `api/proto/task/v1/task.proto` → generated web stubs `services/twig-web/src/gen/task/v1/`

Per the project constitution (Principle II), this document records the **consumed surface** and the **web-facing UI contract** (routes, components, query/mutation behavior) that the implementation must conform to.

## 1. Consumed RPCs (existing `goal.v1.GoalService`)

| RPC | Used for | Request | Response (web-relevant) |
|-----|----------|---------|--------------------------|
| `ListGoals` | List view grouping | `{}` | `{ goals: Goal[] }` ordered by state then position |
| `GetGoal` | Detail header fields | `{ id }` | `{ goal: Goal }`; `NotFound` when missing/not owned |
| `ListGoalStatusUpdates` | Latest + history | `{ goalId }` | `{ updates: StatusUpdate[] }` newest-first |
| `AddGoalStatusUpdate` | Record new update | `{ goalId, body }` | `{ update: StatusUpdate }`; `InvalidArgument` on empty body |
| `UpdateGoalStatusUpdate` | Edit update text | `{ id, body }` | `{ update: StatusUpdate }`; `createdAt` preserved |
| `DeleteGoalStatusUpdate` | Remove update | `{ id }` | `{}` |

**Not used by this feature** (goals are read-only on web): `CreateGoal`, `UpdateGoal`, `SetGoalState`, `ReorderGoal`, `DeleteGoal`.

From `task.v1.TaskService`: `ListTasks` (existing, already consumed) is read for associated-task display via `goalId` filtering. No task mutations.

## 2. Error → message mapping (Principle IV copy in `messages.ts`)

| Condition | ConnectRPC code | UI behavior |
|-----------|-----------------|-------------|
| Goal not found / not owned | `NotFound` (GetGoal) | Show "goal wandered off" not-found state with a link back to `/goals`. |
| Empty/whitespace status body | guarded client-side; `InvalidArgument` as backstop | Inline validation message; no RPC sent on client guard. |
| Network/server failure | any other | `ErrorBanner` with retry-friendly connectivity message. |
| Add/edit/delete success | OK | Toast confirmation (playful). |

## 3. Web route contract (`App.tsx`)

| Route | Component | Purpose |
|-------|-----------|---------|
| `/goals` | `GoalsPage` | Goals grouped by state; show-hidden toggle; empty state. |
| `/goals/:id` | `GoalDetailPage` | Read-only goal fields; status latest + history; add/edit/delete updates; associated tasks (read-only). |

Navigation: `AppHeader` gains a "Goals" `NavLink` (and `HeaderMenu` gains the mobile entry), styled with the existing `navLinkClass`.

## 4. Dev proxy contract (`vite.config.ts`)

Add `"/goal.v1": "http://localhost:8080"` so goal-service RPCs stay same-origin (required for the `SameSite=Strict` session cookie). All other proxying is unchanged.

## 5. Component contracts (props)

### `GoalsPage` (page)
- Reads `listGoals({})`, `readShowHiddenGoals()`.
- Renders `goalGroups(goals, showHidden)`; per group a heading + `GoalListItem` rows.
- Toggle button flips + persists the preference (`writeShowHiddenGoals`).
- Empty state when no visible goals (playful copy); when goals exist only in hidden groups, offers a "show all" affordance (mirrors Tasks).

### `GoalListItem`
```
{ goal: Goal; onOpen: (id: bigint) => void }
```
- Shows name (inline markdown), optional due date, state-appropriate styling. Clicking navigates to `/goals/:id`. No mutation controls.

### `GoalDetailPage` (page)
- `useParams` → `id`; `getGoal({ id })`, `listGoalStatusUpdates({ goalId: id })`, `listTasks({})`.
- Renders: back button, goal header (name inline-md, description block-md, due, state), "latest status" (= `updates[0]`) or empty-state, status history via `StatusUpdateList`, associated tasks (filter `goalId === id`) as links, and an "add update" affordance using `StatusUpdateForm`.
- Handles `NotFound` and generic error states.

### `StatusUpdateForm`
```
{ initialBody?: string; submitLabel: string; loading?: boolean;
  onSubmit: (body: string) => void; onCancel: () => void }
```
- One textarea + Save/Cancel. Trims and blocks empty submit with validation message. Used for both add (empty initial) and edit (seeded).

### `StatusUpdateList`
```
{ updates: StatusUpdate[]; onEdit: (u: StatusUpdate) => void; onDelete: (id: bigint) => void }
```
- Newest-first list; each item shows human-readable `createdAt` + body (block markdown), with Edit and Delete affordances. Delete asks for confirmation (warm but measured, per Principle IV). Long bodies wrap/scroll, never truncated.

## 6. Behavioral guarantees the implementation must meet

- Completed/Archived goals are hidden until the toggle is on (FR-003).
- Group order: Committed, Incubating, [Completed, Archived] (FR-002/049-FR-010).
- Status updates always newest-first; latest = first (FR-011, FR-009).
- Empty/whitespace updates rejected on add and edit (FR-014).
- Edits preserve original timestamp; deletes remove only the targeted update (FR-015/FR-016).
- All goal-mutation controls are absent (FR-019); associated tasks are display-only (FR-008).
- Changes are reflected across interfaces because they write through the shared service (FR-018).
