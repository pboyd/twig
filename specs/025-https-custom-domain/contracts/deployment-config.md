# Contract: Deployment Configuration (LAN / DNS-01)

**Feature**: 025-https-custom-domain | **Date**: 2026-05-31 (revised)

This is the configuration contract the deployment exposes to operators, plus the Caddyfile / compose / Containerfile structure the templates must produce. Per Constitution Principle II, this contract is fixed **before** the role/template edits; implementation conforms to it.

## 1. Operator-facing variable contract

| Variable | Required | Allowed values | Default | Behavior |
|----------|----------|----------------|---------|----------|
| `twig_domain` | Yes | valid hostname / FQDN | — | The domain the deployment serves and obtains a cert for. Missing/invalid → play fails before any restart (FR-009). |
| `twig_cloudflare_api_token` | Yes | non-empty string (secret) | — | Cloudflare API token (scope: Zone → DNS → Edit for the zone) used for the DNS-01 challenge. Missing/empty → play fails before any restart, with `no_log` (FR-016, FR-017). |
| `twig_acme_environment` | No | `production`, `staging` | `production` | `staging` issues from Let's Encrypt staging (untrusted, high limits) for non-prod bring-ups (FR-013). |
| `twig_acme_email` | No | email string or empty | empty | When set, used as the ACME account contact. |
| `twig_acme_dns_resolver` | No | resolver IP | `1.1.1.1` | Public resolver Caddy uses for DNS-01 propagation checks (split-horizon safety, D11). |
| `twig_https_port` | No | 1–65535 | `443` | LAN host port published for HTTPS. |
| `twig_http_port` | No | 1–65535 | `80` | LAN host port for the HTTP→HTTPS redirect (no longer used for any ACME challenge). |

**Guarantees:**
- C1. Setting `twig_domain` + `twig_cloudflare_api_token` (all else default) yields a working, publicly-trusted, TLS-1.3 HTTPS deployment for that domain **even though the host is on a private LAN IP** (FR-001…FR-008, FR-012, FR-014, SC-008).
- C2. Changing only `twig_domain` and redeploying serves the new domain and obtains its cert (FR-004, FR-005, US2-3), provided the token's zone covers it.
- C3. `twig_acme_environment: staging` produces an (untrusted) cert from the staging endpoint via DNS-01 without consuming production rate limits (FR-013).
- C4. An undefined/empty/malformed `twig_domain` **or** an undefined/empty `twig_cloudflare_api_token` fails the play with a message naming the variable (FR-009, FR-017). The token value never appears in output (FR-016).

## 2. Caddyfile structure contract

The rendered Caddyfile MUST have this shape (Jinja2 conditionals shown):

```caddyfile
{
    {% if twig_acme_email %}email {{ twig_acme_email }}
    {% endif %}acme_dns cloudflare {env.CLOUDFLARE_API_TOKEN}
    {% if twig_acme_environment == 'staging' %}acme_ca https://acme-staging-v02.api.letsencrypt.org/directory
    {% endif %}
}

{{ twig_domain }} {
    tls {
        protocols tls1.3
        resolvers {{ twig_acme_dns_resolver }}
    }

    # Route ConnectRPC and auth paths to the Go server (unchanged routing).
    @backend path_regexp "^/(auth/|task[.]v1[.]|health[.]v1[.])"
    handle @backend {
        reverse_proxy twig_server:8080
    }

    # Serve the SPA — fall back to index.html for client-side routing (unchanged).
    handle {
        root * /srv/www
        try_files {path} /index.html
        file_server
    }
}
```

**Invariants:**
- The global options block emits `acme_dns cloudflare {env.CLOUDFLARE_API_TOKEN}` **always** — this selects the DNS-01 challenge for every certificate (FR-014, FR-015, D1). The literal `{env.CLOUDFLARE_API_TOKEN}` placeholder is rendered verbatim (Caddy reads the env var at runtime); the **token value is never written into the Caddyfile** (FR-016).
- The site address is the bare hostname `{{ twig_domain }}` (NOT `:80`/`:443`), which keeps Caddy's automatic HTTPS, issuance, and HTTP→HTTPS redirect (FR-001, FR-002, FR-006, FR-008).
- `email`/`acme_ca` lines are emitted **only** when their conditions hold.
- The site `tls` block sets `protocols tls1.3` (FR-012) and `resolvers {{ twig_acme_dns_resolver }}` (D11), and does not disable automatic certificate management.
- The existing `@backend` regex and SPA `handle` blocks are preserved verbatim (no routing regression).

## 3. Compose service contract (caddy)

The `caddy` service in the rendered `compose.yaml` MUST:
- Use the **custom image** from the on-host registry (e.g. `localhost:5000/twig-caddy:2-cloudflare`), NOT stock `docker.io/library/caddy:2-alpine` (D9).
- Reference the token env file: `env_file: ["{{ twig_host_data_dir }}/app/caddy.env"]` — the token MUST NOT appear inline under `environment:` (FR-016, D10).
- Publish both `"{{ twig_http_port }}:80"` and `"{{ twig_https_port }}:443"` (LAN access; public reachability not required, D2).
- Mount the Caddyfile (`:ro,z`) and SPA dir (`:ro,z`) as today.
- Keep the **read-write** bind mount `"{{ twig_host_data_dir }}/caddy:/data:z"` for certificate/ACME persistence (D4).
- Keep `depends_on: twig_server` and `restart: unless-stopped`.

## 4. Custom Caddy image contract (Containerfile)

`deploy/roles/app/files/Containerfile.caddy` MUST:
- Use the official `docker.io/library/caddy:<ver>-builder` stage to `xcaddy build --with github.com/caddy-dns/cloudflare`.
- Copy the resulting binary into a `docker.io/library/caddy:<ver>-alpine` runtime stage.
- Be built and pushed to the on-host registry by the `app` role, **guarded** so a re-run with an existing tag is a no-op (mirrors the `twig-server` build/push idempotence).

## 5. Secret-handling contract (caddy.env)

- `{{ twig_host_data_dir }}/app/caddy.env` is rendered with mode `0600`, owner `root`, containing exactly `CLOUDFLARE_API_TOKEN={{ twig_cloudflare_api_token }}`.
- The rendering task sets `no_log: true`. The token MUST NOT appear in `compose.yaml`, the Caddyfile, Ansible stdout, or any committed file (FR-016).

## 6. Host contract (base role)

- The directories `{{ twig_host_data_dir }}/caddy` and `{{ twig_host_data_dir }}/app` exist and are writable before the stack starts (unchanged).
- TCP **80 and 443** are reachable **from the LAN** (host firewall). If `firewalld` is active, the play opens the `http`/`https` services for LAN clients; **inbound public-internet reachability is NOT required** (DNS-01 needs only outbound access to Let's Encrypt and the Cloudflare API).
