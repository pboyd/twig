# Phase 0 Research: Hide Completed Tasks

No `NEEDS CLARIFICATION` markers remained in the spec. The decisions below resolve the
open design choices implied by the spec's assumptions and the existing codebase patterns.

## Decision 1: Filter on the client, not the server

**Decision**: Hide completed tasks entirely in the React client by pruning the already-built
task tree. `ListTasks` continues to return all tasks.

**Rationale**: The web app already fetches the full task list via
`task.v1.TaskService.ListTasks` and builds the tree client-side (`buildTree`). Each `Task`
carries `completedAt`, which is all the filter needs. Adding a server-side filter parameter
would require a proto change, regenerated stubs, handler/query changes, and a migration of
the API contract — violating Principle I (Simplicity) and Principle II's "no backend
modification" stance for the web app (per CLAUDE.md). Toggling is also instant with no
network round-trip.

**Alternatives considered**:
- *Server-side `include_completed` flag on `ListTasks`*: Rejected — needs proto + handler +
  contract changes for a purely presentational concern; slower toggle (refetch on every
  switch); contradicts the web app's "backend is not modified" constraint.

## Decision 2: Prune completed *subtrees*, keep partially-complete branches

**Decision**: A task node is removed from the rendered tree when it is completed
**and** has no visible (incomplete) descendants. Concretely: filter the tree bottom-up —
keep a node if it is incomplete, OR if it has at least one kept descendant.

**Rationale**: The backend enforces that a parent may only be completed once all sub-tasks
are complete (surfaced by `completeBlockedBySubtasks`). Therefore a completed parent always
has a fully-completed subtree, and pruning it (with its descendants) is correct and matches
FR-004's intent. Conversely, an *incomplete* parent with a mix of completed and incomplete
children must stay visible along with its incomplete children, while its completed children
are hidden. The bottom-up "keep if incomplete or has a kept descendant" rule handles both
cases. The defensive "has a kept descendant" clause also avoids accidentally hiding an
incomplete child that happens to sit under a (data-inconsistent) completed parent.

**Alternatives considered**:
- *Flat filter: drop every completed task regardless of children*: Rejected — could orphan
  an incomplete child whose parent is completed, or drop a node needed as a structural
  parent. The bottom-up rule is barely more code and is robust.
- *Hide only top-level completed tasks*: Rejected — does not satisfy FR-004 for completed
  sub-tasks under incomplete parents.

## Decision 3: Persist the preference in `localStorage`

**Decision**: Store the show/hide boolean in `localStorage` under a namespaced key
(e.g. `twig-show-completed`). Absence of the key means hidden (the default).

**Rationale**: US3 describes "reload the page or return later in the same browser" and the
spec's success criteria require the choice to survive reloads. `localStorage` survives tab
close and browser restart, matching "return later." The existing expanded/collapsed state
uses `sessionStorage` because it is transient per-session view state; a view *preference* is
conceptually more durable, so `localStorage` is the better fit while still being per-browser
(not synced across devices, per the spec assumption). Read/write is wrapped in try/catch
exactly like the existing `readExpandedIds`/`writeExpandedIds` helpers so private-mode or
quota errors degrade gracefully to the default.

**Alternatives considered**:
- *`sessionStorage` (mirror expanded-state)*: Rejected — cleared when the tab closes, so it
  fails the "return later" expectation in US3.
- *Server-side per-user preference*: Rejected — requires backend changes; out of scope and
  over-engineered for a single boolean (Principle I). The spec explicitly scopes the
  preference to per-browser.

## Decision 4: Toggle placement and styling

**Decision**: Add a "Show completed" / "Hide completed" toggle in the existing header row of
`TaskTreePage` (the row that currently holds the "Tasks" title and "+ Add task" button),
using the existing `Button` component (`variant="secondary"`) and Tailwind theme tokens.

**Rationale**: Principle III (UI/UX Consistency) requires reuse of the shared component
library and spacing system. The header row is already the page's control strip and is always
rendered regardless of task content, satisfying FR-009 (control remains accessible even when
no tasks are visible).

**Alternatives considered**:
- *A bespoke switch component*: Rejected — introduces a one-off style; the existing `Button`
  conveys the toggle clearly.
- *Per-row "show completed children" controls*: Rejected — adds complexity and a confusing
  mixed model; the spec calls for a single list-level control.

## Decision 5: Distinct empty states

**Decision**: Distinguish three render states on the task list: (a) no tasks exist at all →
existing `EmptyState`; (b) tasks exist but all visible ones are filtered out because they are
completed and hidden → a new playful message plus an obvious way to reveal them (the
always-present toggle, optionally reinforced with an inline "show completed" action); (c)
normal list. State (b) directly satisfies FR-008 and SC-005.

**Rationale**: FR-008 forbids implying the list is empty when completed tasks are merely
hidden. Reusing the generic `EmptyState` ("Nothing here yet…") for case (b) would be
misleading. A dedicated message in `src/theme/messages.ts` keeps tone centralized
(Principle IV) and consistent.

**Alternatives considered**:
- *Reuse the generic empty state*: Rejected — misleads the user into thinking they have no
  tasks (violates FR-008).

## Decision 6: Reactivity on completion changes

**Decision**: Rely on the existing React Query invalidation already wired into the
complete/uncomplete mutations in `TreeRow`. Because filtering is derived from the query data
on each render, a completion change re-runs the filter automatically with no extra plumbing.

**Rationale**: `TreeRow.handleToggleComplete` already invalidates the `listTasks` query key,
triggering a refetch and re-render. Deriving the filtered tree during render means FR-006
("update without manual reload") is satisfied for free.

**Alternatives considered**:
- *Manual local mutation of view state*: Rejected — duplicates state already managed by
  React Query and risks drift.
