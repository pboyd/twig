---
description: "Task list for Download CLI Binary from Web App"
---

# Tasks: Download CLI Binary from Web App

**Input**: Design documents from `/specs/047-download-cli-binary/`

**Prerequisites**: plan.md, spec.md, research.md, data-model.md, contracts/cli-download-api.md

**Tests**: Included — the plan's Testing section explicitly calls for server handler tests and web page tests.

**Organization**: Tasks are grouped by user story (US1 = download the binary, US2 = verify version/checksum) so each can be implemented and tested independently.

## Format: `[ID] [P?] [Story] Description`

- **[P]**: Can run in parallel (different files, no dependencies on incomplete tasks)
- **[Story]**: US1 or US2 (Setup/Foundational/Polish carry no story label)

## Path Conventions

Full-stack repo: server module at `services/twig/`, web SPA at `services/twig-web/`, CLI source at repo root (`cmd/twig`, `internal/`). Build via `services/twig/Dockerfile` + `compose.yaml`.

---

## Phase 1: Setup (Build Wiring)

**Purpose**: Wire the build-time version argument so the image build can stamp a version.

- [X] T001 [P] Add a `CLI_VERSION` build arg (default `dev`) to the `server` service `build` block in `compose.yaml`
- [X] T002 [P] Update the `build` target in `Makefile` to pass `--build-arg CLI_VERSION=$(git describe --tags --always)`

---

## Phase 2: Foundational (Blocking Prerequisites)

**Purpose**: Produce the binary inside the server image and create the shared handler that loads its metadata. Both user stories depend on these.

**⚠️ CRITICAL**: No user story work can begin until this phase is complete.

- [X] T003 Modify `services/twig/Dockerfile`: in the builder stage, COPY the root module source (`go.mod`, `go.sum`, `cmd/`, `internal/`) and cross-compile `CGO_ENABLED=0 GOOS=linux GOARCH=amd64 go build -ldflags "-X main.version=${CLI_VERSION}" -o /cli/twig ./cmd/twig`; write `${CLI_VERSION}` to `/cli/version`; declare `ARG CLI_VERSION=dev`; in the final distroless stage `COPY --from=builder /cli /cli`
- [X] T004 Create `services/twig/internal/handler/cli.go`: define the `CLIBinary` struct and `NewCLIBinary(dir string) *CLIBinary` loader that locates `<dir>/twig`, computes size via `os.Stat`, computes the lowercase-hex SHA-256 of the bytes, reads `<dir>/version` (trimmed; falls back to `dev`), and records an `available` flag (false when the binary is missing/unreadable). Hold computed metadata (version, filename `twig`, os `linux`, arch `amd64`, label `Linux (x86-64)`, size, sha256) in memory per [data-model.md](./data-model.md)

**Checkpoint**: The image ships a `linux/amd64` `twig` binary at `/cli/twig` and the server can describe it. Endpoints are added per story below.

---

## Phase 3: User Story 1 - Download the CLI binary (Priority: P1) 🎯 MVP

**Goal**: A signed-in user opens the download page from the header and downloads a runnable Linux (x86-64) binary.

**Independent Test**: Sign in → click **Download** in the header → click the download button → receive a runnable `twig` binary (`./twig --help`). Also: `curl -H "Authorization: Bearer $KEY" /cli/download` returns the binary; without a credential it returns `401`.

### Implementation for User Story 1

- [X] T005 [US1] Add a `ServeDownload` method to `services/twig/internal/handler/cli.go` that streams the binary with `Content-Type: application/octet-stream`, `Content-Disposition: attachment; filename="twig"`, and `Content-Length`; returns `503` (never an empty/partial body) when `available` is false (FR-006)
- [X] T006 [US1] In `services/twig/cmd/server/main.go`, instantiate `handler.NewCLIBinary("/cli")` and register `GET /cli/download` on `taskMux` (inherits `auth.Middleware`, so it is auth-gated per FR-008/FR-009)
- [X] T007 [P] [US1] Handler test in `services/twig/internal/handler/cli_test.go`: build a temp-dir fixture (`twig` + `version` files), assert `ServeDownload` returns the exact bytes with the attachment header, and returns `503` when the binary is absent
- [X] T008 [P] [US1] Add `"/cli": "http://localhost:8080"` to the dev proxy in `services/twig-web/vite.config.ts` (same-origin for the `SameSite=Strict` cookie)
- [X] T009 [P] [US1] Add download-page copy to `services/twig-web/src/theme/messages.ts` (warm/playful heading, helper text, button label, and the `Linux (x86-64)` platform label) per Constitution Principle IV
- [X] T010 [US1] Create `services/twig-web/src/pages/DownloadPage.tsx`: render inside `AppHeader`, show the static `Linux (x86-64)` label and copy, and offer the download via `<a href="/cli/download" download>` styled with the existing `Button`
- [X] T011 [US1] Add `<Route path="/download" element={<DownloadPage />}>` in `services/twig-web/src/App.tsx` and a `Download` `NavLink` in `services/twig-web/src/components/AppHeader.tsx` matching the existing Tasks/Plan link styling
- [X] T012 [P] [US1] Test `services/twig-web/src/pages/DownloadPage.test.tsx`: asserts the platform label renders and the download link points at `/cli/download`

**Checkpoint**: Download works end to end — this is the MVP and is independently demoable.

---

## Phase 4: User Story 2 - Verify what I'm downloading (Priority: P2)

**Goal**: The download page shows the binary's version, size, and SHA-256 so users can verify integrity before running it.

**Independent Test**: Open the download page → it displays version, size, and a SHA-256 that equals `sha256sum` of the downloaded file. If `/cli/info` fails, a friendly error state appears instead.

### Implementation for User Story 2

- [X] T013 [US2] Add a `ServeInfo` method to `services/twig/internal/handler/cli.go` returning the metadata JSON (`version`, `filename`, `os`, `arch`, `label`, `size`, `sha256`) per [contracts/cli-download-api.md](./contracts/cli-download-api.md); returns `503` with `{"error":"binary unavailable"}` when unavailable
- [X] T014 [US2] Register `GET /cli/info` on `taskMux` in `services/twig/cmd/server/main.go` (reuses the `CLIBinary` instance from T006)
- [X] T015 [P] [US2] Handler test in `services/twig/internal/handler/cli_test.go`: assert `/cli/info` JSON fields are correct and that `sha256` matches a hash computed over the same fixture bytes (guards SC-002); assert `503` when the binary is absent
- [X] T016 [P] [US2] Create `services/twig-web/src/lib/cliInfo.ts`: a typed `fetchCliInfo()` wrapper calling `fetch("/cli/info", { credentials: "include" })`, returning the metadata type and throwing on non-200
- [X] T017 [P] [US2] Unit test `services/twig-web/src/lib/cliInfo.test.ts`: mocks fetch for success (parsed metadata) and failure (throws)
- [X] T018 [P] [US2] Add loading/error copy for the metadata fetch to `services/twig-web/src/theme/messages.ts` (playful "couldn't fetch the download just now" message) per Constitution Principle IV
- [X] T019 [US2] Enhance `services/twig-web/src/pages/DownloadPage.tsx` to call `fetchCliInfo()`, show a `Spinner` while loading, render version + human-readable size + SHA-256 on success, and render `ErrorBanner` with the playful copy on failure/`503`
- [X] T020 [P] [US2] Update `services/twig-web/src/pages/DownloadPage.test.tsx`: mock `/cli/info` success (version + checksum shown) and failure (error state shown, download still offered or gated per the rendered design)

**Checkpoint**: Both stories work independently — download (US1) and verification (US2).

---

## Phase 5: Polish & Cross-Cutting Concerns

**Purpose**: Validation and consistency checks across both stories.

- [X] T021 Run the [quickstart.md](./quickstart.md) end-to-end validation: `make dev`, `curl` `/cli/info` + `/cli/download` (verify `sha256sum` match and `401` without auth), then the web flow at `:5173`
- [X] T022 [P] Tone review of all new web copy in `services/twig-web/src/theme/messages.ts` against Constitution Principle IV
- [X] T023 [P] Update the Frontend section of `CLAUDE.md` to note the new `/download` page and that the server now exposes `/cli/*` endpoints consumed by the web app

---

## Dependencies & Execution Order

### Phase Dependencies

- **Setup (Phase 1)**: No dependencies — can start immediately.
- **Foundational (Phase 2)**: T003 depends on T001/T002 (build arg) only loosely (the arg defaults to `dev`, so T003 can proceed in parallel, but stamping a real version needs T001/T002). T004 has no dependencies.
- **User Stories (Phase 3, 4)**: Both depend on Foundational (T003 produces the binary; T004 provides the loader). US1 and US2 can then proceed in parallel if staffed.
- **Polish (Phase 5)**: Depends on the user stories being complete.

### User Story Dependencies

- **US1 (P1)**: Depends only on Foundational. Fully independent MVP — does not call `/cli/info`.
- **US2 (P2)**: Depends only on Foundational. Adds `/cli/info` and augments the existing download page; it reuses the `CLIBinary` instance wired in T006 and the page created in T010, so run US1 first if working sequentially.

### Within Each User Story

- Server method (cli.go) → route registration (main.go) → handler test.
- Web: messages + lib (parallel) → page → route/header → page test.

### Parallel Opportunities

- T001 and T002 (Setup) run in parallel.
- Within US1: T007, T008, T009, T012 are parallel (distinct files); T005→T006 sequential (same file then main.go); T010→T011 sequential (page before route import).
- Within US2: T015, T016, T017, T018, T020 are parallel (distinct files); T013→T014 sequential; T019 depends on T016 + T018.
- With two developers, US1 and US2 can be built concurrently once Phase 2 is done (coordinate the shared edits to `cli.go` and `main.go`).

---

## Parallel Example: User Story 1

```bash
# After T005/T006 land the download endpoint, run these together:
Task: "Handler test for /cli/download in services/twig/internal/handler/cli_test.go"   # T007
Task: "Add /cli proxy entry in services/twig-web/vite.config.ts"                        # T008
Task: "Add download-page copy in services/twig-web/src/theme/messages.ts"               # T009
Task: "DownloadPage render/link test in services/twig-web/src/pages/DownloadPage.test.tsx" # T012
```

---

## Implementation Strategy

### MVP First (User Story 1 only)

1. Phase 1: Setup (build arg wiring).
2. Phase 2: Foundational (binary in image + metadata loader) — **blocks everything**.
3. Phase 3: User Story 1 — header link → page → download.
4. **STOP and VALIDATE**: sign in, download the binary, run `./twig --help`.
5. Demo/deploy the MVP.

### Incremental Delivery

1. Setup + Foundational → binary ships and is described.
2. US1 → download works (MVP). Test independently → demo.
3. US2 → version + checksum verification. Test independently → demo.

---

## Notes

- [P] = different files, no incomplete dependencies.
- `cli.go` and `main.go` are each touched by both stories — sequence those edits; do not parallelize same-file tasks.
- The server computes the SHA-256 over the exact bytes it streams, so the displayed checksum always matches the download (SC-002) — keep that single source of truth.
- Endpoints sit behind `auth.Middleware` automatically (everything except `/auth/` is wrapped); no per-endpoint auth code is needed.
- Commit after each task or logical group.
