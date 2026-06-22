# Quickstart: Goals in the Web App

Manual verification for the Goals web feature. Assumes the full stack is running and you have a provisioned user with some goals (create them via the TUI/CLI if needed).

## Prerequisites

```bash
# From repo root — start server + postgres
make dev

# Frontend dev server (from services/twig-web/)
cd services/twig-web
npm install
npm run dev    # http://localhost:5173
```

Seed data via CLI if you have none:

```bash
twig goal add "Buy a new car" --description "Researching EVs"
twig goal commit <id>        # move one to Committed
twig goal status <id> "Narrowed it down to two EVs; test drive next week."
```

## Run the tests

```bash
cd services/twig-web
npm test          # Vitest — new goalGroups / formatTimestamp / page tests must pass
npm run build     # tsc -b + vite build — must type-check clean
```

## Walkthrough

### 1. Navigation & list (US1)
1. Sign in at `http://localhost:5173`.
2. Click **Goals** in the header (and verify it appears in the mobile menu at a narrow width).
3. Confirm goals are grouped: **Committed** then **Incubating**. Completed/Archived are hidden.
4. Click **Show all** → Completed and Archived groups appear; click again → they hide. Reload the page → your choice persisted.
5. With no goals (or none visible), confirm a friendly empty state shows instead of a blank screen.

### 2. Goal detail & reading status (US1, US2)
1. Click a goal → `/goals/:id`.
2. Confirm name, description (markdown rendered), due date, and state are shown. No edit/delete/state controls on the goal itself.
3. Confirm the **latest status update** shows with a human-readable timestamp, markdown rendered.
4. Open the **history** → all updates newest-first, each timestamped. A long update wraps/scrolls fully (no truncation).
5. A goal with no updates shows the "no status updates yet" message.
6. If the goal has associated tasks, confirm they list (read-only) and link to `/tasks/:id`. No link/unlink controls.

### 3. Write a status update (US3)
1. On a goal, add a status update with some markdown → Save.
2. It becomes the latest status with the current timestamp and appears at the top of history.
3. Try saving an empty/whitespace update → rejected with a clear message; nothing recorded.

### 4. Edit & delete (US4)
1. Edit an existing update's text → Save → new text renders; timestamp unchanged.
2. Edit to empty → rejected; original retained.
3. Delete an update (confirm prompt) → it disappears; others remain.
4. Delete the only update → detail returns to "no status updates yet".

### 5. Cross-interface consistency (FR-018)
1. Add an update on the web; open the same goal in the TUI (`twig`, Goals tab) → the update is there.
2. Add/edit an update in the TUI; refresh the web detail → it reflects the change.

## Error paths
- Stop the server, then load `/goals` or save an update → a clear, retry-friendly error (no ambiguous success).
- Visit `/goals/<nonexistent-id>` → friendly not-found with a link back to `/goals`.

## Done when
- All acceptance scenarios in `spec.md` pass by inspection here, `npm test` is green, and `npm run build` type-checks clean.
