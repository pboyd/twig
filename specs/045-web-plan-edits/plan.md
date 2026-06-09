# Implementation Plan: Basic Planning Edits on the Web App

**Branch**: `045-web-plan-edits` | **Date**: 2026-06-09 | **Spec**: [spec.md](./spec.md)

**Input**: Feature specification from `/specs/045-web-plan-edits/spec.md`

## Summary

Bring three planning edits to the React web app, which is currently a read-only companion to the TUI:

1. **Add a task to a plan from the tasks list** — a one-tap "add to today" on each task row, plus a day-picker (Today / Tomorrow / arbitrary date) for other days. Added items are untimed entries.
2. **Complete a task from the day planner** — a per-entry complete control on task-linked entries, reusing the same completion flow the tasks tab already has.
3. **Remove a plan entry** — a per-entry remove control on the planner, for any entry kind.

All required server operations already exist on the wire: `plan.v1.PlanService.AddPlanTask` (omit `start_minute` for an untimed entry), `RemovePlanEntry`, and `task.v1.TaskService.CompleteTask`. **No proto/server/DB changes.** The server already rejects a duplicate untimed entry for the same (task, day) with a `FailedPrecondition` and a playful message — the web app surfaces that rejection rather than implementing its own membership check.

The only genuinely new UI primitive is a small **toast** for transient success confirmation (FR-005); everything else reuses existing components (`Button`, `Field`, `ErrorBanner`), the design tokens, and the established connect-query mutation + `invalidateQueries` pattern.

## Technical Context

**Language/Version**: TypeScript 5.x, React 19 (web SPA). No Go changes.

**Primary Dependencies**: Vite, React Router, `@connectrpc/connect-query` + `@tanstack/react-query` (data fetching/mutations), Tailwind. Vitest + Testing Library for tests. No new dependencies.

**Storage**: PostgreSQL via the existing server — **not touched**.

**Testing**: `npm test` (Vitest) in `services/twig-web/`. Component/interaction tests with Testing Library; unit tests for any new pure helper.

**Target Platform**: Modern mobile/desktop browser (the SPA is mobile-first; primary use is a phone).

**Project Type**: Web frontend of an existing multi-module full-stack app. Brownfield change to `services/twig-web/` only.

**Performance Goals**: No change — a day's plan and a user's task list are tens of items; mutations invalidate and refetch one or two queries.

**Constraints**: Same-origin only (dev proxy keeps the `SameSite=Strict` cookie working); generated code in `src/gen/` is not hand-edited. Today's "add" must be a single action (SC-001); future-day add ≤ 3 actions (SC-002).

**Scale/Scope**: Small. One new `AddToPlanControl`, one new `Toast`/`useToast` primitive, edits to `TreeRow`, `PlanPage`, `PlanEntryRow`, and `messages.ts`; plus their tests. No API surface change.

## Constitution Check

*GATE: Must pass before Phase 0 research. Re-check after Phase 1 design.*

| Principle | Status | Notes |
|-----------|--------|-------|
| I. Simplicity / YAGNI | ✅ | Reuses the existing mutation + `invalidateQueries` pattern and existing components. The one new abstraction — a minimal toast — is required by FR-005 (transient success confirmation) and has no existing equivalent; it is the simplest thing that satisfies the requirement, not speculative. Duplicate-prevention needs **zero** client logic (server enforces it). |
| II. API-First Design | ✅ | No contract change. Every operation already exists in `api/proto/plan` and `api/proto/task`. The consumed contracts (`AddPlanTask`, `RemovePlanEntry`, `CompleteTask`, `ListPlanEntries`) and their error codes are documented in `contracts/` before implementation. |
| III. UI/UX Consistency | ✅ | New controls use the shared `Button`/`Field` components and design tokens; the planner's complete control mirrors the tasks-tab completion interaction (same `CompleteTask` call, same `FailedPrecondition` handling and copy). The toast uses the shared token palette. No per-page one-off styling. |
| IV. Playful User Messages | ✅ | All new copy (add/remove confirmations, "already on that day", failure notices) lives in `src/theme/messages.ts` and follows the warm/playful tone. The duplicate message the server returns is already playful and may be surfaced as-is or mapped to a local playful string. |

**Result**: PASS (initial). Re-checked after Phase 1 design — still PASS. No Complexity Tracking entries.

## Project Structure

### Documentation (this feature)

```text
specs/045-web-plan-edits/
├── plan.md              # This file
├── research.md          # Phase 0 output
├── data-model.md        # Phase 1 output (client view models + request shapes)
├── quickstart.md        # Phase 1 output (run + manual test by user story)
├── contracts/           # Phase 1 output (documents the existing consumed APIs)
│   └── consumed-apis.md
├── checklists/
│   └── requirements.md  # From /speckit-specify
└── tasks.md             # Phase 2 output (/speckit-tasks — NOT created here)
```

### Source Code (repository root)

```text
services/twig-web/src/
├── components/
│   ├── Toast.tsx              # NEW: presentational transient toast (uses tokens)
│   ├── AddToPlanControl.tsx   # NEW: per-task "add to today" + day-picker popover
│   ├── TreeRow.tsx            # EDIT: mount AddToPlanControl in the task row actions
│   ├── PlanEntryRow.tsx       # EDIT: add Complete (task-linked) + Remove controls
│   └── AppHeader.tsx          # (unchanged)
├── context/
│   └── ToastProvider.tsx      # NEW: ToastProvider + useToast() hook, mounted in App
├── lib/
│   ├── planDays.ts            # NEW: day-target helpers (today/tomorrow/iso) — unit-tested
│   ├── planDays.test.ts       # NEW
│   └── planView.ts            # (unchanged — grouping/format already in place)
├── pages/
│   ├── PlanPage.tsx           # EDIT: wire remove/complete mutations + per-entry handlers
│   ├── PlanPage.test.tsx      # EDIT: cover complete + remove interactions
│   ├── TaskTreePage.test.tsx  # EDIT: cover add-to-today + add-to-day
│   └── TaskDetailPage.tsx     # (unchanged)
├── theme/
│   └── messages.ts            # EDIT: new playful copy for add/remove/complete outcomes
└── App.tsx                    # EDIT: wrap routes in <ToastProvider>
```

**Structure Decision**: Frontend-only change confined to `services/twig-web/`. New work is two small primitives (`Toast`/`ToastProvider`, `AddToPlanControl`) and one pure helper module (`planDays.ts`), plus edits to the three existing surfaces (`TreeRow`, `PlanPage`, `PlanEntryRow`). The backend modules and `api/` are untouched, consistent with the web app's companion role.

## Complexity Tracking

> No Constitution Check violations. Table intentionally empty.
