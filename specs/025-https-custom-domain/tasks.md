---
description: "Task list for HTTPS with Custom Domain (LAN / DNS-01)"
---

# Tasks: HTTPS with Custom Domain (LAN / DNS-01)

**Input**: Design documents from `/specs/025-https-custom-domain/`

**Prerequisites**: plan.md, spec.md, research.md (revised), data-model.md, contracts/deployment-config.md, contracts/tls-behavior.md, quickstart.md

**Tests**: This is a deployment/infrastructure feature with no application-code or unit-test surface. Per plan.md (Testing), verification is **Ansible idempotence** plus **external TLS checks** (`curl --resolve`/`openssl s_client`) against the host's LAN address using the configured domain SNI, mapped to `contracts/tls-behavior.md` (TB-1…TB-8). These appear as explicit verification tasks in the Polish phase, not as a TDD test-first phase.

**Organization**: Tasks are grouped by user story to enable independent implementation and testing of each story.

## Format: `[ID] [P?] [Story] Description`

- **[P]**: Can run in parallel (different files, no dependencies)
- **[Story]**: Which user story this task belongs to (US1, US2, US3)
- All paths are repo-relative; this feature touches **only** `deploy/` (no `services/` changes).

## Revision context (DNS-01 pivot)

This list **supersedes** the prior HTTP-01 task list. The operator clarified the host is on a **private LAN, not publicly reachable**, so issuance now uses the **DNS-01 challenge via the Cloudflare API** (research.md D1–D11). The tasks below are the delta from the **current repo state** to the DNS-01 design.

### Already in place (retained from prior work — do NOT redo)

These exist in `deploy/` and carry over unchanged:

- `Caddyfile.j2`: named-host site `{{ twig_domain }}`, global `email`/`acme_ca` (staging) conditionals, `tls { protocols tls1.3 }` (FR-012/D5).
- `compose.yaml.j2`: caddy publishes `80` + `443`, `/data` bind mount (D4), `depends_on`/`restart`.
- `app/tasks/main.yml`: `twig_domain` fail-fast assert (FR-009/D7); twig-server build/push pipeline (the pattern T003 mirrors).
- `base/tasks/main.yml`: creates `{{ twig_host_data_dir }}/caddy`; opens `http`/`https` in firewalld.
- `group_vars/all.yml`: `twig_acme_environment`, `twig_acme_email`, `twig_http_port`, `twig_https_port`.

## ⚠️ Shared-file note (read before parallelizing)

Several tasks edit the **same files**, so they are frequently **not** parallelizable:

- `deploy/roles/app/tasks/main.yml` — edited by **Foundational** (build/push custom caddy image, T003), **US1** (smoke check, T008), **US2** (token assert T009, `caddy.env` render T010).
- `deploy/roles/app/templates/Caddyfile.j2` — edited within **US1** (T005 → T006), sequentially.
- `deploy/roles/app/templates/compose.yaml.j2` — edited only by **US1** (T007).
- `deploy/README.md` — edited by **US2** (T012) and **US3** (T013) and Polish (T018).

Treat `[P]` as "different file, no incomplete dependency" only; respect the shared-file constraints in Dependencies below.

---

## Phase 1: Setup (Shared Configuration)

**Purpose**: Declare the new per-deployment configuration consumed by the DNS-01 flow.

- [X] T001 In `deploy/group_vars/all.yml`, add `twig_acme_dns_resolver` (default `"1.1.1.1"`, for DNS-01 propagation checks under split-horizon DNS, D11). **Do NOT** add `twig_cloudflare_api_token` here — it is a required secret with no default and must be set per host/group (via Ansible Vault) so an unset value fails the play (FR-017, data-model.md). Add a comment pointing operators to set the token in vaulted `host_vars`. Existing vars (`twig_acme_environment`, `twig_acme_email`, ports) are unchanged.

**Checkpoint**: Config surface defined per `contracts/deployment-config.md` §1.

---

## Phase 2: Foundational (Blocking Prerequisites)

**Purpose**: Produce the DNS-01-capable Caddy image and adjust host posture — shared by all stories. Stock `caddy:2-alpine` cannot do DNS-01, so nothing issues until this lands.

**⚠️ CRITICAL**: No user story is verifiable until this phase is complete.

- [X] T002 Create `deploy/roles/app/files/Containerfile.caddy`: a two-stage build using `docker.io/library/caddy:2-builder` to `xcaddy build --with github.com/caddy-dns/cloudflare`, copying the resulting `/usr/bin/caddy` into a `docker.io/library/caddy:2-alpine` runtime stage (contracts/deployment-config.md §4; research.md D9).
- [X] T003 In `deploy/roles/app/tasks/main.yml`, add a **registry-guarded** build & push of the custom Caddy image to the on-host registry as `localhost:5000/twig-caddy:2-cloudflare`, mirroring the existing twig-server build/push idempotence (check the registry `tags/list` first; build with `podman build -f deploy/roles/app/files/Containerfile.caddy`; push over the existing SSH tunnel; skip when the tag is already present so re-runs are `changed=0`) (research.md D9; plan.md Testing).
- [X] T004 [P] In `deploy/roles/base/tasks/main.yml`, update the firewall task's comment to reflect that opening `http`/`https` now grants **LAN** access only — inbound public reachability is no longer required for issuance (DNS-01 needs only outbound to Let's Encrypt + Cloudflare) (research.md D2; contracts/deployment-config.md §6). The firewalld task itself is unchanged.

**Checkpoint**: A DNS-01-capable Caddy image is in the on-host registry; host posture documented as LAN-only.

---

## Phase 3: User Story 1 - End user reaches the app securely (Priority: P1) 🎯 MVP

**Goal**: An end user on the LAN visiting `https://{{ twig_domain }}` gets an encrypted, publicly-trusted (Let's Encrypt) connection over TLS 1.3 — even though the host has a private IP — with plain HTTP redirected to HTTPS and existing ConnectRPC/auth/SPA routing preserved. Trust is achieved by obtaining the cert via DNS-01.

**Independent Test**: Deploy to a LAN host with the token configured; with `HOST` = the host's LAN IP, `curl -sSI --resolve "$DOMAIN:443:$HOST" https://$DOMAIN/` succeeds with no `-k` (trusted), `openssl x509` shows a Let's Encrypt issuer with `$DOMAIN` in the SAN, the HTTP form returns `308` to `https://`, and `openssl -tls1_3` succeeds while `-tls1_2` fails (contracts/tls-behavior.md TB-1…TB-4). **Note:** requires US2's token plumbing to actually issue (see Implementation Strategy).

- [X] T005 [US1] In `deploy/roles/app/templates/Caddyfile.j2`, add `acme_dns cloudflare {env.CLOUDFLARE_API_TOKEN}` to the **global options block** (always emitted, not conditional) so every certificate uses the DNS-01 challenge. Render the literal `{env.CLOUDFLARE_API_TOKEN}` placeholder verbatim — the token value MUST NOT appear in the file (contracts/deployment-config.md §2; FR-014, FR-015, FR-016/D1).
- [X] T006 [US1] In `deploy/roles/app/templates/Caddyfile.j2`, add `resolvers {{ twig_acme_dns_resolver }}` inside the site `tls` block (alongside the existing `protocols tls1.3`) so DNS-01 propagation is checked against a public resolver rather than the LAN's split-horizon DNS (contracts/deployment-config.md §2; research.md D11).
- [X] T007 [US1] In `deploy/roles/app/templates/compose.yaml.j2`, change the `caddy` service `image` from `docker.io/library/caddy:2-alpine` to `localhost:5000/twig-caddy:2-cloudflare` (the DNS-01-capable build, D9) **and** add `env_file: ["{{ twig_host_data_dir }}/app/caddy.env"]` so the Cloudflare token reaches the process as `CLOUDFLARE_API_TOKEN` without appearing in compose (D10). Keep ports `80`/`443`, the `/data` mount, `depends_on: twig_server`, and `restart: unless-stopped` (contracts/deployment-config.md §3; FR-016).
- [X] T008 [US1] In `deploy/roles/app/tasks/main.yml`, replace the `uri`-based HTTPS smoke check with an `ansible.builtin.command` running `curl -fsS --resolve {{ twig_domain }}:{{ twig_https_port }}:{{ ansible_host }} https://{{ twig_domain }}/health.v1.HealthService/Check -X POST -d '{}'` (add `-k` when `twig_acme_environment == 'staging'`), so the check does not depend on the control node's DNS view. Widen the retry/delay window to bound first DNS-01 issuance to ~10 minutes; a failed first issuance must fail the play (FR-010/D8, SC-003; contracts/tls-behavior.md TB-5).

**Checkpoint**: With the token in place (US2), HTTPS serving, redirect, and the TLS-1.3 floor work for a configured domain on a private IP — the MVP is testable via TB-1…TB-4.

---

## Phase 4: User Story 2 - Operator configures the domain and credentials for a deployment (Priority: P1)

**Goal**: An operator sets `twig_domain` (existing) **and** supplies a `twig_cloudflare_api_token` secret to choose the deployment's domain and authorize DNS-01; a missing/invalid value for either fails the play fast (token handled with `no_log`); the credential never lands in cleartext files or logs.

**Independent Test**: Run the play with `twig_cloudflare_api_token` unset → it fails immediately naming the variable, without printing any token value. After a deploy, grep `compose.yaml`/`Caddyfile` for the token → absent; `caddy.env` is `0600` (contracts/deployment-config.md C4; contracts/tls-behavior.md TB-8).

- [X] T009 [US2] At the **start** of `deploy/roles/app/tasks/main.yml` (near the existing `twig_domain` assert, before any template render or restart), add an `ansible.builtin.assert` that fails when `twig_cloudflare_api_token` is undefined or empty after trim, with `no_log: true` and a `fail_msg` naming the variable and recommending a vaulted `host_vars` entry (FR-017/D7; contracts/deployment-config.md C4).
- [X] T010 [US2] In `deploy/roles/app/tasks/main.yml`, render `{{ twig_host_data_dir }}/app/caddy.env` containing exactly `CLOUDFLARE_API_TOKEN={{ twig_cloudflare_api_token }}` with `owner: root`, `mode: "0600"`, and `no_log: true`, **before** the stack (re)start; `register` it and include its change in the existing "(re)start when changed" condition so a rotated token redeploys Caddy (FR-016/D10; contracts/deployment-config.md §5). The token MUST NOT appear in any other rendered file.
- [X] T011 [P] [US2] Rewrite the per-host comments in `deploy/inventories/lan.ini` and `deploy/inventories/test.ini`: remove the public-IP / DNS-only-grey-cloud / ports-80-443-public preconditions; state that a **private LAN IP is fine**, that `twig_domain` and a vaulted `twig_cloudflare_api_token` are required, and that LAN clients must resolve the domain to the host. Point to `specs/025-https-custom-domain/quickstart.md` (FR-004, FR-005, FR-014, FR-016).
- [X] T012 [P] [US2] Update `deploy/README.md` "Domain configuration" / HTTPS prerequisites to the DNS-01 model: Cloudflare API token (scope Zone → DNS → Edit, via Ansible Vault), domain hosted in Cloudflare, LAN name resolution, **no public reachability required**; point operators at `quickstart.md` (FR-014, FR-015, FR-016).

**Checkpoint**: Domain + credential are validated, per-deployment config points; the secret is never exposed — US1 + US2 together yield a trusted cert on a LAN host.

---

## Phase 5: User Story 3 - Certificate stays valid over time without manual work (Priority: P2)

**Goal**: Issued certificates persist across container/host restarts and redeploys (no re-issuance, avoiding Let's Encrypt rate limits — `/data` mount already in place), and DNS-01 issuance/renewal failures are observable to the operator.

**Independent Test**: Deploy, confirm a cert is issued, then restart the stack / re-run the play and confirm the **same** cert is served (no new ACME request in `podman logs caddy`); `{{ twig_host_data_dir }}/caddy` retains cert material across restart (research.md D4, FR-007).

- [X] T013 [US3] Update the renewal + failure-observability operator workflow in `deploy/README.md` to reflect **DNS-01** renewal: Caddy renews automatically at ~2/3 lifetime via the Cloudflare API; check `podman logs caddy` / `journalctl -u twig.service`; certs persist under `/var/lib/twig/caddy`. Cross-reference `quickstart.md` "Renewal" and "Troubleshooting" (FR-007, FR-010/D8). (The `/data` mount itself is already implemented and is retained, not re-added.)

**Checkpoint**: Certs persist and DNS-01 renewal/observability are documented — all three stories hold.

---

## Phase 6: Polish & Cross-Cutting Concerns (Verification)

**Purpose**: Prove idempotence and the externally-observable TLS + secret-handling contract; finish docs.

- [ ] T014 Idempotence check: run `ansible-playbook -i inventories/<inv>.ini site.yml` a **second** time and confirm `changed=0` for the custom-image build/push (tag already present, T003), the `caddy.env` render (T010), and the Caddyfile/compose render — and that the postgres container is **not** recreated (existing guards still pass) (plan.md Testing).
- [ ] T015 [P] Production-mode external TLS verification against the live `$DOMAIN` via the host's LAN IP per `contracts/tls-behavior.md`: TB-1 (trusted HTTPS on a private IP + Let's Encrypt issuer + SAN), TB-2 (SAN matches), TB-3 (`308` HTTP→HTTPS), TB-4 (TLS 1.3 succeeds, TLS 1.2 refused), TB-6 (no HSTS). Use the `curl --resolve`/`openssl -connect $HOST` commands from `quickstart.md` "Verify".
- [ ] T016 [P] Secret-handling verification (TB-8): confirm the token value is absent from `/var/lib/twig/app/compose.yaml` and `/var/lib/twig/app/Caddyfile` (only the `{env.CLOUDFLARE_API_TOKEN}` placeholder appears), `caddy.env` is `0600`, and the token-render/assert tasks ran with `no_log` (no token in Ansible output) (FR-016).
- [ ] T017 [P] Staging-mode dry-run verification (TB-7): set `twig_acme_environment: staging`, deploy, confirm DNS-01 issuance with `curl -k --resolve ...` and a staging issuer marker, then flip back to `production` and redeploy for the trusted cert (FR-013; quickstart.md "Dry-run against Let's Encrypt staging").
- [X] T018 [P] Final docs pass: ensure `deploy/README.md`, `quickstart.md`, and any deploy notes in `CLAUDE.md` reflect the DNS-01 / LAN / Cloudflare-token model (no stale public-reachability or HTTP-01 language), and confirm `quickstart.md` commands match the rendered templates.

---

## Dependencies & Execution Order

### Phase Dependencies

- **Setup (Phase 1)**: No dependencies — start immediately (T001).
- **Foundational (Phase 2)**: Depends on Setup. T002 (Containerfile) → T003 (build/push needs the file). T004 is independent `[P]`. Blocks all user stories (no DNS-01 image ⇒ no issuance).
- **User Stories (Phase 3–5)**: All depend on Foundational completion.
  - **US1 (P1)** and **US2 (P1)** are mutually dependent for a *working* result: US1 wires the mechanism (`acme_dns`, custom image, `env_file`), US2 supplies/asserts the credential and renders `caddy.env`. Land them together for the MVP.
  - **US3 (P2)** is mostly documentation (persistence already implemented); independent.
- **Polish (Phase 6)**: Depends on the user stories being deployed (T014–T018 verify a running deployment).

### Within / across stories (shared files)

- `app/tasks/main.yml` is edited by T003, T008, T009, T010 — apply these in a coordinated pass; order: asserts (T009) and `caddy.env` (T010) must precede the stack (re)start, and the smoke check (T008) is the last step. T003's build/push sits with the other build steps.
- `Caddyfile.j2`: T005 → T006 (same file, sequential).
- `compose.yaml.j2`: T007 only.
- `README.md`: T012 (US2), T013 (US3), T018 (Polish) — sequence or batch.

### Parallel Opportunities

- T004 `[P]` (base firewall comment) is independent of the app-role work.
- US2: T011 `[P]` (inventories) and T012 `[P]` (README) are docs, parallel to the `app/tasks` edits T009–T010.
- Polish: T015, T016, T017, T018 `[P]` are independent verification/docs streams. **Run T017 (staging) before T015 (production)** to avoid burning production rate limits while iterating.

---

## Parallel Example: User Story 2

```bash
# Within US2, docs can proceed concurrently with the app/tasks edits (different files):
Task: "T011 Rewrite LAN/DNS-01/token comments in deploy/inventories/{lan,test}.ini"
Task: "T012 Update DNS-01 prerequisites in deploy/README.md"

# Meanwhile the app/tasks/main.yml edits run as a coordinated sequential stream:
#   T009 (token assert, no_log) → T010 (render caddy.env, 0600, no_log)
```

---

## Implementation Strategy

### MVP (User Story 1 + User Story 2 together)

For this feature the true MVP is **Foundational + US1 + US2**, because a publicly-trusted cert on a LAN host needs both the DNS-01 mechanism (US1) and the Cloudflare credential (US2):

1. Phase 1 Setup (T001) → Phase 2 Foundational (T002–T004): DNS-01-capable image in the registry.
2. US1 (T005–T008) + US2 (T009–T012): `acme_dns` directive, resolver, custom image + `env_file`, token assert + `caddy.env`, smoke check, docs.
3. **STOP and VALIDATE**: deploy to a LAN test host (use `staging` to iterate, T017) and run TB-1…TB-4 + TB-8. This is a shippable, secure, private-IP deployment.

### Incremental Delivery

1. Setup + Foundational → DNS-01 image ready.
2. US1 + US2 → trusted HTTPS at the domain on a private IP (MVP) → validate TB-1…TB-4, TB-8.
3. US3 → DNS-01 renewal/observability docs (persistence already in place) → validate restart keeps the cert.
4. Polish → idempotence (T014), full external TLS contract (T015), secret handling (T016), staging rehearsal (T017), docs (T018).

### Notes

- `[P]` = different files, no incomplete dependency; honor the shared-file constraints above.
- Every task touches only `deploy/` — no `services/twig` or `services/twig-web` changes (plan.md Project Structure).
- Iterate against `twig_acme_environment: staging` to avoid Let's Encrypt production rate limits; the persistent `/data` mount (already present) further protects against re-issuance across restarts.
- Never echo `twig_cloudflare_api_token`; keep `no_log: true` on the assert (T009) and render (T010), and verify TB-8 before claiming done.
- Commit after each logical group; re-run `site.yml` to confirm `changed=0` before claiming done.
