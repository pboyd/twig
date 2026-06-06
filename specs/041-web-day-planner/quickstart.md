# Quickstart: Web Day Planner (View)

Frontend-only feature. All work happens in `services/twig-web/`. No backend, proto, or codegen changes.

## Prerequisites

- Backend running (`make dev` from repo root) — provides `plan.v1.PlanService` on `:8080`.
- A user provisioned and signed in via the web app.
- A plan with a few entries for today, created from the TUI/CLI (e.g. `twig plan task <id> 9:00am 1h`, `twig plan event "Lunch" 12:00pm`, plus an untimed `twig plan task <id>`), so there's something to view.

## Dev loop

```bash
cd services/twig-web
npm install          # if not already
npm run dev          # http://localhost:5173 (proxies /plan.v1 → :8080 after the config edit)
npm test             # Vitest — unit + interaction tests
npm run build        # type-check + production build
```

> No `npm run gen` needed — `plan.v1` stubs already exist in `src/gen/plan/v1/`.

## What you'll build (see plan.md for the file map)

1. **`vite.config.ts`** — add `"/plan.v1": "http://localhost:8080"` to `server.proxy`.
2. **`src/lib/planView.ts`** (+ test) — pure helpers: `todayString()`, `addDays()`, `isToday()`, `formatDayLabel()`, `formatMinute()`, `formatTimeRange()`, `resolveEntries()`, `groupPlan()`.
3. **`src/theme/messages.ts`** — add `planEmpty` and `planError` copy (warm/playful tone).
4. **`src/components/PlanEntryRow.tsx`** — one entry row: time label, name, event/task styling, completed marker; task rows are links to `/tasks/:taskId`, event rows are plain.
5. **`src/pages/PlanPage.tsx`** (+ test) — day state (default today), prev/next/today controls, `listPlanEntries` + `listTasks` queries, timed & untimed sections, loading/error/empty states.
6. **`src/components/AppHeader.tsx`** — add Tasks / Plan navigation.
7. **`src/App.tsx`** — add `<Route path="/plan" element={<PlanPage />} />`.

## Manual verification (maps to acceptance scenarios)

1. Sign in on a mobile-sized viewport → open **Plan** from the header. Today's timed entries appear in time order with start–end labels. *(US1 AS1–AS2)*
2. A task-linked entry shows the task's name; an event ("Lunch") is visually distinct. *(US1 AS3–AS4, FR-007)*
3. A completed task-linked entry is marked done. *(US2 AS3, FR-008)*
4. An untimed entry appears in a separate "untimed" section, not in the timeline. *(US2 AS2, FR-009)*
5. A gap between two timed entries is apparent from their start/end times. *(US2 AS5, FR-010)*
6. Tap a task-linked entry → lands on that task's detail page; tapping an event does nothing. *(US2 AS4–AS5, FR-011, FR-021)*
7. Step to next/previous day; "Today" returns to today; a day with no plan shows the empty state naming the date. *(US3, FR-012, FR-018)*
8. Confirm there is **no** way to add/edit/move/delete entries from this view. *(FR-013, SC-006)*
9. Times read identically regardless of the device timezone (planned wall-clock). *(FR-020)*
10. Kill the server, reload the plan → a retryable error banner appears. *(FR-017)*

## Test focus

- **`planView.test.ts`**: `formatMinute` boundaries (0 → 12:00 am, 720 → 12:00 pm, 1439 → 11:59 pm), `addDays`/month rollover, `isToday`, `resolveEntries` name fallback (override → task name → generic), `groupPlan` timed/untimed split + `isEmpty`.
- **`PlanPage.test.tsx`**: renders timed/untimed sections from a mocked `listPlanEntries`; empty state on no entries; error + retry; day stepping changes the query input; task row navigates, event row is inert.
