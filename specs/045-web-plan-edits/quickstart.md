# Quickstart: Basic Planning Edits on the Web App

## Prerequisites

Backend + DB running (from repo root):

```bash
make dev   # builds/starts the server; postgres starts automatically
```

Provision a user if you don't have an API key (see CLAUDE.md). The web app authenticates via the `/auth/*` cookie flow through the dev proxy.

## Run the web app

```bash
cd services/twig-web
npm install            # first time only
npm run dev            # → http://localhost:5173  (proxies /auth, /plan.v1, /task.v1, /health.v1 → :8080)
```

No `npm run gen` needed — this feature changes no proto.

## Test

```bash
cd services/twig-web
npm test               # Vitest: component/interaction + helper unit tests
npm run build          # type-check + production build
```

## Manual verification by user story

Sign in first. Then:

### US1 — Add a task to today's plan (P1)
1. On the **Tasks** tab, tap a task's **add-to-plan** (today) control.
2. Expect a toast confirming it was added to today's plan.
3. Switch to the **Plan** tab (today) → the task appears in the **Untimed** group.
4. *Single-action check*: adding to today took exactly one tap, with no day prompt (SC-001).
5. *Duplicate*: tap add-to-today again for the same task → toast says it's already on today's plan; no second entry appears (FR-006).
6. *Failure*: stop the server, tap add → a failure toast appears, the task list is unchanged (FR-014).

### US2 — Complete a task from the planner (P2)
1. With a task-linked entry on today's plan, tap its **complete** control on the **Plan** tab.
2. Untimed entry → it disappears from the Untimed group (044 rule). Timed entry → it stays, marked done.
3. Confirm on the **Tasks** tab the task is now complete (no separate edit screen was opened — SC-003).
4. *Blocked*: try to complete an entry whose task has incomplete sub-tasks → inline message "finish its sub-tasks first"; task stays incomplete (FR-013).
5. A standalone **event** entry shows **no** complete control (FR-012).

### US3 — Add a task to a future day (P2)
1. On the **Tasks** tab, open a task's **other-day** picker.
2. Choose **Tomorrow** (or pick an arbitrary date), then **Add** (≤ 3 actions — SC-002).
3. Toast names the day it was added to (FR-005).
4. Navigate the **Plan** tab to that day → the task is in the Untimed group.
5. *Past day*: pick a past date → still adds (not blocked — FR-004).

### US4 — Remove a plan entry (P3)
1. On the **Plan** tab, tap an entry's **remove** control (≤ 2 actions — SC-004).
2. The entry disappears from the day's plan.
3. On the **Tasks** tab, the linked task still exists and is unchanged (FR-009).
4. Remove works for untimed entries, timed entries, and events (FR-010).

## What changed (files)

- New: `components/Toast.tsx`, `context/ToastProvider.tsx`, `components/AddToPlanControl.tsx`, `lib/planDays.ts`.
- Edited: `App.tsx` (wrap in `ToastProvider`), `components/TreeRow.tsx` (add control), `pages/PlanPage.tsx` + `components/PlanEntryRow.tsx` (complete/remove), `theme/messages.ts` (new playful copy).
- No backend, proto, or generated-code changes.
