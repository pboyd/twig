# Implementation Plan: Task Parity for the Web App

**Branch**: `073-web-task-parity` | **Date**: 2026-08-02 | **Spec**: [spec.md](spec.md)

**Input**: Feature specification from `/specs/073-web-task-parity/spec.md`

## Summary

Bring four task capabilities that exist in the TUI (and on the server) to the web SPA: delete a task (with cascade confirmation), set/clear a due date, set/clear a snooze date, and link/unlink a task with a goal. No server or proto changes — the SPA calls existing RPCs (`DeleteTask`, `UpdateTask`, `SetTaskGoal`) whose TypeScript stubs are already generated in `src/gen/`. Work is confined to `services/twig-web/`: extend `TaskDetailPage` and `TaskForm`, extend `buildUpdatePayload`, add small tested lib helpers for date⇄Timestamp conversion and effective-goal resolution, and add new copy to `messages.ts`.

## Technical Context

**Language/Version**: TypeScript 5.x, React 19 (frontend only; Go server untouched)

**Primary Dependencies**: @connectrpc/connect-query + @tanstack/react-query (RPC + cache), @bufbuild/protobuf (Timestamp), react-router, Tailwind CSS

**Storage**: N/A (server persistence already exists; no schema change)

**Testing**: Vitest + Testing Library (`npm test` in `services/twig-web/`)

**Target Platform**: Browser SPA served by the twig server / Vite dev server

**Project Type**: Web application (frontend-only change)

**Performance Goals**: Standard SPA interaction expectations; views refresh via query invalidation without manual reload (SC-004)

**Constraints**: `UpdateTask` is full-replace — every editable field must be carried through or it gets cleared (`src/lib/updatePayload.ts`); `snooze_until` is pinned to midnight UTC of the chosen calendar day; `goal_id` is read-only on `UpdateTask` writes — goal links go through `SetTaskGoal` only

**Scale/Scope**: 4 user stories, ~2 pages/2 components touched, 2 new lib modules, no new routes

## Constitution Check

*GATE: Must pass before Phase 0 research. Re-check after Phase 1 design.*

| Principle | Status | Notes |
|-----------|--------|-------|
| I. Simplicity / YAGNI | ✅ | No new abstractions: extend existing form/page/payload helper; two small pure lib modules only where logic needs unit tests (date conversion, effective-goal walk) per existing `src/lib/` convention. |
| II. API-First Design | ✅ | No API changes. The RPC usage contract for each action is committed in `contracts/web-task-actions.md` before implementation; it references the authoritative proto comments and specs/072 goal rules. |
| III. UI/UX Consistency | ✅ | Reuses `Button`, `Field`, `ErrorBanner`, `Toast`, existing card layout and Tailwind tokens; date inputs reuse the pattern already in `TaskForm`'s plan-date picker; snooze marker reuses the existing 💤 convention from `TreeRow`. |
| IV. Playful User Messages | ✅ | All new copy (delete confirmation, goal-rule rejections, empty goal list, success toasts) added to `src/theme/messages.ts` with warm tone; none hardcoded in components. |

Post-design re-check: still ✅ on all four — design added no new abstractions or surfaces beyond the above.

## Project Structure

### Documentation (this feature)

```text
specs/073-web-task-parity/
├── plan.md              # This file
├── research.md          # Phase 0 output
├── data-model.md        # Phase 1 output
├── quickstart.md        # Phase 1 output
├── contracts/
│   └── web-task-actions.md   # RPC usage contract per action
└── tasks.md             # Phase 2 output (/speckit-tasks — NOT created by /speckit-plan)
```

### Source Code (repository root)

```text
services/twig-web/src/
├── components/
│   ├── TaskForm.tsx            # + due-date and snooze-date fields (edit mode)
│   └── TaskForm.test.tsx       # + coverage for new fields
├── pages/
│   ├── TaskDetailPage.tsx      # + due/snooze display, delete flow, goal section
│   └── TaskDetailPage.test.tsx # + coverage for delete/goal/date flows
├── lib/
│   ├── updatePayload.ts        # extended: carries edited due/snoozeUntil
│   ├── updatePayload.test.ts
│   ├── dateFields.ts           # NEW: ISO day string ⇄ protobuf Timestamp (midnight UTC)
│   ├── dateFields.test.ts      # NEW
│   ├── effectiveGoal.ts        # NEW: resolve a task's direct-or-inherited goal from ListTasks
│   └── effectiveGoal.test.ts   # NEW
└── theme/
    └── messages.ts             # + delete/goal/due/snooze copy
```

**Structure Decision**: Frontend-only change inside the existing `services/twig-web/` module. Non-trivial logic goes in `src/lib/` with unit tests per repo convention; components stay presentational. No new routes, no server or proto edits, no new proxy entries needed (task.v1 and goal.v1 are already proxied in `vite.config.ts`).

## Complexity Tracking

No Constitution Check violations — table intentionally empty.
