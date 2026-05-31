# Phase 1 Data Model: HTTPS with Custom Domain

**Feature**: 025-https-custom-domain | **Date**: 2026-05-31

This feature has no database schema. Its "data model" is the **deployment configuration** — the per-deployment inputs that drive HTTPS behavior, plus the persisted certificate state. These map to the spec's Key Entities (*Deployment domain configuration*, *CA environment toggle*, *TLS certificate*).

## Configuration variables (Ansible)

Defined in `deploy/group_vars/all.yml` (defaults) and overridable per host/group in inventory or `host_vars`.

| Variable | Type | Required | Default | Validation | Maps to |
|----------|------|----------|---------|------------|---------|
| `twig_domain` | string (hostname/FQDN) | **Yes** | *(none — must be set)* | Defined, non-empty after trim, matches hostname regex; no scheme/path/spaces (FR-009, D7) | *Deployment domain configuration* (FR-003, FR-004, FR-005) |
| `twig_acme_environment` | enum `production` \| `staging` | No | `production` | Must be exactly `production` or `staging` | *CA environment toggle* (FR-013, D3) |
| `twig_acme_email` | string (email) | No | `""` (empty) | If non-empty, a plausible email address | ACME account contact (D6) |
| `twig_https_port` | int | No | `443` | 1–65535 | Host port published for HTTPS (D2) |
| `twig_http_port` | int | No | `80` (existing) | 1–65535 | Host port for HTTP redirect + HTTP-01 (existing var, reused) |

Notes:
- `twig_domain` intentionally has **no default**: an unset value must fail the play (FR-009), not silently pick a placeholder.
- `twig_acme_environment` is the single binary toggle from FR-013 — not a free-form CA URL (Principle I).

### Derived/templated values

| Derived value | Source | Used in |
|---------------|--------|---------|
| `acme_ca` directive presence | `twig_acme_environment == 'staging'` → Let's Encrypt staging directory URL; else omitted | Caddyfile global options |
| `email` directive presence | `twig_acme_email` non-empty | Caddyfile global options |
| Caddy site address | `twig_domain` | Caddyfile site block (replaces bare `:{{ twig_http_port }}`) |

## Persisted state

| Entity | Location (host) | Mount (container) | Lifecycle |
|--------|-----------------|-------------------|-----------|
| Caddy certificate + ACME account state (*TLS certificate*) | `{{ twig_host_data_dir }}/caddy` (`/var/lib/twig/caddy`) | `/data` (`:z` SELinux relabel) | Created on first issuance; auto-renewed by Caddy; **must persist** across restart/redeploy to avoid re-issuance (D4) |
| PostgreSQL data (unchanged) | `{{ twig_host_data_dir }}/postgres` | `/var/lib/postgresql/data` | Unchanged — existing guards must continue to pass |
| Rendered Caddyfile (unchanged location) | `{{ twig_host_data_dir }}/app/Caddyfile` | `/etc/caddy/Caddyfile:ro` | Re-rendered each run; change triggers stack restart |

### TLS certificate — observable attributes (the spec's *TLS certificate* entity)

These are not stored by us in a schema; they are properties of the cert Caddy obtains, asserted via external checks:

| Attribute | Expected value (production mode) |
|-----------|----------------------------------|
| Subject Alternative Name | includes `twig_domain` (FR-003) |
| Issuer | Let's Encrypt (publicly-trusted chain) (FR-002) |
| Trust | chains to a publicly-trusted root; no browser warning (FR-002, FR-011) |
| Validity | short-lived; auto-renewed ahead of expiry (FR-007) |
| Negotiated protocol | TLS 1.3; TLS ≤1.2 refused (FR-012) |

## State transitions (deployment lifecycle)

```text
[play start]
   │  assert twig_domain valid ──fail fast──> [ERROR: clear message]  (FR-009)
   ▼
[render Caddyfile + compose] ── changed? ──> [restart stack]
   ▼
[Caddy starts with named-host site]
   │  /data has valid cert for domain? ──yes──> serve immediately
   │  no ──> ACME issuance (HTTP-01/TLS-ALPN-01 via Let's Encrypt)
   │           │ success ──> cert written to /data ──> serve HTTPS (TLS 1.3)
   │           │ failure ──> logged; HTTPS smoke check retries then fails play  (FR-010)
   ▼
[HTTPS smoke check: https://twig_domain returns 200/401 within retry window]  (SC-003)
   ▼
[steady state]
   └─ Caddy auto-renews at ~2/3 lifetime; renewal failures logged  (FR-007, FR-010)
```
