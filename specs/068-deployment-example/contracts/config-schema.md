# Contract: Deployment Configuration Schema

**Feature**: 068-deployment-example
**Consumers**: operators of the example deployment; the Ansible roles that read these variables
**Stability**: this is the example's public surface. Renaming or removing a variable is a breaking
change for anyone who copied it, and belongs in the deployment README's changelog section.

The configuration surface is deliberately small. Everything below is an Ansible variable; precedence
follows Ansible's normal rules (inventory host vars beat group vars beat the example's defaults).

---

## 1. `inventory.ini` — which host

Copied from `inventory.example.ini`. Operator-owned.

```ini
[twig_servers]
twig-host ansible_host=192.168.1.50 ansible_user=operator
```

| Variable | Type | Required | Default | Notes |
|---|---|---|---|---|
| `ansible_host` | IP or hostname | **yes** | — | reachable over SSH from the control node; also used by the smoke test's `curl --resolve` |
| `ansible_user` | string | **yes** | — | key-based SSH login with `sudo` rights |
| `ansible_become_password` | string | no | — | prefer `-K` at run time; if set, put it in `vault.yml`, never here |

Exactly one host. The play fails if the `twig_servers` group is empty or the run targets `all`.

---

## 2. `group_vars/twig_servers/main.yml` — the knobs

Operator-owned, cleartext, safe to commit in the operator's own repository.

### Required

| Variable | Type | Default | Description |
|---|---|---|---|
| `twig_domain` | hostname / FQDN | — | the domain Twig is served on, e.g. `twig.example.com`. No scheme, no path, no port. Rejected if malformed. |

### Common

| Variable | Type | Default | Description |
|---|---|---|---|
| `twig_image` | image ref | `ghcr.io/pboyd/twig-server` | published server image (bundles the web UI) |
| `twig_image_tag` | string | `latest` | **pin this to a `vX.Y.Z` tag once releases exist.** The one value you change to upgrade. |
| `twig_tls_mode` | `internal` \| `letsencrypt` | `internal` | see §5 |
| `twig_acme_email` | email | `""` | contact address for Let's Encrypt. Required when `twig_tls_mode: letsencrypt`. |
| `twig_tunnel_enabled` | bool | `false` | run a Cloudflare Tunnel connector for access from outside a NAT'd network |

### Rarely changed

| Variable | Type | Default | Description |
|---|---|---|---|
| `twig_http_port` | int | `80` | host port mapped to Caddy's `:80` |
| `twig_https_port` | int | `443` | host port mapped to Caddy's `:443` |
| `twig_data_dir` | absolute path | `/var/lib/twig` | root of all persistent state on the host. Changing it after the first deploy will trip the data-safety guard — see the README's migration note. |
| `twig_postgres_user` | string | `twig` | database role |
| `twig_postgres_db` | string | `twig` | database name |
| `twig_postgres_image` | image ref | `docker.io/library/postgres:17-alpine` | pinned major version; changing the major requires a manual `pg_upgrade` |
| `twig_caddy_image` | image ref | `docker.io/library/caddy:2-alpine` | stock Caddy; no plugins required |
| `twig_cloudflared_image` | image ref | `docker.io/cloudflare/cloudflared:latest` | only used when the tunnel is enabled |
| `twig_smoke_test_timeout` | int (seconds) | `600` | how long to wait for the site to answer; the ceiling matters mainly for first ACME issuance |

---

## 3. `group_vars/twig_servers/vault.yml` — the secrets

Operator-owned, **encrypted** with `ansible-vault`. Created by copying `vault.example.yml` and
encrypting it. The example repository contains only the placeholder file.

| Variable | Type | Required | Description |
|---|---|---|---|
| `twig_postgres_password` | string | **yes** | database password. Read only at container start; changing it after the database is initialized does **not** change the existing role's password — see the README. |
| `twig_tunnel_token` | string | when `twig_tunnel_enabled` | Cloudflare Tunnel connector token from the Zero Trust dashboard |
| `ansible_become_password` | string | no | alternative to `-K` for unattended runs |

```bash
cp group_vars/twig_servers/vault.example.yml group_vars/twig_servers/vault.yml
ansible-vault encrypt group_vars/twig_servers/vault.yml
ansible-vault edit    group_vars/twig_servers/vault.yml
```

---

## 4. Validation contract

Every check below runs **before** the first task that modifies the host. Each failure names the
offending variable and the file to set it in. Checks on secrets run with `no_log: true`.

| Check | Failure condition |
|---|---|
| target is explicit | `twig_servers` empty, or the run targets `all` |
| `twig_domain` | undefined, empty, or not a valid hostname/FQDN |
| `twig_tls_mode` | not one of `internal`, `letsencrypt` |
| `twig_acme_email` | empty while `twig_tls_mode == letsencrypt` |
| `twig_postgres_password` | undefined or empty |
| `twig_tunnel_token` | undefined or empty while `twig_tunnel_enabled` |
| `twig_image_tag` | undefined or empty |

---

## 5. TLS and access matrix

`twig_tls_mode` and `twig_tunnel_enabled` are independent. What each combination gives you:

| `twig_tls_mode` | `twig_tunnel_enabled` | Who can reach it | Publicly trusted certificate | Prerequisites |
|---|---|---|---|---|
| `internal` | `false` | clients that resolve `twig_domain` to the host (LAN / split-horizon DNS / hosts file) | no — Caddy's local CA; browsers warn once | none |
| `letsencrypt` | `false` | anyone on the internet | yes | public DNS A/AAAA → host; inbound `80` **and** `443` |
| `internal` | `true` | anyone, through the Cloudflare edge; LAN clients still direct | yes for tunnel traffic (Cloudflare edge cert); no for direct LAN | Cloudflare account, tunnel + public hostname configured in the dashboard |
| `letsencrypt` | `true` | anyone, by either path | yes on both | all of the above, including inbound `80` for the ACME challenge |

**Known limitation**: `letsencrypt` uses the HTTP-01 / TLS-ALPN challenge, which needs inbound port 80.
There is no DNS-01 option, so a host behind NAT **cannot** obtain an origin certificate with this
example. That combination is unnecessary in practice — with the tunnel on, Cloudflare already presents
a trusted certificate to public clients — so use `internal` + `tunnel`. If you genuinely need DNS-01,
the extension is a custom Caddy image containing your DNS provider's plugin plus an `acme_dns` block;
the README says where to add it.

**Manual steps the example cannot automate**: creating the Cloudflare Tunnel, and mapping its public
hostname to `http://caddy:80`. Both happen in the Cloudflare dashboard (Zero Trust → Tunnels).

---

## 6. Run-time interface

| Command | Effect |
|---|---|
| `make deploy` | `ansible-playbook -K -i inventory.ini site.yml` |
| `make check` | `ansible-playbook -K --check --diff -i inventory.ini site.yml` — preview only |
| `make deps` | `ansible-galaxy install -r requirements.yml` |

`--check` is meaningful for **redeploys**. On a host that has never been deployed it reports failures
for tasks that inspect state which does not exist yet; that is expected, not a bug.

---

## 7. Values consumed but not configurable

| Value | Why fixed |
|---|---|
| server listen port `8080` | hardcoded in the Twig server (`main.go`) |
| `DATABASE_URL` | composed from the postgres variables; the server's only required env var |
| migration execution | the server runs migrations itself on every start |
| compose project location | `{{ twig_data_dir }}/app` |
| systemd unit name | `twig.service` |
