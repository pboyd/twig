# Phase 0 Research: Real-World Deployment Example

**Feature**: 068-deployment-example
**Date**: 2026-07-25

All open questions from the spec's Assumptions section are resolved below. Facts about Twig's own
artifacts were read out of this repository; facts about the deployment shape come from the working
reference implementation.

---

## R1: What exactly gets deployed

**Decision**: Deploy the published multi-arch-agnostic image `ghcr.io/pboyd/twig-server`, pinned by tag,
plus `docker.io/library/postgres:17-alpine` and `docker.io/library/caddy:2-alpine`. No image is built
by the operator.

**Rationale**: `.github/workflows/release.yml` publishes `ghcr.io/<owner>/twig-server` with
`:latest` on every push to `main` and `:vX.Y.Z` on version tags. `services/twig/Dockerfile` bakes the
built SPA into `/web` and the CLI into `/cli`, so one container serves the API *and* the web UI —
there is no separate frontend service to build, sync, or host. Building from source on the operator's
workstation would drag in a private-registry push path (the single largest chunk of the reference's
complexity) for zero benefit.

**Facts established from the repo**:

| Fact | Value | Source |
|---|---|---|
| Image | `ghcr.io/pboyd/twig-server:<tag>` | `.github/workflows/release.yml` |
| Entrypoint | `/server` | `services/twig/Dockerfile` |
| Base image | `gcr.io/distroless/static-debian12` — **no shell** | `services/twig/Dockerfile` |
| Required env | `DATABASE_URL` only; fatal if unset | `services/twig/cmd/server/main.go:39` |
| Listen address | hardcoded `:8080` (not configurable) | `services/twig/cmd/server/main.go:93` |
| Migrations | run automatically on every start, from `/db/migrations` | `main.go:43-49` |
| Provisioning | `/server --provision-user name:password`, prints the API key and **exits** | `main.go:61-74` |
| Unauthenticated route | `GET /` → SPA `index.html` (everything else is behind auth) | `services/twig/cmd/server/router.go:59` |

**Alternatives rejected**:
- *Pin to `:latest` as the recommended long-term practice*: rejected. `:latest` moves under the
  operator's feet and weakens FR-023 (idempotent redeploys) — the rendered compose file is unchanged
  while the image behind the tag is not.

**Tag caveat**: the repository currently has **no version tags** (`git tag` is empty), so `:latest`
from `main` is the only tag published today. The example therefore ships `twig_image_tag: latest` as
the default, with a prominent note to pin a `vX.Y.Z` tag as soon as one exists. When that happens the
default should be changed to the newest release.

**Consequences for the design**:
- The default tag must be a real published version, and the docs must tell the operator to bump it.
- Distroless means **no container healthcheck for `twig_server`** (no shell, no curl). Liveness is
  established instead by (a) the server exiting fatally when the DB is unreachable, so a running
  container implies a reachable DB, and (b) the playbook's own HTTP smoke test.
- Provisioning a user is `podman exec twig_server /server --provision-user alice:secret` — exec'ing
  the binary directly works fine without a shell.

---

## R2: Privilege escalation with a password

**Decision**: Standard `become: true` with the password supplied at run time via
`ansible-playbook -K` (`--ask-become-pass`). Support `ansible_become_password` in the vault as the
unattended alternative. Do **not** require passwordless sudo.

**Rationale**: The request states the sudo password must be configurable. `-K` prompts once per run
and never touches disk, which is the right default for a hands-on deploy; the vaulted variable exists
for operators automating it.

**Consequences**: `Makefile` targets pass `-K`. SSH itself stays key-based (FR: operator has SSH key
access); password SSH is out of scope.

---

## R3: TLS modes and the certificate/tunnel interaction

This was the least obvious part of the design, because the reference implementation solves it with ACME **DNS-01** via a
custom Caddy build containing the Cloudflare DNS plugin — which requires building and distributing a
non-standard image, i.e. the registry machinery R1 just deleted.

**Decision**: A single `twig_tls_mode` variable with two values, and the tunnel as an orthogonal
toggle:

| `twig_tls_mode` | Caddy behavior | Requires |
|---|---|---|
| `internal` (default) | `tls internal` — Caddy's own CA issues for the domain | nothing |
| `letsencrypt` | Caddy's default automatic HTTPS (HTTP-01 / TLS-ALPN) | public DNS → host, inbound 80+443 |

`twig_tunnel_enabled` (default `false`) adds a `cloudflared` container and a second `http://<domain>`
site block in the Caddyfile for the tunnel hop.

**Rationale**:
- The stock `caddy:2-alpine` image covers both TLS modes. DNS-01 is the *only* thing that would force
  a custom image, and it buys nothing that matters here: when the tunnel is on, Cloudflare's edge
  already terminates public TLS with a publicly trusted certificate, so an origin-side Let's Encrypt
  certificate is redundant for public clients.
- `internal` as the default means the P1 story (US1) needs no public DNS, no open ports, and no
  external account — it works on a LAN VM out of the box, which is what "bare Fedora machine" implies.

**The honest limitation, to be documented (not hidden)**: `letsencrypt` + `tunnel_enabled` together
still requires inbound port 80 for the HTTP-01 challenge. An operator who is behind NAT *and* wants a
trusted certificate at the origin needs DNS-01, which this example does not ship. The documentation
will say so explicitly and point at the reference implementations's approach (custom Caddy build with a DNS plugin) as the
extension path. This satisfies SC-009 as written — each combination works *for the access pattern it
targets* — with the prerequisite stated up front:

| tls_mode | tunnel | Who can reach it | Trusted cert? |
|---|---|---|---|
| `internal` | off | LAN clients that resolve the domain to the host | no (Caddy local CA) |
| `letsencrypt` | off | anyone; needs public DNS + inbound 80/443 | yes |
| `internal` | on | anyone, via Cloudflare edge; LAN clients direct | yes for public (Cloudflare edge), no for direct LAN |
| `letsencrypt` | on | anyone, both paths; still needs inbound 80 for issuance | yes |

**Redirect-loop trap**: the tunnel hop
site must be declared with an explicit `http://<domain>` scheme. A bare `:80` block loses to the named
`:443` site's host-matched auto-redirect and produces an infinite redirect through the tunnel. This is
a hard-won detail and belongs in a template comment, not just here.

**Alternatives rejected**:
- *Ship DNS-01 with a custom Caddy image*: reintroduces build+push+registry, one provider's API token,
  and a DNS-provider dependency for every operator. Violates Principle I for a minority case.
- *Plain HTTP when Let's Encrypt is off*: rejected — the session cookie is `SameSite=Strict` and the
  app is a credentialed web UI; serving it over cleartext by default is bad guidance even on a LAN.

---

## R4: Persistence, SELinux, and the postgres UID

**Decision**: Bind-mount `/var/lib/twig/postgres` into `postgres:17-alpine`, created by the playbook
as `owner: 70, group: 70, mode: 0700`, with the `:z` SELinux relabel flag on the mount.

**Rationale**: All three details are load-bearing and were learned the hard way in the reference implementation
(`roles/base/tasks/main.yml`, `roles/app/templates/compose.yaml.j2`):
- `postgres:17-alpine` runs as uid/gid **70**, not 999 — the Debian-based image's UID. Getting this
  wrong makes `initdb` fail on first boot.
- `:z` relabels the host directory `container_share_t`; without it SELinux (enforcing by default on
  Fedora) denies the container access.
- A **bind mount rather than a named volume**, deliberately: the data lands at a predictable host path
  that `tar`, `rsync`, and `pg_dump` can see, which is what makes FR-011 and the documented backup
  procedure (SC-010) actually usable. Named volumes hide under podman's storage tree.

Caddy's `/data` (ACME account + certificates + renewal state) is bind-mounted to
`/var/lib/twig/caddy` for the same reason and to satisfy FR-018.

---

## R5: Service lifecycle and boot persistence

**Decision**: One `podman-compose` project at `/var/lib/twig/app/compose.yaml`, driven by a
`twig.service` systemd unit (`Type=oneshot`, `RemainAfterExit=yes`,
`ExecStart=/usr/bin/podman-compose up -d`), enabled for `multi-user.target`. Rootful podman.

**Rationale**: Directly proven in the reference, minus the `Requires=registry.service` dependency that R1
removes. `restart: unless-stopped` on each service handles crashes; the unit handles reboots (FR-012).

**Alternatives rejected**:
- *`podman generate systemd` / Quadlet per container*: better modern practice, but the request
  specifies podman-compose, and one unit is simpler to explain in an example.
- *Rootless podman*: cleaner isolation, but binding ports 80/443 and getting bind-mount ownership
  right add friction to a starting-point example. Explicitly out of scope in the spec.

---

## R6: Idempotence and the "did anything change" signal

**Decision**: Register the result of every rendered artifact (compose.yaml, Caddyfile, the env files,
the systemd unit) and gate `podman pull` + `podman-compose up -d` on `changed` in any of them.

**Rationale**: This is what makes a second run report zero changes and cause zero restarts (FR-023,
SC-003). Copied from the reference, which demonstrates it working.

**Preview runs (FR-025)**: `ansible-playbook --check --diff`. A known and documented caveat: check
mode on a *never-deployed* host reports failures for tasks that read host state which does not exist
yet. Check mode is for reviewing redeploys, not for dry-running a first install — the docs must say
this rather than let operators discover it.

---

## R7: Verifying the deployment actually works

**Decision**: After the stack is up, assert from the control node that
`GET https://<domain>/` returns HTTP 200, using `curl --resolve <domain>:443:<host_ip>` so the check
does not depend on the operator's own DNS. Retry with a bounded timeout (10 minutes) to cover
first-issuance latency in `letsencrypt` mode. Pass `-k` in `internal` mode (the cert is
deliberately untrusted). Fail the play if the check never passes.

**Rationale**: `GET /` is the one unauthenticated route (`router.go:59`) and returns the SPA shell, so
a 200 proves Caddy → twig_server → (implicitly) postgres are all healthy. The reference's smoke test only
checks `curl`'s exit code against a ConnectRPC endpoint, which would pass on a `401` — a weaker check
this example should not inherit. Assert on the **HTTP status code**, not just the exit code.

---

## R8: First-user provisioning

**Decision**: Not automated. The playbook's closing message tells the operator to run
`podman exec twig_server /server --provision-user <name>:<password>` and keep the printed API key.

**Rationale**: The API key is printed exactly once and is unrecoverable afterwards. Burying that in
playbook output — or, worse, in a vaulted variable the operator never reads — loses it. Making it a
deliberate manual step the operator sees is the safer design, and matches the spec's assumption.

---

## R9: Where the example lives and how it is copied

**Decision**: `deploy/` at the repository root, self-contained: `ansible.cfg`, `site.yml`,
`requirements.yml`, `Makefile`, `inventory.example.ini`, `group_vars/`, `roles/{base,twig}/`, and
`README.md`. Copy-and-own is `cp -r deploy/ ~/my-infra/twig-deploy/` plus filling in two files.

**Rationale**: FR-001 requires a single copyable directory. Keeping `ansible.cfg` *inside* it means the
copy works from its new location with no path fixing. `deploy/` beats `examples/deployment/` for
discoverability from the README (FR-031).

**Copy-and-own documentation (FR-027, FR-028)**: the README will state which files are the operator's
(`inventory.ini`, `group_vars/twig_servers/main.yml`, `vault.yml`), which are the example's logic
(roles, templates), the support posture, and how to diff against upstream later
(`git remote add upstream` + compare `deploy/`, or simply re-download and diff).

**Alternatives rejected**: a separate repository (splits the docs from the app, and the request asks
for it alongside Twig); a git submodule (worse for the copy-and-own goal, which is about divergence).

---

## R10: Automated checking in CI

**Decision**: Add `ansible-playbook --syntax-check` plus `ansible-lint` to `.github/workflows/ci.yml`
as one lightweight job. Do **not** attempt to provision a VM in CI.

**Rationale**: The spec puts automated deployment testing out of scope, and it would be slow and
flaky. But a syntax/lint gate is seconds of CI time and catches the failure mode most likely to reach
a user: a template or YAML error in an example nobody re-ran. Verification of behavior stays manual
against a throwaway Fedora VM, per the spec.

---

## Divergences from the reference, summarized

| reference | This example | Why |
|---|---|---|
| Two apps (Twig + Cardigan) | Twig only | scope |
| On-host private registry + local image builds | published `ghcr.io` image only | R1 — deletes the largest subsystem |
| Custom Caddy image with Cloudflare DNS plugin | stock `caddy:2-alpine` | R3 — no custom image to distribute |
| ACME DNS-01, Cloudflare token mandatory | HTTP-01, optional; `tls internal` default | R3 — no external account needed to start |
| Tunnel and certificates always on | both opt-in, off by default | FR-016, FR-019 |
| Smoke test checks curl exit code | asserts HTTP 200 | R7 — the old check passes on 401 |
| Vault token asserts, postgres data guards, `:z`, uid 70, redirect-loop fix, idempotence gating | **kept** | proven, load-bearing |
