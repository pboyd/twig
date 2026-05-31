# Quickstart: Deploy twig over HTTPS with a custom domain

**Feature**: 025-https-custom-domain | **Date**: 2026-05-31

Operator runbook for bringing up (or migrating) a twig deployment onto HTTPS with a Let's Encrypt certificate at a Cloudflare-hosted domain.

## Prerequisites

1. **A public host.** The target host must have a public, internet-routable IP and be reachable on **TCP 80 and 443**. (Let's Encrypt production cannot validate a domain pointing at a private `192.168.x` address — use `staging` first if you only have a private host, see below.)
2. **Cloudflare DNS, DNS-only.** In the Cloudflare dashboard for your zone, create an `A` (and/or `AAAA`) record:
   - Name: your subdomain (or `@` for the apex), e.g. `twig`.
   - Value: the host's public IP.
   - Proxy status: **DNS only (grey cloud)** — *not* proxied/orange. This is required so Let's Encrypt validates the origin and end users get the origin's cert over TLS 1.3.
3. Wait for the record to resolve: `dig +short twig.example.com` returns the host IP.

## Configure

Set the domain (and optionally the ACME contact) in inventory `host_vars`/`group_vars`, e.g. in `deploy/group_vars/all.yml` or a per-host file:

```yaml
twig_domain: "twig.example.com"      # required
twig_acme_email: "ops@example.com"   # optional but recommended for expiry notices
# twig_acme_environment: production    # default; set to "staging" while testing
```

> Forgetting `twig_domain` is safe: the play fails immediately with a clear message rather than starting in a broken state.

## (Recommended) Dry-run against Let's Encrypt staging first

Production rate limits are strict. Validate the whole pipeline against staging (issues an **untrusted** cert, generous limits):

```yaml
twig_acme_environment: "staging"
```

Run the deploy, then confirm issuance ignoring trust:

```bash
curl -kI https://twig.example.com/        # 200/redirect proves the cert was issued
openssl s_client -connect twig.example.com:443 -servername twig.example.com </dev/null 2>/dev/null \
  | openssl x509 -noout -issuer            # issuer shows a staging marker
```

Then flip back to `production` (or remove the line) and redeploy to get the trusted cert.

## Deploy

From `deploy/`:

```bash
ansible-playbook -i inventories/<your-inventory>.ini site.yml
```

The play will: assert `twig_domain` is valid → render the Caddyfile (named-host site, TLS 1.3 floor, ACME options) and compose (ports 80+443, persistent `/data`) → (re)start the stack → wait for `https://twig_domain` to respond.

## Verify (production mode)

```bash
# Trusted HTTPS, no -k needed:
curl -sSI https://twig.example.com/

# Cert is Let's Encrypt and matches the domain:
echo | openssl s_client -connect twig.example.com:443 -servername twig.example.com 2>/dev/null \
  | openssl x509 -noout -issuer -subject -ext subjectAltName

# HTTP redirects to HTTPS (expect 308 + Location: https://):
curl -sSI http://twig.example.com/

# TLS 1.3 required, TLS 1.2 refused:
openssl s_client -connect twig.example.com:443 -servername twig.example.com -tls1_3 </dev/null  # succeeds
openssl s_client -connect twig.example.com:443 -servername twig.example.com -tls1_2 </dev/null  # fails
```

All checks correspond to `contracts/tls-behavior.md` (TB-1…TB-4).

## Troubleshooting

- **Cert not issued / play smoke check fails**: check Caddy logs on the host —
  `podman logs caddy` or `journalctl -u twig.service`. Common causes: record still proxied (orange cloud), DNS not yet propagated, ports 80/443 not open, or production rate limit hit (use staging to iterate).
- **Browser trust error in production**: confirm `twig_acme_environment` is `production` (staging certs are untrusted by design).
- **Rate-limited by Let's Encrypt**: you have likely been re-issuing. The persistent `/data` mount prevents this across restarts; wait out the limit window and iterate on `staging` meanwhile.

## Renewal

Automatic — Caddy renews well before expiry and persists certs to `/var/lib/twig/caddy`. To confirm later, check that the cert's `notAfter` has advanced and review Caddy logs for renewal entries. No operator action is required in normal operation.
