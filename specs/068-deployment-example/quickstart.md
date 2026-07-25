# Quickstart: Deploying Twig to a Fedora Host

**Feature**: 068-deployment-example
**Date**: 2026-07-25

This is the walkthrough the shipped `deploy/README.md` is built from — the ten-minute path from a bare
Fedora machine to a working Twig instance. It doubles as the manual verification script for this
feature.

---

## Before you start

**On your workstation**: `ansible-core` ≥ 2.15, `git`, `curl`, and an SSH key that logs into the target
host.

```bash
cd deploy
make deps          # installs the containers.podman and ansible.posix collections
```

**On the target host**: a fresh Fedora Server install (or equivalent minimal install), SSH enabled, and
a login account with `sudo` access. You need that account's sudo password. Nothing else — no podman, no
database, no manual preparation.

**Domain**: pick the name you want Twig on, e.g. `twig.example.com`. For the default `internal` TLS
mode it does not need to be public; it only has to resolve to the host for the clients you use. Adding
a line to your workstation's `/etc/hosts` is enough to try it out.

---

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

Full variable reference: `contracts/config-schema.md`.

---

## 2. Deploy

```bash
make deploy        # prompts for the vault password, then the sudo password
```

The play installs podman, creates `/var/lib/twig`, renders the compose file and Caddyfile, installs and
enables `twig.service`, brings the stack up, and then waits for `https://twig.example.com/` to return
HTTP 200 before declaring success. First run takes a few minutes, mostly pulling images.

If it fails on a configuration value, nothing has been changed on the host — fix the value and re-run.

---

## 3. Create your first user

The API key is printed once and cannot be recovered. Save it.

```bash
ssh operator@192.168.1.50 \
  "sudo podman exec twig_server /server --provision-user alice:secret"
# → provisioned user "alice" — API key: twig_...
```

---

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

---

## 5. Verify it survives a reboot

```bash
ssh operator@192.168.1.50 "sudo reboot"
sleep 60
curl -k -o /dev/null -w '%{http_code}\n' https://twig.example.com/    # → 200
twig task list                                                        # → your task is still there
```

---

## Optional: a publicly trusted certificate

Requires public DNS pointing at the host and inbound ports 80 and 443.

```yaml
# group_vars/twig_servers/main.yml
twig_tls_mode: letsencrypt
twig_acme_email: you@example.com
```

`make deploy` again. Caddy obtains and thereafter auto-renews the certificate; the state lives in
`/var/lib/twig/caddy` and survives redeploys, so re-running does not re-issue.

## Optional: reach it from outside a NAT'd network

In the Cloudflare dashboard (Zero Trust → Tunnels): create a tunnel, copy its connector token, and add
a public hostname mapping `twig.example.com → http://caddy:80`. Then:

```yaml
# group_vars/twig_servers/main.yml
twig_tunnel_enabled: true
```

```yaml
# vault.yml
twig_tunnel_token: <connector token>
```

`make deploy` again. No inbound ports are opened; the connector dials out. Cloudflare's edge presents
the publicly trusted certificate, so you can leave `twig_tls_mode: internal`.

---

## Upgrading

One value, then re-run:

```yaml
twig_image_tag: v1.2.0
```

```bash
make check         # preview the diff
make deploy
```

Only `twig_server` is recreated. Migrations run on start. The database is untouched, and the play
aborts if it detects otherwise.

---

## Day two

Backup, restore, logs, restart, and database access: see `contracts/host-layout.md`, which the shipped
README reproduces.

---

## Making it your own

```bash
cp -r deploy/ ~/my-infra/twig-deploy/
cd ~/my-infra/twig-deploy && git init && git add . && git commit -m "twig deployment, mine now"
```

Yours to edit: `inventory.ini`, `group_vars/twig_servers/main.yml`, `vault.yml`.
The example's logic: `site.yml` and `roles/`.

The copy is a fork, not a dependency — nothing phones home and nothing auto-updates. To see what
changed upstream later, diff your copy against `deploy/` in a fresh checkout of Twig. Expect the
compose template and the image tag default to move as Twig evolves; the variable names in
`config-schema.md` are treated as a stable surface and changes to them are called out in the
deployment README.

---

## Manual verification checklist

The acceptance path for this feature. Run against a throwaway Fedora VM.

| # | Check | Spec |
|---|---|---|
| 1 | Fresh VM → `make deploy` → success, `https://<domain>/` returns 200 | US1 |
| 2 | Web login works; CLI with the provisioned key lists tasks | US1 |
| 3 | Missing `twig_domain` → fails before any host change | US1 / FR-007 |
| 4 | Wrong sudo password → fails at escalation, host untouched | Edge case |
| 5 | Reboot → stack returns, data intact | US1 / FR-012 |
| 6 | Second unchanged `make deploy` → zero changed tasks, no restart | US3 / FR-023 |
| 7 | Bump `twig_image_tag` → new version, tasks intact | US3 / SC-004 |
| 8 | Move `postgres/` aside, re-run → play fails loudly | US3 / FR-024 |
| 9 | `twig_tls_mode: letsencrypt` on a public domain → trusted cert; re-run does not re-issue | US4 |
| 10 | Tunnel enabled → reachable from off-LAN, no inbound ports | US5 |
| 11 | Tunnel enabled with no token → fails before any host change | US5 |
| 12 | Tunnel disabled → no `cloudflared` container on the host | US5 / FR-020 |
| 13 | `pg_dump` → fresh deploy → restore → all tasks, goals, plans back | SC-010 |
| 14 | A reader following only `deploy/README.md` can change domain + DB password and deploy | US2 / SC-007 |
