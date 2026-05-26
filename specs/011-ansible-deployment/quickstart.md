# Quickstart: Deploy to the test VM

End-to-end procedure to deploy the todo stack to `user@192.0.2.20` and drive it from the CLI.

## Prerequisites (workstation)

- Ansible ≥ 2.15: `dnf install ansible-core`
- Podman: `dnf install podman`
- Collection: `ansible-galaxy collection install containers.podman`
- SSH key trusted by the VM (already configured per spec assumptions)

## One-time check

```bash
ansible -i deploy/inventories/test.ini todo_servers -m ping
```

Expected: `test-vm | SUCCESS => {"ping": "pong", ...}`.

## Deploy

```bash
# From repo root.
ansible-playbook -i deploy/inventories/test.ini deploy/site.yml
```

Or via the Makefile wrapper:

```bash
make deploy INVENTORY=deploy/inventories/test.ini
```

Expected outcome:

- `PLAY RECAP` shows `test-vm: ok=N changed=M unreachable=0 failed=0`
- The first run reports several `changed=` tasks; a second back-to-back run reports `changed=0` (idempotence; satisfies FR-009 and SC-004).

## Verify the deploy

1. **HTTP reachability**: from the workstation,

   ```bash
   curl -o /dev/null -w "%{http_code}" http://192.0.2.20/health.v1.HealthService/Check \
     -X POST -H "Content-Type: application/json" -d '{}'
   ```

   Expected: `401` (Unauthorized). The auth middleware responding with 401 confirms both
   Caddy and the todo-server are up. (There is no unauthenticated `/healthz` endpoint;
   a 401 is the reliable server-is-alive signal.)

2. **Image came from the on-host registry**: on the VM,

   ```bash
   ssh user@192.0.2.20 -- podman inspect todo_server --format '{{.ImageName}}'
   ```

   Expected: starts with `localhost:5000/todo-server:`.

3. **Provision a user and drive the CLI**: on the VM,

   ```bash
   ssh user@192.0.2.20 -- 'sudo podman exec todo_server /usr/local/bin/todo-server --provision-user alice:secret'
   ```

   Capture the printed API key, then from the workstation:

   ```bash
   export TODO_API_KEY=<key>
   export TODO_ADDR=http://192.0.2.20
   ./services/todo/todo task add "smoke test"
   ./services/todo/todo task        # lists tasks (no subcommand = list)
   ```

   Expected: the added task appears in the list (satisfies SC-001, SC-006).

## Verify persistence (SC-002)

```bash
ssh user@192.0.2.20 -- sudo reboot
# wait ~30s
./services/todo/todo task list  # still shows the smoke task
```

## Verify redeploy preserves data (SC-003)

```bash
# Touch a source file to force a new image, then redeploy.
git commit --allow-empty -m "smoke"
ansible-playbook -i deploy/inventories/test.ini deploy/site.yml
./services/todo/todo task list  # smoke task still present
```

## Verify build-failure safety (SC-005)

Introduce a syntax error in `services/todo/cmd/server/main.go`, run the deploy, confirm:

- The play fails at the build step
- `curl http://192.0.2.20/healthz` still returns `200 OK` (previous stack still serving)

Revert the syntax error before continuing.

## Dry run (check mode, SC-004 / T032)

Run against an already-deployed VM to confirm no unexpected changes would occur:

```bash
ansible-playbook --check -i deploy/inventories/test.ini deploy/site.yml
```

Expected: `PLAY RECAP` shows `changed=0 ... failed=0`. Any non-zero `changed` count means the playbook would modify the VM; investigate before running for real. Note: the build/push steps are skipped in check mode (podman commands are not check-mode-aware), so the check is most meaningful for the render/systemd/config tasks.

## Teardown (development convenience, not part of acceptance)

```bash
ansible -i deploy/inventories/test.ini todo_servers -b -m shell \
  -a 'systemctl stop todo.service registry.service && rm -rf /var/lib/todo'
```

This wipes data. Don't run against a host you care about.
