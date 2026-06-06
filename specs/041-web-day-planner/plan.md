# Implementation Plan: Web Day Planner (View)

**Branch**: `041-web-day-planner` | **Date**: 2026-06-06 | **Spec**: [spec.md](./spec.md)

**Input**: Feature specification from `/specs/041-web-day-planner/spec.md`

## Summary

Add a read-only day planner view to the React web app so a user can check the daily plan they built in the TUI from a mobile browser. The view shows one day's plan — timed entries in chronological order with their wall-clock times, untimed entries grouped separately, task-linked entries distinguished from events and marked when complete — and lets the user step to adjacent days and back to today. Task-linked entries link through to the existing task detail page; event entries are display-only.

**Technical approach**: Frontend-only. The `plan.v1.PlanService.ListPlanEntries` RPC already exists, is registered behind the auth middleware on the server, and its TypeScript/connect-query stubs are already generated in `src/gen/plan/v1/`. No backend, proto, or codegen changes are required. Work is: a new route/page, a small pure helper module (date math + minute→time formatting + timed/untimed grouping), navigation between the task tree and the planner, a Vite dev-proxy entry for `/plan.v1`, and playful copy for the empty/error states.

## Technical Context

**Language/Version**: TypeScript 5.x, React 19

**Primary Dependencies**: `@connectrpc/connect-web`, `@connectrpc/connect-query`, `@tanstack/react-query`, `react-router` v7, Tailwind CSS v4 (all already in `services/twig-web`)

**Storage**: N/A (read-only client; data served by existing PostgreSQL-backed `PlanService`)

**Testing**: Vitest + Testing Library (`npm test` in `services/twig-web`)

**Target Platform**: Mobile-first browsers (primary), desktop browsers (secondary)

**Project Type**: Web frontend (SPA) consuming an existing ConnectRPC backend

**Performance Goals**: Standard SPA responsiveness; a single `ListPlanEntries` call per day view plus the shared `ListTasks` query (cached by React Query). No special targets.

**Constraints**: Backend MUST NOT be modified (per CLAUDE.md and constitution). Same-origin only — calls go through the Vite proxy / shared origin so the `SameSite=Strict` session cookie works. View is strictly read-only.

**Scale/Scope**: One day's plan (≤ a few dozen entries). One new page, one helper module, minor edits to `App.tsx`, `AppHeader.tsx`, `vite.config.ts`, `messages.ts`.

## Constitution Check

*GATE: Must pass before Phase 0 research. Re-check after Phase 1 design.*

| Principle | Status | Notes |
|-----------|--------|-------|
| I. Simplicity / YAGNI | ✅ | One page + one small pure helper. No new abstractions; reuses existing components (AppHeader, Spinner, ErrorBanner), transport, and query patterns. No backend work. |
| II. API-First Design | ✅ | Contract already defined and committed (`api/proto/plan/v1/plan.proto`, `plan.v1.PlanService.ListPlanEntries`). Consumed contract documented in `contracts/`. No contract changes. |
| III. UI/UX Consistency | ✅ | New page uses the existing component library, Tailwind tokens/spacing, and `AppHeader`. Navigation pattern shared between tasks and planner. No one-off styles. |
| IV. Playful User Messages | ✅ | New empty-state and error copy added to `theme/messages.ts` in the established warm/playful tone; reviewed alongside existing messages. |

**Result**: PASS — no violations, Complexity Tracking not required.

## Project Structure

### Documentation (this feature)

```text
specs/041-web-day-planner/
├── plan.md              # This file
├── research.md          # Phase 0 output
├── data-model.md        # Phase 1 output
├── quickstart.md        # Phase 1 output
├── contracts/           # Phase 1 output (consumed-contract reference)
│   └── plan-service.md
└── tasks.md             # Phase 2 output (/speckit-tasks — not created here)
```

### Source Code (repository root)

```text
services/twig-web/
├── src/
│   ├── App.tsx                       # ADD route: /plan → PlanPage
│   ├── components/
│   │   ├── AppHeader.tsx             # EDIT: add nav between Tasks and Plan
│   │   └── PlanEntryRow.tsx          # NEW: renders one plan entry (timed/untimed, task/event, completed)
│   ├── pages/
│   │   ├── PlanPage.tsx              # NEW: day planner view (date stepping, timed/untimed sections, states)
│   │   └── PlanPage.test.tsx         # NEW: interaction tests
│   ├── lib/
│   │   ├── planView.ts               # NEW: pure helpers (date strings, minute→time, group timed/untimed, gaps)
│   │   └── planView.test.ts          # NEW: unit tests for helpers
│   ├── theme/
│   │   └── messages.ts               # EDIT: add planEmpty / planError copy
│   └── gen/plan/v1/                  # EXISTING generated stubs — no change
└── vite.config.ts                    # EDIT: add "/plan.v1" proxy entry
```

**Structure Decision**: Web frontend only. All changes live under `services/twig-web/`. The CLI, server, proto, and generated code are untouched. New rendering logic that is pure (date/time math, grouping) is extracted into `src/lib/planView.ts` so it is unit-testable in isolation, following the existing `lib/tree.ts` / `lib/updatePayload.ts` convention.

## Complexity Tracking

> No Constitution Check violations — no entries required.
