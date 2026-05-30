# Contract: Auth HTTP endpoints (web app usage)

**Source of truth**: `services/todo/internal/auth/handler.go` (feature 003-user-auth). Plain HTTP, outside ConnectRPC. **No backend changes** in this feature.

These endpoints are reverse-proxied by Caddy (prod) / Vite (dev) on the same origin as the SPA.

## POST /auth/login (FR-001, FR-002)

- **Request**: `Content-Type: application/json`, body `{"username": "...", "password": "..."}`.
- **Success** `200 OK`: empty body; `Set-Cookie: todo_session=<token>` with attributes `HttpOnly; Secure; SameSite=Strict; Path=/; Max-Age=<sessionLifetime>`.
- **Invalid credentials** `401 Unauthorized`: `{"error":"invalid username or password"}`. The message does not reveal whether the username or password was wrong (FR-002).
- **Missing fields** `400 Bad Request`: `{"error":"username and password are required"}`.
- **Wrong method** `405`.

**Client behavior**:
- The SPA posts the login form here. On `200`, it navigates to the `next` target (or `/tasks`).
- On `401`, it shows a playful, non-revealing error and keeps the username field populated.
- The cookie is `HttpOnly` → JS cannot read it. The SPA never inspects the cookie; it infers auth from API responses.

## POST /auth/logout (FR-003)

- **Request**: `POST` (no body required). The browser sends the `todo_session` cookie automatically.
- **Response** `200 OK`: clears the cookie (`Max-Age=0`). Idempotent — succeeds even with no/expired session.

**Client behavior**: on sign-out, POST here, clear the React Query cache, and redirect to `/login`.

## Session lifetime & expiry (FR-003)

- Lifetime is server-configured; default **30 days** (003-user-auth). The cookie's `Max-Age` reflects it, so the session persists across reloads and revisits until it expires or the user signs out.
- Mid-use expiry surfaces as `Unauthenticated` on the next ConnectRPC call (see `task-service.md`), which redirects to `/login?next=<current path>` so the user returns where they were after re-auth.

## Security notes (inherited, not changed)

- `SameSite=Strict` + same-origin deployment provides CSRF protection; the client sends **no CSRF token**.
- `Secure` cookie: works over `http://localhost` in dev via the browser's localhost secure-context exception (research R2).
- `HttpOnly`: mitigates token theft via XSS; the price is that auth state is inferred from API responses, not read from the cookie.
