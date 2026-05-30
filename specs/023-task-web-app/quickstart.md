# Quickstart: Task Web App (development)

How to run and develop the web app locally against the existing Go backend. Nothing here changes the backend.

## Prerequisites

- Node.js 20+ and npm.
- The backend running locally: from `services/todo/`, `make dev` (starts Postgres + the Go server on `:8080`).
- A provisioned user (for login). Provision one if needed:
  ```bash
  cd services/todo
  ./todo-server --provision-user me:secretpassword   # prints an API key; remember the password for web login
  ```

## First-time setup

```bash
cd services/todo-web
npm install
npm run gen        # buf generate via buf.gen.web.yaml → src/gen/  (regenerate when the proto changes)
```

## Run the dev server

```bash
cd services/todo-web
npm run dev        # Vite on http://localhost:5173, proxying /auth, /task.v1, /health.v1 → :8080
```

Open `http://localhost:5173` on a phone-sized viewport (browser devtools device mode). The Vite proxy keeps the SPA and API on one origin so the `SameSite=Strict` session cookie works.

## Verify the primary flows (maps to spec user stories)

1. **Sign in (US1 / FR-001)**: go to `/login`, enter the provisioned username + password → lands on `/tasks`.
2. **Capture a task (US1 / FR-008, FR-009)**: add a top-level task; open it; add a sub-task → both appear in the tree, nested correctly.
3. **Review (US2 / FR-005–FR-007)**: expand/collapse branches; tap a task to see its detail; an account with no tasks shows the playful empty state.
4. **Edit & complete (US3 / FR-011, FR-012, FR-017)**:
   - Edit a task's title/description and save → persists (and its due/parent are preserved — full-replace handled).
   - Mark a leaf complete, then reopen it → toggles.
   - Try to complete a parent with an incomplete sub-task → blocked with a friendly message.
5. **Cross-client (SC-005)**: create a task in the TUI (`services/todo`), refresh the web app → it appears.

## Build for production

```bash
cd services/todo-web
npm run build      # outputs static assets to dist/
```

Deployment (Ansible/Caddy) ships `dist/` to the host and bind-mounts it into Caddy at `/srv/www`; Caddy serves the SPA and proxies `/auth/*`, `/task.v1.*`, `/health.v1.*` to `todo_server:8080` on a single origin. See `plan.md` → Deployment / Ops.

## Tests

```bash
cd services/todo-web
npm test           # Vitest: tree builder (lib/tree.ts) + UpdateTask payload builder + key interactions
```

## Gotchas

- **One origin only**: never point the SPA directly at `:8080` from a different origin — the `SameSite=Strict` cookie won't be sent and every call 401s. Use the Vite proxy.
- **Editing**: always send the full task on `UpdateTask` (resend `due` + `parent_id`), or you'll wipe them — the edit form does this for you.
- **Auth is inferred**: the cookie is `HttpOnly`; the app detects "logged out" from an `Unauthenticated` API response, not by reading the cookie.
- **Regenerate after proto changes**: re-run `npm run gen` if `services/todo/proto` changes.
