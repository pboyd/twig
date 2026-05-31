# Quickstart: Deploy twig over HTTPS on a LAN (DNS-01)

**Feature**: 025-https-custom-domain | **Date**: 2026-05-31 (revised)

Operator runbook for bringing up a twig deployment over HTTPS with a Let's Encrypt certificate, when the host lives on a **private LAN** (internal IP, not reachable from the public internet). Certificates are obtained via the **DNS-01 challenge through Cloudflare**, so the host never needs an inbound public connection.

## Prerequisites

1. **A LAN host with outbound internet.** The host needs outbound HTTPS to Let's Encrypt and the Cloudflare API. It does **not** need a public IP or inbound 80/443 from the internet.
2. **Domain hosted in Cloudflare.** The configured domain's zone is managed in Cloudflare.
3. **A Cloudflare API token.** Create a scoped token (My Profile → API Tokens → Create Token → *Edit zone DNS*), scoped to **Zone → DNS → Edit** for your zone only. Keep it secret.
4. **LAN name resolution.** LAN clients must resolve the domain to the host's internal address — e.g. a split-horizon entry on your router/internal DNS pointing `twig.example.com` → `192.0.2.10`. (The certificate is publicly trusted regardless, so browsers show no warning.)

## Configure

Set the domain and token in inventory `host_vars`/`group_vars`. **Store the token with Ansible Vault**, not in plaintext:

```bash
# create/edit an encrypted vars file
ansible-vault create deploy/inventories/host_vars/<host>/vault.yml
```

```yaml
# vault.yml (encrypted)
twig_cloudflare_api_token: "cf_xxx_your_scoped_token"
```

```yaml
# host_vars/<host>/main.yml (or group_vars), plaintext is fine for non-secrets
twig_domain: "twig.example.com"      # required
twig_acme_email: "ops@example.com"   # optional but recommended for expiry notices
# twig_acme_environment: production    # default; set to "staging" while testing
# twig_acme_dns_resolver: 1.1.1.1      # default; override only on unusual networks
```

> Forgetting `twig_domain` or `twig_cloudflare_api_token` is safe: the play fails immediately with a clear message (the token is handled with `no_log`, so it never prints) rather than starting in a broken state.

## (Recommended) Dry-run against Let's Encrypt staging first

Production rate limits are strict. Validate the whole DNS-01 pipeline against staging (issues an **untrusted** cert, generous limits):

```yaml
twig_acme_environment: "staging"
```

Run the deploy, then confirm issuance ignoring trust (pin resolution to the host's LAN IP):

```bash
HOST=192.0.2.10
curl -kI --resolve "twig.example.com:443:$HOST" https://twig.example.com/   # 200/redirect proves issuance
openssl s_client -connect "$HOST:443" -servername twig.example.com </dev/null 2>/dev/null \
  | openssl x509 -noout -issuer            # issuer shows a staging marker
```

Then flip back to `production` (or remove the line) and redeploy to get the trusted cert.

## Deploy

From `deploy/` (add `--ask-vault-pass` or `--vault-password-file` for the encrypted token):

```bash
ansible-playbook -i inventories/<your-inventory>.ini site.yml --ask-vault-pass
```

The play will: assert `twig_domain` and `twig_cloudflare_api_token` are set → build & push the custom Caddy image (with the `caddy-dns/cloudflare` module) to the on-host registry if not already present → render `caddy.env` (`0600`), the Caddyfile (DNS-01 global option, TLS 1.3 floor, resolver) and compose (custom image, `env_file`, persistent `/data`, ports 80/443) → (re)start the stack → wait (up to ~10 min) for HTTPS to come up.

## Verify (production mode)

Pin resolution to the host's LAN IP so the checks don't depend on your workstation's DNS view:

```bash
HOST=192.0.2.10
D=twig.example.com

# Trusted HTTPS, no -k needed (private IP, public cert):
curl -sSI --resolve "$D:443:$HOST" "https://$D/"

# Cert is Let's Encrypt and matches the domain:
echo | openssl s_client -connect "$HOST:443" -servername "$D" 2>/dev/null \
  | openssl x509 -noout -issuer -subject -ext subjectAltName

# HTTP redirects to HTTPS (expect 308 + Location: https://):
curl -sSI --resolve "$D:80:$HOST" "http://$D/"

# TLS 1.3 required, TLS 1.2 refused:
openssl s_client -connect "$HOST:443" -servername "$D" -tls1_3 </dev/null  # succeeds
openssl s_client -connect "$HOST:443" -servername "$D" -tls1_2 </dev/null  # fails

# Secret is not on disk in the clear:
grep -ri "$twig_cloudflare_api_token" /var/lib/twig/app/compose.yaml /var/lib/twig/app/Caddyfile || echo "token absent (expected)"
```

All checks correspond to `contracts/tls-behavior.md` (TB-1…TB-8).

## Troubleshooting

- **Cert not issued / play smoke check fails**: check Caddy logs on the host — `podman logs caddy` or `journalctl -u twig.service`. Common causes: wrong/expired Cloudflare token, token not scoped to the zone, `_acme-challenge` TXT not propagating (try `dig TXT _acme-challenge.$D @1.1.1.1`), or production rate limit hit (use staging to iterate).
- **`dns module not found` / Caddy won't start with `acme_dns`**: the running image is stock Caddy, not the custom build. Confirm the custom `twig-caddy` image was built/pushed and that compose references it.
- **Browser trust error in production**: confirm `twig_acme_environment` is `production` (staging certs are untrusted by design).
- **LAN clients can't reach the app**: the cert is fine, but the domain isn't resolving to the host on the LAN. Add/fix the split-horizon DNS record → `$HOST`.

## Renewal

Automatic — Caddy renews via DNS-01 well before expiry and persists certs to `/var/lib/twig/caddy`. To confirm later, check that the cert's `notAfter` has advanced and review Caddy logs for renewal entries. No operator action is required in normal operation.
