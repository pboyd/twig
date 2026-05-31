# Contract: Externally-Observable TLS / HTTPS Behavior

**Feature**: 025-https-custom-domain | **Date**: 2026-05-31

This contract defines what an external client observes against a deployment in **production mode** with a correctly-pointed DNS-only Cloudflare record. Each item is the acceptance check for the referenced requirement/success criterion. `$DOMAIN` = the configured `twig_domain`.

## TB-1 — HTTPS reachable with trusted cert (FR-001, FR-002, FR-011, SC-001, SC-005)

```bash
curl -sSI https://$DOMAIN/
```
- Expected: TLS handshake succeeds with **no certificate-trust error** (no `-k` needed).
- Expected: HTTP `200` for the SPA root (or `301/302` to a route Caddy serves), proving the proxy still routes.

```bash
echo | openssl s_client -connect $DOMAIN:443 -servername $DOMAIN 2>/dev/null \
  | openssl x509 -noout -issuer -subject -ext subjectAltName
```
- Expected issuer: a **Let's Encrypt** issuer (publicly-trusted chain).
- Expected SAN: includes `$DOMAIN` (FR-003).

## TB-2 — Certificate matches the configured domain (FR-003, SC-005)

- The cert's SubjectAltName list contains exactly the deployment's `$DOMAIN`. A mismatch is a failure.

## TB-3 — HTTP redirects to HTTPS (FR-008, SC-006)

```bash
curl -sSI http://$DOMAIN/
```
- Expected: `308` (Caddy's permanent redirect) with `Location: https://$DOMAIN/...`. No application content is served over plain HTTP.

## TB-4 — TLS 1.3 negotiated, TLS 1.2 refused (FR-012, SC-007)

```bash
openssl s_client -connect $DOMAIN:443 -servername $DOMAIN -tls1_3 </dev/null   # MUST succeed
openssl s_client -connect $DOMAIN:443 -servername $DOMAIN -tls1_2 </dev/null   # MUST fail (handshake error)
```
- Expected: TLS 1.3 handshake completes; TLS 1.2 handshake is rejected.

## TB-5 — First-issuance timing (SC-003)

- On a fresh deploy where `$DOMAIN` already resolves to the host and ports 80/443 are open, TB-1 succeeds within **5 minutes** of the stack starting (the Ansible HTTPS smoke check enforces this via its retry/delay window).

## TB-6 — No HSTS header (Q2 / Assumptions)

```bash
curl -sSI https://$DOMAIN/ | grep -i strict-transport-security || echo "no HSTS (expected)"
```
- Expected: **no** `Strict-Transport-Security` header is emitted.

## TB-7 — Staging mode (FR-013)

When `twig_acme_environment: staging`:
- TB-1's trust check is **expected to fail** (staging root is untrusted) — verify issuance succeeded with `curl -k` and an issuer containing a staging marker (e.g. `(STAGING)` / `Pretend Pear` / `Fake LE`). This confirms the pipeline works without consuming production quota.

## Renewal (FR-007) — operational check

Not externally point-in-time testable on demand; verified operationally:
- `podman logs caddy` / `journalctl -u twig.service` show renewal activity ahead of expiry; the persisted `/data` cert's `notAfter` advances after renewal (D4, D8).
