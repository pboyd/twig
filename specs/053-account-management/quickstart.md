# Quickstart: Self-Serve Account Management

How to build, wire, and exercise this feature end-to-end. Assumes the dev stack from
`CLAUDE.md` (PostgreSQL + server via `make dev`).

## 1. Generate the API stubs (contract first)

```bash
# Copy the reviewed contract into the proto tree
cp specs/053-account-management/contracts/account.proto api/proto/account/v1/account.proto

# Regenerate Go server stubs (api/gen/account/v1/...)
make proto

# Regenerate the web client (services/twig-web/src/gen/account/v1/...)
cd services/twig-web && npm run gen && cd -
```

## 2. Add DB queries and regenerate sqlc

Edit `services/twig/db/queries/auth.sql` to add `GetUserByID`, `UpdateUserPasswordByID`,
`ListApiKeysByUser`, and `DeleteApiKeyForUser` (see `data-model.md`), then:

```bash
cd services/twig && sqlc generate && cd -
```

## 3. Implement and register the handler

- Implement `services/twig/internal/handler/account.go` (`AccountService`), holding
  `*db.Queries` and the shared `*auth.LoginLimiter`.
- Register it in `services/twig/cmd/server/main.go` alongside the task/plan/goal handlers,
  behind `auth.Middleware`, passing the same `loginLimiter` instance used by `LoginHandler`.

```bash
cd services/twig && go test ./... && cd -   # handler tests, no live DB
```

## 4. Wire the web app

- Add `"/account.v1": "http://localhost:8080"` to the proxy in
  `services/twig-web/vite.config.ts`.
- Add the `/account` route in `src/App.tsx` and an "Account" link in
  `src/components/AppHeader.tsx`.
- Build `src/pages/AccountPage.tsx` from existing components; put all copy in
  `src/theme/messages.ts`.

```bash
cd services/twig-web && npm test && cd -
```

## 5. Add the production reverse-proxy route

The dev proxy only fixes local dev. In production, Caddy routes backend paths via an explicit
allowlist regex — add `account[.]v1[.]` so `/account.v1.AccountService/*` reaches the server
instead of falling through to the SPA:

```jinja
# deploy/roles/app/templates/Caddyfile.j2
@backend path_regexp "^/(auth/|cli/|task[.]v1[.]|health[.]v1[.]|plan[.]v1[.]|goal[.]v1[.]|account[.]v1[.])"
```

Re-run the `app` ansible role to render and reload the Caddyfile on deploy.

## 6. Manual end-to-end check

```bash
make dev                                   # start postgres + server
# Provision a throwaway account to log in with:
podman-compose exec server ./twig-server --provision-user demo:demopass
cd services/twig-web && npm run dev         # → http://localhost:5173
```

Then, signed in as `demo`:

1. **Change password** (US1): Account → enter `demopass` + a new 8+ char password twice →
   success toast. Sign out; confirm the new password works and `demopass` is rejected
   (SC-002). Confirm you were **not** signed out by the change (FR-006).
2. **Create key** (US2): create a key with a label → full secret shown once with a
   "can't see this again" warning. Copy it and authenticate the CLI:
   `TWIG_API_KEY=<secret> TWIG_ADDR=http://localhost:8080 ./twig task list` (SC-003).
   Reload the page → only label + date remain, no secret (US2 scenario 4, SC-005).
3. **Revoke key** (US3): revoke that key (confirm the prompt) → it disappears; the same CLI
   call now fails while any other key still works (SC-004). Revoke the *last* key → observe
   the "this removes all CLI/API access" warning before it proceeds (FR-016).

## Acceptance trace

| Story | Verified by |
|---|---|
| US1 Change password | Step 6.1 + `account_test.go` ChangePassword cases |
| US2 Create API key | Step 6.2 + `account_test.go` CreateApiKey cases + `AccountPage.test.tsx` |
| US3 View/revoke keys | Step 6.3 + `account_test.go` List/Revoke cases + `AccountPage.test.tsx` |
| Authz / unauth (SC-006) | `auth.Middleware` rejects unauthenticated calls (covered by middleware tests) |
