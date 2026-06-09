# Consumed API Contracts: Basic Planning Edits on the Web App

Per Constitution Principle II, the data contracts this feature depends on are documented and reviewed before implementation. **This feature introduces NO new or changed contracts** — every operation already exists in `api/proto/plan/v1/plan.proto` and `api/proto/task/v1/task.proto` and is already used by the TUI/CLI. The web app consumes them via the generated connect-query hooks in `services/twig-web/src/gen/`.

## 1. `plan.v1.PlanService.AddPlanTask`

Appends an entry linked to a task. **Used for adding an untimed task to a day's plan from the tasks list.**

**Request** `AddPlanTaskRequest`:
| Field | Value this feature sends |
|-------|--------------------------|
| `day` | target day, `YYYY-MM-DD` (today, tomorrow, or any picked date) |
| `task_id` | the task being added (int64) |
| `start_minute` | **omitted** → server creates an **untimed** entry |
| `duration_minute` | `0` → server chooses its default |

**Response** `AddPlanTaskResponse { PlanEntry entry }` — the created untimed entry (server-allocated per-day `id`).

**Errors consumed**:
- `FailedPrecondition` — a second **untimed** entry for the same (task, day) is rejected. Message (server-authored, playful): *"That one's already parked here without a time — it can only wait in one spot."* The web app maps this code to its "already on that day" confirmation (FR-006). This is the **only** `FailedPrecondition` path for an untimed add, so the mapping is unambiguous.
- Transport/other → generic failure copy (FR-014).

## 2. `plan.v1.PlanService.RemovePlanEntry`

Deletes the entry identified by `(day, id)`. **Used for removing a plan entry from the day planner** (any kind — untimed/timed/event, FR-008/FR-010).

**Request** `RemovePlanEntryRequest { string day; int32 id }`.

**Response** `RemovePlanEntryResponse {}` (empty).

**Behavior relied upon**: removal affects only the plan entry; any linked `Task` is untouched (FR-009).

## 3. `task.v1.TaskService.CompleteTask`

Marks the identified task complete. **Used to complete a task-linked entry directly from the planner** (FR-011).

**Request** `CompleteTaskRequest { int64 id }` — the entry's `task_id`.

**Response** `CompleteTaskResponse` — the task row (idempotent if already complete).

**Errors consumed**:
- `FailedPrecondition` — the task has at least one incomplete descendant. Mapped to the existing `messages.completeBlockedBySubtasks` copy and shown inline on the entry (FR-013); the task stays incomplete.
- Transport/other → generic failure copy (FR-015).

**Not offered**: standalone event entries (`task_id == 0`) expose no complete control (FR-012).

## 4. `plan.v1.PlanService.ListPlanEntries` (read; already in use)

Returns every entry for a day, including the read-only `completed` flag. **Used to render the planner and to re-read state after a mutation** (invalidate + refetch, FR-014). Query key is day-scoped: `createConnectQueryKey({ schema: listPlanEntries, input: { day }, cardinality: "finite" })`.

---

## Contract stability statement

No `.proto` files change. No `make proto` / `npm run gen` regeneration is required. If a future iteration needs an explicit "is task on day" probe or a bulk edit, that would be a new contract added here first — out of scope for this feature.
