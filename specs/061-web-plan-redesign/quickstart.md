# Quickstart: Web Plan Tab Day-Planner Redesign

All work happens in `services/twig-web/` (React 19 + TypeScript SPA). No backend changes.

## Setup

```bash
# Backend stack (PostgreSQL + server on :8080)
make dev                      # from repo root, via podman-compose

# Frontend
cd services/twig-web
npm install
npm run dev                   # Vite dev server → http://localhost:5173 (proxies /auth, /plan.v1, /task.v1)
```

Log in with a provisioned user (see CLAUDE.md for `--provision-user`).

## Seeding a day to look at

Use the CLI (or the web Tasks tab's add-to-plan control):

```bash
export TWIG_API_KEY=...        # from provisioning
./twig task add "Find azaleas"
./twig plan add --task 1 --at 08:00 --for 30m   # timed entry
./twig plan add --task 2                         # untimed entry
```

## Verify

1. Open the Plan tab at phone width (DevTools responsive mode, e.g., 390px).
2. Untimed checklist appears **above** an hour-ruled timeline; timed entries are blocks sized by duration; gaps are empty ruled rows.
3. The done control is the same circle as the Tasks tab; tapping toggles complete/re-open both ways.
4. On today only, an accent line marks the current time.

## Tests

```bash
cd services/twig-web
npm test                      # vitest (unit + interaction)
npm run build                 # tsc -b && vite build — must stay clean
```

Key test files: `src/lib/planView.test.ts` (window/slot math), `src/components/PlanTimeline.test.tsx`, `src/components/CompletionToggle.test.tsx`, `src/pages/PlanPage.test.tsx`.
