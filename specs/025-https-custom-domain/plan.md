# Implementation Plan: HTTPS with Custom Domain (LAN / DNS-01)

**Branch**: `025-https-custom-domain` | **Date**: 2026-05-31 (revised) | **Spec**: [spec.md](./spec.md)

**Input**: Feature specification from `/specs/025-https-custom-domain/spec.md`

## Summary

Serve the twig stack over HTTPS at an operator-configured domain with a publicly-trusted Let's Encrypt certificate, **for a deployment that lives on a LAN behind a private/internal IP and is not reachable from the public internet**. Because the host cannot answer an inbound ACME challenge, certificates are obtained and renewed using the **DNS-01 challenge through the Cloudflare API**: Caddy creates the validation TXT record in the domain's Cloudflare zone using an operator-supplied API token, Let's Encrypt validates the record (never contacting the host), and the host is issued a trusted cert for its domain. LAN browsers resolve the domain to the host's internal address (split-horizon/internal DNS) and see no certificate warning.

This **reverses the prior public-reachability design** (HTTP-01/TLS-ALPN-01 over public 80/443, stock Caddy image, DNS-only-but-public Cloudflare record). The concrete deltas versus that design: (1) run a **custom Caddy image** built with the `caddy-dns/cloudflare` module (stock `caddy:2-alpine` has no DNS plugins); (2) configure Caddy global `acme_dns cloudflare {env.CLOUDFLARE_API_TOKEN}`; (3) introduce a **secret** `twig_cloudflare_api_token` delivered to the container via a `0600` env file (never in `compose.yaml`/logs); (4) set an explicit public DNS resolver for propagation checks (split-horizon safety); (5) drop the "publicly reachable on 80/443" precondition — ports 80/443 are now LAN-only, with 80 retained solely for the HTTP→HTTPS redirect; (6) make the post-deploy smoke check independent of the control node's DNS view via `curl --resolve`. The staging/production toggle, TLS 1.3 floor, `/data` certificate persistence, optional ACME email, and fail-fast domain validation all carry over. See [research.md](./research.md) for the full trade-offs (D1–D11).

## Technical Context

**Language/Version**: No application code change. Infrastructure-as-config: Ansible (YAML) playbook/roles, Jinja2 templates, Caddy 2 Caddyfile syntax, a Containerfile (xcaddy build). No Go (`services/twig`) or TypeScript (`services/twig-web`) changes.

**Primary Dependencies**: Caddy 2 built with `github.com/caddy-dns/cloudflare` (via `docker.io/library/caddy:2-builder` xcaddy stage → `docker.io/library/caddy:2-alpine` runtime); Let's Encrypt (ACME CA, production + staging endpoints) over the **DNS-01** challenge; Cloudflare (authoritative DNS **and** API for the DNS-01 challenge); podman + podman-compose; on-host OCI registry (existing); Ansible (`community.general`/`ansible.posix` already used).

**Storage**: Caddy ACME/certificate state persisted to host bind mount `{{ twig_host_data_dir }}/caddy` → `/data` (unchanged). Cloudflare API token stored in `{{ twig_host_data_dir }}/app/caddy.env` (`0600`). PostgreSQL bind mount unchanged and must not be disturbed.

**Testing**: Ansible idempotence (a second `site.yml` run reports `changed=0` for cert/template/image steps and does not recreate the postgres container); external TLS verification via `curl --resolve`/`openssl s_client` against the host's LAN address using the configured domain SNI (chain to Let's Encrypt root, SAN matches domain, TLS 1.3 negotiated, TLS 1.2 refused, HTTP→HTTPS 308 redirect). No unit-test framework applies to this layer.

**Target Platform**: Fedora host running rootful podman on a **private LAN** (e.g. `192.168.0.0/16`). The host does **not** need a public IP or inbound public connectivity. Outbound HTTPS to Let's Encrypt and the Cloudflare API is required. The domain is hosted in Cloudflare; LAN clients resolve it to the host's internal address.

**Project Type**: Deployment / infrastructure feature (extends the existing `deploy/` Ansible roles + Caddy config).

**Performance Goals**: First-time HTTPS readiness within ~10 minutes of bring-up (SC-003), allowing for DNS-01 challenge-record propagation. Renewal is automatic and ahead of expiry (Caddy renews at ~2/3 of lifetime, via DNS-01).

**Constraints**: TLS 1.3 minimum, TLS 1.2 and below refused (FR-012). Missing/invalid `twig_domain` **or** missing `twig_cloudflare_api_token` must fail the play fast (FR-009, FR-017). The API token must never appear in rendered files, logs, or status output (FR-016). Must preserve existing idempotence guarantees and the postgres data directory. Staging-mode certs are intentionally untrusted (FR-013).

**Scale/Scope**: One host, one domain per deployment.

## Constitution Check

*GATE: Must pass before Phase 0 research. Re-check after Phase 1 design.*

| Principle | Status | Notes |
|-----------|--------|-------|
| I. Simplicity / YAGNI | ✅ | DNS-01 is the minimal mechanism that works behind a private IP. New surface is small: one Containerfile (official xcaddy stage), two new vars (`twig_cloudflare_api_token`, `twig_acme_dns_resolver`) plus the existing toggles, and an env-file mount. The custom image reuses the **existing** build→registry→pull pipeline rather than adding a new delivery path. No new abstractions or roles. The added build complexity is justified in Complexity Tracking. |
| II. API-First Design | ✅ | No request/response schema or endpoint changes (no proto, no new ConnectRPC routes). The relevant contract is the *deployment configuration contract* (Ansible vars + Caddyfile structure + externally-observable TLS behavior), fixed in `contracts/` before role/template edits. |
| III. UI/UX Consistency | ✅ | No TUI/CLI/web surface added or changed. The only new human-facing text is Ansible `assert` failure messages (operator-facing infra output), outside Principle III's application surfaces. |
| IV. Playful User Messages | ✅ | No new *application* user-facing text. Operator-facing Ansible assertion/log messages stay clear and actionable per FR-009/FR-017; Principle IV governs end-user app copy, which is untouched. |

No unjustified violations. One justified complexity item (custom Caddy build) is recorded in Complexity Tracking.

## Project Structure

### Documentation (this feature)

```text
specs/025-https-custom-domain/
├── plan.md              # This file
├── research.md          # Phase 0 output (revised for DNS-01)
├── data-model.md        # Phase 1 output — configuration schema
├── quickstart.md        # Phase 1 output — operator runbook
├── contracts/           # Phase 1 output
│   ├── deployment-config.md   # Ansible var + Caddyfile + compose structure contract
│   └── tls-behavior.md        # Externally-observable HTTPS/TLS contract
└── tasks.md             # Phase 2 output (/speckit-tasks — regenerated for DNS-01)
```

### Source Code (repository root)

This feature touches only the deployment layer. No `src/` application code changes.

```text
deploy/
├── group_vars/
│   └── all.yml                         # + twig_cloudflare_api_token (no default), twig_acme_dns_resolver (default 1.1.1.1);
│                                        #   keep twig_acme_environment / twig_acme_email / ports
├── roles/
│   └── app/
│       ├── files/
│       │   └── Containerfile.caddy     # NEW: xcaddy build of caddy + caddy-dns/cloudflare
│       ├── tasks/main.yml              # + assert twig_cloudflare_api_token (no_log); build/push custom caddy image
│       │                                #   (registry-guarded); render caddy.env (0600, no_log); smoke-check via curl --resolve
│       └── templates/
│           ├── Caddyfile.j2            # global acme_dns cloudflare {env.CLOUDFLARE_API_TOKEN}; site tls { protocols tls1.3; resolvers ... }
│           └── compose.yaml.j2         # caddy: use custom image from registry; env_file: caddy.env; keep /data + ports 80/443
├── inventories/                        # rewrite comments: LAN/private IP OK; Cloudflare API token required; no public reachability
└── README.md                           # update HTTPS prerequisites to DNS-01 / token / LAN
```

**Structure Decision**: The single existing Ansible `app` role is extended in place; no new role is introduced (Principle I). The custom Caddy image is built and shipped through the **existing** on-host registry pipeline used for `twig-server`, so no new delivery mechanism appears. The Caddy data directory and the new `caddy.env` live under the established `{{ twig_host_data_dir }}` convention, keeping host state in one backup-friendly location.

## Complexity Tracking

| Violation | Why Needed | Simpler Alternative Rejected Because |
|-----------|------------|-------------------------------------|
| Custom Caddy image (xcaddy build with `caddy-dns/cloudflare`) instead of stock `caddy:2-alpine` | Stock Caddy ships no DNS provider modules; DNS-01 with Cloudflare — the only ACME challenge that works behind a private IP (FR-014) — requires the `caddy-dns/cloudflare` module compiled in | Stock image cannot do DNS-01 at all. An unofficial prebuilt `caddy-cloudflare` image avoids the build but is an unvetted supply-chain dependency. The official xcaddy builder stage is the trusted, self-owned path and reuses the existing registry pipeline. |
