# Contract: Deployment Configuration

**Feature**: 025-https-custom-domain | **Date**: 2026-05-31

This is the configuration contract the deployment exposes to operators, plus the Caddyfile structure the templates must produce. Per Constitution Principle II, this contract is fixed **before** the role/template edits; implementation conforms to it.

## 1. Operator-facing variable contract

| Variable | Required | Allowed values | Default | Behavior |
|----------|----------|----------------|---------|----------|
| `twig_domain` | Yes | valid hostname / FQDN | — | The domain the deployment serves and obtains a cert for. Missing/invalid → play fails with an actionable message before any restart. |
| `twig_acme_environment` | No | `production`, `staging` | `production` | `staging` issues from Let's Encrypt staging (untrusted, high limits) for non-prod bring-ups. |
| `twig_acme_email` | No | email string or empty | empty | When set, used as the ACME account contact. |
| `twig_https_port` | No | 1–65535 | `443` | Host port published for HTTPS. |
| `twig_http_port` | No | 1–65535 | `80` | Host port for the HTTP→HTTPS redirect and HTTP-01 challenge. |

**Guarantees:**
- C1. Setting only `twig_domain` (all else default) yields a working, publicly-trusted, TLS-1.3 HTTPS deployment for that domain (FR-001…FR-008, FR-012).
- C2. Changing only `twig_domain` and redeploying serves the new domain and obtains its cert (FR-004, FR-005, US2-3).
- C3. `twig_acme_environment: staging` produces an (untrusted) cert from the staging endpoint without consuming production rate limits (FR-013).
- C4. An undefined/empty/malformed `twig_domain` fails the play with a message naming the variable (FR-009).

## 2. Caddyfile structure contract

The rendered Caddyfile MUST have this shape (Jinja2 conditionals shown):

```caddyfile
{
    {% if twig_acme_email %}email {{ twig_acme_email }}
    {% endif %}{% if twig_acme_environment == 'staging' %}acme_ca https://acme-staging-v02.api.letsencrypt.org/directory
    {% endif %}
}

{{ twig_domain }} {
    tls {
        protocols tls1.3
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
- The site address is the bare hostname `{{ twig_domain }}` (NOT `:80`/`:443`). This is what activates Caddy automatic HTTPS, ACME issuance, and the HTTP→HTTPS redirect (FR-001, FR-002, FR-006, FR-008).
- The global options block emits `email`/`acme_ca` lines **only** when their conditions hold; an empty global block (`{ }`) is valid when production + no email.
- `tls { protocols tls1.3 }` is present (FR-012) and does not disable automatic certificate management.
- The existing `@backend` regex and SPA `handle` blocks are preserved verbatim (no routing regression).

## 3. Compose service contract (caddy)

The `caddy` service in the rendered `compose.yaml` MUST:
- Publish both `"{{ twig_http_port }}:80"` and `"{{ twig_https_port }}:443"`.
- Mount the Caddyfile (`:ro,z`) and SPA dir (`:ro,z`) as today.
- Add a **read-write** bind mount `"{{ twig_host_data_dir }}/caddy:/data:z"` for certificate/ACME persistence (D4).
- Keep `depends_on: twig_server` and `restart: unless-stopped`.

## 4. Host contract (base role)

- The directory `{{ twig_host_data_dir }}/caddy` exists and is writable by the Caddy container before the stack starts.
- TCP **80 and 443** are reachable from the public internet (host firewall / cloud security group). If `firewalld` is active on the host, the play opens the `http` and `https` services; otherwise this is an operator precondition (documented in quickstart).
