# Phase 1 Data Model: Ansible Deployment

This feature does not introduce any new application-level data. "Data model" here means the **deployment topology and host artifacts** that the playbook creates and manages.

## Host filesystem entities (on the VM)

| Path | Owner | Purpose | Persistence |
|---|---|---|---|
| `/var/lib/todo/postgres/` | `999:999` (postgres container user) | PostgreSQL data directory; bind-mounted into the postgres container as `/var/lib/postgresql/data` | Survives container recreation, redeploy, and VM reboot |
| `/var/lib/todo/registry/` | `root:root` (registry runs as root in container) | Registry blob storage; bind-mounted as `/var/lib/registry` | Survives container recreation, redeploy, and VM reboot |
| `/var/lib/todo/app/compose.yaml` | `root:root` | Rendered production compose file used by `todo.service` | Overwritten on each deploy |
| `/var/lib/todo/app/Caddyfile` | `root:root` | Rendered reverse proxy config | Overwritten on each deploy |
| `/var/lib/todo/registry/compose.yaml` | `root:root` | Rendered registry compose file used by `registry.service` | Overwritten only if the registry config template changes |

## systemd units

| Unit | Type | WorkingDirectory | What it does | Ordering |
|---|---|---|---|---|
| `registry.service` | `oneshot` with `RemainAfterExit=yes` | `/var/lib/todo/registry/` | `ExecStart=/usr/bin/podman-compose up -d`, `ExecStop=/usr/bin/podman-compose down` | `After=network-online.target` |
| `todo.service` | `oneshot` with `RemainAfterExit=yes` | `/var/lib/todo/app/` | `ExecStart=/usr/bin/podman-compose up -d`, `ExecStop=/usr/bin/podman-compose down` | `After=registry.service`, `Requires=registry.service` |

Both are `enabled` so they fire on boot. Container-level `restart: unless-stopped` (in the compose files) handles crash recovery within a running stack; systemd handles reboot recovery.

## Container topology (running on the VM)

```text
[ host port 80 ]
       │
       ▼
┌───────────────┐
│   caddy       │  reverse proxy, container in "todo" compose
└───────┬───────┘
        │  (compose network: todo_default)
        ▼
┌───────────────┐
│  todo-server  │  pulls localhost:5000/todo-server:<sha>
└───────┬───────┘
        │
        ▼
┌───────────────┐
│  postgres     │  binds /var/lib/todo/postgres → /var/lib/postgresql/data
└───────────────┘

────── separate compose project: "registry" ──────
[ 127.0.0.1:5000 on host ] ──▶ ┌───────────┐
                               │ registry  │ binds /var/lib/todo/registry → /var/lib/registry
                               └───────────┘
```

## Inventory model

| Group | Hosts | Variables |
|---|---|---|
| `todo_servers` | `test-vm` (`192.0.2.20`) | `ansible_user=user`, `ansible_become=true` |

`group_vars/all.yml` defines:

```yaml
todo_image: "todo-server"
todo_image_tag: "{{ lookup('pipe', 'git rev-parse --short HEAD') }}"
todo_host_data_dir: "/var/lib/todo"
todo_http_port: 80
todo_registry_port: 5000
todo_postgres_user: "todo"
todo_postgres_password: "todo"
todo_postgres_db: "todo"
```

`todo_postgres_password` matches the dev default. **Not a secret in this feature** (matches the dev environment exactly per the spec); when production warrants it, the Ansible Vault wrapper is a future change, not part of this feature.

## State transitions during a deploy

```text
[idle]
   │  ansible-playbook -i inventories/test.ini site.yml
   ▼
[role: base]      install podman + podman-compose, create /var/lib/todo/*, ensure user in podman group
   │
   ▼
[role: registry]  render registry compose, ensure registry.service is enabled+started
   │
   ▼
[role: app]
   ├─ delegate_to: localhost — podman build services/todo, tag as localhost:5000/todo-server:<sha>
   ├─ open SSH -L 5000:127.0.0.1:5000 (background, scoped to this play)
   ├─ delegate_to: localhost — podman push (both <sha> and :latest)
   ├─ close SSH tunnel
   ├─ render /var/lib/todo/app/compose.yaml and Caddyfile (pinned to <sha>)
   ├─ ensure todo.service is enabled
   └─ podman-compose up -d  (no-op if image tag unchanged)
   │
   ▼
[done]            run a smoke check: HTTP GET http://VM/healthz returns 200
```

If the **build/push** step fails, `app` aborts before touching `compose.yaml` or restarting the stack — satisfies FR-011.

## What this feature does NOT model

- Application database schema (managed by the server's own migrations on startup; unchanged).
- API contracts (no new endpoints).
- User accounts on the deployed system (still provisioned via `todo-server --provision-user` against the running container, same as dev).
