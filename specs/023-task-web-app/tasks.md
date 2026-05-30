---

description: "Task list for Task Web App implementation"
---

# Tasks: Task Web App

**Input**: Design documents from `specs/023-task-web-app/`

**Prerequisites**: plan.md, spec.md, research.md, data-model.md, contracts/ (task-service.md, auth.md, ui-contract.md), quickstart.md

**Tests**: Targeted only. Per plan.md → Technical Context and research R8, the high-risk pure logic (`lib/tree.ts`, the `UpdateTask` full-replace payload builder) gets unit tests, plus two key interaction tests. No exhaustive E2E this round. The backend is unchanged, so no backend tests.

**Organization**: Tasks are grouped by user story. All paths are under the new `services/todo-web/` frontend; the Go backend (`services/todo/`) is NOT modified.

## Format: `[ID] [P?] [Story] Description`

- **[P]**: Can run in parallel (different files, no dependencies on incomplete tasks)
- **[Story]**: US1 / US2 / US3 (Setup, Foundational, Polish have no story label)

---

## Phase 1: Setup (Shared Infrastructure)

**Purpose**: Scaffold the new `services/todo-web/` SPA and its toolchain.

- [ ] T001 Create the frontend project skeleton in `services/todo-web/`: `package.json`, `tsconfig.json`, `index.html`, `.gitignore` (ignore `node_modules/`, `dist/`), and stub `src/main.tsx` + `src/App.tsx`
- [ ] T002 Declare and install dependencies in `services/todo-web/package.json` — runtime: `react@19`, `react-dom@19`, `react-router@7`, `@connectrpc/connect-web`, `@connectrpc/connect-query`, `@tanstack/react-query@5`; dev: `vite@6`, `typescript@5`, `tailwindcss@4`, `@tailwindcss/vite`, `vitest`, `@testing-library/react`, `@testing-library/jest-dom`, `jsdom`, `@bufbuild/protoc-gen-es`, `@connectrpc/protoc-gen-connect-query`
- [ ] T003 [P] Configure Vite with the dev proxy in `services/todo-web/vite.config.ts` — proxy `/auth`, `/task.v1`, `/health.v1` to `http://localhost:8080` (single-origin dev so the `SameSite=Strict` cookie works)
- [ ] T004 [P] Configure Tailwind v4 in `services/todo-web/tailwind.config.ts` and `services/todo-web/src/index.css` (`@import "tailwindcss";`), mobile-first defaults
- [ ] T005 [P] Configure Vitest + React Testing Library in `services/todo-web/vitest.config.ts` and `services/todo-web/src/test/setup.ts` (jsdom env, jest-dom matchers)
- [ ] T006 Add `services/todo-web/buf.gen.web.yaml` (plugins: `buf.build/bufbuild/es`, `buf.build/connectrpc/query-es`; out: `src/gen`) and a `"gen"` script in `package.json`; run it to generate TypeScript from `services/todo/proto` into `services/todo-web/src/gen/`

**Checkpoint**: `npm run dev` serves an empty app; `npm run gen` produces typed `task.v1` clients.

---

## Phase 2: Foundational (Blocking Prerequisites)

**Purpose**: Transport, providers, routing, auth guard, and the shared theme/components every story depends on (Principles III & IV).

**⚠️ CRITICAL**: No user-story work can begin until this phase is complete.

- [ ] T007 Implement the Connect transport in `services/todo-web/src/lib/transport.ts` — `createConnectTransport` at same-origin base URL, `fetch` with `credentials: "include"`, and an interceptor that detects `Code.Unauthenticated` and redirects to `/login?next=<current path>`
- [ ] T008 Wire root providers in `services/todo-web/src/main.tsx` — `QueryClientProvider` (React Query), connect-query `TransportProvider` (using T007), and `BrowserRouter`
- [ ] T009 Define the route table and auth handling in `services/todo-web/src/App.tsx` — routes `/login`, `/tasks`, `/tasks/:id`; default redirect `/` → `/tasks`; centralized handling of the `Unauthenticated` redirect from T007
- [ ] T010 [P] Create shared theme tokens in `services/todo-web/src/theme/tokens.ts` and wire them into `tailwind.config.ts` — colors (`bg/surface/text/muted/accent/danger/success`), typography, spacing; light + dark via `prefers-color-scheme` (per contracts/ui-contract.md)
- [ ] T011 [P] Create the centralized tone strings in `services/todo-web/src/theme/messages.ts` — all user-facing copy from contracts/ui-contract.md (Principle IV)
- [ ] T012 [P] Build shared primitives in `services/todo-web/src/components/`: `Button.tsx` (variants + loading/disabled), `Field.tsx` (labeled input/textarea + inline validation), `Spinner.tsx` — touch targets ≥44px (per contracts/ui-contract.md)
- [ ] T013 [P] Build `services/todo-web/src/components/ErrorBanner.tsx` — connectivity/expiry/precondition error display with a retry affordance, sourcing copy from `messages.ts` (FR-015)

**Checkpoint**: Providers, routing, auth redirect, theme, and primitives are ready — user stories can begin.

---

## Phase 3: User Story 1 - Capture a task on the go (Priority: P1) 🎯 MVP

**Goal**: A signed-in user can capture a new top-level task and a sub-task from a phone.

**Independent Test**: On a mobile viewport, sign in with valid credentials, add a top-level task, then add a sub-task under it; confirm both appear nested in the list. Invalid credentials are rejected.

### Tests for User Story 1

- [ ] T014 [P] [US1] Unit test the tree builder in `services/todo-web/src/lib/tree.test.ts` — roots, nesting by `parent_id`, id-ascending sibling order, defensive orphan handling

### Implementation for User Story 1

- [ ] T015 [P] [US1] Implement the flat→nested tree builder in `services/todo-web/src/lib/tree.ts` producing `TaskNode { task, children, depth }` (per data-model.md)
- [ ] T016 [US1] Implement `services/todo-web/src/pages/LoginPage.tsx` — username/password form (uses `Field`/`Button`), `POST /auth/login` JSON, on `200` redirect to `?next` or `/tasks`, on `401` show the non-revealing login-failure message and keep the username (FR-001, FR-002; contracts/auth.md)
- [ ] T017 [US1] Add an app header with a sign-out action in `services/todo-web/src/components/AppHeader.tsx` — `POST /auth/logout`, clear the React Query cache, redirect to `/login` (FR-003)
- [ ] T018 [US1] Implement `services/todo-web/src/pages/TaskTreePage.tsx` — call `ListTasks` via connect-query, build the tree (T015), render rows with loading (`Spinner`) and error (`ErrorBanner`) states (FR-005)
- [ ] T019 [US1] Implement `services/todo-web/src/components/TreeRow.tsx` (initial) — render task name, indent by `depth`, show completed-state indicator, and an "add sub-task" affordance
- [ ] T020 [US1] Implement `services/todo-web/src/components/TaskForm.tsx` — reusable create form; name required + empty/whitespace-title validation with the playful message (FR-010), description optional
- [ ] T021 [US1] Wire the create flow in `TaskTreePage`/`TreeRow` — "add task" (top-level) and "add sub-task" (sets `parent_id`) call `CreateTask`; invalidate the `ListTasks` query on success so the new task appears (FR-008, FR-009; contracts/task-service.md)

**Checkpoint**: MVP — sign in, capture top-level tasks and sub-tasks, see them in the list. Deployable/demoable.

---

## Phase 4: User Story 2 - Review existing tasks (Priority: P2)

**Goal**: A signed-in user can browse the task tree, expand/collapse branches, and open a task to read its details.

**Independent Test**: With nested tasks present, confirm the hierarchy renders, branches expand/collapse, tapping a task shows its detail (title, description, status, sub-tasks), and an account with no tasks shows the empty state.

### Implementation for User Story 2

- [ ] T022 [US2] Add expand/collapse to the tree — manage an expanded-id `Set` in `services/todo-web/src/pages/TaskTreePage.tsx` and render chevrons + conditional children in `services/todo-web/src/components/TreeRow.tsx` (FR-006)
- [ ] T023 [P] [US2] Implement `services/todo-web/src/components/EmptyState.tsx` and show it in `TaskTreePage` when the user has no tasks, using the whimsical empty-list copy (FR-016)
- [ ] T024 [US2] Implement `services/todo-web/src/pages/TaskDetailPage.tsx` — call `GetTask`, display title/description/completion status and the task's sub-tasks; map `NotFound` to the "wandered off" message (FR-007; contracts/task-service.md)
- [ ] T025 [US2] Wire navigation — tapping a `TreeRow` routes to `/tasks/:id`; back returns to `/tasks` with expand state preserved (lift expand state or persist in the URL/session as needed)

**Checkpoint**: Browsing and detail viewing work end-to-end alongside US1.

---

## Phase 5: User Story 3 - Update and complete tasks (Priority: P3)

**Goal**: A signed-in user can edit a task's title/description and toggle its completion, with the parent/child guard surfaced.

**Independent Test**: Edit a task's title/description and save (its due/parent are preserved); discard restores originals; toggle a leaf complete/incomplete; completing a parent with an incomplete sub-task is blocked with a friendly message.

### Tests for User Story 3

- [ ] T026 [P] [US3] Unit test the `UpdateTask` payload builder in `services/todo-web/src/lib/updatePayload.test.ts` — verifies edited name/description plus **preserved** `due` and `parentId` (full-replace; research R6)
- [ ] T027 [P] [US3] Interaction test in `services/todo-web/src/pages/TaskDetailPage.test.tsx` — empty-title save is blocked (FR-010) and a `FailedPrecondition` on complete renders the "finish its sub-tasks first" message (FR-017)

### Implementation for User Story 3

- [ ] T028 [P] [US3] Implement the full-replace payload builder in `services/todo-web/src/lib/updatePayload.ts` — from the loaded task, build `UpdateTaskRequest` resending `due` and `parentId` unchanged (data-model.md, contracts/task-service.md)
- [ ] T029 [US3] Add edit mode to `services/todo-web/src/pages/TaskDetailPage.tsx` — reuse `TaskForm` for editing, call `UpdateTask` via T028, invalidate `GetTask(id)` + `ListTasks` on save; discard restores original values (FR-011)
- [ ] T030 [US3] Implement the completion toggle in `services/todo-web/src/components/TreeRow.tsx` and `TaskDetailPage.tsx` — call `CompleteTask`/`UncompleteTask`, reflect completed state in tree + detail, and map `FailedPrecondition` (complete → "finish its sub-tasks first"; uncomplete → "reopen its parent first") to playful messages (FR-012, FR-017; contracts/task-service.md)

**Checkpoint**: All three user stories are independently functional.

---

## Phase 6: Polish & Cross-Cutting Concerns

**Purpose**: Production deployment, mobile/consistency/tone review, and validation.

- [ ] T031 Verify the production build — `npm run build` in `services/todo-web/` emits static assets to `dist/`; confirm bundle is reasonable for mobile
- [ ] T032 Update the Caddy config in `deploy/roles/app/templates/Caddyfile.j2` — serve the SPA (`root /srv/www`, `try_files {path} /index.html`, `file_server`) and reverse-proxy `/auth/*`, `/task.v1.*`, `/health.v1.*` to `todo_server:8080` on one origin (plan.md → Deployment)
- [ ] T033 Update `deploy/roles/app/templates/compose.yaml.j2` and `deploy/roles/app/tasks/main.yml` — add the frontend build/ship step (`npm ci && npm run build`) and bind-mount `dist/` into Caddy at `/srv/www`
- [ ] T034 [P] Mobile QA pass at 320–430px widths — no horizontal scroll or pinch-zoom, touch targets ≥44px across login, tree, detail, and forms (SC-004)
- [ ] T035 [P] Tone review (Principle IV) — confirm every user-facing string flows through `src/theme/messages.ts` and reads warm + actionable; no stray dry/terse copy
- [ ] T036 [P] UI consistency review (Principle III) — confirm pages compose shared components/tokens only; no per-page one-off colors or spacing (contracts/ui-contract.md)
- [ ] T037 Run the `quickstart.md` end-to-end validation, including the cross-client check (create a task in the TUI, refresh the web app — it appears) (SC-005)
- [ ] T038 [P] Update repo docs — note `services/todo-web/` and its `dev`/`gen`/`build`/`test` scripts in `CLAUDE.md` (and a `services/todo-web/README.md`)

---

## Dependencies & Execution Order

### Phase Dependencies

- **Setup (Phase 1)**: No dependencies — start immediately. T006 (codegen) depends on T001/T002.
- **Foundational (Phase 2)**: Depends on Setup. BLOCKS all user stories. T007→T008→T009 are sequential; T010–T013 are parallel.
- **User Stories (Phase 3–5)**: All depend on Foundational. US1 is the MVP. US2 and US3 build on US1's `TaskTreePage`/`TreeRow` (shared files), so they are most safely done in priority order rather than fully parallel.
- **Polish (Phase 6)**: Depends on the desired user stories being complete.

### User Story Dependencies

- **US1 (P1)**: Only needs Foundational. Self-contained MVP.
- **US2 (P2)**: Needs Foundational; extends US1's `TaskTreePage`/`TreeRow` (expand/collapse, detail page). Independently testable but touches US1 files.
- **US3 (P3)**: Needs Foundational; extends `TaskDetailPage` (US2) and `TreeRow` (US1) with edit + completion. Independently testable.

### Within Each User Story

- Pure-logic helpers (and their tests) before the components that use them.
- Components before the pages that compose them; pages before the flows that wire mutations.

### Parallel Opportunities

- Setup: T003, T004, T005 in parallel (distinct config files) after T001/T002.
- Foundational: T010, T011, T012, T013 in parallel (distinct files) after T007–T009.
- US1: T014 + T015 (tree builder + its test) in parallel with each other; then UI tasks.
- US3: T026, T027, T028 in parallel (distinct files) before T029/T030.
- Polish: T034, T035, T036, T038 in parallel.

---

## Parallel Example: Foundational shared assets

```bash
# After the transport/providers/routing chain (T007→T008→T009):
Task: "Create theme tokens in services/todo-web/src/theme/tokens.ts"          # T010
Task: "Create tone strings in services/todo-web/src/theme/messages.ts"        # T011
Task: "Build Button/Field/Spinner in services/todo-web/src/components/"        # T012
Task: "Build ErrorBanner in services/todo-web/src/components/ErrorBanner.tsx"  # T013
```

---

## Implementation Strategy

### MVP First (User Story 1 only)

1. Phase 1: Setup → 2. Phase 2: Foundational → 3. Phase 3: US1.
4. **STOP and VALIDATE**: sign in, capture a task + sub-task on a mobile viewport.
5. Deploy/demo (this is the core "capture before I forget" value).

### Incremental Delivery

1. Setup + Foundational → foundation ready.
2. US1 → test → deploy (MVP).
3. US2 (browse + detail + empty state) → test → deploy.
4. US3 (edit + completion toggle) → test → deploy.
5. Polish → production deploy via Caddy/Ansible.

---

## Notes

- **No backend changes**: every task is under `services/todo-web/` or `deploy/`. Do not edit `services/todo/` source or proto.
- **Single origin is mandatory**: never call `:8080` cross-origin — the `SameSite=Strict` cookie won't be sent (research R1/R2).
- **Edit = full replace**: T028/T029 must resend `due` + `parentId` or they get wiped (research R6).
- **Generated code is read-only**: regenerate `src/gen/` via `npm run gen` if the proto changes (T006).
- Commit after each task or logical group; validate at each checkpoint.
