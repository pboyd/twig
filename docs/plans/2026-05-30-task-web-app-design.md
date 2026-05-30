# Task Web App — Technology Design

**Date**: 2026-05-30
**Feature**: [023-task-web-app](../../specs/023-task-web-app/spec.md)
**Status**: Approved

## Goal

A mobile-first React SPA for capturing and managing tasks away from the terminal. Reuses the existing backend, accounts, and session mechanism. Primary target: mobile browsers.

## Technology Choices

| Layer | Choice | Rationale |
|---|---|---|
| Framework | React 19 + TypeScript | Largest ecosystem; first-class `@connectrpc/connect-query` support |
| Build | Vite | Fast dev server with proxy; straightforward static build |
| Styling | Tailwind CSS v4 | Mobile-first utilities; tiny production bundle after tree-shaking |
| API client | `@connectrpc/connect-web` + `@connectrpc/connect-query` | Typed hooks directly from the existing proto definitions; React Query semantics |
| Client routing | React Router v7 | Lightweight; needed for `/login` vs authenticated app routes |
| Server state | connect-query (React Query under the hood) | Handles caching, loading/error states, and refetch for all gRPC calls |
| Prod serving | Caddy (static files + API proxy) | Already in production; no new infrastructure |
| Dev serving | Vite dev server + proxy to Go `:8080` | Mirrors production routing locally |

## Repository Layout

```
services/todo-web/       ← new: React application
  src/
    main.tsx
    App.tsx
    components/
    pages/
  index.html
  vite.config.ts
  tailwind.config.ts
  tsconfig.json
  package.json
```

The frontend is a sibling to `services/todo/` (the Go backend), consistent with the existing monorepo layout.

## Architecture

```
Browser
  └── Caddy :80
        ├── /auth/*                    → reverse_proxy todo_server:8080
        ├── /task.v1.TaskService/*     → reverse_proxy todo_server:8080
        └── everything else            → file_server /srv/www (React SPA)
                                           try_files {path} /index.html
```

Client-side routing (`/login`, `/tasks`, `/tasks/:id`) is handled by React Router. Caddy's `try_files … /index.html` fallback ensures deep-links and refreshes work.

### API Communication

`@connectrpc/connect-web` calls the existing `TaskService` ConnectRPC endpoints directly from the browser using the Connect protocol (HTTP/JSON). No translation layer or new endpoints are needed. Generated TypeScript types come from the existing proto definitions via `buf generate`.

### Authentication

Auth reuses the existing session cookie mechanism (feature 003-user-auth):

- **Sign in**: `POST /auth/login` (username + password form) → Go sets an `HttpOnly` session cookie.
- **API calls**: The browser sends the cookie automatically; the Go auth middleware accepts it.
- **Sign out**: `POST /auth/logout` → Go clears the cookie; React Router redirects to `/login`.
- **Session expiry**: Any 401 response from the API triggers a redirect to `/login`, preserving the current URL as a `?next=` param.

### State Management

connect-query covers all server state (task list, task detail, mutations). No additional state library is needed. Local UI state (form inputs, tree expand/collapse) uses React `useState`.

## Production Deployment

The built `dist/` output is bind-mounted into the Caddy container:

```yaml
# compose.yaml.j2 additions
  caddy:
    volumes:
      - "{{ todo_host_data_dir }}/app/Caddyfile:/etc/caddy/Caddyfile:ro,z"
      - "{{ todo_host_data_dir }}/app/dist:/srv/www:ro,z"
```

The Ansible playbook gets a build step that runs `npm ci && npm run build` in `services/todo-web/` and copies `dist/` to the host before reloading Caddy.

Updated Caddyfile:

```caddy
:{{ todo_http_port }} {
    handle /auth/* {
        reverse_proxy todo_server:8080
    }
    handle /task.v1.* {
        reverse_proxy todo_server:8080
    }
    handle /health.v1.* {
        reverse_proxy todo_server:8080
    }
    handle {
        root * /srv/www
        try_files {path} /index.html
        file_server
    }
}
```

## Development

Vite proxies API and auth routes to the Go server:

```ts
// vite.config.ts
proxy: {
  '/auth': 'http://localhost:8080',
  '/task.v1': 'http://localhost:8080',
  '/health.v1': 'http://localhost:8080',
}
```

Workflow: `make dev` starts the Go server + postgres; `npm run dev` in `services/todo-web/` starts the Vite dev server.

## Key Constraints

- **Mobile-first**: All layouts and touch targets sized for phones. Desktop usable but not polished.
- **Online-only**: No service worker, no offline queue. A connectivity error shows a retry prompt.
- **Task management only**: Pomodoro, planning, and other terminal features are out of scope.
- **No deletion**: The spec explicitly excludes task deletion from this round.
