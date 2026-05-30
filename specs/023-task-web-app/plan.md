# Implementation Plan: Task Web App

**Branch**: `023-task-web-app` | **Date**: 2026-05-30 | **Spec**: [spec.md](./spec.md)

**Input**: Feature specification from `specs/023-task-web-app/spec.md`

**Design source**: [docs/plans/2026-05-30-task-web-app-design.md](../../docs/plans/2026-05-30-task-web-app-design.md) (Approved)

## Summary

A mobile-first React + TypeScript single-page app for capturing and managing tasks away from the terminal. It reuses the existing Go backend without changes: it talks to the existing `task.v1.TaskService` ConnectRPC endpoints directly from the browser and authenticates with the existing session-cookie mechanism (feature 003-user-auth). A single Caddy origin serves the static SPA and reverse-proxies `/auth/*` and the ConnectRPC routes to the Go server, which is what makes the `SameSite=Strict` session cookie usable from the browser. Scope is task management only — view tree, view detail, add task/sub-task, edit, and toggle completion. No backend, proto, or schema changes.

## Technical Context

**Language/Version**: TypeScript 5.x on React 19 (frontend, new). Go backend unchanged.

**Primary Dependencies**: React 19, Vite 6, Tailwind CSS v4, `@connectrpc/connect-web`, `@connectrpc/connect-query`, `@tanstack/react-query` v5, React Router v7. Codegen via `buf generate` with `@bufbuild/protoc-gen-es` + `protoc-gen-connect-query`.

**Storage**: None client-side. All persistence is the existing backend (PostgreSQL via the Go service). No local/offline storage.

**Testing**: Vitest + React Testing Library for component/logic tests (the tree-building and edit-payload helpers are the highest-value unit targets). No new backend tests — the backend is unchanged.

**Target Platform**: Modern evergreen mobile browsers (primary); desktop browsers (usable, unpolished).

**Project Type**: Web application — new frontend SPA (`services/todo-web/`) over the existing backend (`services/todo/`).

**Performance Goals**: Returning user captures a task in <20s (SC-001); small production bundle for fast load over mobile networks; UI interactions feel instant via React Query cache + optimistic-free refetch.

**Constraints**: Online-only (no service worker / offline queue); mobile-first layout with touch-sized targets, no horizontal scroll or pinch-zoom (SC-004); single-origin deployment is mandatory because the session cookie is `SameSite=Strict`.

**Scale/Scope**: ~1–2 users per instance (per 003-user-auth). 3 routes (`/login`, `/tasks`, `/tasks/:id`), one task tree, a handful of shared components.

## Constitution Check

*GATE: Must pass before Phase 0 research. Re-check after Phase 1 design.*

| Principle | Status | Notes |
|-----------|--------|-------|
| I. Simplicity / YAGNI | ✅ | Standard, minimal SPA stack; each dependency maps to a concrete need (routing, typed RPC, server-state cache, mobile styling). No state library beyond React Query; no offline layer; no new backend code or abstractions. |
| II. API-First Design | ✅ | No new API. Contracts are the existing `task.v1` proto plus the existing `/auth/*` HTTP endpoints, restated in `contracts/` before implementation. Frontend conforms to them. |
| III. UI/UX Consistency | ✅ | This is the first web surface; the plan establishes a single shared Tailwind theme (tokens) + small shared component set (Button, Field, TreeRow, etc.). Per-page one-off styles are prohibited (see `contracts/ui-contract.md`). |
| IV. Playful User Messages | ✅ | Empty state, validation, connectivity/expiry errors, and the sub-task-blocks-completion message all carry a light, warm tone while staying actionable. Tone strings are centralized for review. |

No violations → Complexity Tracking table omitted.

## Project Structure

### Documentation (this feature)

```text
specs/023-task-web-app/
├── plan.md              # This file
├── research.md          # Phase 0 output
├── data-model.md        # Phase 1 output
├── quickstart.md        # Phase 1 output
├── contracts/           # Phase 1 output
│   ├── task-service.md  # TaskService RPCs used by the web app (from existing proto)
│   ├── auth.md          # /auth/login + /auth/logout HTTP contract
│   └── ui-contract.md   # Shared theme + component contract (Principle III/IV)
└── tasks.md             # Phase 2 (/speckit-tasks — not created here)
```

### Source Code (repository root)

```text
services/todo-web/                 # NEW — React SPA, sibling to services/todo/
├── index.html
├── package.json
├── vite.config.ts                 # dev proxy: /auth, /task.v1, /health.v1 → :8080
├── tsconfig.json
├── tailwind.config.ts
├── buf.gen.web.yaml               # TS codegen (es + connect-query) → src/gen/
└── src/
    ├── main.tsx                   # React Query + Connect transport + Router providers
    ├── App.tsx                    # route table, auth guard
    ├── gen/                       # generated TS from proto (do not edit by hand)
    ├── lib/
    │   ├── transport.ts           # connect-web transport (credentials: include, 401 → /login)
    │   └── tree.ts                # flat ListTasks → nested tree builder (unit-tested)
    ├── theme/                     # shared tokens + tone strings (Principle III/IV)
    ├── components/                # Button, Field, TreeRow, TaskForm, EmptyState, ...
    └── pages/
        ├── LoginPage.tsx
        ├── TaskTreePage.tsx       # /tasks
        └── TaskDetailPage.tsx     # /tasks/:id

services/todo/                     # UNCHANGED (backend, proto, db)
```

**Structure Decision**: New `services/todo-web/` directory, a sibling to the Go module `services/todo/`, matching the existing monorepo convention (one directory per service under `services/`). The backend is not modified; the frontend consumes the existing proto and auth endpoints.

### Deployment / Ops (existing Ansible + Caddy)

- Add an `npm ci && npm run build` step to the deploy playbook; ship `services/todo-web/dist/` to the host and bind-mount it into the Caddy container at `/srv/www`.
- Update the Caddyfile to serve the SPA (`try_files {path} /index.html`) and reverse-proxy `/auth/*`, `/task.v1.*`, `/health.v1.*` to `todo_server:8080` (single origin — required for the `SameSite=Strict` cookie).

## Complexity Tracking

> No Constitution Check violations — section intentionally empty.
