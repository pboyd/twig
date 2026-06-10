# Data Model: Download CLI Binary from Web App

**Feature**: 047-download-cli-binary | **Date**: 2026-06-10

This feature introduces **no database entities and no schema migrations**. The only data is
a small, in-memory description of the binary the server already has on disk. It is computed
once at server startup and served read-only.

## Entity: CLI Binary Metadata

Describes the single downloadable artifact. Derived from the binary file in the image and
the build-time version file; held in memory by the server.

| Field | Type | Source | Description |
|-------|------|--------|-------------|
| `version` | string | `/cli/version` file (build-arg `CLI_VERSION`) | Human-readable version label of the offered binary (e.g., a `git describe` string; `dev` in local builds). |
| `filename` | string | constant | Suggested download filename: `twig`. |
| `size` | int64 | `os.Stat` of the binary | Size of the binary in bytes. |
| `sha256` | string | SHA-256 over the binary bytes | Lowercase hex digest for user-side integrity verification. |
| `os` | string | constant | Target operating system: `linux`. |
| `arch` | string | constant | Target architecture: `amd64`. |
| `label` | string | derived | Human-readable platform label: `Linux (x86-64)`. |

### Validation / invariants

- `sha256` MUST be computed over the exact bytes streamed by `/cli/download` (single source
  of truth — satisfies SC-002).
- If the binary file is absent or unreadable at startup, the metadata is considered
  **unavailable**; endpoints respond `503` rather than serving partial/empty data (FR-006).
- Metadata is immutable for the lifetime of the server process; it changes only when a new
  image is built and deployed (FR-007 — a new version ships with a new image).

### Lifecycle

```text
server start ──► read /cli/version ──► stat + sha256 /cli/twig ──► cache metadata in memory
                                   └► (file missing) ──► mark unavailable ──► endpoints 503
```

No state transitions occur at runtime; the metadata is load-once, read-many.

## Relationships

None. This metadata is standalone and unrelated to tasks, plans, users, or sessions. Access
is gated by the existing auth middleware (any authenticated user may read it and download).
