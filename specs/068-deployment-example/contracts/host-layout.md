# Contract: Target Host Layout

**Feature**: 068-deployment-example
**Consumers**: operators (for backup, debugging, and disaster recovery); the Ansible roles that create
these paths

What the deployment puts on the host, and what the operator can rely on finding there. This is the
other half of the example's public surface: paths here appear in the documented backup, restore, and
troubleshooting procedures, so moving one is a breaking change for anyone following those docs.

---

## Filesystem

```text
/var/lib/twig/                          # twig_data_dir — root of all persistent state
├── app/                                # rendered configuration (regenerated every run)
│   ├── compose.yaml                    # 0644 root:root
│   ├── Caddyfile                       # 0644 root:root
│   └── cloudflared.env                 # 0600 root:root — only when the tunnel is enabled
├── postgres/                           # 70:70, 0700 — THE TASK DATABASE. Back this up.
├── caddy/                              # certificates + ACME account + renewal state
└── .deployed                           # marker: a successful deploy has happened here

/etc/systemd/system/twig.service        # 0644 root:root
```

**Guarantees**:

| Path | Guarantee |
|---|---|
| `app/` | rewritten on every run; nothing operator-authored should be kept here |
| `postgres/` | never touched by the playbook after creation; readable by host backup tools |
| `caddy/` | survives redeploys, so certificates are not re-issued (avoids ACME rate limits) |
| `.deployed` | written only after the smoke test passes; its presence enables the data-safety guards |

The database password does **not** appear in any file on the host except the rendered `compose.yaml`
(`0644`) as part of `DATABASE_URL` — the same exposure `podman inspect` already gives to any user who
can run podman. The tunnel token, which grants network access to the operator's Cloudflare account, is
kept in a `0600` env file instead.

---

## Packages installed

`podman`, `podman-compose`, `python3-pip` — via `dnf`, `state: present`. Nothing is removed or
upgraded beyond what these require.

## Firewall

When `firewalld` is active: the `http` and `https` services are enabled, permanently and immediately.
When `firewalld` is absent or inactive: skipped without failing. No other rule is changed.

## SELinux

Left enforcing. Bind mounts use the `:z` relabel flag; no policy, boolean, or mode change is made.

---

## Containers

| Name | Image | Ports on host | Restart policy |
|---|---|---|---|
| `twig_postgres` | `postgres:17-alpine` | none | `unless-stopped` |
| `twig_server` | `ghcr.io/pboyd/twig-server:<tag>` | none | `unless-stopped` |
| `twig_caddy` | `caddy:2-alpine` | `twig_http_port`→80, `twig_https_port`→443 | `unless-stopped` |
| `twig_cloudflared` | `cloudflare/cloudflared:latest` | none | `unless-stopped` |

Container names are fixed (not left to compose's project-name prefixing) because they appear in the
documented operator commands below.

## systemd

`twig.service` — `Type=oneshot`, `RemainAfterExit=yes`, `WorkingDirectory=/var/lib/twig/app`,
`ExecStart=/usr/bin/podman-compose up -d`, `ExecStop=/usr/bin/podman-compose down`, enabled for
`multi-user.target`.

---

## Operator commands these paths support

```bash
# Provision a user (prints the API key once — save it)
sudo podman exec twig_server /server --provision-user alice:secret

# Logs
sudo podman logs -f twig_server
sudo podman logs -f twig_caddy

# Restart / stop the whole stack
sudo systemctl restart twig.service
sudo systemctl stop    twig.service

# Database shell
sudo podman exec -it twig_postgres psql -U twig

# Backup (logical — preferred; safe while running)
sudo podman exec twig_postgres pg_dump -U twig twig | gzip > twig-$(date +%F).sql.gz

# Restore onto a freshly deployed host
gunzip -c twig-2026-07-25.sql.gz | sudo podman exec -i twig_postgres psql -U twig twig

# Backup (filesystem — only with the stack stopped)
sudo systemctl stop twig.service
sudo tar czf twig-data-$(date +%F).tar.gz -C /var/lib/twig postgres
sudo systemctl start twig.service
```

---

## What the deployment never does

- Modify the SSH configuration, users, or `sudoers`.
- Disable or reconfigure SELinux.
- Remove or downgrade existing packages.
- Touch anything outside `/var/lib/twig`, `/etc/systemd/system/twig.service`, the packages listed
  above, and the two firewall services.
- Delete the `postgres/` directory — under any code path, including failure and rollback.
