# Implementation Plan: Hide Completed Tasks

**Branch**: `026-hide-completed-tasks` | **Date**: 2026-05-31 | **Spec**: [spec.md](./spec.md)

**Input**: Feature specification from `/specs/026-hide-completed-tasks/spec.md`

## Summary

Add a client-side filter to the web app's task tree so completed tasks are hidden by
default, with a single toggle to reveal them on demand. The toggle's state is persisted
per-browser so it survives reloads and return visits. No backend, proto, or database
changes are required — the existing `task.v1.TaskService.ListTasks` already returns every
task with its `completedAt` timestamp, and the filtering happens entirely in the React
client after the tree is built.

## Technical Context

**Language/Version**: TypeScript 5.8 (React 19)

**Primary Dependencies**: React 19, @connectrpc/connect-query + @tanstack/react-query (data fetching), react-router 7, Tailwind CSS v4

**Storage**: Browser `localStorage` for the show/hide preference (durable across sessions). No server-side storage; task data continues to come from PostgreSQL via the existing API.

**Testing**: Vitest + @testing-library/react (unit/interaction). Pure filtering logic unit-tested in `src/lib/`.

**Target Platform**: Modern browsers (web SPA served at the app origin)

**Project Type**: Web application — frontend-only change within `services/twig-web/`. Backend (`services/twig/`) is untouched.

**Performance Goals**: Filtering and toggling are instantaneous (operate on already-fetched, in-memory task list; no network round-trip on toggle).

**Constraints**: Must reuse existing components, theme tokens, and message conventions. No new API endpoints. `SameSite=Strict` session model unchanged.

**Scale/Scope**: Single page affected (`TaskTreePage`), one new pure helper (`filterTree`), one new persisted preference, one new playful empty-state message. Personal task volumes (tens to low hundreds of tasks).

## Constitution Check

*GATE: Must pass before Phase 0 research. Re-check after Phase 1 design.*

| Principle | Status | Notes |
|-----------|--------|-------|
| I. Simplicity / YAGNI | ✅ | Pure client-side filter + one boolean preference. No new abstractions, no backend changes, no generalized "filtering framework." A single `filterTree` helper mirrors the existing `buildTree`/`updatePayload` pattern. |
| II. API-First Design | ✅ | No API change. The existing `task.v1.TaskService` contract is the source of truth and already exposes `completedAt`. Documented in `contracts/` as a "no API change" record plus the client-side UI contract. |
| III. UI/UX Consistency | ✅ | Toggle uses the existing `Button` component and Tailwind theme tokens; placement mirrors the existing header row in `TaskTreePage`. No one-off styles. |
| IV. Playful User Messages | ✅ | New "all tasks done, completed hidden" empty-state copy added to `src/theme/messages.ts` in the established warm/playful tone. |

No violations — Complexity Tracking table omitted.

## Project Structure

### Documentation (this feature)

```text
specs/026-hide-completed-tasks/
├── plan.md              # This file
├── research.md          # Phase 0 output
├── data-model.md        # Phase 1 output
├── quickstart.md        # Phase 1 output
├── contracts/           # Phase 1 output (UI contract + no-API-change record)
│   └── ui-contract.md
└── checklists/
    └── requirements.md  # From /speckit-specify
```

### Source Code (repository root)

All changes are confined to the existing `services/twig-web/` React app:

```text
services/twig-web/src/
├── lib/
│   ├── tree.ts                  # existing — unchanged
│   ├── filterTree.ts            # NEW — pure helper: prune completed tasks from a TaskNode[]
│   └── filterTree.test.ts       # NEW — unit tests for the filtering rules
├── lib/
│   └── showCompletedPref.ts     # NEW — read/write the persisted show/hide preference (localStorage)
├── pages/
│   ├── TaskTreePage.tsx         # MODIFIED — add toggle, wire preference + filtering, all-done empty state
│   └── TaskTreePage.test.tsx    # NEW (or extended) — interaction test for hide/show + persistence
├── theme/
│   └── messages.ts              # MODIFIED — add playful "all done / completed hidden" message
└── components/
    └── (reuse Button; no new component required)
```

**Structure Decision**: Web-application layout, frontend tier only. The pure filtering and
preference logic live in `src/lib/` (consistent with `tree.ts` and `updatePayload.ts`,
which are unit-tested in isolation), and the wiring lives in the affected page
`TaskTreePage.tsx`. The backend Go module is not part of this feature's source footprint.

## Complexity Tracking

> No Constitution Check violations — section intentionally empty.
