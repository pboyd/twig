# Phase 0 Research: Goals in the Web App

All "unknowns" here are decisions about *how* to consume an already-stable contract using already-established web-app patterns. There are no open NEEDS CLARIFICATION items.

## Decision 1 — Backend is reused unchanged; consume the existing GoalService

- **Decision**: Use the existing `goal.v1.GoalService` and its already-generated web stubs (`src/gen/goal/v1/goal-GoalService_connectquery.ts`, `goal_pb.ts`). No proto edits, no `npm run gen`, no server changes.
- **Rationale**: The full surface this feature needs already exists and ships in the TUI/CLI: `ListGoals`, `GetGoal`, `ListGoalStatusUpdates`, `AddGoalStatusUpdate`, `UpdateGoalStatusUpdate`, `DeleteGoalStatusUpdate`. The web-app convention (CLAUDE.md) is explicitly "the backend is not modified."
- **Alternatives considered**: Regenerating stubs — unnecessary, the proto is unchanged and stubs are present. Adding a web-specific aggregation endpoint — rejected (backend change + violates Simplicity).

## Decision 2 — Dev proxy must include `/goal.v1`

- **Decision**: Add `"/goal.v1": "http://localhost:8080"` to `vite.config.ts`.
- **Rationale**: `vite.config.ts` currently proxies `/auth`, `/task.v1`, `/health.v1`, `/plan.v1`, `/cli`, `/account.v1` — but **not** `/goal.v1`. Without it, ConnectRPC calls to the goal service would hit the SPA origin and 404 (and break the `SameSite=Strict` cookie model). This is the one config change that makes the whole feature reachable in dev.
- **Alternatives considered**: Cross-origin calls to `:8080` — explicitly prohibited by CLAUDE.md (breaks the session cookie).

## Decision 3 — Route-per-view (`/goals`, `/goals/:id`) rather than two-pane

- **Decision**: A list route `/goals` and a detail route `/goals/:id`, registered in `App.tsx`, mirroring `/tasks` and `/tasks/:id`.
- **Rationale**: The web app already uses route-per-view for Tasks (list + `:id` detail) and the `AppHeader`/back-button navigation idiom. Matching it satisfies Principle III (consistency) and keeps responsive/mobile behavior identical. The TUI's two-pane layout is a TUI convention, not a web one.
- **Alternatives considered**: Single-page master/detail (two-pane) — adds layout complexity and diverges from the established web pattern for no user benefit.

## Decision 4 — Status display has a single source of truth: the updates list

- **Decision**: The detail page fetches `ListGoalStatusUpdates(goal_id)` and treats `updates[0]` as the "latest status"; the same list renders the history. `Goal.latest_status_update` from `GetGoal` is ignored for rendering.
- **Rationale**: `Goal.latest_status_update` and `ListGoalStatusUpdates` overlap. Driving both "latest" and "history" from one query avoids reconciling two caches after add/edit/delete and makes invalidation trivial (invalidate one list). Newest-first ordering is guaranteed by the server contract.
- **Alternatives considered**: Use `goal.latestStatusUpdate` for the latest and the list for history — two sources to keep in sync after mutations; more invalidation surface, more bugs.

## Decision 5 — Goal state grouping & hidden filter live in a pure `lib/goalGroups.ts`

- **Decision**: A pure function takes the flat `ListGoals` result + a `showHidden` flag and returns ordered groups: Committed, Incubating, then (only when `showHidden`) Completed, Archived — each preserving server `position` order.
- **Rationale**: Mirrors the existing pattern of extracting pure view logic (`tree.ts`, `planView.ts`, `filterTree.ts`) so it is unit-tested without rendering. Grouping order matches the TUI (FR-010 of feature 049): Committed first, then Incubating.
- **Alternatives considered**: Inline grouping in the page component — harder to unit test, repeats logic.

## Decision 6 — Show-hidden preference persists like the Tasks toggle

- **Decision**: New `lib/showHiddenGoalsPref.ts` with `readShowHiddenGoals()` / `writeShowHiddenGoals()` backed by `localStorage` key `twig-show-hidden-goals`, mirroring `showCompletedPref.ts`.
- **Rationale**: Same UX as the Tasks "Show all" toggle and same persistence story. A separate key avoids coupling goal visibility to task visibility (different semantics: goals hide Completed **and** Archived).
- **Alternatives considered**: Reuse `twig-show-completed` — wrong semantics and surprising cross-surface coupling. Component-only state (no persistence) — inconsistent with Tasks.

## Decision 7 — Associated tasks shown read-only from the task list

- **Decision**: On the detail page, query `listTasks({})` and show tasks whose `goalId === goal.id` (the association roots). Render names with `<Markdown mode="inline">`, link each to `/tasks/:id`. No link/unlink controls.
- **Rationale**: Tasks already carry an optional `goal_id` (set on the association root; descendants inherit). Filtering the existing task list is the simplest read-only display and reuses the cached `listTasks` query already used elsewhere. Showing association roots is sufficient context; the whole subtree belongs to the goal by definition.
- **Alternatives considered**: Expanding to render full subtrees inline — more rendering logic than "context" warrants (Simplicity); a user can click through to the task to see its subtree.

## Decision 8 — Status update form is a small dedicated component

- **Decision**: `StatusUpdateForm` = one `<textarea>` (body) + Save/Cancel, used for both add and edit (seeded with existing text when editing). Client-side guard rejects empty/whitespace bodies before mutating (FR-014), surfacing a playful validation message; the server also enforces.
- **Rationale**: `TaskForm` is name+description specific; status updates are a single markdown body. A focused component is simpler than overloading `TaskForm`. Reuses `Field`/`Button` primitives and the toast/error patterns.
- **Alternatives considered**: Reusing `TaskForm` — wrong field shape; bending it would add props/branches (Simplicity violation).

## Decision 9 — Timestamps and due dates: `lib/formatTimestamp.ts`

- **Decision**: A small helper formats protobuf `Timestamp` (`google.protobuf.Timestamp`, accessed via the generated message) into human-readable absolute date/time using `Intl.DateTimeFormat`; a companion formats a goal's due date (date-only). Unit-tested with fixed inputs.
- **Rationale**: FR-009/FR-011 require human-readable timestamps; FR-005 shows due date. No date lib is present in `package.json`, so native `Intl` keeps the dependency surface flat (Simplicity).
- **Alternatives considered**: Adding `date-fns`/`dayjs` — unnecessary dependency for two format calls.

## Decision 10 — Mutations invalidate the relevant queries (no optimistic cache surgery)

- **Decision**: After add/edit/delete, `invalidateQueries` for `listGoalStatusUpdates({ goalId })` (drives latest + history) and `getGoal({ id })` and `listGoals({})` (latest-status preview on rows, if shown), using `createConnectQueryKey`, exactly as `TaskDetailPage` does.
- **Rationale**: Matches the proven pattern in the codebase; correctness over micro-optimization (Simplicity). Re-fetches are cheap at this scale.
- **Alternatives considered**: Manual optimistic cache writes — more code and edge cases for negligible perceived-latency gains.

## Decision 11 — Testing approach

- **Decision**: Vitest + React Testing Library, mirroring existing `*.test.tsx` (mock the connect-query transport) and `*.test.ts` (pure libs). Cover: grouping/hidden-filter (`goalGroups`), timestamp formatting, empty states, add/edit/delete flows incl. empty-body rejection, not-found/error rendering.
- **Rationale**: Matches existing test conventions; no DB/integration infra needed.
- **Alternatives considered**: E2E (Playwright) — not part of the current stack; out of scope.
