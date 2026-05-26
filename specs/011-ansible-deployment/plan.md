# Implementation Plan: Ansible Deployment to Dedicated VM

**Branch**: `011-ansible-deployment` | **Date**: 2026-05-25 | **Spec**: [spec.md](./spec.md)

**Input**: Feature specification from `/specs/011-ansible-deployment/spec.md`

## Summary

Provide a one-command Ansible deployment that takes a fresh Fedora Server VM (passwordless SSH + sudo already set up) to a fully running todo stack: PostgreSQL with persistent on-host data, the todo server, a reverse proxy in front of the server, and a private container image registry running on the VM itself. The maintainer's workstation builds the application image(s) locally, pushes them to the on-host registry, then renders a production compose file on the VM and starts the stack with podman-compose. A systemd unit ensures the stack auto-starts on VM boot. The test target is `user@192.0.2.20`.

## Technical Context

**Language/Version**: Ansible (playbooks/roles, YAML; Ansible ≥ 2.15 on workstation). Application code (Go) is unchanged by this feature.

**Primary Dependencies**:

- Workstation: Ansible, podman (for `podman build` and `podman push`), an SSH key trusted by the VM
- VM (installed by playbook): `podman`, `podman-compose`, `python3` (for Ansible modules)
- Container images pulled from the public internet during initial setup: `postgres:17-alpine`, `registry:2`, `caddy:2-alpine` (reverse proxy)
- Application images: built locally on workstation, pushed to on-host registry

**Storage**: PostgreSQL data and registry blob storage both live on the VM host filesystem under `/var/lib/todo/` (subdirs `postgres/` and `registry/`). Bind-mounted into containers.

**Testing**: Manual acceptance against `user@192.0.2.20` per the quickstart. No automated test infrastructure beyond `ansible-playbook --check` and an end-to-end script that runs the quickstart steps and asserts `todo task list` succeeds.

**Target Platform**: Fedora Server (current stable) on a single VM. Single-host deployment; not clustered.

**Project Type**: Operational/infrastructure feature. Adds `deploy/` directory at repo root; does not change `services/todo/` source.

**Performance Goals**: N/A — single-user personal deployment. Acceptance is functional ("CLI can drive it"), not throughput-based.

**Constraints**:

- Must remain idempotent (FR-009): every task converges; PostgreSQL container must not be recreated on a no-op redeploy.
- Must not require any pre-staged state on the VM beyond what the spec states (provisioned, SSH, sudo).
- Must support `--check` mode for dry runs against the test VM.

**Scale/Scope**: One VM. One application image (the server). One database. One registry. No fanout.

## Constitution Check

*GATE: Must pass before Phase 0 research. Re-check after Phase 1 design.*

| Principle | Status | Notes |
|---|---|---|
| I. Simplicity / YAGNI | PASS | Single playbook with a small set of roles (`base`, `registry`, `app`). No abstraction over Ansible itself; no custom modules; no Ansible Vault unless a real secret appears (none in this feature). Reverse proxy is `caddy` chosen specifically because its config is one file. |
| II. API-First Design | N/A | This feature exposes no new application API. The CLI ↔ server contract is unchanged; the deploy only repackages the existing services. No new entries under `contracts/` are needed. Documenting this explicitly so the gate is consciously evaluated, not silently skipped. |

Both gates pass. No entries required in Complexity Tracking.

## Project Structure

### Documentation (this feature)

```text
specs/011-ansible-deployment/
├── plan.md              # This file
├── spec.md              # Feature spec (already exists)
├── research.md          # Phase 0 output
├── data-model.md        # Phase 1 output (host filesystem layout, not application data)
├── quickstart.md        # Phase 1 output (test against 192.0.2.20)
├── contracts/           # Not used for this feature (no new API surface)
└── tasks.md             # Created by /speckit-tasks
```

### Source Code (repository root)

```text
deploy/
├── ansible.cfg                 # inventory path, ssh defaults, retry behavior
├── inventories/
│   └── test.ini                # holds user@192.0.2.20
├── site.yml                    # entry point: imports the three roles in order
├── group_vars/
│   └── all.yml                 # defaults: ports, paths, image tags
└── roles/
    ├── base/                   # podman + podman-compose install, /var/lib/todo dirs
    │   └── tasks/main.yml
    ├── registry/               # registry:2 container + persistent storage + systemd unit
    │   ├── tasks/main.yml
    │   └── templates/registry-compose.yaml.j2
    └── app/                    # build+push locally, render compose, bring up stack
        ├── tasks/main.yml
        └── templates/
            ├── compose.yaml.j2
            └── Caddyfile.j2

services/todo/                  # UNCHANGED — Go source for server + CLI
compose.yaml                    # UNCHANGED — dev compose
Makefile                        # +1 target: `make deploy` wraps ansible-playbook
```

**Structure Decision**: New top-level `deploy/` directory. Keeps infrastructure code out of the Go module and preserves the existing dev workflow untouched. Three Ansible roles, in order: `base` (prereqs + host directories) → `registry` (independent service so the app role can push to it) → `app` (build, push, render compose, bring up stack). One inventory file for now (`test.ini`); production inventory is just another file with the same shape when a production VM exists.

## Complexity Tracking

> No Constitution Check violations to justify.

Not applicable for this feature.
