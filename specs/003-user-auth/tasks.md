---
description: "Task list for feature 003-user-auth implementation"
---

# Tasks: User Authentication

**Input**: Design documents from `/specs/003-user-auth/`

**Prerequisites**: plan.md, spec.md, research.md, data-model.md, contracts/auth-http.md

**Tests**: Test tasks ARE included — the plan's Testing section and the existing
codebase (every package has `_test.go`) make tests part of the definition of
done. Within each phase, test tasks precede the implementation they cover and
should fail before that implementation exists.

**Organization**: Tasks are grouped by user story. All paths are under
`services/todo/` (the existing Go module); paths below are written in full.

## Format: `[ID] [P?] [Story] Description`

- **[P]**: Can run in parallel (different files, no dependency on an incomplete task)
- **[Story]**: US1 / US2 / US3 — user story from spec.md (story phases only)

---

## Phase 1: Setup (Shared Infrastructure)

**Purpose**: Add the one new dependency the feature needs.

- [X] T001 Add `golang.org/x/crypto` and run `go mod tidy` in `services/todo/` so `golang.org/x/crypto/bcrypt` is available (research Decision 3)

---

## Phase 2: Foundational (Blocking Prerequisites)

**Purpose**: Schema, generated DB code, credential helpers, the auth middleware,
and per-user task scoping — everything every user story consumes.

**⚠️ CRITICAL**: No user story work can begin until this phase is complete.

- [X] T002 Create `services/todo/db/migrations/000003_auth.up.sql` — `users`, `sessions`, `api_keys` tables and the `tasks.user_id` column plus `tasks_user_id_idx`, `sessions` and `api_keys` `user_id` indexes (per data-model.md)
- [X] T003 [P] Create `services/todo/db/migrations/000003_auth.down.sql` — drop `tasks.user_id` and its index, then drop `api_keys`, `sessions`, `users`
- [X] T004 [P] Modify `services/todo/db/queries/task.sql` — add a `user_id` parameter and `WHERE`/`AND user_id = $n` predicate to `CreateTask`, `GetTask`, `ListTasks`, `UpdateTask`, `DeleteTask`, `TaskExists` (data-model.md "Query changes")
- [X] T005 [P] Create `services/todo/db/queries/auth.sql` — `CreateUser`, `GetUserByUsername`, `UpdateUserPassword`, `CreateSession`, `GetSession`, `DeleteSession`, `CreateApiKey`, `GetApiKeyByHash`, `DeleteApiKeysByUser` (data-model.md "auth.sql")
- [X] T006 Run `sqlc generate` in `services/todo/` to regenerate `internal/db/models.go` (adds `User`, `Session`, `ApiKey`; `Task.UserID`), `internal/db/task.sql.go`, and create `internal/db/auth.sql.go` — depends on T002, T004, T005
- [X] T007 [P] Write unit tests in `services/todo/internal/auth/credential_test.go` — bcrypt hash/verify round-trip, random token length/uniqueness, SHA-256 key hashing, constant-time compare
- [X] T008 Implement `services/todo/internal/auth/credential.go` — `HashPassword`/`VerifyPassword` (bcrypt cost 12), `GenerateToken` (32 bytes `crypto/rand`, hex), `HashAPIKey` (`crypto/sha256` hex), constant-time compare helper — make T007 pass
- [X] T009 Write tests in `services/todo/internal/auth/middleware_test.go` — `/auth/*` bypass, missing credential → 401, unknown/expired session → 401, unknown/revoked API key → 401, `Authorization` header wins when both present, resolved `user_id` reaches context
- [X] T010 Implement `services/todo/internal/auth/middleware.go` — `http.Handler` middleware: bypass `/auth/login` and `/auth/logout`, resolve `Authorization: Bearer` (SHA-256 → `GetApiKeyByHash`) then session cookie (`GetSession`, check `expires_at`), inject `user_id` via `context.WithValue`, exported `UserID(ctx)` accessor, 401 + rejection logging on failure — depends on T006, T008; make T009 pass
- [X] T011 Update `services/todo/internal/handler/task_test.go` — supply a `user_id` context (via `auth.UserID` seam) to existing handler tests and add cross-user isolation cases (a second user cannot get/update/delete the first user's task by id) — depends on T006, T010
- [X] T012 Modify `services/todo/internal/handler/task.go` — read `user_id` from context with `auth.UserID`, pass it to the user-scoped `db` queries in `CreateTask`/`GetTask`/`ListTasks`/`UpdateTask`/`DeleteTask` and the `parentChainContains`/`TaskExists` checks — make T011 pass
- [X] T013 [P] Wire the auth middleware around the task mux in `services/todo/cmd/server/main.go` — wrap the Connect handler mux with `auth.Middleware` before `h2c` — depends on T010

**Checkpoint**: Schema, generated code, credential helpers, middleware, and
per-user task scoping are in place. Task RPCs now reject every request with
`401` until a credential-issuing story is added.

---

## Phase 3: User Story 1 - Browser user signs in to manage their tasks (Priority: P1) 🎯 MVP

**Goal**: Username/password sign-in issues a session cookie; sign-out ends it.
The middleware (Phase 2) already consumes session cookies — this story delivers
their creation and destruction.

**Independent Test**: With a user row inserted directly, `POST /auth/login` with
the correct password returns `200` + a `HttpOnly; Secure; SameSite=Strict`
cookie; a task RPC carrying that cookie succeeds; `POST /auth/logout` deletes the
session and the cookie then returns `401`. Wrong password returns a generic
`401` with no cookie.

- [X] T014 [P] [US1] Write tests in `services/todo/internal/auth/handler_test.go` — `POST /auth/login` success (200 + cookie attributes), wrong password and unknown user (generic 401, no cookie), bad JSON (400), non-POST (405); `POST /auth/logout` deletes the session and clears the cookie, and is idempotent for a missing/expired cookie (per contracts/auth-http.md §2–§3)
- [X] T015 [US1] Implement `services/todo/internal/auth/handler.go` — `LoginHandler` (parse JSON, `GetUserByUsername` + `VerifyPassword`, `CreateSession` with configurable lifetime default 30 days, `Set-Cookie todo_session` with `HttpOnly; Secure; SameSite=Strict; Max-Age`) and `LogoutHandler` (`DeleteSession`, clear cookie), generic 401 message, login/logout event logging (FR-018) — make T014 pass
- [X] T016 [US1] Register `POST /auth/login` and `POST /auth/logout` on the mux in `services/todo/cmd/server/main.go`, before the middleware-wrapped task handlers — depends on T015, T013

**Checkpoint**: Browser-style session login/logout works end to end and is the
deployable MVP slice (a user must exist — insert directly, or provision via US3).

---

## Phase 4: User Story 2 - CLI tool authenticates with an API key (Priority: P2)

**Goal**: The `todo` CLI sends its API key as a `Bearer` token on every request.
The middleware (Phase 2) already resolves API keys — this story delivers the
client side.

**Independent Test**: With an `api_keys` row inserted directly, running a `todo`
task command with `TODO_API_KEY` set reaches the backend and operates on that
key's user; an unset or wrong `TODO_API_KEY` fails with a clear error / `401`.

- [X] T017 [P] [US2] Write tests in `services/todo/internal/cli/cli_test.go` — a task command against an `httptest` fake server asserts the request carries `Authorization: Bearer <TODO_API_KEY>`; an unset `TODO_API_KEY` produces a clear non-zero-exit error
- [X] T018 [US2] Modify `services/todo/internal/cli/cli.go` — read `TODO_API_KEY`, attach `Authorization: Bearer <key>` to every task RPC via a Connect interceptor (or `http.RoundTripper` wrapper), and fail with a clear message when the variable is unset — make T017 pass

**Checkpoint**: The CLI authenticates with an API key; US1 and US2 both work
independently.

---

## Phase 5: User Story 3 - Operator provisions accounts at deploy time (Priority: P3)

**Goal**: A `--provision-user` server flag creates accounts and API keys at
deploy time, printing each raw key once, then exits.

**Independent Test**: Starting the server with `--provision-user=alice:s3cr3t`
creates the `users` row and one `api_keys` row, prints the raw key exactly once,
and exits without listening. Re-running for an existing username updates the
password and issues a fresh key (old key stops working).

- [X] T019 [P] [US3] Write tests in `services/todo/internal/auth/provision_test.go` — provisioning a new user creates the account + one API key and returns the raw key once; provisioning an existing username updates the password and replaces the API key (FR-015, FR-019)
- [X] T020 [US3] Implement `services/todo/internal/auth/provision.go` — for each `name:password`: `GetUserByUsername` then `CreateUser` or `UpdateUserPassword` (bcrypt), `DeleteApiKeysByUser`, generate a key, `CreateApiKey` with its SHA-256 hash, return the raw key for one-time display — make T019 pass
- [X] T021 [US3] Add a repeatable `--provision-user=name:password` flag (custom `flag.Value`) to `services/todo/cmd/server/main.go`; when present, run migrations, call provisioning, print each raw key once, and exit before `ListenAndServe` — depends on T020

**Checkpoint**: All three user stories are independently functional.

---

## Phase 6: Polish & Cross-Cutting Concerns

**Purpose**: Verification across the whole feature.

- [X] T022 [P] Run `go build ./...` and `go vet ./...` in `services/todo/` and fix any issues
- [ ] T023 Run the `specs/003-user-auth/quickstart.md` walkthrough end to end — provision, browser login, CLI with API key, and the cross-user isolation check
- [X] T024 [P] Verify FR-018 logging — confirm auth events (login success/failure, logout, middleware rejections) are logged and that passwords, raw API keys, and session IDs never appear in logs, across `handler.go` and `middleware.go`

---

## Dependencies & Execution Order

### Phase Dependencies

- **Setup (Phase 1)**: No dependencies — start immediately.
- **Foundational (Phase 2)**: Depends on Setup — BLOCKS all user stories.
- **User Stories (Phases 3–5)**: All depend on Foundational completion. Once it
  is done, US1, US2, and US3 are independent and may proceed in parallel or in
  priority order (P1 → P2 → P3).
- **Polish (Phase 6)**: Depends on all desired user stories being complete.

### Key Task Dependencies

- T006 (sqlc generate) depends on T002, T004, T005.
- T008 depends on T007; T010 depends on T006 + T008 (and T009 tests); T012
  depends on T011; T011 depends on T006 + T010; T013 depends on T010.
- T015 depends on T014 + Phase 2; T016 depends on T015 + T013.
- T018 depends on T017 + Phase 2.
- T020 depends on T019 + Phase 2; T021 depends on T020.
- `cmd/server/main.go` is edited by T013, T016, and T021 — these three must be
  sequenced relative to one another (not run in parallel).

### User Story Dependencies

- **US1 (P1)**: Foundational only. Independently testable.
- **US2 (P2)**: Foundational only. Independently testable.
- **US3 (P3)**: Foundational only. Independently testable.

No user story depends on another — the middleware in Phase 2 consumes both
credential types, so each story only adds one credential's issuing/sending side.

### Parallel Opportunities

- Phase 2: T003, T004, T005 run in parallel; T007 runs in parallel with them.
  After T010, T011 and T013 can run in parallel.
- Phase 3/4/5 lead test tasks T014, T017, T019 are each `[P]` and, after
  Foundational, the three stories can be developed in parallel by different
  people.
- Phase 6: T022 and T024 run in parallel.

---

## Parallel Example: Phase 2 Foundational

```bash
# After T002 (migration up) is drafted, these touch different files:
Task: "Create db/migrations/000003_auth.down.sql"          # T003
Task: "Modify db/queries/task.sql for user_id scoping"     # T004
Task: "Create db/queries/auth.sql"                         # T005
Task: "Write internal/auth/credential_test.go"             # T007
```

## Parallel Example: User stories after Foundational

```bash
# Once Phase 2 is complete, the three stories are independent:
Developer A: T014 → T015 → T016   # US1 — session login
Developer B: T017 → T018          # US2 — CLI API key
Developer C: T019 → T020 → T021   # US3 — provisioning
```

---

## Implementation Strategy

### MVP First (User Story 1)

1. Phase 1: Setup (T001).
2. Phase 2: Foundational (T002–T013) — CRITICAL, blocks all stories.
3. Phase 3: User Story 1 (T014–T016).
4. **STOP and VALIDATE**: sign in, call a task RPC with the cookie, sign out.
   For a real deploy (not just tests) provision a user via US3 first.

### Incremental Delivery

1. Setup + Foundational → task RPCs are credential-gated.
2. Add US1 → browser session login works → demo.
3. Add US2 → CLI API-key auth works → demo.
4. Add US3 → deploy-time provisioning works → demo.
5. Phase 6 → verify build, quickstart, and logging.

---

## Notes

- `[P]` = different files, no dependency on an incomplete task.
- Test tasks precede the implementation they cover; confirm they fail first.
- `task.proto` and the generated Connect code are NOT changed — per-user scoping
  is server-side only (see plan.md and contracts/auth-http.md).
- Commit after each task or logical group.
- Total: 24 tasks — Setup 1, Foundational 12, US1 3, US2 2, US3 3, Polish 3.
