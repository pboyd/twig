# Contract: Externally-Observable TLS / HTTPS Behavior (LAN / DNS-01)

**Feature**: 025-https-custom-domain | **Date**: 2026-05-31 (revised)

This contract defines what a **LAN client** observes against a deployment in **production mode** whose certificate was issued via DNS-01. Each item is the acceptance check for the referenced requirement/success criterion.

- `$DOMAIN` = the configured `twig_domain`.
- `$HOST` = the deployment's LAN address (the inventory `ansible_host`, e.g. `192.0.2.10`).

Because the host is on a private LAN and the control node may not resolve `$DOMAIN` to `$HOST` (split-horizon DNS), checks pin resolution with `curl --resolve` / `openssl -connect $HOST`. From a real LAN client whose DNS already resolves `$DOMAIN` to `$HOST`, the plain `https://$DOMAIN/` forms work without `--resolve`.

## TB-1 — HTTPS reachable with trusted cert (FR-001, FR-002, FR-011, FR-014, SC-001, SC-005, SC-008)

```bash
curl -sSI --resolve "$DOMAIN:443:$HOST" "https://$DOMAIN/"
```
- Expected: TLS handshake succeeds with **no certificate-trust error** (no `-k` needed) — even though `$HOST` is a private IP.
- Expected: HTTP `200` for the SPA root (or a redirect to a route Caddy serves), proving the proxy still routes.

```bash
echo | openssl s_client -connect "$HOST:443" -servername "$DOMAIN" 2>/dev/null \
  | openssl x509 -noout -issuer -subject -ext subjectAltName
```
- Expected issuer: a **Let's Encrypt** issuer (publicly-trusted chain).
- Expected SAN: includes `$DOMAIN` (FR-003).

## TB-2 — Certificate matches the configured domain (FR-003, SC-005)

- The cert's SubjectAltName list contains the deployment's `$DOMAIN`. A mismatch is a failure.

## TB-3 — HTTP redirects to HTTPS (FR-008, SC-006)

```bash
curl -sSI --resolve "$DOMAIN:80:$HOST" "http://$DOMAIN/"
```
- Expected: `308` (Caddy's permanent redirect) with `Location: https://$DOMAIN/...`. No application content over plain HTTP.

## TB-4 — TLS 1.3 negotiated, TLS 1.2 refused (FR-012, SC-007)

```bash
openssl s_client -connect "$HOST:443" -servername "$DOMAIN" -tls1_3 </dev/null   # MUST succeed
openssl s_client -connect "$HOST:443" -servername "$DOMAIN" -tls1_2 </dev/null   # MUST fail (handshake error)
```
- Expected: TLS 1.3 handshake completes; TLS 1.2 handshake is rejected.

## TB-5 — First-issuance timing without public reachability (SC-003, FR-014)

- On a fresh deploy with the Cloudflare token configured and `$HOST` on a private LAN (no inbound public connectivity), TB-1 succeeds within **~10 minutes** of the stack starting — the Ansible HTTPS smoke check enforces this via its retry/delay window, allowing for DNS-01 TXT-record propagation.

## TB-6 — No HSTS header (Q2 / Assumptions)

```bash
curl -sSI --resolve "$DOMAIN:443:$HOST" "https://$DOMAIN/" | grep -i strict-transport-security || echo "no HSTS (expected)"
```
- Expected: **no** `Strict-Transport-Security` header is emitted.

## TB-7 — Staging mode (FR-013)

When `twig_acme_environment: staging`:
- TB-1's trust check is **expected to fail** (staging root is untrusted) — verify issuance succeeded with `curl -k --resolve ...` and an issuer containing a staging marker (e.g. `(STAGING)` / `Pretend Pear` / `Fake LE`). Confirms the DNS-01 pipeline works without consuming production quota.

## TB-8 — Secret never exposed (FR-016)

```bash
grep -ri "$CLOUDFLARE_API_TOKEN" "$TWIG_HOST_DATA_DIR/app/compose.yaml" "$TWIG_HOST_DATA_DIR/app/Caddyfile" 2>/dev/null && echo "LEAK" || echo "token absent from compose/Caddyfile (expected)"
ls -l "$TWIG_HOST_DATA_DIR/app/caddy.env"   # expect mode 0600
```
- Expected: the token value does **not** appear in `compose.yaml` or the Caddyfile (only the `{env.CLOUDFLARE_API_TOKEN}` placeholder); `caddy.env` is `0600`. Ansible output shows the token-rendering task as `no_log`.

## Renewal (FR-007) — operational check

Not externally point-in-time testable on demand; verified operationally:
- `podman logs caddy` / `journalctl -u twig.service` show DNS-01 renewal activity ahead of expiry; the persisted `/data` cert's `notAfter` advances after renewal (D4, D8).
