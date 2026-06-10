# Implementation Plan: Download CLI Binary from Web App

**Branch**: `047-download-cli-binary` | **Date**: 2026-06-10 | **Spec**: [spec.md](./spec.md)

**Input**: Feature specification from `/specs/047-download-cli-binary/spec.md`

## Summary

Let signed-in web-app users download the `linux/amd64` twig CLI binary. The binary is
cross-compiled inside the existing server container image; the server exposes two
authenticated, same-origin plain-HTTP endpoints — `GET /cli/info` (JSON metadata: version,
size, SHA-256, platform label) and `GET /cli/download` (the binary as a file attachment).
A new web page at `/download`, linked from the app header, fetches the metadata, displays
version/size/checksum for the "Linux (x86-64)" build, and offers the download. The server
computes the checksum and size from the on-disk binary at startup so the displayed checksum
always matches the delivered bytes. No database changes; no ConnectRPC/proto changes.

## Technical Context

**Language/Version**: Go 1.26 (server + CLI); TypeScript / React 19 (web)

**Primary Dependencies**: Server — stdlib `net/http`, `crypto/sha256`, existing
`internal/auth` middleware (no new Go deps). Web — existing React 19, react-router,
@connectrpc/connect-web, Tailwind; metadata fetched with plain `fetch` (no new deps).

**Storage**: N/A — no DB entities, no migrations. Binary metadata is computed once at server
startup and held in memory (see [data-model.md](./data-model.md)).

**Testing**: `cd services/twig && go test ./...` (handler tests over a temp-dir fixture,
following the `export_test.go` shim convention); `cd services/twig-web && npm test`
(download-page render / download-link / error-state).

**Target Platform**: Server runs on `linux/amd64` (distroless); the downloadable artifact is
`linux/amd64`. Web app runs in the browser, same origin as the server.

**Project Type**: Full-stack web — Go server module + React SPA + Dockerfile build change.

**Performance Goals**: Metadata is an in-memory read; download is a straight file stream. No
performance concerns. SC-001 (obtain binary in <30s, ≤2 interactions) is a UX target met by
the header link → page → download button flow.

**Constraints**: Same-origin only (the `SameSite=Strict` `todo_session` cookie must be sent),
so `/cli/*` is served by the twig server and proxied via Vite in dev. Endpoints sit behind
the existing `auth.Middleware` (everything except `/auth/` is wrapped). Must never serve an
empty/corrupt file (FR-006). New user-facing copy must follow the warm/playful tone.

**Scale/Scope**: One Dockerfile change, one small server handler (`internal/handler/cli.go`)
+ registration in `cmd/server/main.go`, one new web page + header link + messages + a Vite
proxy entry. Single target platform (linux/amd64); single "current" version.

## Constitution Check

*GATE: Must pass before Phase 0 research. Re-check after Phase 1 design.*

| Principle | Status | Notes |
|-----------|--------|-------|
| I. Simplicity / YAGNI | ✅ | No new dependencies, no DB, no proto. Plain-HTTP endpoints reuse the existing auth-wrapped mux; metadata is a load-once in-memory struct. No multi-platform abstraction (single linux/amd64 target, as specified). Adding the CLI build to the existing Dockerfile builder is the minimal way to produce the binary in-image. |
| II. API-First Design | ✅ | The HTTP contract (`/cli/info`, `/cli/download` request/response shapes) is committed in [contracts/cli-download-api.md](./contracts/cli-download-api.md) before implementation. Binary streaming cannot be ConnectRPC; a proto service for one read-only metadata call was rejected for simplicity (see research.md R4). |
| III. UI/UX Consistency | ✅ | The download page reuses existing components (`AppHeader`, `Button`, `Spinner`, `ErrorBanner`) and Tailwind tokens; the header link uses the same `NavLink` styling as Tasks/Plan. |
| IV. Playful User Messages | ✅ | New copy (page heading, helper text, error, post-download hint) added to `src/theme/messages.ts` with the warm tone; tone obligations recorded in the contract. Server error JSON is internal and mapped to friendly copy by the web app. |

**Post-Phase-1 re-check**: ✅ All four principles still hold after design. No Complexity
Tracking entries required.

## Project Structure

### Documentation (this feature)

```text
specs/047-download-cli-binary/
├── plan.md              # This file (/speckit-plan command output)
├── research.md          # Phase 0 output — build/serve/version decisions
├── data-model.md        # Phase 1 output — CLI Binary Metadata (in-memory, no DB)
├── quickstart.md        # Phase 1 output — build, verify, test walkthrough
├── contracts/
│   └── cli-download-api.md   # HTTP contract: /cli/info + /cli/download
└── tasks.md             # Phase 2 output (/speckit-tasks — NOT created here)
```

### Source Code (repository root)

```text
services/twig/
├── Dockerfile                     # MODIFIED: cross-compile cmd/twig (linux/amd64) into the
│                                  #   builder; COPY /cli/{twig,version} into the final image
├── cmd/server/main.go             # MODIFIED: register /cli/info & /cli/download on taskMux
└── internal/handler/
    ├── cli.go                     # NEW: CLIBinary handler — load metadata at startup
    │                              #   (stat + sha256 + version file), serve info + download
    ├── cli_test.go                # NEW: handler tests over a temp-dir fixture
    └── export_test.go             # MODIFIED (if needed): shim for constructing the handler

compose.yaml                       # MODIFIED: pass CLI_VERSION build arg (default "dev")
Makefile                           # MODIFIED (optional): build target stamps git describe

services/twig-web/
├── vite.config.ts                 # MODIFIED: proxy "/cli" → http://localhost:8080
├── src/App.tsx                    # MODIFIED: add <Route path="/download">
├── src/components/AppHeader.tsx    # MODIFIED: add "Download" NavLink
├── src/pages/
│   ├── DownloadPage.tsx           # NEW: fetch /cli/info, render metadata + download button
│   └── DownloadPage.test.tsx      # NEW: render / download-link / error-state tests
├── src/lib/cliInfo.ts             # NEW: typed fetch wrapper for GET /cli/info
├── src/lib/cliInfo.test.ts        # NEW: unit test for the fetch wrapper
└── src/theme/messages.ts          # MODIFIED: download page copy (playful tone)
```

**Structure Decision**: Full-stack change across the existing modules. The binary is produced
by the `services/twig` image build and served by a new plain-HTTP handler in
`services/twig/internal/handler` (sibling to the ConnectRPC handlers, but a bare
`http.Handler` since it streams a file). The web change is a self-contained page plus a header
link and a proxy entry, reusing existing components and the established `fetch(..., {
credentials: "include" })` pattern already used for `/auth/logout`. No `api/proto` or `db`
changes.

## Complexity Tracking

> No Constitution Check violations — table intentionally empty.
