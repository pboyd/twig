# Implementation Plan: Goals in the Web App

**Branch**: `058-add-web-goals` | **Date**: 2026-06-22 | **Spec**: [spec.md](./spec.md)

**Input**: Feature specification from `/specs/058-add-web-goals/spec.md`

## Summary

Add a Goals area to the React web app (`services/twig-web/`) that lets a signed-in user **view** their goals (grouped by state, completed/archived hidden by default) and fully **manage status updates** (read latest + history, add, edit, delete). Goals themselves are read-only on the web for this iteration. The web app's goal detail also lists the goal's associated tasks read-only.

The server already exposes the complete `goal.v1.GoalService` (used by the TUI/CLI) and the TypeScript ConnectRPC stubs are already generated in `src/gen/goal/v1/`. **No backend changes.** This is a frontend-only feature that consumes existing contracts, mirroring the established Tasks/Plan page patterns.

## Technical Context

**Language/Version**: TypeScript 5.x, React 19

**Primary Dependencies**: `@connectrpc/connect-query` v2 + `@tanstack/react-query` v5 (data), `react-router` v7 (routing), `react-markdown` + `remark-gfm` (rendering, via existing `<Markdown>`), Tailwind CSS v4 (styling), Vitest + React Testing Library (tests)

**Storage**: None client-side beyond `localStorage` for the show-hidden preference (mirrors `showCompletedPref.ts`). All data lives server-side via `GoalService`.

**Testing**: `npm test` (Vitest) from `services/twig-web/`; component/interaction tests in `*.test.tsx`, lib unit tests in `*.test.ts`, following existing conventions.

**Target Platform**: Modern browsers (SPA served by Vite dev server / static build), single-origin with the server via the dev proxy.

**Project Type**: Web application frontend (consumes existing web service).

**Performance Goals**: Standard SPA responsiveness; list and detail render instantly on cached data, mutations reflect via query invalidation.

**Constraints**: Same-origin only — all RPC traffic flows through the Vite proxy so the `SameSite=Strict` session cookie works; never call `:8080` cross-origin. Backend MUST NOT be modified. Reuse shared components/theme/messages (no per-page one-off styles).

**Scale/Scope**: Tens to low-hundreds of goals per user; tens of status updates per goal. Two new routes, ~2 pages, ~3 small components, 2 lib modules, 1 date helper, message additions, one proxy entry.

## Constitution Check

*GATE: Must pass before Phase 0 research. Re-check after Phase 1 design.*

| Principle | Status | Notes |
|-----------|--------|-------|
| I. Simplicity / YAGNI | ✅ | Reuses existing page/query/mutation patterns (TaskDetailPage), shared components (`Markdown`, `Button`, `ErrorBanner`, `Spinner`, `EmptyState`), and the `showCompletedPref` pattern. No new abstractions or layers. Status display uses a single source of truth (the status-updates list) rather than reconciling two. |
| II. API-First Design | ✅ | Contract already defined and committed: `api/proto/goal/v1/goal.proto` with generated web stubs in `src/gen/goal/v1/`. No new/changed contract. The consumed surface is documented in `contracts/web-goals.md`. |
| III. UI/UX Consistency | ✅ | New Goals nav entry uses the same `AppHeader`/`HeaderMenu` patterns; pages reuse theme tokens, layout shells, and the `<Markdown>` renderer used by Tasks. State grouping/hidden-toggle mirror the TUI and the Tasks "Show all" control. |
| IV. Playful User Messages | ✅ | All new user-facing copy (empty states, validation, errors, confirmations, save/delete toasts) added to `src/theme/messages.ts` in the established warm/playful tone. |

No violations — Complexity Tracking is empty.

## Project Structure

### Documentation (this feature)

```text
specs/058-add-web-goals/
├── plan.md              # This file
├── research.md          # Phase 0 output
├── data-model.md        # Phase 1 output (view models derived from goal.proto)
├── quickstart.md        # Phase 1 output (manual verification steps)
├── contracts/
│   └── web-goals.md     # Phase 1 output: consumed RPCs + web routes/components contract
└── tasks.md             # Phase 2 output (/speckit-tasks — NOT created here)
```

### Source Code (repository root)

All work is under `services/twig-web/` (frontend module). Existing generated stubs (`src/gen/goal/v1/`) are reused as-is.

```text
services/twig-web/
├── vite.config.ts                        # ADD "/goal.v1" proxy entry
├── src/
│   ├── App.tsx                           # ADD routes: /goals, /goals/:id
│   ├── components/
│   │   ├── AppHeader.tsx                 # ADD "Goals" NavLink
│   │   ├── HeaderMenu.tsx                # ADD "Goals" item (mobile menu)
│   │   ├── GoalListItem.tsx              # NEW: a goal row in the grouped list
│   │   ├── StatusUpdateForm.tsx          # NEW: body textarea + save/cancel (add & edit)
│   │   └── StatusUpdateList.tsx          # NEW: history list w/ edit/delete affordances
│   ├── pages/
│   │   ├── GoalsPage.tsx                 # NEW: grouped list + show-hidden toggle + empty state
│   │   └── GoalDetailPage.tsx            # NEW: goal fields (read-only) + status updates + linked tasks
│   ├── lib/
│   │   ├── goalGroups.ts                 # NEW: group/sort goals by state, apply hidden filter (unit-tested)
│   │   ├── showHiddenGoalsPref.ts        # NEW: localStorage pref (mirrors showCompletedPref.ts)
│   │   └── formatTimestamp.ts            # NEW: human-readable date/time + due formatting (unit-tested)
│   └── theme/
│       └── messages.ts                   # ADD goal/status-update copy (Principle IV)
└── (corresponding *.test.tsx / *.test.ts for each new file)
```

**Structure Decision**: Frontend-only change confined to `services/twig-web/`. Two routes mirror the Tasks pattern (`/goals` list + `/goals/:id` detail) rather than a single two-pane screen, because the existing web app already uses route-per-view for Tasks and this keeps mobile/responsive behavior consistent (the TUI's two-pane layout is a TUI-specific convention, not a web one). Pure logic (grouping, formatting, preference) is extracted into `lib/` modules so it is unit-testable without rendering, matching `tree.ts`/`planView.ts`/`showCompletedPref.ts`.

## Complexity Tracking

> No Constitution Check violations. No entries.

| Violation | Why Needed | Simpler Alternative Rejected Because |
|-----------|------------|-------------------------------------|
| — | — | — |
