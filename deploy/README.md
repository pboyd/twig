# Deploying Twig

A self-contained Ansible example that turns a bare Fedora host into a working Twig instance.

Two roles, four configuration values, and a first run that needs no external service.

## Prerequisites

**On your workstation**:
- `ansible-core` >= 2.15
- `git`, `curl`
- An SSH key that logs into the target host

```bash
cd deploy
make deps          # installs the containers.podman and ansible.posix collections
```

**On the target host**:
- A fresh Fedora Server install (or equivalent minimal install)
- SSH enabled, with a login account that has `sudo` access
- That account's sudo password

Nothing else — no podman, no database, no manual preparation.

**Domain**: pick the name you want Twig on, e.g. `twig.example.com`. For the default `internal` TLS
mode it does not need to be public; it only has to resolve to the host for the clients you use. Adding
a line to your workstation's `/etc/hosts` is enough to try it out.

## 1. Configure

Two files. Copy the examples, fill in four values.

```bash
cp inventory.example.ini inventory.ini
cp group_vars/twig_servers/vault.example.yml group_vars/twig_servers/vault.yml
```

`inventory.ini`:

```ini
[twig_servers]
twig-host ansible_host=192.168.1.50 ansible_user=operator
```

`group_vars/twig_servers/main.yml`:

```yaml
twig_domain: twig.example.com
```

`group_vars/twig_servers/vault.yml` — set the database password, then encrypt it:

```bash
ansible-vault encrypt group_vars/twig_servers/vault.yml
```

## 2. Deploy

```bash
make deploy        # prompts for the vault password, then the sudo password
```

The play installs podman, creates `/var/lib/twig`, renders the compose file and Caddyfile, installs and
enables `twig.service`, brings the stack up, and then waits for `https://twig.example.com/` to return
HTTP 200 before declaring success. First run takes a few minutes, mostly pulling images.

If it fails on a configuration value, nothing has been changed on the host — fix the value and re-run.

## 3. Create your first user

The API key is printed once and cannot be recovered. Save it.

```bash
ssh operator@192.168.1.50 \
  "sudo podman exec twig_server /server --provision-user alice:secret"
# → provisioned user "alice" — API key: twig_...
```

## 4. Use it

**Web**: open `https://twig.example.com` and log in as `alice`. In `internal` TLS mode your browser
warns about the certificate — expected; it is Caddy's local CA. Accept it, or switch to
`twig_tls_mode: letsencrypt` if the domain is public.

**CLI**:

```bash
export TWIG_ADDR=https://twig.example.com
export TWIG_API_KEY=twig_...
twig task add "try out my own twig server"
twig                      # the TUI
```

## 5. Verify it survives a reboot

```bash
ssh operator@192.168.1.50 "sudo reboot"
sleep 60
curl -k -o /dev/null -w '%{http_code}\n' https://twig.example.com/    # → 200
twig task list                                                        # → your task is still there
```

---

## Configuration reference

### Required values

| Variable | Type | Description |
|---|---|---|
| `twig_domain` | hostname | The domain Twig is served on, e.g. `twig.example.com`. No scheme, no path, no port. |

Set in `group_vars/twig_servers/main.yml`.

### Common knobs

| Variable | Type | Default | Description |
|---|---|---|---|
| `twig_image` | image ref | `ghcr.io/pboyd/twig-server` | Published server image (bundles the web UI) |
| `twig_image_tag` | string | `latest` | **Pin this to a `vX.Y.Z` tag once releases exist.** The one value you change to upgrade. |
| `twig_tls_mode` | `internal` \| `letsencrypt` | `internal` | Certificate mode. See [TLS and access](#tls-and-access) below. |
| `twig_acme_email` | email | `""` | Contact address for Let's Encrypt. Required when `twig_tls_mode: letsencrypt`. |
| `twig_tunnel_enabled` | bool | `false` | Run a Cloudflare Tunnel connector for access from outside a NAT'd network. |

### Rarely changed

| Variable | Type | Default | Description |
|---|---|---|---|
| `twig_http_port` | int | `80` | Host port mapped to Caddy's `:80` |
| `twig_https_port` | int | `443` | Host port mapped to Caddy's `:443` |
| `twig_data_dir` | path | `/var/lib/twig` | Root of all persistent state on the host |
| `twig_postgres_user` | string | `twig` | Database role |
| `twig_postgres_db` | string | `twig` | Database name |

### Secrets (vault.yml)

| Variable | Required | Description |
|---|---|---|
| `twig_postgres_password` | **yes** | Database password |
| `twig_tunnel_token` | when tunnel enabled | Cloudflare Tunnel connector token |
| `ansible_become_password` | no | Alternative to `-K` prompt |

---

## TLS and access

`twig_tls_mode` and `twig_tunnel_enabled` are independent. What each combination gives you:

| TLS mode | Tunnel | Who can reach it | Trusted cert |
|---|---|---|---|
| `internal` | off | Clients that resolve the domain to the host (LAN / hosts file) | no (Caddy local CA) |
| `letsencrypt` | off | Anyone on the internet | yes |
| `internal` | on | Anyone, via Cloudflare edge; LAN clients direct | yes for tunnel traffic |
| `letsencrypt` | on | Anyone, by either path | yes on both |

### Let's Encrypt

Requires public DNS pointing at the host and inbound ports 80 and 443.

```yaml
# group_vars/twig_servers/main.yml
twig_tls_mode: letsencrypt
twig_acme_email: you@example.com
```

Caddy obtains and auto-renews the certificate. The state lives in `/var/lib/twig/caddy` and survives
redeploys, so re-running does not re-issue.

**Known limitation**: Let's Encrypt uses HTTP-01, which needs inbound port 80. A host behind NAT
cannot obtain an origin certificate with this example. Use `internal` + tunnel instead — Cloudflare's
edge already presents a trusted certificate to public clients.

### Cloudflare Tunnel (no inbound ports)

Makes the instance reachable from outside a NAT'd network. No inbound ports are opened — the
connector dials out to Cloudflare. Cloudflare's edge supplies the publicly trusted certificate, so
you can leave `twig_tls_mode: internal`.

In the Cloudflare dashboard (Networking → Tunnels): create a tunnel, copy its connector token, and
add a public hostname mapping `twig.example.com → http://caddy:80`.

```yaml
# group_vars/twig_servers/main.yml
twig_tunnel_enabled: true
```

```yaml
# group_vars/twig_servers/vault.yml (encrypted)
twig_tunnel_token: <connector token>
```

**Manual steps** (cannot be automated — they happen outside the host):
1. Create a tunnel in the Cloudflare dashboard (Networking Trust → Tunnels).
2. Add a public hostname: map your domain to `http://caddy:80`.
3. Copy the connector token into `vault.yml`.

---

## Copy and own

```bash
cp -r deploy/ ~/my-infra/twig-deploy/
cd ~/my-infra/twig-deploy && git init && git add . && git commit -m "twig deployment, mine now"
```

**Your files** (edit these): `inventory.ini`, `group_vars/twig_servers/main.yml`, `vault.yml`

**The example's logic** (copy, but you own the copy): `site.yml` and `roles/`

This copy is a fork, not a dependency — nothing phones home and nothing auto-updates. To see what
changed upstream later, diff your copy against `deploy/` in a fresh checkout of Twig. The compose
template and the image tag default may move as Twig evolves; the variable names in this README are
treated as a stable surface and changes to them are called out in the changelog.

---

## Day two

### Backup (logical — preferred; safe while running)

```bash
sudo podman exec twig_postgres pg_dump -U twig twig | gzip > twig-$(date +%F).sql.gz
```

### Restore onto a freshly deployed host

```bash
gunzip -c twig-2026-07-25.sql.gz | sudo podman exec -i twig_postgres psql -U twig twig
```

### Backup (filesystem — only with the stack stopped)

```bash
sudo systemctl stop twig.service
sudo tar czf twig-data-$(date +%F).tar.gz -C /var/lib/twig postgres
sudo systemctl start twig.service
```

### Logs

```bash
sudo podman logs -f twig_server
sudo podman logs -f twig_caddy
```

### Restart / stop the whole stack

```bash
sudo systemctl restart twig.service
sudo systemctl stop    twig.service
```

### Database shell

```bash
sudo podman exec -it twig_postgres psql -U twig twig
```

### Provision additional users

```bash
sudo podman exec twig_server /server --provision-user <name>:<password>
```

The API key is printed once and cannot be recovered.

---

## Domain resolution

When `twig_tls_mode` is `internal`, the domain does not need to be public. Your clients just need to
resolve `twig_domain` to the host's IP. Two options:

- **Split-horizon DNS**: point `twig.example.com` at the host in your internal DNS server.
- **`/etc/hosts`**: add `192.168.1.50 twig.example.com` to each client.

In `internal` mode, your browser will warn about the certificate — expected; it is Caddy's local CA.
You can accept the warning once, or switch to `twig_tls_mode: letsencrypt` if the domain is public.

---

## Upgrading

One value, then re-run:

```yaml
# group_vars/twig_servers/main.yml
twig_image_tag: v1.2.0
```

```bash
make check         # preview the diff
make deploy
```

Only `twig_server` is recreated. Migrations run on start. The database is untouched, and the play
aborts if it detects otherwise.

An unchanged `make deploy` is a no-op — zero changed tasks, zero restarts.

Use `make check` to preview what a redeploy would change. Caveat: `--check` is meaningful for
**redeploys**. On a host that has never been deployed it reports failures for tasks that inspect
state which does not exist yet; that is expected, not a bug.

**Note**: changing `twig_postgres_password` after the database is initialized does **not** change the
existing role's password. The password is only read at container start.
