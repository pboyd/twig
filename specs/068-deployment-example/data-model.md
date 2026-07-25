# Phase 1 Data Model: Real-World Deployment Example

**Feature**: 068-deployment-example
**Date**: 2026-07-25

This feature stores no application data. Its "model" is the configuration surface the operator owns,
the state that lives on the target host, and the relationships between them. The authoritative
variable list is `contracts/config-schema.md`; the authoritative host paths are
`contracts/host-layout.md`. This document describes the entities and their rules.

---

## Entities

### Target host

The Fedora machine being deployed to.

| Attribute | Where it lives | Notes |
|---|---|---|
| address | `inventory.ini` → `ansible_host` | IP or resolvable name; also used by the smoke test's `--resolve` |
| login account | `inventory.ini` → `ansible_user` | key-based SSH; must have `sudo` |
| escalation password | prompted (`-K`) or `ansible_become_password` in vault | never written to disk by the example |
| inventory group | `[twig_servers]` | the playbook targets this group and refuses a bare `all` |

**Rules**: exactly one host per deployment (single-instance, per spec). The playbook fails fast if the
`twig_servers` group is empty or the run was not targeted explicitly.

---

### Deployment configuration

Operator-owned values, held apart from deployment logic. Three files, one purpose each.

| File | Owns | Encrypted |
|---|---|---|
| `inventory.ini` | which host, which account | no |
| `group_vars/twig_servers/main.yml` | domain, image tag, ports, toggles, DB user/name | no |
| `group_vars/twig_servers/vault.yml` | every secret | **yes** (`ansible-vault`) |
| `group_vars/all/defaults.yml` | the example's defaults — not operator-owned | no |

**Required before the first run** (4 values, satisfying SC-002 ≤ 5):
`ansible_host`, `ansible_user`, `twig_domain`, `twig_postgres_password`.

**Validation rules** (all enforced before any host change, FR-007):

| Value | Rule | On failure |
|---|---|---|
| `twig_domain` | defined, non-empty, valid hostname/FQDN — no scheme, no path, no port | fail naming the value and the file to set it in |
| `twig_postgres_password` | defined, non-empty | fail; message points at `vault.yml` |
| `twig_tls_mode` | one of `internal`, `letsencrypt` | fail listing the valid values |
| `twig_acme_email` | required and non-empty **when** `twig_tls_mode == letsencrypt` | fail explaining Let's Encrypt needs a contact address |
| `twig_tunnel_token` | required and non-empty **when** `twig_tunnel_enabled` | fail; message points at `vault.yml` |
| `twig_image_tag` | defined, non-empty | fail |

Secret validations run with `no_log: true` so the value cannot leak into output (FR-009).

---

### Secret store

One location: `group_vars/twig_servers/vault.yml`, encrypted with `ansible-vault`.

| Secret | Consumed by | Required |
|---|---|---|
| `twig_postgres_password` | postgres + the server's `DATABASE_URL` | always |
| `twig_tunnel_token` | `cloudflared` | only when `twig_tunnel_enabled` |
| `ansible_become_password` | privilege escalation | optional; `-K` prompt otherwise |

**Rules**: the repository ships `vault.example.yml` in cleartext with placeholder values only
(FR-008). Secrets reach the host only through files written `0600, root:root`, and every task that
touches one sets `no_log: true`.

---

### Service stack

Containers in one `podman-compose` project, managed by one systemd unit.

| Service | Image | Exposed on host | Depends on | Present when |
|---|---|---|---|---|
| `postgres` | `docker.io/library/postgres:17-alpine` | no | — | always |
| `twig_server` | `ghcr.io/pboyd/twig-server:<tag>` | no | `postgres` (healthy) | always |
| `caddy` | `docker.io/library/caddy:2-alpine` | `80`, `443` | `twig_server` | always |
| `cloudflared` | `docker.io/cloudflare/cloudflared:latest` | no | `caddy` | `twig_tunnel_enabled` |

**Rules**:
- Only `caddy` publishes ports (FR-013). `postgres` and `twig_server` are reachable only on the
  compose network.
- Every service carries `restart: unless-stopped`; the systemd unit covers reboots (FR-012).
- `twig_server` has **no** container healthcheck — the image is distroless and has no shell. Its
  liveness is implied (the process exits fatally without a reachable database) and confirmed by the
  playbook's HTTP smoke test.
- Toggled-off services are absent from the rendered compose file entirely, not merely stopped
  (FR-020).

---

### Persistent data

State that must outlive containers, redeploys, and reboots.

| Data | Host path | Recoverability if lost | Guard |
|---|---|---|---|
| Task database | `/var/lib/twig/postgres` | **unrecoverable** without a backup | `PG_VERSION` presence check + container-recreation check on redeploys |
| Certificate / ACME state | `/var/lib/twig/caddy` | recoverable by re-issuance, but rate-limited | persisted bind mount |
| Rendered config | `/var/lib/twig/app` | regenerated on every run | — |

**Rules**:
- The database directory is created `70:70 / 0700` before first start (the `postgres:17-alpine` UID)
  and mounted with `:z` for SELinux.
- Bind mounts, not named volumes, so ordinary host tools can back the data up (FR-011, SC-010).
- On a redeploy (detected by `/var/lib/twig/.deployed`), the play **fails** if `postgres/PG_VERSION` is
  missing or if the `postgres` container's creation timestamp is newer than the play's start — either
  means the data directory was re-initialized (FR-024).

---

### Twig release

The `twig_image_tag` value: the single thing an operator changes to upgrade (SC-004).

**State transition — upgrade**: change tag → re-run → rendered compose changes → image pulled →
`podman-compose up -d` recreates `twig_server` only → the new server applies migrations on start →
`postgres` is untouched and its recreation guard confirms it.

---

## Relationships

```text
inventory.ini ──────────────┐
group_vars/…/main.yml ──────┼──> validation gate ──> rendered artifacts ──> service stack
group_vars/…/vault.yml ─────┘    (fails before          in /var/lib/twig/app     │
                                  touching host)                                 │
                                                                                 v
                       /var/lib/twig/{postgres,caddy}  <── persistent data <── data-safety guards
                                                                                 │
                                                                                 v
                                                                    HTTP 200 smoke test
```

## State transitions

| From | Trigger | To | Guarantee |
|---|---|---|---|
| bare host | first run | running stack + `.deployed` marker | smoke test passed before success is reported |
| running stack | re-run, no config change | unchanged | zero changed tasks, zero restarts (FR-023) |
| running stack | tag change | new version running | migrations applied, data intact (FR-024) |
| running stack | toggle flipped | layer added or removed | rendered compose gains/loses a service (FR-020) |
| running stack | host reboot | running stack | systemd unit + `restart: unless-stopped` (FR-012) |
| running stack | data dir missing/relocated | **play fails** | no empty database is silently started (FR-024) |
