---
description: "Task list for Self-Serve Account Management"
---

# Tasks: Self-Serve Account Management

**Input**: Design documents from `/specs/053-account-management/`

**Prerequisites**: plan.md, spec.md, research.md, data-model.md, contracts/

**Tests**: Included. The project has established test conventions (`services/twig/internal/handler/*_test.go` with `export_test.go` shims; `services/twig-web/src/**/*.test.tsx`) and the spec defines an Independent Test per story, so each story carries handler + web tests.

**Organization**: Tasks are grouped by user story (US1–US3) for independent implementation and testing.

## Format: `[ID] [P?] [Story] Description`

- **[P]**: Can run in parallel (different files, no dependencies on incomplete tasks)
- **[Story]**: US1, US2, US3 (maps to spec.md user stories)

## Path Conventions

Existing monorepo: proto in `api/proto/`, server in `services/twig/`, web SPA in `services/twig-web/`, deployment in `deploy/`. New service domain `account.v1` follows the `task.v1`/`plan.v1`/`goal.v1` pattern.

> **Shared-file note**: The four RPCs live in one handler file (`account.go`), one test file (`account_test.go`), one SQL file (`auth.sql`), and the UI lives in one page (`AccountPage.tsx`). Tasks across stories that touch the same file are therefore **not** parallel with each other — see Dependencies. Implement stories sequentially by priority for the smoothest path.

---

## Phase 1: Setup (Shared Infrastructure)

**Purpose**: Land the API contract and generate stubs for both tiers.

- [X] T001 Copy `specs/053-account-management/contracts/account.proto` to `api/proto/account/v1/account.proto`
- [X] T002 Run `make proto` (repo root) to generate Go server stubs at `api/gen/account/v1/` (incl. `accountv1connect`)
- [X] T003 [P] Run `npm run gen` in `services/twig-web/` to generate the web client at `services/twig-web/src/gen/account/v1/`

---

## Phase 2: Foundational (Blocking Prerequisites)

**Purpose**: Routing + a compilable, registered service skeleton + the page shell, so every story can be filled in independently afterward.

**⚠️ CRITICAL**: No user story work can begin until this phase is complete

- [X] T004 Add `account[.]v1[.]` to the `@backend` `path_regexp` allowlist in `deploy/roles/app/templates/Caddyfile.j2` (production reverse-proxy route)
- [X] T005 [P] Add `"/account.v1": "http://localhost:8080"` to the dev proxy in `services/twig-web/vite.config.ts`
- [X] T006 Create `services/twig/internal/handler/account.go`: define `AccountService` struct holding a `*db.Queries` (or an `AccountQuerier` interface subset, mirroring `auth.HandlerQuerier`) plus a shared `*auth.LoginLimiter`; implement all four RPCs (`ChangePassword`, `ListApiKeys`, `CreateApiKey`, `RevokeApiKey`) as stubs returning `connect.NewError(connect.CodeUnimplemented, …)` so the service compiles and registers
- [X] T007 Register the `AccountService` handler in `services/twig/cmd/server/main.go` (behind `auth.Middleware`, alongside task/plan/goal) and pass the **same** `loginLimiter` instance already used by `LoginHandler`
- [X] T008 [P] Create `services/twig-web/src/pages/AccountPage.tsx` shell with two empty sections (password / API keys); add the `/account` route in `services/twig-web/src/App.tsx`; add an "Account" link in `services/twig-web/src/components/AppHeader.tsx`
- [X] T009 [P] Add an `account` copy namespace to `services/twig-web/src/theme/messages.ts` (warm/playful, per Constitution IV) — section headings + shared labels stories will extend

**Checkpoint**: `go build ./...` (in `services/twig/`) and `npm run build` (in `services/twig-web/`) both succeed; `/account` renders behind auth and its requests route to the backend in dev and prod.

---

## Phase 3: User Story 1 - Change my password (Priority: P1) 🎯 MVP

**Goal**: A signed-in user changes their own password by supplying the current one and a new one; sessions and API keys are unaffected.

**Independent Test**: Sign in → Account → enter correct current password + valid new password (twice) → success; sign out and confirm the new password works while the old is rejected.

### Tests for User Story 1

- [X] T010 [P] [US1] Handler tests for `ChangePassword` in `services/twig/internal/handler/account_test.go`: wrong current password → `InvalidArgument` (+ records a limiter failure); new password < 8 chars → `InvalidArgument`; happy path writes the new hash; repeated failures → `ResourceExhausted`; verify no session/API-key rows are touched (FR-003, FR-005, FR-006, FR-007, FR-019)
- [X] T011 [P] [US1] Web test in `services/twig-web/src/pages/AccountPage.test.tsx`: confirmation mismatch blocks submit (FR-004); sub-8-char new password shows guidance; happy path shows the success toast and the user stays signed in

### Implementation for User Story 1

- [X] T012 [US1] Add `GetUserByID` (`:one`) and `UpdateUserPasswordByID` (`:exec`) to `services/twig/db/queries/auth.sql`; run `sqlc generate` in `services/twig/` to regenerate `internal/db`
- [X] T013 [US1] Implement `ChangePassword` in `services/twig/internal/handler/account.go`: resolve caller via `GetUserByID(auth.UserID(ctx))`; gate on the limiter key `user:<username>`; `VerifyPassword` the current password (record failure + `InvalidArgument` on mismatch); enforce `len(new) >= 8`; `HashPassword` + `UpdateUserPasswordByID`; reset the limiter on success; do not touch sessions or keys
- [X] T014 [US1] Build the password-change section in `services/twig-web/src/pages/AccountPage.tsx` using `Button`/`Field`/`ErrorBanner` + a connect-query `ChangePassword` mutation; client-side confirm-match and length checks; success toast; add the section's copy to `theme/messages.ts`

**Checkpoint**: US1 fully functional and independently testable — the MVP.

---

## Phase 4: User Story 2 - Create a new API key (Priority: P2)

**Goal**: A signed-in user mints a new API key (optional label) and sees the full secret exactly once.

**Independent Test**: Sign in → Account → create a key with a label → full secret shown once with a "can't retrieve again" warning; use that secret to authenticate a CLI request successfully.

### Tests for User Story 2

- [X] T015 [P] [US2] Handler tests for `CreateApiKey` in `services/twig/internal/handler/account_test.go`: response carries the raw secret and metadata; empty label → default label stored (FR-010); provided label preserved; stored `key_hash` equals `HashAPIKey(secret)` so it authenticates via the existing lookup (FR-012)
- [X] T016 [P] [US2] Web test in `services/twig-web/src/pages/AccountPage.test.tsx`: after create, the full secret is shown once with the warning, and the new key appears in the list; re-rendering the list no longer shows the secret (US2 scenario 4)

### Implementation for User Story 2

- [X] T017 [US2] Implement `CreateApiKey` in `services/twig/internal/handler/account.go`: `GenerateToken` → raw secret; `HashAPIKey` → stored hash; resolve label (provided, else default constant per research Decision 6); insert via the existing `CreateApiKey` query with `created_at = now`; return `ApiKeyMetadata` + raw `secret`
- [X] T018 [US2] Build the "create key" UI in `services/twig-web/src/pages/AccountPage.tsx`: optional label input + connect-query `CreateApiKey` mutation; reveal the secret once with a copy affordance and the "this won't be shown again" warning; add the copy to `theme/messages.ts`

**Checkpoint**: US1 and US2 both work independently.

---

## Phase 5: User Story 3 - View and revoke API keys (Priority: P2)

**Goal**: A signed-in user lists their keys (non-secret metadata only) and revokes any of them.

**Independent Test**: Sign in → view keys (label + creation date, never the secret) → revoke one → confirm it no longer authenticates while other keys still work.

### Tests for User Story 3

- [X] T019 [P] [US3] Handler tests for `ListApiKeys` and `RevokeApiKey` in `services/twig/internal/handler/account_test.go`: list returns only `id`/`label`/`created_at` (never the secret/hash — FR-009); revoke is scoped by `user_id` (a non-owned/absent id is an idempotent success no-op — FR-014, concurrent-revocation edge case); other keys remain after a revoke
- [X] T020 [P] [US3] Web test in `services/twig-web/src/pages/AccountPage.test.tsx`: list shows label + date only; revoking prompts for confirmation (FR-013); revoking when exactly one key remains surfaces the "removes all CLI/API access" warning first (FR-016)

### Implementation for User Story 3

- [X] T021 [US3] Add `ListApiKeysByUser` (`:many`, projecting `id, label, created_at` only) and `DeleteApiKeyForUser` (`:execrows`, `WHERE id = $1 AND user_id = $2`) to `services/twig/db/queries/auth.sql`; run `sqlc generate` in `services/twig/`
- [X] T022 [US3] Implement `ListApiKeys` and `RevokeApiKey` in `services/twig/internal/handler/account.go`: list user-scoped metadata ordered by `created_at, id`; delete scoped by `id` + `auth.UserID(ctx)`, treating zero rows affected as success (idempotent)
- [X] T023 [US3] Build the key list + revoke UI in `services/twig-web/src/pages/AccountPage.tsx`: render each key's label + creation date with a per-key revoke action behind a confirmation; show the last-key warning when only one key remains; refresh the list via the `ListApiKeys` query after create/revoke; add the copy to `theme/messages.ts`

**Checkpoint**: All three stories independently functional.

---

## Phase 6: Polish & Cross-Cutting Concerns

**Purpose**: Validation and quality gates spanning all stories.

- [X] T024 [P] Tone review of all new `theme/messages.ts` copy against Constitution Principle IV (warm-but-measured for destructive confirmations; clarity preserved)
- [X] T025 [P] UI/UX consistency check (Principle III): AccountPage uses only shared components + `theme/tokens.ts`, no one-off styles
- [X] T026 Run quickstart.md steps 5–6 end-to-end: provision a demo user, change password (old rejected/new accepted, still signed in), create a key + authenticate the CLI with it, revoke it + confirm CLI failure while another key works
- [X] T027 Run full suites: `cd services/twig && go test ./...`; `cd services/twig-web && npm test`; `go build ./...` (repo root + server)

---

## Dependencies & Execution Order

### Phase Dependencies

- **Setup (Phase 1)**: No dependencies. T002 depends on T001; T003 is independent of T001/T002.
- **Foundational (Phase 2)**: Depends on Setup. T006 depends on T002 (generated `accountv1connect`); T007 depends on T006; T008 depends on T003. **Blocks all user stories.**
- **User Stories (Phase 3–5)**: All depend on Foundational completion. Independently testable, but they edit shared files (`account.go`, `account_test.go`, `auth.sql`, `AccountPage.tsx`, `messages.ts`) — so run them **sequentially by priority** to avoid conflicts.
- **Polish (Phase 6)**: Depends on all targeted stories being complete.

### User Story Dependencies

- **US1 (P1)**: After Foundational. No dependency on US2/US3.
- **US2 (P2)**: After Foundational. Independent of US1. (Its created keys naturally appear in US3's list, but each is testable alone.)
- **US3 (P2)**: After Foundational. Independent of US1/US2.

### Within Each User Story

- Write the tests (handler + web) first and watch them fail before implementing.
- DB queries (`auth.sql` + `sqlc generate`) → handler method → web UI.
- Story complete before moving to the next priority.

### Parallel Opportunities

- Setup: T003 ∥ (T001→T002).
- Foundational: T004, T005, T008, T009 are parallel; T006→T007 are sequential.
- Within a story, the two test tasks ([P]) can be written in parallel (different files: `account_test.go` vs `AccountPage.test.tsx`).
- **Cross-story caution**: because the handler, page, SQL, and messages are each single files, do **not** parallelize T013/T017/T022 (all `account.go`), T014/T018/T023 (all `AccountPage.tsx`), T010/T015/T019 (all `account_test.go`), or T012/T021 (both `auth.sql`). Sequence by story.

---

## Parallel Example: User Story 1

```bash
# The two US1 test tasks touch different files — write them together:
Task: "Handler tests for ChangePassword in services/twig/internal/handler/account_test.go"
Task: "Web test for password form in services/twig-web/src/pages/AccountPage.test.tsx"
```

---

## Implementation Strategy

### MVP First (User Story 1 only)

1. Phase 1: Setup → 2. Phase 2: Foundational (CRITICAL) → 3. Phase 3: US1 → **STOP & VALIDATE** (sign-out/in proves the change) → demo. Password self-service is the highest-value, independently shippable slice.

### Incremental Delivery

1. Setup + Foundational → foundation ready.
2. US1 (change password) → test → demo (MVP).
3. US2 (create key) → test → demo.
4. US3 (view/revoke keys) → test → demo.

### Notes

- [P] = different files, no incomplete dependencies.
- Server handler tests need no live DB — use a fake querier (mirror `auth.HandlerQuerier`/`ProvisionQuerier`).
- Reuse, don't reinvent: `HashPassword`/`VerifyPassword`/`HashAPIKey`/`GenerateToken`, the `LoginLimiter`, and the existing `CreateApiKey`/`GetApiKeyByHash` queries.
- The acting user is always `auth.UserID(ctx)`; no request carries an account id (FR-017).
- Commit after each task or logical group.
