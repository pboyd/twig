# Quickstart: Hide Completed Tasks

A frontend-only change in `services/twig-web/`. No backend, proto, DB, or migration steps.

## Prerequisites

```bash
# Backend running for live data (optional for unit tests):
make dev                       # from repo root — Postgres + server on :8080

cd services/twig-web
npm install
```

## Where the work happens

| File | Change |
|------|--------|
| `src/lib/filterTree.ts` | NEW — pure `filterTree(nodes, showCompleted)` (see contracts/ui-contract.md). |
| `src/lib/filterTree.test.ts` | NEW — unit tests for every row of the filtering contract. |
| `src/lib/showCompletedPref.ts` | NEW — `readShowCompleted` / `writeShowCompleted` (localStorage, default false). |
| `src/pages/TaskTreePage.tsx` | MODIFIED — add toggle (reuse `Button`), wire preference + `filterTree`, add all-done empty state. |
| `src/pages/TaskTreePage.test.tsx` | NEW/extended — interaction test: hidden-by-default, toggle reveals, persistence, all-done state. |
| `src/theme/messages.ts` | MODIFIED — add playful "all done / completed hidden" message. |

## Develop

```bash
cd services/twig-web
npm run dev        # http://localhost:5173 (proxies API → :8080)
```

Manual smoke test:
1. Create a few tasks; complete some of them.
2. Confirm completed tasks vanish from the default view (FR-001).
3. Click **Show completed** → completed tasks appear, marked done (FR-002, FR-005).
4. Reload the page → completed tasks still shown (FR-007).
5. Click **Hide completed**, then complete the last visible task → list shows the playful
   "all done, completed hidden" message with a way to reveal them, *not* the generic empty
   state (FR-008), and the toggle is still reachable (FR-009).

## Test

```bash
cd services/twig-web
npm test                          # all Vitest suites
npm test -- filterTree            # the pure filtering logic
```

## Build (verify before merge)

```bash
cd services/twig-web
npm run build                     # tsc -b && vite build — must pass clean
```

## Definition of done

- `filterTree` and `showCompletedPref` unit tests pass and cover the contract rows.
- `TaskTreePage` interaction test covers default-hidden, toggle, persistence, and all-done
  empty state.
- `npm run build` succeeds with no type errors.
- New user-facing copy reviewed for playful tone (Constitution Principle IV).
- Toggle styling reuses `Button` + theme tokens (Principle III).
