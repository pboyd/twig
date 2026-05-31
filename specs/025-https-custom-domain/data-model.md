# Phase 1 Data Model: HTTPS with Custom Domain (LAN / DNS-01)

**Feature**: 025-https-custom-domain | **Date**: 2026-05-31 (revised)

This feature has no database schema. Its "data model" is the **deployment configuration** — the per-deployment inputs that drive HTTPS behavior, plus the persisted certificate state and the DNS-provider secret. These map to the spec's Key Entities (*Deployment domain configuration*, *CA environment toggle*, *DNS-provider credentials*, *TLS certificate*).

## Configuration variables (Ansible)

Defined in `deploy/group_vars/all.yml` (defaults) and overridable per host/group in inventory or `host_vars`. Secrets should be supplied via Ansible Vault.

| Variable | Type | Required | Default | Validation | Maps to |
|----------|------|----------|---------|------------|---------|
| `twig_domain` | string (hostname/FQDN) | **Yes** | *(none — must be set)* | Defined, non-empty after trim, matches hostname regex; no scheme/path/spaces (FR-009, D7) | *Deployment domain configuration* (FR-003, FR-004, FR-005) |
| `twig_cloudflare_api_token` | string (secret) | **Yes** | *(none — must be set)* | Defined, non-empty after trim; asserted with `no_log: true` (FR-016, FR-017, D7, D10) | *DNS-provider credentials* (FR-015, FR-016) |
| `twig_acme_environment` | enum `production` \| `staging` | No | `production` | Must be exactly `production` or `staging` | *CA environment toggle* (FR-013, D3) |
| `twig_acme_email` | string (email) | No | `""` (empty) | If non-empty, a plausible email address | ACME account contact (D6) |
| `twig_acme_dns_resolver` | string (IP) | No | `1.1.1.1` | A reachable public DNS resolver | Propagation-check resolver, split-horizon safety (D11) |
| `twig_https_port` | int | No | `443` | 1–65535 | Host (LAN) port published for HTTPS (D2) |
| `twig_http_port` | int | No | `80` | 1–65535 | Host (LAN) port for the HTTP→HTTPS redirect only (D2) |

Notes:
- `twig_domain` and `twig_cloudflare_api_token` intentionally have **no default**: an unset value must fail the play (FR-009, FR-017), not silently pick a placeholder or start without a way to issue certs.
- `twig_acme_environment` is the single binary toggle from FR-013 — not a free-form CA URL (Principle I).
- The DNS provider is fixed to **Cloudflare** for this deployment (concrete instance of the spec's generic "DNS provider"); a free-form provider variable is intentionally not introduced (Principle I / YAGNI).

### Derived/templated values

| Derived value | Source | Used in |
|---------------|--------|---------|
| `acme_dns cloudflare {env.CLOUDFLARE_API_TOKEN}` | always (DNS-01 is the only challenge) | Caddyfile global options (D1) |
| `acme_ca` directive presence | `twig_acme_environment == 'staging'` → Let's Encrypt staging directory URL; else omitted | Caddyfile global options (D3) |
| `email` directive presence | `twig_acme_email` non-empty | Caddyfile global options (D6) |
| `resolvers {{ twig_acme_dns_resolver }}` | always | Caddyfile site `tls` block (D11) |
| Caddy site address | `twig_domain` | Caddyfile site block (D5) |
| `CLOUDFLARE_API_TOKEN` env var | `twig_cloudflare_api_token` | `caddy.env` (`env_file:` for the caddy service) (D10) |
| Custom Caddy image ref | on-host registry tag (e.g. `localhost:5000/twig-caddy:2-cloudflare`) | compose `caddy.image` (D9) |

## Persisted state

| Entity | Location (host) | Mount (container) | Lifecycle |
|--------|-----------------|-------------------|-----------|
| Caddy certificate + ACME account state (*TLS certificate*) | `{{ twig_host_data_dir }}/caddy` (`/var/lib/twig/caddy`) | `/data` (`:z` SELinux relabel) | Created on first issuance; auto-renewed by Caddy via DNS-01; **must persist** across restart/redeploy to avoid re-issuance (D4) |
| Cloudflare API token (*DNS-provider credentials*) | `{{ twig_host_data_dir }}/app/caddy.env` (`0600`) | `env_file` → `CLOUDFLARE_API_TOKEN` env var | Rendered with `no_log: true`; never written into `compose.yaml` or logs (FR-016, D10) |
| PostgreSQL data (unchanged) | `{{ twig_host_data_dir }}/postgres` | `/var/lib/postgresql/data` | Unchanged — existing guards must continue to pass |
| Rendered Caddyfile | `{{ twig_host_data_dir }}/app/Caddyfile` | `/etc/caddy/Caddyfile:ro` | Re-rendered each run; references `{env.CLOUDFLARE_API_TOKEN}` (no secret in file); change triggers stack restart |

### TLS certificate — observable attributes (the spec's *TLS certificate* entity)

These are not stored by us in a schema; they are properties of the cert Caddy obtains via DNS-01, asserted via external checks:

| Attribute | Expected value (production mode) |
|-----------|----------------------------------|
| Subject Alternative Name | includes `twig_domain` (FR-003) |
| Issuer | Let's Encrypt (publicly-trusted chain) (FR-002) |
| Trust | chains to a publicly-trusted root; no browser warning — even though the host has a private IP (FR-002, FR-011, SC-008) |
| Validity | short-lived; auto-renewed ahead of expiry via DNS-01 (FR-007) |
| Negotiated protocol | TLS 1.3; TLS ≤1.2 refused (FR-012) |

## State transitions (deployment lifecycle)

```text
[play start]
   │  assert twig_domain valid ───────fail fast──> [ERROR: clear message]  (FR-009)
   │  assert twig_cloudflare_api_token set (no_log) ─fail fast──> [ERROR: clear message]  (FR-017)
   ▼
[build/push custom caddy image if tag absent in registry]  (D9, idempotent)
   ▼
[render caddy.env (0600), Caddyfile, compose] ── changed? ──> [restart stack]
   ▼
[Caddy starts; global acme_dns = cloudflare]
   │  /data has valid cert for domain? ──yes──> serve immediately
   │  no ──> DNS-01 issuance: create _acme-challenge TXT via Cloudflare API
   │           │ propagation checked against {{ twig_acme_dns_resolver }}  (D11)
   │           │ success ──> cert written to /data ──> serve HTTPS (TLS 1.3)
   │           │ failure ──> logged; HTTPS smoke check retries then fails play  (FR-010)
   ▼
[HTTPS smoke check: curl --resolve twig_domain:443:ansible_host → 200/401 within ~10-min window]  (SC-003, D8)
   ▼
[steady state]
   └─ Caddy auto-renews via DNS-01 at ~2/3 lifetime; renewal failures logged  (FR-007, FR-010)
```
