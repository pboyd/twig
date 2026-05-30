# Phase 0 Research: Task Web App

All Technical Context unknowns are resolved below. Findings are grounded in the existing backend code (`services/todo/internal/auth/`, `services/todo/proto/task/v1/task.proto`) and the approved design doc.

## R1: Authentication & session cookie integration

**Decision**: Reuse the existing `/auth/login` and `/auth/logout` HTTP endpoints and the `todo_session` cookie as-is. No backend auth changes; no CSRF token handling on the client.

**Findings (from `internal/auth/handler.go` and `middleware.go`)**:
- Login: `POST /auth/login`, JSON body `{"username","password"}`. Success → `200` (empty body) + `Set-Cookie: todo_session`. Bad credentials → `401` `{"error":"invalid username or password"}`. Missing fields → `400`.
- The cookie is `HttpOnly`, `Secure`, `SameSite=Strict`, `Path=/`, `MaxAge = sessionLifetime`. It is therefore invisible to JS (cannot be read to "check if logged in").
- Logout: `POST /auth/logout` clears the cookie, returns `200`.
- The ConnectRPC middleware accepts either `Authorization: Bearer <api-key>` or the `todo_session` cookie. The browser will use the cookie.

**Rationale / consequences**:
- **No CSRF token needed**: protection comes from `SameSite=Strict`, so the client sends no token. This keeps the frontend simpler.
- **Single origin is mandatory**: `SameSite=Strict` means the cookie is only sent on same-site requests, so the SPA and the API must share an origin (Caddy in prod, Vite proxy in dev). Calling the Go server cross-origin would silently drop auth.
- **Auth state is inferred, not read**: because the cookie is `HttpOnly`, the SPA cannot inspect it. It treats "logged in" as the default and reacts to a `401`/`Unauthenticated` from any API call by redirecting to `/login?next=<path>`. A bare `/tasks` load issues `ListTasks`; a `401` there bounces to login.

**Alternatives considered**: A token-in-localStorage scheme (rejected — would require new backend endpoints and abandons the existing secure cookie); a `/auth/me` probe endpoint (rejected — doesn't exist, and the `ListTasks` call already serves as the implicit probe).

## R2: Local-dev cookie behavior (`Secure` over http://localhost)

**Decision**: Develop via the Vite dev server with a proxy to `:8080`; rely on the browser's localhost secure-context exception.

**Rationale**: The cookie has `Secure: true`, so it is normally sent only over HTTPS. Modern browsers (Chrome, Firefox, Safari) treat `http://localhost` as a secure context and accept/return `Secure` cookies there, so login works in dev without TLS. The Vite proxy keeps everything on the `localhost:5173` origin, satisfying `SameSite=Strict`.

**Alternatives considered**: Running local HTTPS with mkcert (rejected as unnecessary overhead given the localhost exception); pointing the SPA directly at `:8080` (rejected — different origin drops the Strict cookie).

## R3: Browser ConnectRPC client + server-state management

**Decision**: `@connectrpc/connect-web` transport wired into `@connectrpc/connect-query` (which sits on `@tanstack/react-query` v5). Configure the transport's `fetch` with `credentials: "include"` so the session cookie rides along, and centralize a response interceptor that maps an `Unauthenticated` Connect error to a `/login` redirect.

**Rationale**: connect-query generates typed query/mutation hooks directly from the proto, giving caching, loading, and error states for free — which directly serve FR-015 (surface failures) and the loading/empty UX. No bespoke fetch layer or extra state library is needed (Principle I).

**Alternatives considered**: Hand-rolled fetch + manual cache (rejected — reinvents React Query); Redux/Zustand (rejected — no client state beyond server cache + transient form/expand state).

## R4: TypeScript code generation from the existing proto

**Decision**: Add `services/todo-web/buf.gen.web.yaml` using `buf.build/bufbuild/es` (`protoc-gen-es`) and `buf.build/connectrpc/query-es` (connect-query) plugins, output to `services/todo-web/src/gen/`. Generate against the existing `services/todo/proto` module. The existing Go-only `buf.gen.yaml` is left untouched.

**Rationale**: A separate gen file keeps Go and TS generation independent and avoids touching the backend's pipeline. Generated TS is committed and treated as read-only (mirrors the repo's "do not edit generated code" convention).

**Alternatives considered**: Extending the existing `buf.gen.yaml` with TS plugins (rejected — couples backend regen to the frontend and changes a stable file); ts-proto (rejected — connect-query plugins integrate better with the chosen client).

## R5: Task tree construction & ordering

**Decision**: `ListTasks` returns a flat list ordered by `id` ascending; build the nested tree on the client in `lib/tree.ts` by linking `parent_id → id`, preserving the server's id-ascending order among siblings. Unit-test this helper.

**Rationale**: The backend has no tree endpoint and orders by id; the spec defers ordering to existing backend behavior. Client-side assembly is trivial and keeps the backend untouched. Id-ascending = roughly creation order, which is a sensible, predictable sibling order on mobile.

**Alternatives considered**: A new server-side tree RPC (rejected — out of scope, backend frozen); manual ordering UI (rejected — not in spec).

## R6: Edit semantics — `UpdateTask` is full-replace

**Decision**: The task edit form loads the task via `GetTask`, lets the user change `name`/`description`, and on save sends `UpdateTask` with the **full** current state — re-sending the existing `due` and `parent_id` — so unedited fields are preserved.

**Rationale**: `UpdateTask` has documented full-replace semantics ("omitted optional fields are cleared"). Sending only name/description would silently wipe a task's due date or re-parent it to top level. This is the single most important correctness detail in the feature and is captured in `data-model.md` and `contracts/task-service.md`.

**Alternatives considered**: A partial-update / field-mask RPC (rejected — doesn't exist; backend frozen).

## R7: Completion toggle & parent/child guard

**Decision**: Use `CompleteTask` / `UncompleteTask` for the toggle. Surface the backend's `FailedPrecondition` errors as playful, actionable messages:
- `CompleteTask` on a task with incomplete descendants → "Hold on — finish its sub-tasks first." (FR-017)
- `UncompleteTask` on a task whose parent is complete → "Reopen its parent first to reopen this one."

**Rationale**: The rules (FR-012, FR-017) are already enforced server-side; the client's job is to call the right RPC and translate the error code into friendly copy (Principle IV). `CompleteTask` is idempotent, so double-taps are safe.

**Alternatives considered**: Client-side pre-checking descendant completeness before calling (rejected — the server is the source of truth and already validates; pre-checking duplicates logic and can race).

## R8: Frontend testing approach

**Decision**: Vitest + React Testing Library. Focus tests on pure logic (`lib/tree.ts` tree-building, the `UpdateTask` payload builder) and a few component interaction tests (form validation blocks empty title; completion error renders friendly copy).

**Rationale**: Vitest is the native test runner for Vite projects with near-zero config. The highest-risk logic is the tree builder and the full-replace payload builder (R5, R6); those get unit coverage. Exhaustive UI E2E is out of scope for this round.

**Alternatives considered**: Jest (rejected — extra config vs. Vite-native Vitest); Playwright E2E (rejected — heavier than warranted for a 3-route app this round).

## Resolved unknowns summary

| Unknown | Resolution |
|---------|------------|
| Auth/session mechanism | Reuse `/auth/*` + `todo_session` cookie; no CSRF token (R1) |
| Dev cookie over http | localhost secure-context exception; Vite proxy (R2) |
| Client/server-state libs | connect-web + connect-query/React Query (R3) |
| TS codegen | `buf.gen.web.yaml`, es + query-es → `src/gen/` (R4) |
| Tree + ordering | Client-side build from flat id-ordered list (R5) |
| Edit semantics | Full-replace: resend due + parent_id (R6) |
| Completion guard UX | Map `FailedPrecondition` to playful copy (R7) |
| Testing | Vitest + RTL on logic + key interactions (R8) |
