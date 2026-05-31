---
description: "Task list for HTTPS with Custom Domain"
---

# Tasks: HTTPS with Custom Domain

**Input**: Design documents from `/specs/025-https-custom-domain/`

**Prerequisites**: plan.md, spec.md, research.md, data-model.md, contracts/deployment-config.md, contracts/tls-behavior.md, quickstart.md

**Tests**: This is a deployment/infrastructure feature with no application-code or unit-test surface. Per plan.md (Testing), verification is **Ansible idempotence** plus **external TLS checks** (`curl`/`openssl s_client`) against the live domain, mapped to `contracts/tls-behavior.md` (TB-1…TB-7). These appear as explicit verification tasks in the Polish phase, not as a TDD test-first phase.

**Organization**: Tasks are grouped by user story to enable independent implementation and testing of each story.

## Format: `[ID] [P?] [Story] Description`

- **[P]**: Can run in parallel (different files, no dependencies)
- **[Story]**: Which user story this task belongs to (US1, US2, US3)
- All paths are repo-relative; this feature touches **only** `deploy/` (no `services/` changes).

## ⚠️ Shared-file note (read before parallelizing)

Several stories edit the **same two files**, so tasks in different stories are frequently **not** parallelizable:

- `deploy/roles/app/templates/compose.yaml.j2` — edited by **US1** (publish 443, T007) and **US3** (`/data` mount, T011).
- `deploy/roles/app/tasks/main.yml` — edited by **US1** (HTTPS smoke check, T008) and **US2** (fail-fast assert, T009).
- `deploy/roles/app/templates/Caddyfile.j2` — edited only within **US1** (T004→T005→T006), sequentially.

Treat `[P]` markers as "different file, no incomplete dependency" only; respect the shared-file constraints in Dependencies below.

---

## Phase 1: Setup (Shared Configuration)

**Purpose**: Declare the new per-deployment configuration variables consumed by every story.

- [X] T001 Add new deployment variables to `deploy/group_vars/all.yml`: `twig_acme_environment` (default `production`), `twig_acme_email` (default `""`), and `twig_https_port` (default `443`). **Do NOT** add `twig_domain` here — it intentionally has no default and must be set per host/group so an unset value fails the play (FR-009, data-model.md). `twig_http_port` already exists and is reused.

**Checkpoint**: Config surface defined per `contracts/deployment-config.md` §1.

---

## Phase 2: Foundational (Blocking Prerequisites)

**Purpose**: Host-level prerequisites that must exist before the Caddy container starts issuing/serving — shared by all stories.

**⚠️ CRITICAL**: No user story work is verifiable until this phase is complete.

- [X] T002 In `deploy/roles/base/tasks/main.yml`, create the Caddy state directory `{{ twig_host_data_dir }}/caddy` (add `"{{ twig_host_data_dir }}/caddy"` to the existing `/var/lib/twig` directory loop) so certificate/ACME state (the `/data` mount target, D4) has a home before the stack starts (contracts/deployment-config.md §4).
- [X] T003 [P] In `deploy/roles/base/tasks/main.yml`, open the host firewall for the `http` and `https` services (TCP 80/443), guarded to run only when `firewalld` is active (e.g. check the service is running first). Ports 80 (HTTP-01 + redirect) and 443 (TLS serving) must be publicly reachable (research.md D2, contracts/deployment-config.md §4).

**Checkpoint**: Host has the cert directory and open ports — Caddy can issue and serve.

---

## Phase 3: User Story 1 - End user reaches the app securely (Priority: P1) 🎯 MVP

**Goal**: An end user visiting `https://{{ twig_domain }}` gets an encrypted, publicly-trusted (Let's Encrypt) connection over TLS 1.3, plain HTTP is redirected to HTTPS, and existing ConnectRPC/auth/SPA routing is preserved.

**Independent Test**: Deploy to a resolving domain; `curl -sSI https://$DOMAIN/` succeeds with no `-k` (trusted), `openssl x509` shows a Let's Encrypt issuer with `$DOMAIN` in the SAN, `curl -sSI http://$DOMAIN/` returns `308` to `https://`, and `openssl s_client -tls1_3` succeeds while `-tls1_2` fails (contracts/tls-behavior.md TB-1…TB-4).

- [X] T004 [US1] In `deploy/roles/app/templates/Caddyfile.j2`, replace the bare `:{{ twig_http_port }}` site address with the named-host site address `{{ twig_domain }}` (this activates Caddy automatic HTTPS, ACME issuance, and the HTTP→HTTPS redirect). Preserve the `@backend path_regexp` block and the SPA `handle` block **verbatim** — no routing regression (contracts/deployment-config.md §2; FR-001, FR-006, FR-008).
- [X] T005 [US1] In `deploy/roles/app/templates/Caddyfile.j2`, add the Caddy **global options block** at the top that conditionally emits `email {{ twig_acme_email }}` when `twig_acme_email` is non-empty and `acme_ca https://acme-staging-v02.api.letsencrypt.org/directory` when `twig_acme_environment == 'staging'`; an empty `{ }` block is valid (production + no email). Matches the Jinja2 shape in contracts/deployment-config.md §2 (FR-013/D3, D6).
- [X] T006 [US1] In `deploy/roles/app/templates/Caddyfile.j2`, add `tls { protocols tls1.3 }` inside the site block to set the TLS 1.3 floor (rejects TLS ≤1.2 without disabling automatic ACME), and delete the now-stale "To add TLS in the future…" comment block (FR-012/D5).
- [X] T007 [P] [US1] In `deploy/roles/app/templates/compose.yaml.j2`, publish the HTTPS port on the `caddy` service: add `"{{ twig_https_port }}:443"` alongside the existing `"{{ twig_http_port }}:80"` (research.md D2; contracts/deployment-config.md §3).
- [X] T008 [US1] In `deploy/roles/app/tasks/main.yml`, upgrade the post-deploy "Wait for server to respond" smoke check to probe `https://{{ twig_domain }}/health.v1.HealthService/Check` (accept `[200, 401]`, `validate_certs` per environment) with a retry/delay window that bounds first-issuance to ≤5 minutes, so a failed first issuance fails the play instead of passing silently (FR-010/D8, SC-003).

**Checkpoint**: HTTPS serving, redirect, and TLS-1.3 floor work for a configured domain — the MVP is independently testable via TB-1…TB-4.

---

## Phase 4: User Story 2 - Operator configures the domain for a deployment (Priority: P1)

**Goal**: An operator sets a single value (`twig_domain`) to choose the deployment's domain; a missing/invalid value fails the play fast with an actionable message; two deployments differ only by that value.

**Independent Test**: Run the play with `twig_domain` unset → it fails immediately naming the variable. Set two different domains on two deployments → each serves and obtains a cert for its own domain with no code changes (contracts/deployment-config.md C2/C4).

- [X] T009 [US2] At the **start** of `deploy/roles/app/tasks/main.yml` (before any template render or restart), add an `ansible.builtin.assert` that fails when `twig_domain` is undefined, empty after trim, or does not match the hostname regex from research.md D7 (`^(?!-)[A-Za-z0-9-]{1,63}(?<!-)(\.(?!-)[A-Za-z0-9-]{1,63}(?<!-))*$`; no scheme/path/spaces). The `fail_msg` must name `twig_domain` and say where to set it (FR-009/D7, contracts/deployment-config.md C4).
- [X] T010 [P] [US2] Document per-deployment domain configuration: add a commented `twig_domain` example (and the public-IP / DNS-only-Cloudflare precondition) to the inventory files `deploy/inventories/lan.ini` and `deploy/inventories/test.ini` (or accompanying `host_vars`), and update `deploy/README.md` to point operators at `specs/025-https-custom-domain/quickstart.md` (FR-004, FR-005, US2 acceptance).

**Checkpoint**: Domain is a single, validated, per-deployment config point — US1 + US2 both hold.

---

## Phase 5: User Story 3 - Certificate stays valid over time without manual work (Priority: P2)

**Goal**: Issued certificates persist across container/host restarts and redeploys (no re-issuance, avoiding Let's Encrypt rate limits), and issuance/renewal failures are observable to the operator.

**Independent Test**: Deploy, confirm a cert is issued, then restart the stack / re-run the play and confirm the **same** cert is served (no new ACME request in `podman logs caddy`); confirm `{{ twig_host_data_dir }}/caddy` retains cert material across restart (research.md D4, FR-007).

- [X] T011 [US3] In `deploy/roles/app/templates/compose.yaml.j2`, add a **read-write** bind mount `"{{ twig_host_data_dir }}/caddy:/data:z"` to the `caddy` service so issued certs, keys, and ACME account state survive restarts/redeploys (research.md D4; contracts/deployment-config.md §3). Keep `depends_on: twig_server` and `restart: unless-stopped`.
- [X] T012 [P] [US3] Document the renewal + failure-observability operator workflow (automatic renewal at ~2/3 lifetime; check `podman logs caddy` / `journalctl -u twig.service`; persisted certs under `/var/lib/twig/caddy`) in `deploy/README.md`, cross-referencing `quickstart.md` "Renewal" and "Troubleshooting" (FR-007, FR-010/D8).

**Checkpoint**: Certs persist and renewal/observability are documented — all three stories hold independently.

---

## Phase 6: Polish & Cross-Cutting Concerns (Verification)

**Purpose**: Prove idempotence and the externally-observable TLS contract; finish docs.

- [ ] T013 Idempotence check: run `ansible-playbook -i inventories/<inv>.ini site.yml` a **second** time and confirm `changed=0` for the Caddyfile/compose render and cert steps, and that the postgres container is **not** recreated (existing guards in `deploy/roles/app/tasks/main.yml` still pass). (plan.md Testing; preserves prior idempotence guarantees.)
- [ ] T014 [P] Production-mode external TLS verification against the live `$DOMAIN` per `contracts/tls-behavior.md`: TB-1 (trusted HTTPS + Let's Encrypt issuer + SAN), TB-2 (SAN matches domain), TB-3 (`308` HTTP→HTTPS), TB-4 (TLS 1.3 succeeds, TLS 1.2 refused), TB-6 (no HSTS header). Use the `curl`/`openssl` commands from `quickstart.md` "Verify".
- [ ] T015 [P] Staging-mode dry-run verification (TB-7): set `twig_acme_environment: staging`, deploy, confirm issuance with `curl -kI` and a staging issuer marker, then flip back to `production` and redeploy for the trusted cert (FR-013; quickstart.md "Dry-run against Let's Encrypt staging").
- [X] T016 [P] Final docs pass: ensure `deploy/README.md` (and any HTTP-only references in `CLAUDE.md` deploy notes) reflect the HTTPS/named-host model; confirm `quickstart.md` commands match the rendered templates.

---

## Dependencies & Execution Order

### Phase Dependencies

- **Setup (Phase 1)**: No dependencies — start immediately (T001).
- **Foundational (Phase 2)**: Depends on Setup. Blocks all user stories (cert dir + open ports must exist before issuance/serving).
- **User Stories (Phase 3–5)**: All depend on Foundational completion.
  - **US1 (P1)** is the MVP and should land first.
  - **US2 (P1)** edits the **same file** as US1's T008 (`app/tasks/main.yml`), so sequence T008 → T009 (or coordinate a single edit session); otherwise US2 is logically independent.
  - **US3 (P2)** edits the **same file** as US1's T007 (`compose.yaml.j2`), so sequence T007 → T011; otherwise US3 is independent.
- **Polish (Phase 6)**: Depends on the user stories being deployed (T013–T016 verify a running deployment).

### Within Each User Story

- **US1**: T004 → T005 → T006 are the **same file** (`Caddyfile.j2`) — sequential. T007 (`compose.yaml.j2`) and T008 (`app/tasks/main.yml`) are different files and may run alongside the Caddyfile edits.
- **US2**: T009 (assert) and T010 (docs/inventory) are different files — T010 is `[P]`.
- **US3**: T011 (compose) and T012 (docs) are different files — T012 is `[P]`.

### Parallel Opportunities

- T003 `[P]` within Foundational (base role firewall) is independent of T002's dir creation only if edited as separate tasks — both touch `base/tasks/main.yml`, so in practice apply them in one pass.
- US1: T007 `[P]` (compose 443) runs parallel to the Caddyfile sequence T004–T006.
- US2: T010 `[P]` (docs) parallel to T009 (assert).
- US3: T012 `[P]` (docs) parallel to T011 (compose mount).
- Polish: T014, T015, T016 `[P]` are independent verification/docs streams (run T015 staging **before** T014 production to avoid burning production rate limits during iteration).

---

## Parallel Example: User Story 1

```bash
# After Foundational (Phase 2), within US1 these can proceed concurrently
# because they touch different files:
Task: "T007 Publish 443 on the caddy service in deploy/roles/app/templates/compose.yaml.j2"
Task: "T008 Upgrade HTTPS smoke check in deploy/roles/app/tasks/main.yml"

# Meanwhile the Caddyfile edits run as a single sequential stream (same file):
#   T004 → T005 → T006 in deploy/roles/app/templates/Caddyfile.j2
```

---

## Implementation Strategy

### MVP First (User Story 1 only)

1. Phase 1 Setup (T001) → Phase 2 Foundational (T002–T003).
2. Phase 3 US1 (T004–T008): named-host Caddyfile, ACME options, TLS 1.3, publish 443, HTTPS smoke check.
3. **STOP and VALIDATE**: deploy to a resolving test domain (use `staging` to iterate) and run TB-1…TB-4. This is a shippable secure deployment for one domain.

### Incremental Delivery

1. Setup + Foundational → host ready.
2. US1 → trusted HTTPS at the domain (MVP) → validate TB-1…TB-4.
3. US2 → fail-fast domain config + per-deployment overridability → validate C2/C4.
4. US3 → cert persistence + renewal/observability docs → validate restart keeps the cert.
5. Polish → idempotence (T013) + full external TLS contract (T014), staging rehearsal (T015), docs (T016).

### Notes

- `[P]` = different files, no incomplete dependency; honor the shared-file constraints above.
- Every task touches only `deploy/` — no `services/twig` or `services/twig-web` changes (plan.md Project Structure).
- Iterate against `twig_acme_environment: staging` to avoid Let's Encrypt production rate limits; the persistent `/data` mount (T011) further protects against re-issuance across restarts.
- Commit after each logical group; re-run `site.yml` to confirm `changed=0` before claiming done.
