# Research: Download CLI Binary from Web App

**Feature**: 047-download-cli-binary | **Date**: 2026-06-10

The spec's clarifications already fixed the headline decisions (server-hosted, built into
the container image, served from an authenticated same-origin endpoint; a dedicated
download page linked from the header). This document resolves the remaining
implementation-level unknowns.

## R1. How does the `linux/amd64` CLI binary get into the server image?

**Decision**: Extend the existing multi-stage `services/twig/Dockerfile`. The builder stage
already has the Go toolchain and the `api/` + `services/twig/` modules. Add the root module
source (`go.mod`/`go.sum`, `cmd/twig`, `internal/`) and cross-compile the CLI in the same
builder stage:

```dockerfile
RUN CGO_ENABLED=0 GOOS=linux GOARCH=amd64 \
    go build -ldflags "-X main.version=${CLI_VERSION}" -o /cli/twig ./cmd/twig
```

The final distroless stage copies `/cli/` into the image alongside `/server`.

**Rationale**: The server already runs on `linux/amd64`, so the build host architecture
matches the target — no QEMU or separate cross-build infrastructure. Reusing the existing
builder stage keeps one Dockerfile and one image. The root module depends only on `api/`
(via the local `replace` directive), which is already present in the build context at
`/app/api`, so the module graph resolves with `GOWORK=off` (already set in the Dockerfile).

**Alternatives considered**:
- *Separate `go:embed` into the server binary*: rejected — bloats the server binary by the
  full CLI size and couples the two artifacts at compile time for no operational gain over a
  sibling file in the same image.
- *External hosting (GitHub Releases)*: rejected in the clarify step — adds a release
  pipeline and an unauthenticated origin, contradicting FR-009.
- *Build the CLI on the host and `COPY` a prebuilt binary*: rejected — relies on a host
  build step outside the reproducible image build; the in-image build keeps `make build`
  self-contained.

## R2. Where does the displayed version come from?

**Decision**: A build-time `CLI_VERSION` build-arg, injected into the binary via
`-ldflags -X` **and** written to a plain text file `/cli/version` in the image. The server
reads `/cli/version` at startup. `compose.yaml` passes the arg (defaulting to `dev`); the
`make build` target and deployment can pass `git describe --tags --always`.

**Rationale**: Keeps the server decoupled from the binary's internal symbols — it reads a
file rather than parsing the ELF. The version is informational (FR-004/FR-007); it need not
be semver. Defaulting to `dev` means a plain `make dev` still works without git plumbing.

**Alternatives considered**:
- *Server parses the version out of the binary*: rejected — fragile and unnecessary.
- *Hardcoded version constant*: rejected — would drift and violates FR-007 ("reflect the
  currently published version" without manual intervention).

## R3. How are the checksum and size determined and displayed?

**Decision**: The **server computes** the SHA-256 and byte size from the actual binary file
at startup (cached in memory) and exposes them via the metadata endpoint. The checksum is
computed over the exact bytes that `/cli/download` streams.

**Rationale**: Guarantees SC-002 (displayed checksum always matches delivered bytes) by
construction — there is a single source of truth (the file on disk), not a separately
generated manifest that could drift. SHA-256 is the conventional integrity hash for binary
downloads and is verifiable client-side with `sha256sum`.

## R4. What endpoints serve the binary and its metadata?

**Decision**: Two plain-HTTP endpoints registered on the existing auth-wrapped `taskMux`:

- `GET /cli/info` → JSON metadata (version, filename, size, sha256, os, arch).
- `GET /cli/download` → streams the binary with `Content-Type: application/octet-stream`
  and `Content-Disposition: attachment; filename="twig"`.

Both sit behind `auth.Middleware` automatically because everything except `/auth/` is
wrapped (`main.go`: `mux.Handle("/", auth.Middleware(queries)(taskMux))`). The same-origin
session cookie is sent on both the metadata fetch and the download navigation.

**Rationale**: A binary stream cannot be a ConnectRPC unary response, so plain HTTP is
required for the download regardless. Keeping the metadata as a sibling plain-HTTP JSON
endpoint (mirroring the existing `/auth/*` plain-HTTP pattern) is simpler than introducing a
new proto service + generated stubs for a single read-only call. The HTTP contract is
documented in `contracts/cli-download-api.md` to satisfy Constitution Principle II
(API-First).

**Alternatives considered**:
- *A ConnectRPC `CliService.GetInfo`*: rejected for simplicity (Principle I) — the download
  must be plain HTTP anyway, so splitting metadata into ConnectRPC adds a proto + `make
  proto` regen for no benefit.
- *Embedding metadata in `/cli/download` response headers only*: rejected — the page must
  show version/checksum **before** the user downloads, so a separate metadata read is needed.

## R5. How does the web app reach these endpoints (dev vs. prod)?

**Decision**: Add `"/cli": "http://localhost:8080"` to the Vite dev proxy so the SPA and the
binary endpoints share one origin in development (matching the existing `/auth`, `/task.v1`,
etc. proxy entries). In production the twig server already fronts the same origin, so no
change is needed there. The metadata call uses a plain `fetch("/cli/info", { credentials:
"include" })` (same pattern as `AppHeader`'s `/auth/logout` call); the download is a plain
`<a href="/cli/download">`.

**Rationale**: The `SameSite=Strict` session cookie requires one origin; the proxy preserves
that in dev exactly as it does for the existing endpoints.

## R6. Error / unavailable-binary behavior (FR-006)

**Decision**: At startup the server attempts to stat + hash the binary. If it is missing,
the metadata endpoint returns `503` with a small JSON error, and `/cli/download` returns
`503` (never an empty/zero-byte body). The download page renders a playful error state
(Principle IV) using the existing `ErrorBanner` component.

**Rationale**: Fails loudly and never serves a corrupt/partial file, satisfying FR-006 and
SC-004.

## Summary of decisions

| # | Decision |
|---|----------|
| R1 | Cross-compile the CLI in the existing Dockerfile builder stage; copy `/cli/` into the distroless image |
| R2 | Version via `CLI_VERSION` build-arg → `/cli/version` file, read by the server (defaults to `dev`) |
| R3 | Server computes SHA-256 + size from the on-disk binary at startup (single source of truth) |
| R4 | Two plain-HTTP endpoints on the auth-wrapped mux: `GET /cli/info` (JSON), `GET /cli/download` (stream) |
| R5 | Add `/cli` to the Vite dev proxy; plain `fetch` + `<a>` from the SPA; prod already same-origin |
| R6 | Missing binary → `503` + friendly error state; never serve an empty/corrupt file |
