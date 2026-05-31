# Implementation Plan: HTTPS with Custom Domain

**Branch**: `025-https-custom-domain` | **Date**: 2026-05-31 | **Spec**: [spec.md](./spec.md)

**Input**: Feature specification from `/specs/025-https-custom-domain/spec.md`

## Summary

Enable the existing podman-compose deployment to serve the twig stack over HTTPS at an operator-configured domain, using a publicly-trusted Let's Encrypt certificate that is obtained and renewed automatically. The current Caddy reverse proxy already terminates HTTP on port 80 and front-routes ConnectRPC/auth/SPA; this feature switches its site block from a bare `:80` listener to a named-host listener, which turns on Caddy's automatic ACME (Let's Encrypt) issuance, the automatic HTTP→HTTPS redirect, and TLS termination.

Per the user's input, **Let's Encrypt is the CA** (Caddy's default — aligns with the spec's "fixed default public CA" clarification) and **the domain is hosted in Cloudflare**. The chosen approach uses a **DNS-only ("grey-cloud") Cloudflare record** pointing at the host's public IP, so Let's Encrypt validates against the origin directly and end users are served the origin's Let's Encrypt certificate over TLS 1.3 — exactly what FR-002/FR-011/FR-012 require. This avoids a Cloudflare-proxied path that would terminate TLS at Cloudflare's edge (presenting Cloudflare's cert, not Let's Encrypt's, and obscuring the deployment's own TLS version), and avoids needing a custom Caddy build with the Cloudflare DNS plugin. See [research.md](./research.md) for the full trade-off.

New per-deployment configuration (Ansible group/host vars): `twig_domain` (required), `twig_acme_environment` (`production` default / `staging` toggle, per FR-013), and optional `twig_acme_email`. The Caddyfile and compose templates are updated; Caddy gains a persistent `/data` volume so issued certificates survive restarts (protecting against re-issuance rate limits), and the compose service publishes port 443 in addition to 80.

## Technical Context

**Language/Version**: No application code change. Infrastructure-as-config: Ansible (YAML) playbook/roles, Jinja2 templates, Caddy 2 Caddyfile syntax. No Go (`services/twig`) or TypeScript (`services/twig-web`) changes.

**Primary Dependencies**: Caddy 2 (`docker.io/library/caddy:2-alpine`, stock image — no plugins) with built-in ACME client; Let's Encrypt (ACME CA, production + staging endpoints); Cloudflare (authoritative DNS only, DNS-only record); podman + podman-compose; Ansible (`community.general`/`ansible.posix` already used).

**Storage**: Caddy ACME/certificate state persisted to a host bind mount `{{ twig_host_data_dir }}/caddy` → `/data` in the Caddy container. PostgreSQL bind mount is unchanged and must not be disturbed.

**Testing**: Ansible idempotence (a second `site.yml` run reports `changed=0` for cert/template steps and does not recreate the postgres container — existing guards in `roles/app/tasks/main.yml`); external TLS verification via `curl`/`openssl s_client` against the live domain (chain to Let's Encrypt root, SAN matches domain, TLS 1.3 negotiated, TLS 1.2 refused, HTTP→HTTPS 308 redirect). No unit-test framework applies to this layer.

**Target Platform**: Fedora host running rootful podman, publicly reachable on TCP 80 and 443, with a public DNS A/AAAA record for `twig_domain` resolving to it.

**Project Type**: Deployment / infrastructure feature (extends the existing `deploy/` Ansible role + Caddy config).

**Performance Goals**: First-time HTTPS readiness within 5 minutes of bring-up on an already-resolving domain (SC-003). Certificate renewal is automatic and ahead of expiry (Caddy renews at ~2/3 of lifetime).

**Constraints**: TLS 1.3 minimum, TLS 1.2 and below refused (FR-012). Missing/invalid `twig_domain` must fail the play fast (FR-009). Must preserve existing idempotence guarantees and the postgres data directory. Staging-mode certs are intentionally untrusted (FR-013).

**Scale/Scope**: One host, one domain per deployment.

## Constitution Check

*GATE: Must pass before Phase 0 research. Re-check after Phase 1 design.*

| Principle | Status | Notes |
|-----------|--------|-------|
| I. Simplicity / YAGNI | ✅ | Reuses Caddy's built-in ACME by changing the site address to a hostname; stock Caddy image (no custom build/plugin); 2 required + 1 optional new vars. DNS-only Cloudflare path avoids the Cloudflare-DNS-plugin/API-token complexity. No new abstractions. |
| II. API-First Design | ✅ | No request/response schema or endpoint changes (no proto, no new ConnectRPC routes). The relevant contract is the *deployment configuration contract* (Ansible vars + Caddyfile structure + externally-observable TLS behavior), defined in `contracts/` before the role/template edits. |
| III. UI/UX Consistency | ✅ | No TUI/CLI/web surface added or changed. The only new human-facing text is an Ansible `assert` failure message (operator-facing infra output), outside the scope of Principle III's application surfaces. |
| IV. Playful User Messages | ✅ | No new *application* user-facing text. Operator-facing Ansible assertion/log messages are kept clear and actionable per FR-009; Principle IV governs end-user app copy, which is untouched. |

No violations. Complexity Tracking table omitted (nothing to justify).

## Project Structure

### Documentation (this feature)

```text
specs/025-https-custom-domain/
├── plan.md              # This file
├── research.md          # Phase 0 output
├── data-model.md        # Phase 1 output — configuration schema
├── quickstart.md        # Phase 1 output — operator runbook
├── contracts/           # Phase 1 output
│   ├── deployment-config.md   # Ansible var + Caddyfile structure contract
│   └── tls-behavior.md        # Externally-observable HTTPS/TLS contract
└── tasks.md             # Phase 2 output (/speckit-tasks — NOT created here)
```

### Source Code (repository root)

This feature touches only the deployment layer. No `src/` application code changes.

```text
deploy/
├── group_vars/
│   └── all.yml                         # + twig_domain, twig_acme_environment, twig_acme_email, twig_https_port
├── roles/
│   ├── base/tasks/main.yml             # + ensure host data dir for caddy /data; (optional) firewall 80/443
│   └── app/
│       ├── tasks/main.yml              # + assert twig_domain set/valid; create caddy data dir; smoke-check over HTTPS
│       └── templates/
│           ├── Caddyfile.j2            # bare :80 site → named-host site + global ACME options + tls protocols tls1.3
│           └── compose.yaml.j2         # caddy: publish 443 + bind-mount {{ twig_host_data_dir }}/caddy:/data
└── inventories/                        # operator sets twig_domain per host/group (public IP + public DNS required)
```

**Structure Decision**: The single existing Ansible `app` role is extended in place; no new role or project is introduced (Principle I). The Caddy data directory is created under the established `{{ twig_host_data_dir }}` (`/var/lib/twig`) convention used by postgres and the app, keeping all host state in one backup-friendly location.

## Complexity Tracking

> No Constitution Check violations — nothing to justify.
