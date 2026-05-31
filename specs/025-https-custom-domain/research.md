# Phase 0 Research: HTTPS with Custom Domain (LAN / DNS-01)

**Feature**: 025-https-custom-domain | **Date**: 2026-05-31 (revised)

> **Revision note**: This document replaces the earlier public-reachability design. The operator clarified the deployment runs on a **LAN behind a private/internal IP and is not reachable from the public internet**, so the HTTP-01 / TLS-ALPN-01 challenges Caddy uses by default cannot work. Domain control is now proven with a **DNS-01 challenge via the Cloudflare API**, using an operator-supplied API token. Decisions D1 and D2 are reversed from the prior revision; D3–D6 carry over; D7–D8 are extended. New decisions D9–D11 cover the custom Caddy build, the API-token secret, and resolver handling.

---

## D1. ACME challenge method: DNS-01 via Cloudflare

**Decision**: Use the **DNS-01 challenge through the Cloudflare API**. Caddy proves control of `twig_domain` by creating a `_acme-challenge` TXT record in the domain's Cloudflare zone; Let's Encrypt validates that record and never connects to the deployment. Configured via a Caddyfile **global** `acme_dns cloudflare {env.CLOUDFLARE_API_TOKEN}` directive so every certificate uses DNS-01.

**Rationale**:
- The deployment is on a private LAN (e.g. `192.0.2.10`) with no inbound public connectivity, so HTTP-01 (port 80) and TLS-ALPN-01 (port 443) — which require the CA to reach the host — **cannot complete** (FR-014). DNS-01 is the only ACME challenge that does not require the host to be publicly reachable.
- It still yields a **publicly-trusted Let's Encrypt certificate** (FR-002/FR-011): trust derives from proven DNS control, not from the host's reachability. LAN browsers therefore see no warning even though the host has a private IP (SC-008).
- Cloudflare is the authoritative DNS for the domain and exposes a token-scoped API that the `caddy-dns/cloudflare` module drives automatically (FR-006, FR-015).

**Alternatives considered**:
- **HTTP-01 / TLS-ALPN-01 (the prior design)**: requires the host to be publicly reachable on 80/443. Rejected — contradicts the LAN/private-IP reality (FR-014).
- **A self-signed / internal CA cert**: would avoid external dependencies but is not publicly trusted, so LAN browsers would show warnings unless every client installs a custom root. Rejected per FR-002/FR-011.
- **Cloudflare Tunnel / proxied edge**: would expose the app publicly and terminate TLS at Cloudflare's edge (Cloudflare's cert, not Let's Encrypt's, presented to users). Rejected — changes the security posture and does not satisfy "Let's Encrypt cert presented to end users".

---

## D2. Listener ports: 443 for serving, 80 for redirect only

**Decision**: Caddy continues to listen on **443** (HTTPS serving) and **80** (HTTP→HTTPS redirect) for **LAN** clients. Neither port needs to be reachable from the public internet. The compose service keeps publishing `{{ twig_http_port }}:80` and `{{ twig_https_port }}:443`; the base-role `firewalld` rule that opens `http`/`https` now governs **LAN** access, not public access.

**Rationale**:
- With DNS-01 (D1), no ACME challenge arrives on 80/443, so the prior justification for *publicly* opening those ports is gone. Port 80 is retained solely to honor FR-008 (HTTP→HTTPS redirect) for LAN clients; port 443 serves the app.
- Keeping the existing port publishing avoids a needless change and preserves the redirect behavior the spec still requires.

**Implication**: The earlier "host must be publicly reachable on 80/443" precondition is removed from inventory comments, README, and quickstart.

---

## D3. CA selection and the production/staging toggle (FR-013) — carried over

**Decision**: Let's Encrypt remains the CA (Caddy's default issuer). The existing `twig_acme_environment` variable (`production` default / `staging`) still selects the endpoint: `staging` sets `acme_ca https://acme-staging-v02.api.letsencrypt.org/directory` in the global options; `production` omits it.

**Rationale**: Unchanged from the prior design and orthogonal to the challenge type — DNS-01 works against both endpoints. Staging issues an untrusted cert with generous rate limits for pipeline validation (FR-013); production trust requirements (FR-002/FR-011) apply to production mode only.

---

## D4. Certificate persistence across restarts (D4) — carried over

**Decision**: Keep the `{{ twig_host_data_dir }}/caddy` → `/data` bind mount so issued certs, keys, and ACME account state survive container/host restarts and redeploys.

**Rationale**: Unchanged. Persisting `/data` prevents needless re-issuance (and Let's Encrypt rate-limit exposure) on every restart. DNS-01 does not change where Caddy stores material.

---

## D5. TLS 1.3 minimum (FR-012) — carried over

**Decision**: `tls { protocols tls1.3 }` per site for the TLS 1.3 floor. The same `tls` block also carries the per-site DNS challenge config when not set globally; here the DNS provider is set **globally** via `acme_dns` (D1), so the site `tls` block needs only `protocols tls1.3` (plus a `resolvers` line per D11).

**Rationale**: Unchanged — Caddy's default floor is TLS 1.2; FR-012 requires rejecting ≤1.2. Externally verifiable (SC-007).

---

## D6. ACME account email (optional) — carried over

**Decision**: Keep optional `twig_acme_email` → global `email` directive when set.

**Rationale**: Unchanged. Recommended for production expiry/problem notices; optional otherwise.

---

## D7. Fail-fast on missing/invalid configuration (FR-009, FR-017)

**Decision**: Keep the existing `assert` that fails the play when `twig_domain` is undefined/empty/not a valid hostname. **Add** an `assert` that fails when `twig_cloudflare_api_token` is undefined or empty (FR-017). The token assertion uses `no_log: true` so the secret never appears in output.

**Rationale**: FR-009 and FR-017 both require a clear, actionable failure before any container starts, rather than a broken/insecure start. Without the token, DNS-01 cannot proceed, so it is a hard precondition. `no_log` keeps the secret out of Ansible output (FR-016).

---

## D8. Observability + smoke check independent of split-horizon DNS (FR-010, SC-003)

**Decision**: Caddy logs remain the observability surface (issuance/renewal logged to stdout, captured by podman/journald). The post-deploy smoke check is changed to **not depend on the control node's DNS view**: it runs `curl --resolve {{ twig_domain }}:{{ twig_https_port }}:{{ ansible_host }} https://{{ twig_domain }}/health.v1.HealthService/Check`, forcing the connection to the host's known LAN address while still presenting (and, in production, validating) the certificate for `twig_domain`. The retry window is widened to ~10 minutes to absorb DNS challenge-record propagation (SC-003).

**Rationale**:
- On a LAN with split-horizon DNS, the Ansible control node may not resolve `twig_domain` to the host's internal IP. `--resolve` pins the address deterministically and is the most representative check (a LAN client reaching the host and being served the domain's cert). The `ansible.builtin.uri` module cannot pin resolution, so the check uses `ansible.builtin.command` with `curl`.
- DNS-01 issuance includes a propagation wait (the TXT record must be visible to Let's Encrypt's validators), which can take longer than an inbound HTTP-01 check; 10 minutes (per revised SC-003) gives comfortable headroom.
- In `staging` mode the cert is untrusted, so the check adds `-k` (skip trust) and asserts only that HTTPS responds.

---

## D9. Custom Caddy image with the Cloudflare DNS module

**Decision**: Build a **custom Caddy image** that includes `github.com/caddy-dns/cloudflare`, using the official `caddy:<ver>-builder` (xcaddy) stage, and run **that** image instead of stock `caddy:2-alpine`. Build it on the workstation and push it to the existing on-host registry (reusing the SSH-tunnel build/push pattern already used for `twig-server`), tagged by Caddy version + plugin (e.g. `twig-caddy:2-cloudflare`). Build/push only when the tag is absent from the registry (idempotent re-runs).

```dockerfile
FROM docker.io/library/caddy:2-builder AS builder
RUN xcaddy build --with github.com/caddy-dns/cloudflare

FROM docker.io/library/caddy:2-alpine
COPY --from=builder /usr/bin/caddy /usr/bin/caddy
```

**Rationale**:
- Stock `caddy:2-alpine` ships **no DNS provider modules**; DNS-01 with Cloudflare requires the `caddy-dns/cloudflare` module compiled in. The xcaddy builder image is the official, supported way to produce such a build (no third-party prebuilt image, avoiding supply-chain risk — Principle I trades a tiny bit of build complexity for a trusted, self-owned artifact).
- Reusing the established build→registry→pull pipeline (already implemented for `twig-server`) keeps the host pulling from its own registry and fits the existing idempotence model (`app_need_build_push`-style guard), rather than introducing a new delivery mechanism.

**Alternatives considered**:
- **Unofficial prebuilt `caddy-cloudflare` Docker Hub images**: convenient but unvetted; rejected on supply-chain grounds.
- **Building on the host directly**: the workstation-build + registry-push pattern already exists and is consistent; building on the host would diverge from how `twig-server` is delivered.

---

## D10. Cloudflare API token handling (secret) (FR-016)

**Decision**: Introduce `twig_cloudflare_api_token` — a **required, secret** variable the operator supplies (recommended via Ansible Vault). The token is rendered into a host **env file** `{{ twig_host_data_dir }}/app/caddy.env` (mode `0600`, `no_log: true`) containing `CLOUDFLARE_API_TOKEN=...`; the `caddy` compose service references it with `env_file:` so the value is **not** written into `compose.yaml` and the Caddyfile only references `{env.CLOUDFLARE_API_TOKEN}`. Recommended Cloudflare token scope: **Zone → DNS → Edit** for the specific zone only.

**Rationale**:
- FR-016 requires the credential to be a per-deployment secret never exposed in logs or operator output. An `0600` env file written with `no_log: true`, referenced via `env_file`, keeps the token out of the committed/rendered compose file and out of Ansible output, while making it available to the Caddy process as an environment variable (the form `{env.CLOUDFLARE_API_TOKEN}` in the Caddyfile expects).
- A narrowly-scoped token (single zone, DNS edit) limits blast radius if leaked.

**Alternatives considered**:
- **Token inline in `compose.yaml` `environment:`**: would persist the secret in a world-readable rendered file. Rejected (FR-016).
- **Podman secret**: viable but adds a podman-secret lifecycle the rest of the stack does not use; the `0600` env file is simpler and consistent with the repo's plaintext-config-on-host convention. Revisit if a secrets manager is adopted.

---

## D11. DNS resolvers for the challenge (split-horizon safety)

**Decision**: Set an explicit public resolver for ACME DNS propagation checks via the site `tls` block: `tls { resolvers 1.1.1.1 ... }` (exposed as `twig_acme_dns_resolver`, default `1.1.1.1`). 

**Rationale**:
- On a LAN, the host's own resolver may be a split-horizon/internal DNS that does not serve (or lags) the public `_acme-challenge` TXT record. Pointing Caddy's propagation check at a public resolver (Cloudflare's `1.1.1.1`) ensures it verifies the same view Let's Encrypt will, avoiding spurious "record not propagated" stalls. Exposed as a variable so an operator on an unusual network can override it; default covers the common case.

**Alternatives considered**: relying on the system resolver — rejected because split-horizon DNS is exactly the environment here and is the most likely cause of DNS-01 propagation-check failures.

---

## Summary of decisions

| ID | Decision |
|----|----------|
| D1 | **DNS-01 via Cloudflare API** (`acme_dns cloudflare {env.CLOUDFLARE_API_TOKEN}`); no inbound challenge; works behind a private IP |
| D2 | Listen on 443 (serve) + 80 (redirect) for **LAN** clients; no public reachability required |
| D3 | `twig_acme_environment` prod/staging toggle (default production) — carried over |
| D4 | Persist Caddy `/data` via host bind mount — carried over |
| D5 | `tls { protocols tls1.3 }` per site for the TLS 1.3 floor — carried over |
| D6 | Optional `twig_acme_email` → global `email` directive — carried over |
| D7 | Fail-fast `assert` on missing/invalid `twig_domain` **and** on missing `twig_cloudflare_api_token` (`no_log`) |
| D8 | Caddy logs + smoke check via `curl --resolve` (split-horizon-independent), ~10-min retry window |
| D9 | **Custom Caddy image** built with `caddy-dns/cloudflare` (xcaddy), pushed to on-host registry |
| D10 | `twig_cloudflare_api_token` secret → `0600` `caddy.env` via `env_file:`; never in compose/logs |
| D11 | Explicit public `resolvers` (`twig_acme_dns_resolver`, default `1.1.1.1`) for propagation checks |

All NEEDS CLARIFICATION items are resolved. No open questions block Phase 1.
