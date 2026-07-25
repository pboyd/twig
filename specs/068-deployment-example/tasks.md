---

description: "Task list for feature implementation"
---

# Tasks: Real-World Deployment Example

**Input**: Design documents from `/specs/068-deployment-example/`

**Prerequisites**: [plan.md](./plan.md), [spec.md](./spec.md), [research.md](./research.md), [data-model.md](./data-model.md), [contracts/config-schema.md](./contracts/config-schema.md), [contracts/host-layout.md](./contracts/host-layout.md), [quickstart.md](./quickstart.md)

**Tests**: No automated test tasks. The spec puts automated deployment testing out of scope; behavior is
verified manually against a throwaway Fedora VM using the 14-item checklist in `quickstart.md`. Each
story phase ends with an explicit manual verification task naming the checks it covers. The only
automated gate is `ansible-lint` / `--syntax-check` in CI (T042).

**Organization**: Tasks are grouped by user story so each story can be implemented and verified
independently.

## Format: `[ID] [P?] [Story] Description`

- **[P]**: Can run in parallel (different files, no dependencies)
- **[Story]**: Which user story this task belongs to (US1–US5)
- Include exact file paths in descriptions

## Path Conventions

All new work lives under `deploy/` at the repository root (see plan.md → Project Structure). Two files
outside it change, both additively: `README.md` and `.github/workflows/ci.yml`. No Go, TypeScript, SQL,
or proto changes.

**Ordering warning**: `roles/twig/templates/Caddyfile.j2`, `roles/twig/templates/compose.yaml.j2`,
`roles/twig/tasks/main.yml`, `roles/twig/tasks/validate.yml`, and `deploy/README.md` are each edited in
several phases. Tasks touching the same file are never marked `[P]` relative to each other, even across
phases.

---

## Phase 1: Setup (Shared Infrastructure)

**Purpose**: Create the copyable `deploy/` tree and its run-time entry points

- [X] T001 Create the directory skeleton `deploy/{group_vars/all,group_vars/twig_servers,roles/base/tasks,roles/twig/tasks,roles/twig/templates}`
- [X] T002 [P] Create `deploy/ansible.cfg` with `inventory = inventory.ini`, `host_key_checking = False`, `retry_files_enabled = False`, `result_format = yaml` — kept inside `deploy/` so a copied tree runs from its new location with no path fixing
- [X] T003 [P] Create `deploy/requirements.yml` declaring the `containers.podman` and `ansible.posix` collections
- [X] T004 [P] Create `deploy/Makefile` with `deps` (`ansible-galaxy install -r requirements.yml`), `check` (`ansible-playbook -K --check --diff -i inventory.ini site.yml`), and `deploy` (`ansible-playbook -K -i inventory.ini site.yml`) per contracts/config-schema.md §6
- [X] T005 [P] Create `deploy/.gitignore` excluding `inventory.ini`, `group_vars/twig_servers/vault.yml`, and `*.retry` so an operator's own copy cannot commit secrets by accident
- [X] T006 [P] Create `deploy/inventory.example.ini` with a single commented `[twig_servers]` host showing `ansible_host` and `ansible_user`, per contracts/config-schema.md §1
- [X] T007 Create `deploy/site.yml`: a `localhost` pre-play asserting the run targets `twig_servers` explicitly (fails on an empty group or a bare `all`), then the `twig_servers` play with `become: true` and roles `base`, `twig`

---

## Phase 2: Foundational (Blocking Prerequisites)

**Purpose**: The configuration surface, the fail-fast gate, and host preparation — everything every
story depends on

**⚠️ CRITICAL**: No user story work can begin until this phase is complete. In particular T011 must land
before any host-mutating task exists, so that no intermediate commit can touch a host without the
validation gate in place (FR-007, SC-005).

- [X] T008 [P] Create `deploy/group_vars/all/defaults.yml` with every default from contracts/config-schema.md §2 (`twig_image`, `twig_image_tag: latest`, `twig_tls_mode: internal`, `twig_acme_email: ""`, `twig_tunnel_enabled: false`, `twig_http_port`, `twig_https_port`, `twig_data_dir`, `twig_postgres_user`, `twig_postgres_db`, the three pinned image refs, `twig_smoke_test_timeout: 600`). Comment that this file is the example's defaults and not operator-owned
- [X] T009 [P] Create `deploy/group_vars/twig_servers/main.yml` holding only the operator knobs — `twig_domain` (required, empty placeholder), plus commented-out `twig_image_tag`, `twig_tls_mode`, `twig_acme_email`, `twig_tunnel_enabled`. Add a comment that `twig_image_tag: latest` should be pinned to a `vX.Y.Z` tag once Twig publishes releases (research.md R1)
- [X] T010 [P] Create `deploy/group_vars/twig_servers/vault.example.yml` with placeholder values only for `twig_postgres_password`, `twig_tunnel_token`, and a commented `ansible_become_password`, plus header comments giving the `cp` + `ansible-vault encrypt` commands (FR-008)
- [X] T011 Create `deploy/roles/twig/tasks/validate.yml` implementing every check in contracts/config-schema.md §4 as `ansible.builtin.assert` tasks: `twig_domain` defined/non-empty/valid hostname regex, `twig_tls_mode` in `[internal, letsencrypt]`, `twig_postgres_password` defined/non-empty, `twig_image_tag` defined/non-empty. Each `fail_msg` names the variable and the file to set it in; the secret assertions set `no_log: true` (FR-009). Conditional asserts for `twig_acme_email` and `twig_tunnel_token` are added in their own phases (T031, T038)
- [X] T012 Create `deploy/roles/base/tasks/main.yml`: `dnf` install `podman`, `podman-compose`, `python3-pip`; create `{{ twig_data_dir }}` and `{{ twig_data_dir }}/{app,caddy}` as `root:root 0755`; create `{{ twig_data_dir }}/postgres` as `owner: "70"`, `group: "70"`, `mode: "0700"` (the `postgres:17-alpine` UID — research.md R4); detect whether `firewalld` is active and, only if so, enable the `http` and `https` services permanently and immediately, skipping cleanly when it is absent (FR-015)

**Checkpoint**: Configuration surface defined, validation gate in place, host preparation complete. User
story work can begin.

---

## Phase 3: User Story 1 - Deploy Twig to a bare Fedora machine (Priority: P1) 🎯 MVP

**Goal**: A clean Fedora VM plus four configuration values becomes a working Twig instance served over
HTTPS on the configured domain, surviving reboots.

**Independent Test**: quickstart.md checks 1–5 — deploy to a fresh VM, load the web UI, provision a user
and drive the CLI, feed a bad config and confirm it fails before touching the host, reboot and confirm
everything returns with data intact.

### Implementation for User Story 1

- [X] T013 [US1] Create `deploy/roles/twig/templates/compose.yaml.j2` with the `postgres`, `twig_server`, and `caddy` services per data-model.md → Service stack: fixed container names (`twig_postgres`, `twig_server`, `twig_caddy`), `restart: unless-stopped` throughout, postgres bind-mount `{{ twig_data_dir }}/postgres:/var/lib/postgresql/data:z` plus a `pg_isready` healthcheck, `twig_server` with `DATABASE_URL` and `depends_on: postgres (service_healthy)` and **no** healthcheck (distroless, no shell — research.md R1), and `caddy` as the only service publishing ports (`{{ twig_http_port }}:80`, `{{ twig_https_port }}:443`) with the Caddyfile and `{{ twig_data_dir }}/caddy:/data:z` mounted
- [X] T014 [US1] Create `deploy/roles/twig/templates/Caddyfile.j2` with the `{{ twig_domain }}` site reverse-proxying to `twig_server:8080`, `encode zstd gzip`, and `tls internal` for the default mode. Factor the routing into a reusable snippet block so the tunnel-hop site (T036) can import it rather than duplicate it
- [X] T015 [P] [US1] Create `deploy/roles/twig/templates/twig.service.j2`: `Type=oneshot`, `RemainAfterExit=yes`, `After=network-online.target`, `WorkingDirectory={{ twig_data_dir }}/app`, `ExecStart=/usr/bin/podman-compose up -d`, `ExecStop=/usr/bin/podman-compose down`, `WantedBy=multi-user.target` (research.md R5, FR-012)
- [X] T016 [US1] Create `deploy/roles/twig/tasks/main.yml`: include `validate.yml` first; render `compose.yaml` and `Caddyfile` into `{{ twig_data_dir }}/app` (`0644 root:root`) registering each result; install the systemd unit and `daemon_reload` + enable it; then gate `podman pull` of each image and `podman-compose up -d` on any rendered artifact having changed, and finish with `systemd: state: started` — this gating is what makes an unchanged re-run a no-op with no restarts (FR-023, SC-003, research.md R6)
- [X] T017 [US1] Add the smoke test to `deploy/roles/twig/tasks/main.yml`: from `localhost` (`become: false`), `curl -o /dev/null -w '%{http_code}'` against `https://{{ twig_domain }}/` using `--resolve {{ twig_domain }}:{{ twig_https_port }}:{{ ansible_host }}` and `-k` (the `internal` cert is deliberately untrusted), retrying until it returns **200**. Assert on the status code, not curl's exit code — an exit-code check passes on a 401 (research.md R7). Bound the retries by `twig_smoke_test_timeout` and fail the play if it never passes (FR-022)
- [X] T018 [US1] Add the deployment-marker task to `deploy/roles/twig/tasks/main.yml`, writing `{{ twig_data_dir }}/.deployed` only after the smoke test passes, and a closing `debug` message in the project's voice telling the operator to provision their first user with `podman exec twig_server /server --provision-user name:password` and to save the printed API key, which is shown exactly once (research.md R8, FR-032)
- [X] T019 [US1] Verify manually on a clean Fedora VM: quickstart.md checks 1–5 (fresh deploy returns HTTP 200, web login plus CLI both work, a missing `twig_domain` fails before any host change, a wrong sudo password fails at escalation with the host untouched, reboot restores the stack with data intact)

**Checkpoint**: US1 is a complete, deployable, self-hosted Twig. This is the MVP — everything after it
is documentation or an optional layer.

---

## Phase 4: User Story 2 - Copy the example and make it your own (Priority: P2)

**Goal**: `deploy/README.md` carries an operator from prerequisites to a running instance and onward to
owning their copy, without them ever reading the roles.

**Independent Test**: quickstart.md check 14 — someone who has not seen the example copies it, changes
the domain and database password using the documentation alone, and deploys successfully.

**Note**: T020–T024 all edit `deploy/README.md`, so none are parallel to each other. Sections covering
the optional layers are written in their own phases (T028, T033, T039); this phase writes everything
that stands on US1 alone.

### Implementation for User Story 2

- [X] T020 [US2] Create `deploy/README.md` with the opening framing plus prerequisites and first-run walkthrough, adapted from quickstart.md §1–4: what the operator needs on their workstation and on the target host, the two files to copy, the four required values, `make deps` / `make deploy`, provisioning the first user, and connecting the web UI and CLI (FR-026)
- [X] T021 [US2] Add the configuration reference to `deploy/README.md`, reproducing contracts/config-schema.md §2–3 as tables: required values, common knobs, rarely-changed knobs, and the vault contents, each with type and default
- [X] T022 [US2] Add the copy-and-own section to `deploy/README.md` (FR-027, FR-028): the `cp -r deploy/ ~/my-infra/` workflow, an explicit split between the operator's files (`inventory.ini`, `group_vars/twig_servers/main.yml`, `vault.yml`) and the example's logic (`site.yml`, `roles/`), the support posture — a starting point they own, not a supported product surface — what is expected to drift as Twig evolves, that the variable names in the config schema are treated as a stable surface, and how to diff against upstream later
- [X] T023 [US2] Add the day-two operations section to `deploy/README.md` (FR-029), reproducing the command set from contracts/host-layout.md: the host filesystem layout with `postgres/` called out as the thing to back up, `pg_dump`/restore, the stack-stopped `tar` alternative, container logs, `systemctl restart`, `psql`, and provisioning additional users
- [X] T024 [US2] Add the domain-resolution section to `deploy/README.md` (FR-030): how the operator's own clients reach `twig_domain` when public DNS does not point at the host — split-horizon DNS or a `/etc/hosts` entry — and the expected browser warning in `internal` TLS mode with what to do about it
- [X] T025 [US2] Verify manually: quickstart.md check 14 — a reader following only `deploy/README.md` changes the domain and database password and deploys successfully. Also confirm the README's tone and table/code-block conventions match the root `README.md` (Principle III as it applies to docs)

**Checkpoint**: The example is usable by someone who did not write it, and copyable without guesswork.

---

## Phase 5: User Story 3 - Redeploy and upgrade without losing data (Priority: P3)

**Goal**: Re-running is boring — an unchanged run changes nothing, an upgrade is a one-value change, and
a misplaced data directory stops the play instead of starting an empty database.

**Independent Test**: quickstart.md checks 6–8 — a second unchanged run reports zero changes, bumping
`twig_image_tag` upgrades with tasks intact, and moving `postgres/` aside makes the play fail loudly.

### Implementation for User Story 3

- [X] T026 [US3] Create `deploy/roles/twig/tasks/data_guards.yml` implementing both redeploy guards from data-model.md → Persistent data, each conditional on `{{ twig_data_dir }}/.deployed` existing: fail if `postgres/PG_VERSION` is missing (the bind mount is pointing somewhere wrong or empty), and fail if the `twig_postgres` container's creation timestamp is newer than the play's start epoch (the container was recreated, which wipes the data directory). Both `fail_msg` values must state what was expected and where to look (FR-024)
- [X] T027 [US3] Wire the guards into `deploy/roles/twig/tasks/main.yml`: record the play-start epoch and stat the `.deployed` marker before the bring-up tasks, and include `data_guards.yml` after the smoke test but **before** the marker is written (T018), so a failed guard leaves the marker reflecting the last known-good deploy
- [X] T028 [US3] Add the upgrade and idempotence section to `deploy/README.md`: bumping `twig_image_tag` as the one-value upgrade, that only `twig_server` is recreated and migrations run on start, that re-running unchanged is a no-op, the `make check` preview and its honest caveat that `--check` is meaningful for redeploys and reports spurious failures on a never-deployed host (research.md R6, FR-025), and that changing `twig_postgres_password` after the database is initialized does not change the existing role's password
- [X] T029 [US3] Verify manually: quickstart.md checks 6–8 (second unchanged run reports zero changed tasks and no restart, `twig_image_tag` bump preserves all data on the new version, `postgres/` moved aside makes the play fail loudly rather than starting empty)

**Checkpoint**: The example is safe to re-run and safe to upgrade — the precondition for documenting it
as production-shaped.

---

## Phase 6: User Story 4 - Serve a trusted certificate for a public domain (Priority: P4)

**Goal**: `twig_tls_mode: letsencrypt` plus a contact email yields a publicly trusted certificate that
auto-renews and is not re-issued on redeploy.

**Independent Test**: quickstart.md check 9 — deploy with `letsencrypt` against a publicly resolvable
domain, confirm a browser reports a trusted connection, then confirm a redeploy reuses the certificate.

### Implementation for User Story 4

- [X] T030 [US4] Extend `deploy/roles/twig/templates/Caddyfile.j2` with the `letsencrypt` branch: emit the global `email {{ twig_acme_email }}` option and let Caddy's default automatic HTTPS handle issuance (HTTP-01 / TLS-ALPN on stock `caddy:2-alpine` — no plugin, no `acme_dns`), keeping `tls internal` for the default mode. Branch on `twig_tls_mode` inside the one template; do not fork it per mode
- [X] T031 [US4] Add the conditional assertion to `deploy/roles/twig/tasks/validate.yml`: `twig_acme_email` must be defined and non-empty when `twig_tls_mode == 'letsencrypt'`, with a `fail_msg` explaining that Let's Encrypt requires a contact address and naming the file to set it in
- [X] T032 [US4] Make the smoke test's `-k` flag conditional in `deploy/roles/twig/tasks/main.yml` — omitted in `letsencrypt` mode so the check also proves the certificate is trusted, retained in `internal` mode where it is deliberately not. Confirm `twig_smoke_test_timeout` leaves room for first-issuance latency, and that a failed issuance surfaces as a smoke-test failure rather than a reported success (FR-022, scenario US4-4)
- [X] T033 [US4] Add the certificate section to `deploy/README.md`: enabling `letsencrypt`, its prerequisites (public DNS A/AAAA to the host, inbound 80 **and** 443), that state in `{{ twig_data_dir }}/caddy` persists so redeploys do not re-issue (FR-018), and the honest limitation from contracts/config-schema.md §5 — HTTP-01 means a NAT-bound host cannot get an origin certificate, use `internal` + tunnel instead, and DNS-01 is the extension path if genuinely needed (a custom Caddy image with the provider's plugin plus an `acme_dns` block)
- [X] T034 [US4] Verify manually: quickstart.md check 9 (trusted certificate issued against a public domain; a redeploy reuses rather than re-issues it)

**Checkpoint**: Public deployments get real TLS; the limitation is documented rather than hidden.

---

## Phase 7: User Story 5 - Reach a NAT-bound instance from anywhere (Priority: P5)

**Goal**: `twig_tunnel_enabled: true` plus a connector token makes the instance reachable from outside
the network with no inbound ports opened.

**Independent Test**: quickstart.md checks 10–12 — reachable from off-LAN with the tunnel on, fails
before touching the host when the token is missing, and leaves no `cloudflared` container behind when
off.

### Implementation for User Story 5

- [X] T035 [US5] Extend `deploy/roles/twig/templates/compose.yaml.j2` with a `cloudflared` service (`twig_cloudflared`, `command: tunnel --no-autoupdate run`, `env_file` pointing at `cloudflared.env`, `depends_on: caddy`, `restart: unless-stopped`) emitted only when `twig_tunnel_enabled` — when the toggle is off the service must be absent from the rendered file entirely, not merely stopped (FR-020)
- [X] T036 [US5] Extend `deploy/roles/twig/templates/Caddyfile.j2` with the tunnel-hop site, emitted only when `twig_tunnel_enabled`: a site declared with the explicit `http://{{ twig_domain }}` scheme importing the shared routing snippet from T014. Include a comment explaining the trap — a bare `:80` block loses to the named `:443` site's host-matched auto-redirect and produces an infinite redirect through the tunnel — and that the cloudflared↔Caddy hop needs no TLS because it is already inside the tunnel (research.md R3)
- [X] T037 [US5] Add the `cloudflared.env` render task to `deploy/roles/twig/tasks/main.yml`, writing `TUNNEL_TOKEN=` with `mode: "0600"`, `owner: root`, `no_log: true`, only when `twig_tunnel_enabled`, and register it so it participates in the change-gating from T016 (FR-009)
- [X] T038 [US5] Add the conditional assertion to `deploy/roles/twig/tasks/validate.yml`: `twig_tunnel_token` must be defined and non-empty when `twig_tunnel_enabled`, with `no_log: true` and a `fail_msg` naming `vault.yml` and pointing at the Cloudflare Zero Trust dashboard as the source of the token
- [X] T039 [US5] Add the tunnel section to `deploy/README.md`: what it is for (no inbound ports, works behind NAT), the manual steps that cannot be automated because they happen outside the host — creating the tunnel and mapping the public hostname to `http://caddy:80` in the Cloudflare dashboard (FR: US5-4) — where the token goes, and that Cloudflare's edge supplies the publicly trusted certificate so `twig_tls_mode: internal` is the right pairing
- [X] T040 [US5] Verify manually: quickstart.md checks 10–12 (reachable from off-LAN with no inbound ports; missing token fails before any host change; with the toggle off no `cloudflared` container exists on the host). Also confirm the both-layers-on combination works without a redirect loop (FR-021)

**Checkpoint**: All five stories complete; all four TLS/tunnel combinations behave as documented.

---

## Phase 8: Polish & Cross-Cutting Concerns

- [X] T041 [P] Add a deployment pointer to the root `README.md`, from the local-development quick start to `deploy/` — one or two sentences distinguishing `make dev` (local) from `deploy/` (a real host), in the README's existing voice (FR-031)
- [X] T042 [P] Add an `ansible` job to `.github/workflows/ci.yml` running `ansible-playbook --syntax-check` and `ansible-lint` against `deploy/`, matching the existing jobs' structure (research.md R10). Do not attempt to provision a VM in CI
- [X] T043 Run `ansible-lint` over `deploy/` locally and clear its findings, so the CI job added in T042 passes on the first run
- [X] T044 Run the full 14-item verification checklist in quickstart.md against a freshly provisioned Fedora VM, including check 13 (`pg_dump` → fresh deploy → restore recovers all tasks, goals, and plans — SC-010). Record the outcome; this is the feature's acceptance evidence
- [X] T045 Audit `deploy/` before merge against the constitution's Quality Gates: no real secrets anywhere (SC-006), no `[ALL_CAPS_IDENTIFIER]` placeholder tokens left in any committed file (Gate 4), every operator-facing string — `assert` failure messages, the closing provisioning message, README prose — reviewed for Principle IV tone while staying accurate and actionable (FR-032, Gate 5)

---

## Dependencies & Execution Order

### Phase Dependencies

- **Setup (Phase 1)**: no dependencies
- **Foundational (Phase 2)**: depends on Setup — **blocks every story**. T011 (validation) must precede
  every host-mutating task that follows
- **US1 (Phase 3)**: depends on Foundational. Blocks US3, US4, US5 — all three modify files US1 creates
- **US2 (Phase 4)**: depends on US1 (it documents a working deploy). Its optional-layer sections are
  deferred to T028/T033/T039 inside the later phases, so US2 stays independently completable
- **US3 (Phase 5)**: depends on US1
- **US4 (Phase 6)**: depends on US1
- **US5 (Phase 7)**: depends on US1; T036 depends specifically on T014's shared routing snippet
- **Polish (Phase 8)**: T044/T045 depend on every story being complete; T041/T042 depend only on
  `deploy/` existing

### User Story Dependencies

- **US1 (P1)**: independent — the MVP
- **US2 (P2)**: needs US1 to exist to describe; independent of US3–US5
- **US3 (P3)**: independent of US2, US4, US5
- **US4 (P4)**: independent of US2, US3, US5
- **US5 (P5)**: independent of US2, US3, US4

US3, US4, and US5 all edit `roles/twig/tasks/main.yml`, `validate.yml`, and the templates. They are
logically independent but will conflict textually if run concurrently — sequence them, or expect
merges.

### Parallel Opportunities

- Phase 1: T002–T006 in parallel (five distinct new files)
- Phase 2: T008, T009, T010 in parallel; T011 and T012 are distinct files and may also run alongside
- Phase 3: T015 is parallel to T013/T014; T016–T018 are all `main.yml` and strictly sequential
- Phase 8: T041 and T042 in parallel

### Parallel Example: Phase 1

```bash
# After T001 creates the skeleton, launch together:
Task: "Create deploy/ansible.cfg"
Task: "Create deploy/requirements.yml"
Task: "Create deploy/Makefile"
Task: "Create deploy/.gitignore"
Task: "Create deploy/inventory.example.ini"
```

---

## Implementation Strategy

### MVP First (US1 only)

1. Phase 1: Setup
2. Phase 2: Foundational — validation gate before anything can touch a host
3. Phase 3: US1
4. **STOP and VALIDATE**: quickstart.md checks 1–5 against a throwaway VM
5. At this point Twig is genuinely self-hostable; everything after is documentation or an optional layer

### Incremental Delivery

1. Setup + Foundational → configuration surface and host prep
2. US1 → a working deployment (**MVP**)
3. US2 → someone other than the author can use and copy it
4. US3 → safe to re-run and upgrade
5. US4 → trusted certificates for public domains
6. US5 → reachable from behind NAT
7. Polish → discoverability (root README), CI gate, full checklist run, pre-merge audit

### Parallel Team Strategy

After US1 lands, US3, US4, and US5 are logically independent but share files — assign them to one
person in sequence, or accept merge conflicts in `main.yml`, `validate.yml`, and the two templates. US2
(documentation) is the cleanest split for a second person, provided the optional-layer sections
(T028, T033, T039) stay with whoever implements those layers.

---

## Notes

- **No automated behavior tests.** Every host-touching claim is verified manually per quickstart.md.
  Tasks say so explicitly; do not infer a test suite that does not exist.
- `[P]` = different files, no dependencies on incomplete tasks.
- Commit after each task or logical group.
- `twig_image_tag` defaults to `latest` only because the repository has no version tags yet. When the
  first `vX.Y.Z` is published, change the default and update the README note (research.md R1).
- Load-bearing details inherited from the reference deployment, easy to lose and expensive to
  rediscover: postgres UID `70:70` with mode `0700`, `:z` on every bind mount, the explicit
  `http://<domain>` scheme on the tunnel-hop site, and gating bring-up on rendered-artifact changes.
