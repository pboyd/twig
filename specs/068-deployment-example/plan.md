# Implementation Plan: Real-World Deployment Example

**Branch**: `068-deployment-example` | **Date**: 2026-07-25 | **Spec**: [spec.md](./spec.md)

**Input**: Feature specification from `/specs/068-deployment-example/spec.md`

## Summary

Ship `deploy/` — a self-contained Ansible example that turns a bare Fedora host reachable over SSH into
a working Twig instance: podman-compose running PostgreSQL, the published `twig-server` image, and
Caddy as the reverse proxy, with persistent data on the host and two opt-in layers (a Let's Encrypt
certificate, a Cloudflare Tunnel). Alongside it, the documentation that makes copying and owning it the
expected workflow.

## Technical Context

**Language/Version**: YAML — Ansible (`ansible-core` ≥ 2.15), Jinja2 templates. No Go, TypeScript, or
SQL changes.

**Primary Dependencies**: `containers.podman` and `ansible.posix` collections (control node);
`podman` + `podman-compose` + `python3-pip` (target host, installed by the play).

**Storage**: PostgreSQL 17 in a container, bind-mounted to `/var/lib/twig/postgres` on the host. Caddy
certificate state bind-mounted to `/var/lib/twig/caddy`.

**Testing**: `ansible-playbook --syntax-check` and `ansible-lint` in CI; behavior verified manually
against a throwaway Fedora VM using the 14-item checklist in [quickstart.md](./quickstart.md). No
automated VM provisioning — out of scope per the spec.

**Target Platform**: current Fedora Server on x86-64, SELinux enforcing, `dnf`, systemd. Control node:
Linux or macOS with Ansible.

**Project Type**: infrastructure-as-code example plus documentation. Not an application surface.

**Performance Goals**: first deploy under 30 minutes wall-clock, under 10 hands-on (SC-001). Redeploy
with no config change: zero changed tasks, zero restarts (SC-003).

**Constraints**: no more than 5 operator-supplied values before the first successful run (SC-002); no
real secrets in the repository (SC-006); every validation failure precedes any host mutation (SC-005);
the task database is never destroyed by any code path.

**Scale/Scope**: one host, one Twig instance. Roughly 12 files in `deploy/`, two roles, one README.

## Constitution Check

*GATE: Must pass before Phase 0 research. Re-check after Phase 1 design.*

| Principle | Status | Notes |
|-----------|--------|-------|
| I. Simplicity / YAGNI | ✅ | The design is defined by what it drops from the reference: no private registry, no local image builds, no custom Caddy image, no DNS-01, no second application. Two roles (`base`, `twig`), two toggles, four required values. The known gap (no origin certificate behind NAT) is documented rather than engineered around, because the tunnel already provides trusted TLS at the edge. |
| II. API-First Design | ✅ | The example exposes no runtime API, so the contract is its configuration surface. `contracts/config-schema.md` and `contracts/host-layout.md` are committed before any implementation task, and both are declared stable surfaces whose changes are breaking for operators who copied the example. |
| III. UI/UX Consistency | ✅ | No TUI, CLI, or web surface is added or changed. Consistency obligations here are documentation-shaped: `deploy/README.md` follows the existing README/docs voice and reuses the repository's table-and-code-block conventions, and the operator commands it documents match those already in the root README (`--provision-user`, `TWIG_ADDR`, `TWIG_API_KEY`). |
| IV. Playful User Messages | ✅ | Covered by FR-032. The play's `assert` failure messages and its closing "here's what to do next" output are written in the project's warm-but-clear voice while staying strictly actionable — a failed validation must still name the variable and the file. Templates and README prose follow the same register; playfulness never displaces a required detail. |

Re-checked after Phase 1 design: no change. The design added no abstraction beyond the two roles, and
the Complexity Tracking table stays empty.

## Project Structure

### Documentation (this feature)

```text
specs/068-deployment-example/
├── plan.md              # This file
├── spec.md              # Feature specification
├── research.md          # Phase 0 — 10 decisions, divergences from the reference
├── data-model.md        # Phase 1 — configuration/host/state entities and their rules
├── quickstart.md        # Phase 1 — operator walkthrough + manual verification checklist
├── contracts/
│   ├── config-schema.md # Every variable, default, validation rule, TLS/access matrix
│   └── host-layout.md   # What lands on the host and the operator commands it supports
├── checklists/
│   └── requirements.md  # Spec quality checklist (passing)
└── tasks.md             # Phase 2 output (/speckit-tasks — NOT created here)
```

### Source Code (repository root)

```text
deploy/                                   # NEW — the copyable example, self-contained
├── README.md                             # first run, config reference, day two, copy-and-own
├── ansible.cfg                           # inventory = inventory.ini, no host key checking
├── Makefile                              # deps / check / deploy
├── requirements.yml                      # containers.podman, ansible.posix
├── site.yml                              # target assertion + base and twig roles
├── inventory.example.ini                 # copied to inventory.ini (gitignored)
├── group_vars/
│   ├── all/
│   │   └── defaults.yml                  # the example's defaults (not operator-owned)
│   └── twig_servers/
│       ├── main.yml                      # operator knobs — domain, tag, toggles
│       └── vault.example.yml             # placeholder secrets; copied + ansible-vault encrypted
├── roles/
│   ├── base/tasks/main.yml               # packages, host dirs, postgres uid 70:70, firewalld
│   └── twig/
│       ├── tasks/
│       │   ├── main.yml                  # render, systemd, bring up, smoke test, guards
│       │   ├── validate.yml              # all fail-fast assertions (runs first)
│       │   └── data_guards.yml           # PG_VERSION + container-recreation checks
│       └── templates/
│           ├── compose.yaml.j2           # postgres, twig_server, caddy, optional cloudflared
│           ├── Caddyfile.j2              # tls_mode branches + optional tunnel-hop site
│           └── twig.service.j2           # podman-compose oneshot unit
└── .gitignore                            # inventory.ini, vault.yml, *.retry

.github/workflows/ci.yml                  # MODIFIED — add an ansible lint/syntax job
README.md                                 # MODIFIED — link from the quick start to deploy/
```

**Structure Decision**: everything lives under `deploy/` so that `cp -r deploy/ ~/my-infra/` yields a
working, runnable tree with no path fixing — the copy-and-own requirement (FR-001, FR-027) drives the
layout. `ansible.cfg` and the `Makefile` sit inside the directory for the same reason. Only two files
outside `deploy/` change, both additive: a CI job and a README pointer.

The role split follows the reference: `base` prepares the host (packages, directories, ownership,
firewall) and `twig` owns everything about the application stack. Validation and the data-safety guards
are separate task files inside the `twig` role so their ordering is obvious at a glance —
`validate.yml` before anything mutating, `data_guards.yml` before the success marker is written.

## Phase 0 — Research summary

Full detail in [research.md](./research.md). The decisions that shaped the design:

- **R1** — deploy the published `ghcr.io/pboyd/twig-server` image; it bundles the SPA, needs only
  `DATABASE_URL`, runs migrations itself, listens on a hardcoded `:8080`, and is distroless (so: no
  container healthcheck, and `podman exec twig_server /server --provision-user …` for accounts). The
  repo has **no version tags yet**, so `twig_image_tag` defaults to `latest` with a note to pin.
- **R2** — password-based privilege escalation via `-K`; no passwordless sudo requirement.
- **R3** — `twig_tls_mode: internal | letsencrypt` on stock Caddy, tunnel as an orthogonal toggle. The
  `letsencrypt`-behind-NAT gap is documented, not engineered around. Carries over the `http://<domain>`
  scheme fix that prevents a redirect loop through the tunnel.
- **R4** — bind mounts (not named volumes) so backups work with ordinary tools; `70:70` `0700` for the
  postgres directory; `:z` for SELinux.
- **R5** — one `podman-compose` project driven by a `twig.service` oneshot unit for reboot survival.
- **R6** — gate `pull`/`up -d` on the rendered artifacts having changed, which is what makes a second
  run a no-op. `--check` is for redeploys; that caveat gets documented.
- **R7** — the smoke test asserts **HTTP 200** on the unauthenticated `GET /`, from the control node,
  with `curl --resolve` so it does not depend on the operator's DNS. The reference's exit-code-only
  check would have passed on a 401.
- **R8** — first-user provisioning stays a documented manual step; the API key is printed once.
- **R9** — `deploy/` at the repo root, copied wholesale.
- **R10** — syntax-check and lint in CI; no VM provisioning.

No `NEEDS CLARIFICATION` items remain.

## Phase 1 — Design summary

- **[data-model.md](./data-model.md)** — five entities (target host, deployment configuration, secret
  store, service stack, persistent data) plus the release identifier, with the validation rules and the
  state transitions for first deploy, no-op redeploy, upgrade, toggle change, reboot, and the failure
  path when the data directory has moved.
- **[contracts/config-schema.md](./contracts/config-schema.md)** — the variable surface: 4 required
  values, ~15 optional, every validation rule, the four-way TLS/tunnel access matrix with its
  prerequisites and its one honest limitation, and the values that are deliberately not configurable.
- **[contracts/host-layout.md](./contracts/host-layout.md)** — the host contract: paths, permissions,
  container names, the systemd unit, the operator commands those paths support (including the backup
  and restore procedures behind SC-010), and an explicit list of what the deployment never touches.
- **[quickstart.md](./quickstart.md)** — the operator walkthrough that `deploy/README.md` is built
  from, ending in the 14-item manual verification checklist that maps each check to a user story or
  requirement.

## Implementation notes for Phase 2

Sequencing that `/speckit-tasks` should respect:

1. **Contracts are already committed** — implementation may proceed (Constitution Principle II,
   Quality Gate 1).
2. **Scaffold before roles**: `deploy/` skeleton, `ansible.cfg`, `requirements.yml`, `site.yml`,
   `Makefile`, `.gitignore`, and the example config files come first; every later task edits inside
   that tree.
3. **`validate.yml` before any mutating task exists**, so no intermediate commit can touch a host
   without the fail-fast gate in place.
4. **US1 (P1) is the vertical slice**: `base` role → templates → systemd → bring-up → smoke test. It
   must be deployable and verifiable on a VM before the optional layers are written.
5. **Templates are conditional, not duplicated**: `twig_tls_mode` and `twig_tunnel_enabled` branch
   inside `compose.yaml.j2` and `Caddyfile.j2`. Do not fork the templates per mode.
6. **Data guards land with the upgrade story (US3)**, before the example is documented as safe to
   re-run.
7. **Documentation is a task, not a footnote**: `deploy/README.md` is the primary deliverable for
   US2 and carries the copy-and-own, support-posture, and day-two content (FR-026 to FR-030).
8. **Last**: the root README pointer (FR-031) and the CI job (R10).

Verification of anything touching a host is manual, per the quickstart checklist. Tasks should say so
explicitly rather than implying an automated test exists.

## Complexity Tracking

> No Constitution Check violations. Table intentionally empty.
