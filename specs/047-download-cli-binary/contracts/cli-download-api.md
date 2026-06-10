# API Contract: CLI Binary Download

**Feature**: 047-download-cli-binary | **Date**: 2026-06-10

Two plain-HTTP endpoints served by the twig server (`services/twig`), registered on the
auth-wrapped mux so both require authentication. These are **not** ConnectRPC — they mirror
the existing plain-HTTP `/auth/*` pattern. The binary stream cannot be a ConnectRPC unary
response, and the metadata is kept alongside it for cohesion (see `research.md` R4).

All endpoints are same-origin with the web app. Authentication is satisfied by either the
`todo_session` cookie (browser) or an `Authorization: Bearer <api-key>` header, exactly as
for every other non-`/auth/` route. Unauthenticated requests receive `401` from the existing
`auth.Middleware` (FR-008, FR-009).

---

## `GET /cli/info`

Returns metadata about the downloadable binary so the web app can display version, size, and
checksum before the user downloads.

### Request

- Method: `GET`
- Auth: required (cookie or bearer)
- Body: none

### Response — 200 OK

- `Content-Type: application/json`

```json
{
  "version": "foundation-smoke-test-passed-188-gee5b721",
  "filename": "twig",
  "os": "linux",
  "arch": "amd64",
  "label": "Linux (x86-64)",
  "size": 12873728,
  "sha256": "3b1f...e9c2"
}
```

| Field | Type | Notes |
|-------|------|-------|
| `version` | string | Build version label; may be `dev` in local builds. |
| `filename` | string | Always `twig`. |
| `os` | string | Always `linux`. |
| `arch` | string | Always `amd64`. |
| `label` | string | Always `Linux (x86-64)`. |
| `size` | number | Binary size in bytes. |
| `sha256` | string | Lowercase hex SHA-256 of the binary bytes. |

### Response — 401 Unauthorized

Returned by `auth.Middleware` when no valid credential is present. Empty body (matches
existing middleware behavior).

### Response — 503 Service Unavailable

Returned when the binary is missing/unreadable at startup (FR-006).

- `Content-Type: application/json`

```json
{ "error": "binary unavailable" }
```

---

## `GET /cli/download`

Streams the binary as a file attachment.

### Request

- Method: `GET`
- Auth: required (cookie or bearer)
- Body: none

### Response — 200 OK

- `Content-Type: application/octet-stream`
- `Content-Disposition: attachment; filename="twig"`
- `Content-Length: <size>`
- Body: the exact binary bytes (the same bytes hashed by `/cli/info`).

### Response — 401 Unauthorized

From `auth.Middleware`; empty body.

### Response — 503 Service Unavailable

Returned when the binary is missing/unreadable; never returns an empty `200`/partial body
(FR-006).

---

## Web app consumption

- The download page issues `fetch("/cli/info", { credentials: "include" })` to render
  version, label, size, and checksum.
- The download action is a plain `<a href="/cli/download" download>` (browser sends the
  same-origin session cookie automatically).
- A failed `/cli/info` fetch (network error or `503`) renders the playful error state via the
  existing `ErrorBanner` (FR-006, Constitution Principle IV).
- Vite dev proxy MUST route `/cli` → `http://localhost:8080` (same-origin requirement for the
  `SameSite=Strict` cookie).

## Tone (Constitution Principle IV)

User-facing copy on the download page (heading, helper text, error, post-download hint) lives
in `services/twig-web/src/theme/messages.ts` and MUST carry the warm/playful tone, e.g.
"Grab the twig CLI" / "Couldn't fetch the download just now — give it another go?". Server
error strings are minimal JSON (`"binary unavailable"`) and are not surfaced verbatim to the
user; the web app maps them to friendly copy.
