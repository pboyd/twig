# Implementation Plan: Web Plan Tab Day-Planner Redesign

**Branch**: `061-web-plan-redesign` | **Date**: 2026-07-02 | **Spec**: [spec.md](./spec.md)

**Input**: Feature specification from `/specs/061-web-plan-redesign/spec.md`

## Summary

Redesign the web app's Plan tab so timed entries render on an hour-ruled day-planner timeline (blocks positioned by start time, height proportional to duration) instead of a flat list, and unify the task done-control with the Tasks tab's circle toggle (two-way complete/re-open). Untimed entries move to a checklist above the timeline, matching the terminal UI. Frontend-only: no backend, proto, or data-model changes; the page keeps its existing ConnectRPC queries/mutations and gains `uncompleteTask` (already used by the Tasks tab).

## Technical Context

**Language/Version**: TypeScript 5.8, React 19

**Primary Dependencies**: Vite 6, Tailwind CSS 4, `@connectrpc/connect-query` 2 + `@tanstack/react-query` 5 (existing generated clients in `src/gen/`), `react-router` 7

**Storage**: N/A — existing server API only; no schema or proto changes

**Testing**: Vitest 3 + `@testing-library/react` (jsdom), colocated `*.test.tsx`/`*.test.ts`

**Target Platform**: Modern evergreen browsers; narrow (phone-width) viewports are a first-class target

**Project Type**: Web SPA frontend (`services/twig-web/`), presentation-layer change to one page

**Performance Goals**: Timeline renders synchronously from already-fetched data; current-time indicator refreshes once per minute (no continuous animation)

**Constraints**: 44px minimum touch targets (`spacing.touchTarget` convention, `h-11` Tailwind class); shared theme tokens and message catalog; dev traffic must stay on the Vite proxy origin

**Scale/Scope**: One page (`PlanPage`), one replaced component (`PlanEntryRow` → timeline block + untimed row), one extracted shared component (`CompletionToggle`), pure layout helpers in `src/lib/planView.ts`

## Constitution Check

*GATE: Must pass before Phase 0 research. Re-check after Phase 1 design.*

| Principle | Status | Notes |
|-----------|--------|-------|
| I. Simplicity / YAGNI | ✅ | Timeline is a CSS grid of 15-minute slot rows plus pure window/snapping math in `planView.ts`. The only new abstraction is `CompletionToggle`, extracted because three surfaces render it (Tasks tree rows, timeline blocks, untimed rows) — deduplication of existing code, not speculation. No drag/resize/creation machinery. |
| II. API-First Design | ✅ | No API changes. The feature consumes existing `plan.v1` and `task.v1` RPCs (adds no new ones); the UI component contract is committed to `contracts/ui-contract.md` before implementation. |
| III. UI/UX Consistency | ✅ | This feature *is* a consistency fix: same done control as the Tasks tab, untimed-above-grid layout matching the TUI, 15-minute slot geometry mirroring the CLI/TUI planner, styling from existing Tailwind conventions and theme tokens. |
| IV. Playful User Messages | ✅ | No new user-facing messages anticipated; existing `messages.planEmpty`, `messages.completeBlockedBySubtasks`, `messages.reopenBlockedByParent`, and `messages.entryRemoved` are reused. Any new copy must go through `src/theme/messages.ts` and tone review. |

## Project Structure

### Documentation (this feature)

```text
specs/061-web-plan-redesign/
├── plan.md              # This file (/speckit-plan command output)
├── research.md          # Phase 0 output (/speckit-plan command)
├── data-model.md        # Phase 1 output (/speckit-plan command)
├── quickstart.md        # Phase 1 output (/speckit-plan command)
├── contracts/
│   └── ui-contract.md   # Phase 1 output (/speckit-plan command)
└── tasks.md             # Phase 2 output (/speckit-tasks command - NOT created by /speckit-plan)
```

### Source Code (repository root)

```text
services/twig-web/
├── src/
│   ├── lib/
│   │   ├── planView.ts             # MODIFIED: add timeline window + slot-geometry helpers
│   │   └── planView.test.ts        # MODIFIED: unit tests for new helpers
│   ├── components/
│   │   ├── CompletionToggle.tsx    # NEW: circle done-toggle extracted from TreeRow
│   │   ├── CompletionToggle.test.tsx # NEW
│   │   ├── PlanTimeline.tsx        # NEW: hour-ruled grid, entry blocks, now indicator
│   │   ├── PlanTimeline.test.tsx   # NEW
│   │   ├── PlanEntryRow.tsx        # MODIFIED: becomes the untimed-row renderer (circle toggle)
│   │   └── TreeRow.tsx             # MODIFIED: use shared CompletionToggle (no visual change)
│   └── pages/
│       ├── PlanPage.tsx            # MODIFIED: untimed section above timeline; uncomplete mutation
│       └── PlanPage.test.tsx       # MODIFIED: layout, toggle, and indicator coverage
```

**Structure Decision**: All changes live in `services/twig-web/src` (existing SPA module). No new packages, no backend edits, no proto regeneration.

## Complexity Tracking

> No Constitution Check violations — table intentionally empty.
