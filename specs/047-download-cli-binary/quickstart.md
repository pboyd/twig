# Quickstart: Download CLI Binary from Web App

**Feature**: 047-download-cli-binary

How to build, run, and verify the feature end to end.

## Prerequisites

- `podman` / `podman-compose` (for the server + DB stack)
- Node 20+ and `npm` (for the web dev server)
- A provisioned user (for login)

## 1. Build & run the stack

The CLI binary is built into the server image, so a normal build produces it:

```bash
make dev    # podman-compose up -d --build --force-recreate server
```

To stamp a real version (optional — defaults to `dev`):

```bash
podman build --build-arg CLI_VERSION="$(git describe --tags --always)" \
  -t twig-server -f services/twig/Dockerfile .
```

## 2. Verify the server endpoints

Get an API key (or log in via the web app for a cookie):

```bash
# Inside the server container or a local binary with DATABASE_URL set:
./twig-server --provision-user me:secret    # prints an API key
```

Check metadata and download (bearer auth):

```bash
KEY=<api-key>

# Metadata — expect JSON with version, size, sha256, label "Linux (x86-64)"
curl -s -H "Authorization: Bearer $KEY" http://localhost:8080/cli/info | jq

# Download — expect a runnable binary
curl -s -H "Authorization: Bearer $KEY" -o twig-dl http://localhost:8080/cli/download
chmod +x twig-dl

# Integrity — the printed digest MUST equal the sha256 from /cli/info (SC-002)
sha256sum twig-dl

# Sanity — the downloaded binary runs (SC-003)
./twig-dl --help
```

Auth gate (FR-008/FR-009) — without a credential, expect `401`:

```bash
curl -s -o /dev/null -w '%{http_code}\n' http://localhost:8080/cli/info   # → 401
```

## 3. Verify the web flow

```bash
cd services/twig-web
npm install
npm run dev      # http://localhost:5173
```

1. Sign in.
2. Click **Download** in the header → lands on `/download`.
3. Confirm the page shows: platform label "Linux (x86-64)", the version, the byte size, and
   the SHA-256 checksum.
4. Click the download button → the `twig` binary downloads.
5. Verify `sha256sum` of the downloaded file matches the checksum shown on the page.

## 4. Run the tests

```bash
# Server handler tests (metadata + download, served from a temp dir fixture)
cd services/twig && go test ./...

# Web tests (download page render, download link, error state)
cd services/twig-web && npm test
```

## Done when

- [ ] `/cli/info` returns correct metadata behind auth; `401` without auth.
- [ ] `/cli/download` streams a runnable binary whose `sha256sum` matches `/cli/info`.
- [ ] The header **Download** link opens a page showing label, version, size, and checksum.
- [ ] The download button delivers the binary.
- [ ] A missing binary yields a `503` and a friendly error state (no empty file).
