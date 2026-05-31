# Phase 0 Research: HTTPS with Custom Domain

**Feature**: 025-https-custom-domain | **Date**: 2026-05-31

This document resolves the open decisions implied by the spec and the user's planning input ("Let's Encrypt as the CA; domain hosted in Cloudflare"), plus the two items the spec deferred to planning (ACME challenge type, observability specifics).

---

## D1. Cloudflare DNS posture: DNS-only vs. proxied

**Decision**: Use a **DNS-only ("grey-cloud") A/AAAA record** in Cloudflare pointing directly at the host's public IP. Cloudflare is the authoritative DNS host only; it does not proxy or terminate TLS.

**Rationale**:
- The spec requires the certificate **presented to end users** to be a Let's Encrypt cert (FR-002) and the **deployment** to negotiate TLS 1.3 / refuse TLS 1.2 (FR-012). With a DNS-only record, the browser's TLS session terminates at the origin's Caddy, which serves the Let's Encrypt cert and enforces the TLS 1.3 floor — both requirements are satisfied end-to-end and are directly verifiable from an external client.
- If the record were proxied (orange-cloud), Cloudflare would terminate the browser's TLS at its edge and present *Cloudflare's* certificate (issued by Google Trust Services / Let's Encrypt at Cloudflare's discretion, not the deployment's). The deployment's own TLS version would be invisible to the end user, and FR-002/FR-012 would describe the Cloudflare↔origin hop rather than the user-facing one. This breaks the spec's intent.
- DNS-only keeps Caddy's standard automatic HTTPS working with **no custom image and no API credentials** (Principle I).

**Alternatives considered**:
- **Proxied + DNS-01 challenge via Cloudflare API** (`caddy-dns/cloudflare` plugin): would allow issuance behind the proxy and hide the origin IP, but requires a **custom Caddy build** (stock `caddy:2-alpine` has no DNS plugins), a Cloudflare API token mounted as a secret, and still leaves Cloudflare presenting its own edge cert to users. Rejected: more moving parts, an extra secret to manage, and it does not actually satisfy "Let's Encrypt cert presented to end users." Documented here so a future operator who *wants* edge proxying knows the path.
- **Proxied + Cloudflare Origin CA cert**: not publicly trusted outside Cloudflare's edge; fails FR-002 for direct origin access. Rejected.

**Consequences / prerequisites**:
- The host needs a **public, internet-routable IP**, and the Cloudflare record must point at it. The existing `inventories/lan.ini` and `inventories/test.ini` use private RFC-1918 addresses (`192.168.x`, `192.168.122.x`); Let's Encrypt **production** cannot validate a domain that resolves to a private IP. Public HTTPS therefore requires a publicly-reachable host (new/updated inventory). This is called out in quickstart.md as a precondition, not a code change.
- Origin IP is exposed (no Cloudflare proxy hiding it). Acceptable for this feature's scope; revisit if DDoS protection becomes a requirement.

---

## D2. ACME challenge type

**Decision**: Rely on Caddy's **default automatic challenge selection over the public listeners** — i.e. HTTP-01 (port 80) and/or TLS-ALPN-01 (port 443). No DNS-01, no Cloudflare API token.

**Rationale**:
- The spec assumes the host is reachable on standard ports 80 and 443; with a DNS-only record (D1) those challenges reach the origin directly. Caddy negotiates the challenge type itself; the operator does nothing.
- DNS-01 is only needed for wildcard certs or when ports 80/443 are not publicly reachable — both out of scope (single domain; standard ports assumed). Avoiding DNS-01 keeps us on the stock image with no secret (Principle I).

**Implication for compose**: Caddy must publish **both 80 and 443**. Port 80 currently published; **443 must be added**. Port 80 must remain open because (a) Caddy serves the HTTP→HTTPS redirect there and (b) HTTP-01 validation uses it.

**Alternatives considered**: DNS-01 via Cloudflare token — rejected per D1 (custom build + secret, unnecessary here).

---

## D3. CA selection and the production/staging toggle (FR-013)

**Decision**: Let's Encrypt is the CA, selected implicitly because it is Caddy's default issuer. Expose a single Ansible variable `twig_acme_environment` with values `production` (default) and `staging`. In `staging` mode the Caddyfile global options set `acme_ca https://acme-staging-v02.api.letsencrypt.org/directory`; in `production` mode the directive is omitted so Caddy uses its default (Let's Encrypt production, with its usual ZeroSSL fallback).

**Rationale**:
- Let's Encrypt **production** enforces strict issuance rate limits (e.g. certificates-per-registered-domain per week). Operators iterating on bring-up can exhaust them and be locked out for days. The **staging** endpoint has far higher limits and issues certs from an **untrusted** root — perfect for validating the whole pipeline without burning production quota. This is exactly the spec's FR-013 toggle.
- Staging certs are deliberately untrusted, so FR-002/FR-011 (publicly-trusted cert) apply to **production mode only**; the spec already scopes them that way.

**Rationale for default = production**: matches FR-013's stated default and the principle of least surprise for a real deployment.

**Alternatives considered**: a free-form `twig_acme_ca` URL — rejected as over-general (Principle I / YAGNI); the binary prod/staging toggle covers the actual need. Operator-configurable arbitrary CA was explicitly ruled out by the spec's Q1 clarification.

---

## D4. Certificate persistence across restarts

**Decision**: Bind-mount `{{ twig_host_data_dir }}/caddy` to `/data` in the Caddy container so issued certificates, keys, and ACME account state persist across container/host restarts and redeploys.

**Rationale**:
- Caddy stores all certificate material under `/data/caddy`. The current compose mounts only the Caddyfile and the SPA dir — **no `/data`** — so every container recreation would discard certs and re-request them from Let's Encrypt, quickly hitting **production rate limits** (the same hazard D3 mitigates). Persisting `/data` makes restarts and same-commit redeploys free of issuance.
- Uses the existing `{{ twig_host_data_dir }}` convention (alongside `postgres/` and `app/`), so it is captured by the same host backup story. SELinux relabel `:z` is applied as elsewhere in the compose template.

**Alternatives considered**: a podman named volume — rejected to stay consistent with the repo's deliberate choice of host bind mounts for backup visibility (documented in `compose.yaml.j2`).

---

## D5. TLS 1.3 minimum (FR-012)

**Decision**: Set the minimum protocol per site via the Caddyfile `tls` directive: `tls { protocols tls1.3 }`. Caddy still manages the certificate automatically (the `tls` block with only `protocols` does not switch off automatic ACME).

**Rationale**:
- Caddy's default minimum is TLS 1.2; FR-012 requires rejecting 1.2 and below, so an explicit floor of `tls1.3` is needed. With port 80 open for HTTP-01, raising the HTTPS protocol floor does not impede issuance.
- Verifiable externally: `openssl s_client -tls1_2` must fail the handshake; `-tls1_3` must succeed (SC-007).

**Alternatives considered**: global `servers { protocols }` block — also possible but the per-site `tls` directive is the more localized, conventional Caddyfile expression and keeps the change inside the one site block we are already editing.

---

## D6. ACME account email (optional)

**Decision**: Expose optional `twig_acme_email`. When set, render `email {{ twig_acme_email }}` in the Caddyfile global options block (Let's Encrypt expiry/notice contact); when empty, omit it (Caddy registers an ACME account without a contact address).

**Rationale**: An email is recommended for production so the CA can send expiry/renewal-problem notices, but it is not strictly required for issuance. Optional keeps simple deployments frictionless while allowing production hardening.

---

## D7. Fail-fast on missing/invalid domain (FR-009)

**Decision**: Add an Ansible `assert` at the start of the `app` role that fails the play when `twig_domain` is undefined, empty, or not a syntactically valid hostname (single label or dotted FQDN; no scheme, no path, no spaces). The failure message names the variable and where to set it.

**Rationale**: FR-009 requires a clear, actionable failure rather than a broken/insecure start. Asserting before any template is rendered or container is (re)started prevents Caddy from coming up on a bad/empty host value. The pattern mirrors the existing inventory-target assertion in `site.yml`.

**Validation approach**: `twig_domain is defined and (twig_domain | trim) | length > 0` plus a hostname regex (`^(?!-)[A-Za-z0-9-]{1,63}(?<!-)(\.(?!-)[A-Za-z0-9-]{1,63}(?<!-))*$`). Keep the regex pragmatic; the authoritative validity check is the live cert issuance.

---

## D8. Observability of issuance/renewal failures (FR-010, spec-deferred)

**Decision**: Treat Caddy's own logs as the observability surface (no new tooling). Caddy logs ACME issuance and renewal attempts/failures to stdout, captured by podman/journald via the `twig.service` unit. The Ansible **post-deploy smoke check** is upgraded to probe `https://{{ twig_domain }}` and fail the play if a valid HTTPS response is not obtained within the retry window — surfacing first-issuance failures at deploy time.

**Rationale**: Satisfies FR-010 ("observable to the operator") without adding a metrics/alerting stack (Principle I / YAGNI). Renewal happens long after deploy; for that, the documented operator check is `podman logs caddy` / `journalctl -u twig.service`. Quickstart documents both.

**Alternatives considered**: dedicated cert-expiry monitoring/alerting — out of scope for this feature; noted as a future enhancement.

---

## Summary of decisions

| ID | Decision |
|----|----------|
| D1 | Cloudflare **DNS-only** record → origin terminates TLS with the Let's Encrypt cert |
| D2 | Default HTTP-01/TLS-ALPN-01 challenges; publish ports **80 and 443**; no DNS-01/token |
| D3 | `twig_acme_environment` prod/staging toggle (default production); staging sets `acme_ca` |
| D4 | Persist Caddy `/data` via host bind mount to survive restarts (rate-limit safety) |
| D5 | `tls { protocols tls1.3 }` per site for the TLS 1.3 floor |
| D6 | Optional `twig_acme_email` → global `email` directive |
| D7 | Ansible `assert` fails fast on missing/invalid `twig_domain` |
| D8 | Caddy logs + HTTPS smoke check as the observability surface; no new stack |

All NEEDS CLARIFICATION items are resolved. No open questions block Phase 1.
