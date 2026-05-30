# todo-web

Mobile-first React SPA for the todo app. Talks to the existing Go backend (`services/todo/`) via ConnectRPC and session-cookie auth — no backend changes required.

## Quick start

```bash
npm install
npm run gen       # generate TypeScript from proto
npm run dev       # Vite dev server at http://localhost:5173
```

Requires the Go backend running on `:8080` (`make dev` from the repo root).

## Scripts

| Script | What it does |
|--------|-------------|
| `npm run dev` | Vite dev server, proxies `/auth`, `/task.v1`, `/health.v1` → `:8080` |
| `npm run gen` | `buf generate` → typed TS clients in `src/gen/` |
| `npm run build` | Production build to `dist/` |
| `npm test` | Vitest unit + interaction tests |
| `npm run preview` | Preview the production build locally |

## Structure

```
src/
├── gen/          # Generated TS from proto — do not edit by hand
├── lib/          # Pure helpers (tree builder, update-payload builder)
├── theme/        # Design tokens + tone strings
├── components/   # Shared UI components
└── pages/        # LoginPage, TaskTreePage, TaskDetailPage
```

## Key constraints

- **Single origin**: the Vite proxy (dev) and Caddy (production) keep the SPA and API on one origin so the `SameSite=Strict` session cookie is sent. Never call `:8080` directly from the browser.
- **Full replace on edit**: `UpdateTask` requires resending `due` + `parentId` unchanged — `src/lib/updatePayload.ts` handles this.
- **Generated code**: re-run `npm run gen` after any proto change.
