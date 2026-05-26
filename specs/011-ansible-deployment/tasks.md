---
description: "Tasks for Ansible deployment to dedicated VM"
---

# Tasks: Ansible Deployment to Dedicated VM

**Input**: Design documents from `/specs/011-ansible-deployment/`

**Prerequisites**: plan.md, spec.md, research.md, data-model.md, quickstart.md

**Tests**: No automated unit/integration tests are generated for this feature. Acceptance is the quickstart procedure run against `user@192.0.2.20` (per plan.md "Testing"). A `--check` dry-run task and the quickstart smoke checks are the verification surface.

**Organization**: Grouped by user story (US1, US2, US3 from spec.md), priority order.

## Format: `[ID] [P?] [Story] Description`

- **[P]**: Can run in parallel (different files, no dependency on incomplete tasks)
- **[Story]**: Maps to user stories from spec.md (US1, US2, US3)
- File paths are repo-relative

## Path Conventions

All new files live under `deploy/` at the repo root. The Go source under `services/todo/` is **not modified** by this feature. One Makefile entry is added.

---

## Phase 1: Setup (shared infrastructure)

**Purpose**: Create the `deploy/` skeleton and workstation prerequisites.

- [ ] T001 Create directory skeleton: `deploy/`, `deploy/inventories/`, `deploy/group_vars/`, `deploy/roles/{base,registry,app}/{tasks,templates}/` (empty directories acceptable)
- [ ] T002 [P] Add `deploy/ansible.cfg` declaring `inventory = inventories/test.ini`, `host_key_checking = False` (it's a LAN VM), `retry_files_enabled = False`, `stdout_callback = yaml`
- [ ] T003 [P] Add `deploy/inventories/test.ini` with one host `test-vm ansible_host=192.0.2.20 ansible_user=user` under group `[todo_servers]`
- [ ] T004 [P] Add `deploy/group_vars/all.yml` with the variables listed in data-model.md ("Inventory model") — image name, tag (from `git rev-parse --short HEAD`), data dir, ports, postgres creds
- [ ] T005 [P] Add `deploy/site.yml` that targets `todo_servers`, becomes root, and imports roles `base`, `registry`, `app` in that order
- [ ] T006 [P] Add `make deploy` target to `Makefile` that runs `ansible-playbook -i $(INVENTORY) deploy/site.yml`, defaulting `INVENTORY` to `deploy/inventories/test.ini`
- [ ] T007 Document workstation prereqs in `deploy/README.md`: `ansible-core`, `podman`, `ansible-galaxy collection install containers.podman`, SSH key trust expectation

**Checkpoint**: `ansible-playbook --syntax-check deploy/site.yml` succeeds; `ansible -i deploy/inventories/test.ini todo_servers -m ping` succeeds.

---

## Phase 2: Foundational (`base` role — blocking prerequisite for everything)

**Purpose**: VM is prepared with podman, podman-compose, and the host directory layout. Required by both the registry role (US1) and the app role (US1). No user story can run without these.

- [ ] T008 Implement `deploy/roles/base/tasks/main.yml`: install packages `podman`, `podman-compose`, `python3-pip` via `ansible.builtin.dnf`; ensure `systemd` is present (it is, but the task makes it explicit/idempotent)
- [ ] T009 In `deploy/roles/base/tasks/main.yml`: create `/var/lib/todo/`, `/var/lib/todo/postgres/`, `/var/lib/todo/registry/`, `/var/lib/todo/app/` using `ansible.builtin.file state=directory` with `mode=0755`, owner `root`
- [ ] T010 In `deploy/roles/base/tasks/main.yml`: chown `/var/lib/todo/postgres/` to uid:gid `999:999` (postgres-alpine's user) so the bind-mount is writable on first run

**Checkpoint**: Re-running just the `base` role yields `changed=0`.

---

## Phase 3: User Story 1 — First-time deployment to a fresh VM (Priority: P1) 🎯 MVP

**Goal**: Single `ansible-playbook` invocation against a fresh VM brings up the registry, builds and pushes images, renders production compose, starts the stack via systemd, and serves traffic via the reverse proxy on port 80.

**Independent test**: Quickstart steps "One-time check" → "Deploy" → "Verify the deploy" succeed end-to-end on the test VM, including `curl http://192.0.2.20/healthz` returning 200 and the CLI being able to add and list tasks (SC-001, SC-006).

### Registry role (sub-stack)

- [ ] T011 [US1] Create `deploy/roles/registry/templates/registry-compose.yaml.j2`: one service `registry` from `docker.io/library/registry:2`, ports `127.0.0.1:{{ todo_registry_port }}:5000`, volume `{{ todo_host_data_dir }}/registry:/var/lib/registry`, `restart: unless-stopped` (satisfies FR-008a)
- [ ] T012 [US1] Implement `deploy/roles/registry/tasks/main.yml`: render the compose template to `{{ todo_host_data_dir }}/registry/compose.yaml`; create `/etc/systemd/system/registry.service` (Type=oneshot, RemainAfterExit=yes, ExecStart=`/usr/bin/podman-compose up -d`, ExecStop=`/usr/bin/podman-compose down`, WorkingDirectory=`{{ todo_host_data_dir }}/registry`, After=network-online.target, WantedBy=multi-user.target); `systemctl daemon-reload`; `systemctl enable --now registry.service` — all via `ansible.builtin.template`, `ansible.builtin.copy`, and `ansible.builtin.systemd`
- [ ] T013 [US1] Add a wait/poll step at end of registry role: `ansible.builtin.uri url=http://127.0.0.1:{{ todo_registry_port }}/v2/` from the VM, retry until 200, fail loudly otherwise (proves registry is actually serving before app role tries to push)

**Checkpoint after registry**: `ssh user@192.0.2.20 -- curl -fsS http://127.0.0.1:5000/v2/` returns `{}`.

### App role — build & push (runs on workstation via delegate_to)

- [ ] T014 [US1] In `deploy/roles/app/tasks/main.yml`: `delegate_to: localhost`, `become: false` block that runs `podman build -t localhost:5000/{{ todo_image }}:{{ todo_image_tag }} -t localhost:5000/{{ todo_image }}:latest services/todo`. Register result; abort the play on failure (satisfies FR-011, SC-005)
- [ ] T015 [US1] In `deploy/roles/app/tasks/main.yml`: open an SSH local port-forward to the VM's registry port using `ansible.builtin.shell` `delegate_to: localhost` with `ssh -fN -o ExitOnForwardFailure=yes -L {{ todo_registry_port }}:127.0.0.1:{{ todo_registry_port }} {{ ansible_user }}@{{ ansible_host }}`; capture the PID for teardown
- [ ] T016 [US1] In `deploy/roles/app/tasks/main.yml`: `delegate_to: localhost` `podman push localhost:5000/{{ todo_image }}:{{ todo_image_tag }}` and `podman push localhost:5000/{{ todo_image }}:latest`. Use a block with `always:` to kill the SSH forwarding PID from T015 whether push succeeded or failed
- [ ] T017 [US1] After push, verify on the VM with `ansible.builtin.uri url=http://127.0.0.1:{{ todo_registry_port }}/v2/{{ todo_image }}/tags/list` — must list `{{ todo_image_tag }}`

### App role — render production compose

- [ ] T018 [US1] [P] Create `deploy/roles/app/templates/Caddyfile.j2`: a site block listening on `:80` that reverse-proxies to `todo_server:8080`. Comment a one-line snippet showing how a future TLS site would be added (FR-015 "structured so TLS can be added later")
- [ ] T019 [US1] [P] Create `deploy/roles/app/templates/compose.yaml.j2` defining three services:
  - `postgres` from `docker.io/library/postgres:17-alpine`, env from `todo_postgres_*` vars, volume `{{ todo_host_data_dir }}/postgres:/var/lib/postgresql/data`, healthcheck `pg_isready -U {{ todo_postgres_user }}`, `restart: unless-stopped`, no published ports
  - `todo_server` from `localhost:5000/{{ todo_image }}:{{ todo_image_tag }}` (pinned SHA — drives idempotence per research §4), `DATABASE_URL` env wired to the postgres service, `depends_on: postgres (service_healthy)`, `restart: unless-stopped`, no published ports
  - `caddy` from `docker.io/library/caddy:2-alpine`, ports `{{ todo_http_port }}:80`, mounts the rendered Caddyfile, `restart: unless-stopped`, `depends_on: todo_server`
- [ ] T020 [US1] In `deploy/roles/app/tasks/main.yml`: render `compose.yaml.j2` → `{{ todo_host_data_dir }}/app/compose.yaml` and `Caddyfile.j2` → `{{ todo_host_data_dir }}/app/Caddyfile`. Register `changed` from these two templates

### App role — systemd unit & start

- [ ] T021 [US1] In `deploy/roles/app/tasks/main.yml`: create `/etc/systemd/system/todo.service` (Type=oneshot, RemainAfterExit=yes, ExecStart=`/usr/bin/podman-compose up -d`, ExecStop=`/usr/bin/podman-compose down`, WorkingDirectory=`{{ todo_host_data_dir }}/app`, After=`network-online.target registry.service`, Requires=`registry.service`, WantedBy=multi-user.target); `systemctl daemon-reload`; `systemctl enable todo.service`
- [ ] T022 [US1] In `deploy/roles/app/tasks/main.yml`: invoke `podman-compose up -d` in `{{ todo_host_data_dir }}/app` using `ansible.builtin.command` with `chdir`, only if the compose file or systemd unit changed in T020/T021, AND `systemctl start todo.service` (idempotent — no-op if already running with same compose; satisfies FR-009, SC-004)
- [ ] T023 [US1] Smoke-check at end of app role: `ansible.builtin.uri url=http://{{ ansible_host }}/healthz` from `delegate_to: localhost`, expect 200, retry briefly to allow container warmup

**Checkpoint (US1 done = MVP)**: Quickstart "Deploy" + "Verify the deploy" pass cleanly. Acceptance scenarios US1.1, US1.2, US1.3 all hold.

---

## Phase 4: User Story 2 — Data persistence across reboots and redeploys (Priority: P1)

**Goal**: PostgreSQL data and registry blobs survive reboot, container recreation, and redeploys. (Largely a consequence of US1 design — the work here is to **prove** it and prevent regressions.)

**Independent test**: Quickstart "Verify persistence (SC-002)" and "Verify redeploy preserves data (SC-003)" both succeed.

- [ ] T024 [US2] Audit `compose.yaml.j2` (created in T019) to confirm postgres uses the host bind-mount and not a named volume — add a comment to the template explaining why (named volumes are owned by podman's storage, less obvious to back up later)
- [ ] T025 [US2] Audit `registry-compose.yaml.j2` (T011) for the same: bind-mount, not named volume
- [ ] T026 [US2] Add a play assertion at the end of `app` role: `ansible.builtin.stat` on `{{ todo_host_data_dir }}/postgres/PG_VERSION` (created by initdb on first boot); on **second** play run only (when first-run already happened), fail loudly if it's missing — guards against accidentally pointing the bind-mount somewhere wrong in the future. Use a fact stored in `/var/lib/todo/.deployed` to distinguish first run
- [ ] T027 [US2] Document the persistence story in `deploy/README.md`: which directories are stateful, what wiping them does, how to back them up (one paragraph each; no actual backup tooling — out of scope per spec)

**Checkpoint (US2 done)**: After `sudo reboot` of the VM and a redeploy with a new commit SHA, `todo task list` returns the pre-existing tasks.

---

## Phase 5: User Story 3 — Repeatable redeploys for new versions (Priority: P2)

**Goal**: Re-running the playbook with new source upgrades the server cleanly without restarting PostgreSQL, and with no source change it is fully idempotent.

**Independent test**: After US1, make a trivial code change, redeploy, confirm new version is serving and the postgres container has not been recreated. Then redeploy again with no change; PLAY RECAP shows `changed=0`.

- [ ] T028 [US3] In `deploy/roles/app/tasks/main.yml`, ensure the only thing that triggers `podman-compose up -d` is a change to `{{ todo_host_data_dir }}/app/compose.yaml` (image tag flip) or the systemd unit (rare). Concretely: use `notify`/handlers, or `when:` against the registered results of T020/T021. PostgreSQL service's definition does not change between deploys, so podman-compose will leave it alone (idempotence-by-content)
- [ ] T029 [US3] Add an end-of-play assertion: query `podman inspect todo_postgres --format '{{ .Created }}'` via `ansible.builtin.command delegate_to: VM`; on a redeploy run (detected via T026's `.deployed` flag), fail if the `Created` timestamp is newer than the start of this play run — proves postgres was not recreated (covers FR-010 and SC-004)
- [ ] T030 [US3] Add a teardown safety guard: `site.yml` requires `--limit` or `-i` to be explicitly provided. Add an `assert` at the top of `site.yml` that `groups.todo_servers | length > 0` and that the user did not target `all` — satisfies FR-012 ("must not silently default to a production host")

**Checkpoint (US3 done)**: A second back-to-back `make deploy` reports `changed=0`. After a trivial commit, `make deploy` reports `changed >= 1` but T029 confirms postgres untouched.

---

## Phase 6: Polish & cross-cutting

- [ ] T031 [P] Run `ansible-lint deploy/` and fix any findings (or document acceptable violations inline)
- [ ] T032 [P] Add `--check` smoke task to quickstart: document running `ansible-playbook --check -i deploy/inventories/test.ini deploy/site.yml` against an already-deployed VM and confirm it reports no changes — gives maintainers a safe "what would change?" command
- [ ] T033 Add troubleshooting section to `deploy/README.md`: how to read logs (`journalctl -u todo.service`, `podman logs todo_server`), how to manually push an image, how to enter the postgres container
- [ ] T034 Run the full quickstart end-to-end against `user@192.0.2.20` and check off SC-001 through SC-006 explicitly in a closing comment on the feature branch

---

## Dependencies

```text
Phase 1 (Setup, T001–T007)
    │
    ▼
Phase 2 (Foundational base role, T008–T010)   ← blocks everything below
    │
    ▼
Phase 3 (US1 — MVP, T011–T023)
    │
    ├──▶ Phase 4 (US2 — persistence proofs, T024–T027) ── can start once T019, T011 written
    │
    └──▶ Phase 5 (US3 — redeploy idempotence, T028–T030) ── depends on T020/T021 existing

Phase 6 (Polish, T031–T034) ── runs after US1, US2, US3 done
```

US2 and US3 can be developed **in parallel after US1** — they touch different concerns (assertions/comments in templates already authored by US1 vs. handler/idempotence logic in tasks/main.yml).

## Parallel execution opportunities

Within Phase 1: T002, T003, T004, T005, T006 all touch different new files — fully parallel after T001 makes the directories.

Within Phase 3 (US1):
- T018 (Caddyfile.j2) and T019 (compose.yaml.j2) — different files, fully parallel
- T011 (registry template) can be written in parallel with T018/T019, since they're in different role directories
- T014–T017 are a serial pipeline (build → forward → push → verify) — not parallelizable internally

Across stories after US1: T024/T025 (US2 audits) can be done in parallel with T028 (US3 handler refactor).

## MVP scope

**Phase 1 + Phase 2 + Phase 3 (US1)** is the MVP. At that point the test VM is deployable, reachable, and the CLI works against it. Phases 4 and 5 harden the guarantees the spec demands but the system is functionally usable without them.

## Format validation

All tasks above:
- start with `- [ ]`
- have a sequential `T###` ID
- carry a `[Story]` label inside phases 3–5 (none in setup/foundational/polish per the rules)
- reference at least one concrete file path under `deploy/` (or `Makefile`)
