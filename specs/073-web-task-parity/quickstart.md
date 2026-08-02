# Quickstart: Task Parity for the Web App

## Prerequisites

```bash
make dev                          # postgres + server on :8080 (podman-compose)
cd services/twig-web
npm install
npm run dev                       # SPA on http://localhost:5173 (proxies to :8080)
npm test                          # Vitest unit/interaction tests
```

Log in at `http://localhost:5173` with a provisioned user (`./twig-server --provision-user name:password`).

## Manual verification walkthrough

1. **Delete (US1)**: Create a task with a subtask (use the + button on a tree row). Open the parent's detail page → Delete → confirmation mentions subtasks → confirm. You land on `/tasks`; parent and child are gone. Repeat with cancel: nothing changes.
2. **Due date (US2)**: Open a task → Edit → set a due date → Save. The detail card shows the date; reload the page and it's still there. Edit again, clear the date → gone. Name/description untouched throughout.
3. **Snooze (US3)**: Edit a task, set "Snooze until" to tomorrow → Save. Detail card shows "Snoozed until …"; the task tree shows 💤 (under "reveal hidden" if filtered). Clear it → marker disappears.
4. **Goal link (US4)**: Create a goal on the Goals page. On a root task's detail page, pick the goal → task shows the goal; the goal's detail page lists the task. Switch to another goal, then unlink. On a *subtask* of a goal-linked task: the goal shows as inherited with no controls. Try linking a parent whose child has its own goal → friendly rule explanation, nothing changes.

## Key files

- `services/twig-web/src/pages/TaskDetailPage.tsx` — delete flow, goal section, due/snooze display
- `services/twig-web/src/components/TaskForm.tsx` — due/snooze fields (edit mode)
- `services/twig-web/src/lib/updatePayload.ts` — full-replace payload with edited dates
- `services/twig-web/src/lib/dateFields.ts` — ISO day ⇄ Timestamp (midnight UTC)
- `services/twig-web/src/lib/effectiveGoal.ts` — direct vs inherited goal resolution
- `services/twig-web/src/theme/messages.ts` — all new copy
- Contract: `specs/073-web-task-parity/contracts/web-task-actions.md`
