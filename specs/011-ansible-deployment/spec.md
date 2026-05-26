# Feature Specification: Ansible Deployment to Dedicated VM

**Feature Branch**: `011-ansible-deployment`

**Created**: 2026-05-25

**Status**: Draft

## Clarifications

### Session 2026-05-25

- Q: How should the deployed server be reachable from the maintainer's workstation? → A: Reverse proxy on the VM in front of the server (no TLS for now, but architected to add it later without re-architecting).
- Q: Should the on-host image registry's storage persist across VM reboots? → A: Yes — persist registry storage on the host filesystem so images survive reboot and auto-restart works without re-running the deploy.
- Q: How should the stack behave if a container crashes while the VM is running? → A: All services restart automatically on crash (`unless-stopped` policy); only stay down if a maintainer explicitly stopped them.

**Input**: User description: "deployment with ansible. The server will run on dedicated virtual machine. Assume the VM has been provisioned, is running Fedora server, is accessible via ssh without a password, the user can sudo to root without a password. The server and postgresql should run in podman-compose, like the development environment. It is very important that the postgresql data is persisted across reboots. We'll need to run a image registry on the host too and push the built images to it. For testing, use `user@192.0.2.20`."

## User Scenarios & Testing *(mandatory)*

### User Story 1 - First-time deployment to a fresh VM (Priority: P1)

A maintainer with shell access to a freshly provisioned Fedora VM runs a single Ansible command from their workstation. The playbook installs the runtime prerequisites on the VM, starts a private container image registry on the VM, builds the application images locally, pushes them to the VM's registry, writes a production compose configuration, and brings up PostgreSQL and the todo server. After it completes, the server is reachable and serving requests, and a provisioned user can log in with the CLI.

**Why this priority**: Without a working initial deployment, no other deployment scenario is possible. This is the foundational user journey.

**Independent Test**: Point the playbook at the test VM (`user@192.0.2.20`), run the deploy command, and verify (a) the server responds to a health probe over HTTP, (b) the CLI can authenticate against it using a provisioned API key, and (c) at least one task can be created and listed.

**Acceptance Scenarios**:

1. **Given** a freshly provisioned Fedora VM with passwordless SSH and passwordless sudo for the deploy user, **When** the maintainer runs the deploy command targeting that VM, **Then** the playbook completes without manual intervention and the server is reachable on the expected port.
2. **Given** a successful deploy, **When** the maintainer provisions a user via the documented mechanism, **Then** the resulting API key authenticates against the deployed server and the CLI can create and list tasks.
3. **Given** the playbook runs end-to-end, **When** the maintainer inspects the VM, **Then** the application images are present in the on-host registry and the running containers were started from those registry images (not from external sources for the application itself).

---

### User Story 2 - Data persistence across reboots and redeploys (Priority: P1)

A maintainer redeploys the application — for example to ship a new version of the server — or the VM is rebooted (planned or unplanned). After the system comes back up, all tasks, users, plans, and pomodoro history that existed before the event are still present and the system continues serving them.

**Why this priority**: Data loss on reboot or redeploy would make the deployment unusable for real work. The user explicitly called this out as critical.

**Independent Test**: Deploy to the test VM, create a known set of records via the CLI, then (a) reboot the VM and verify the records survive, and (b) re-run the deploy playbook (including image rebuild) and verify the records survive.

**Acceptance Scenarios**:

1. **Given** a deployed system containing user data, **When** the VM is rebooted, **Then** after the system comes back up the same data is queryable through the CLI without any restore step.
2. **Given** a deployed system containing user data, **When** the maintainer re-runs the deploy playbook against the same VM with a new server image, **Then** the existing data remains intact and is served by the new server version.
3. **Given** the application stack is stopped and started, **When** PostgreSQL comes back up, **Then** it reads its data from the same persistent location on the host that it used previously.

---

### User Story 3 - Repeatable redeploys for new versions (Priority: P2)

A maintainer has made changes to the server code on their workstation. They run the same deploy command again. The playbook rebuilds the images, pushes the new versions to the on-host registry, and restarts the application containers using the new images. PostgreSQL is not unnecessarily restarted, and the cutover is brief.

**Why this priority**: Day-to-day operation depends on being able to ship updates without ceremony. Not P1 only because the initial deploy and data persistence must work first.

**Independent Test**: After an initial deploy, make a trivial visible change to the server (for example, a version string), re-run the deploy, and confirm the running server reports the new version while existing data remains intact.

**Acceptance Scenarios**:

1. **Given** a deployed system, **When** the maintainer re-runs the deploy with updated source, **Then** the application container is restarted on the new image version and the database container is not recreated.
2. **Given** a re-run of the deploy with no source changes, **When** the playbook executes, **Then** it completes successfully and is effectively a no-op for the running services (idempotent).

---

### Edge Cases

- The image registry on the VM is unreachable or not yet running on the very first run — the playbook is responsible for starting it before attempting to push.
- A previous deploy left containers in a partially-started state (for example, after an interrupted run); re-running the playbook must converge to a healthy state without manual cleanup.
- The local image build fails on the maintainer's workstation; the playbook must stop before touching the live VM containers so the running system is not disturbed.
- The VM's disk is at or near capacity; the deploy should fail loudly rather than silently produce a broken stack.
- A maintainer accidentally points the playbook at the wrong host; the playbook should require explicit host targeting rather than defaulting to a production host.

## Requirements *(mandatory)*

### Functional Requirements

- **FR-001**: The deployment MUST be invokable from a maintainer's workstation as a single command that targets a specified VM by hostname or IP, with the test target being `user@192.0.2.20`.
- **FR-002**: The deployment MUST assume the target VM is already provisioned, runs Fedora Server, is reachable via passwordless SSH, and grants passwordless sudo to the deploy user; it MUST NOT require any other prior manual setup on the VM.
- **FR-003**: The deployment MUST install and configure on the VM whatever runtime prerequisites are needed to run the application stack via podman-compose, matching how the development environment is run today.
- **FR-004**: The deployment MUST run a container image registry on the target VM itself and use it as the source of the application images consumed by the running stack.
- **FR-005**: The deployment MUST build the application's container images from the current source tree and push them to the on-host registry as part of the same deploy run.
- **FR-006**: The deployment MUST start and manage the application stack (server + PostgreSQL) on the VM via podman-compose, in a configuration consistent with the development compose setup.
- **FR-007**: The deployment MUST persist PostgreSQL data on the host filesystem such that it survives container restart, container recreation, application redeploy, and VM reboot.
- **FR-007a**: The deployment MUST persist the on-host image registry's storage on the host filesystem such that previously pushed application images survive VM reboot and remain pullable by the local container runtime without re-running the deploy.
- **FR-008**: The application stack MUST come back up automatically after a VM reboot without a maintainer re-running the playbook, including being able to pull its application images from the on-host registry using the previously-persisted registry storage.
- **FR-008a**: All services in the deployed stack (server, PostgreSQL, on-host registry, reverse proxy) MUST be configured to restart automatically if their container exits unexpectedly, and MUST remain stopped only when a maintainer explicitly stops them.
- **FR-009**: The deployment MUST be idempotent: running it repeatedly with no source changes converges to the same state without disrupting healthy running services.
- **FR-010**: The deployment MUST support redeploying a new version of the server without losing existing PostgreSQL data and without unnecessarily restarting PostgreSQL.
- **FR-011**: The deployment MUST fail clearly and stop before touching live containers if the image build or push step fails.
- **FR-012**: The deployment MUST require the target host to be specified explicitly per invocation; it MUST NOT silently default to a production host.
- **FR-013**: The deployment MUST produce, on the VM, a runnable production compose configuration that references images by their on-host registry coordinates rather than by paths into the source tree.
- **FR-014**: After a successful deploy, the deployed server MUST be reachable over the network at a documented address and port, and the CLI configured with a provisioned API key MUST be able to authenticate and perform task operations against it.
- **FR-015**: The deployment MUST run a reverse proxy on the VM that fronts the application server, with the proxy being the only network entry point exposed on the VM's host interface. The proxy's configuration MUST be structured so that TLS termination can be added later without re-architecting the stack.

### Key Entities

- **Target VM**: A Fedora Server host identified by SSH-reachable address, with passwordless SSH and passwordless sudo for the deploy user. Hosts the registry, the application server container, the PostgreSQL container, and the persistent PostgreSQL data directory.
- **Application images**: Container images for the server (and any other application components) built from the current source tree and tagged with coordinates that resolve to the on-host registry.
- **On-host image registry**: A container image registry running on the target VM that stores the application images and serves them to the local container runtime.
- **Reverse proxy**: A proxy service running on the VM that accepts inbound HTTP traffic on the host interface and forwards it to the application server container. Provides the seam where TLS termination can later be added.
- **Production compose configuration**: A compose file deployed onto the VM that defines the server, PostgreSQL, and reverse proxy services, references the on-host registry for application images, and mounts a host directory for PostgreSQL data.
- **PostgreSQL data directory**: A directory on the VM's host filesystem that holds the database's data files and persists independently of container lifecycle.
- **Registry storage directory**: A directory on the VM's host filesystem that holds the image registry's blob storage and persists independently of container lifecycle.

## Success Criteria *(mandatory)*

### Measurable Outcomes

- **SC-001**: A maintainer can take a freshly provisioned Fedora VM and reach a working, authenticated CLI session against the deployed server using only the documented deploy command and a user-provisioning step — with no other manual steps on the VM.
- **SC-002**: After a VM reboot, 100% of tasks, users, plans, and pomodoro records that existed before the reboot are still queryable through the CLI, with no restore action required.
- **SC-003**: After a redeploy that ships a new server version, 100% of pre-existing application data is still queryable through the CLI and the server reports the new version.
- **SC-004**: Running the deploy a second time against an already-deployed VM with no source changes completes successfully and results in no restart of the PostgreSQL service.
- **SC-005**: A failed image build on the maintainer's workstation never results in a broken running stack on the VM — the previously running services continue serving requests.
- **SC-006**: The test VM `user@192.0.2.20` can be deployed to end-to-end and exercised via the CLI as the canonical acceptance environment for this feature.

## Assumptions

- The maintainer's workstation has the tooling needed to build the application's container images locally and to run Ansible against the target VM.
- The target VM has outbound network access sufficient to install OS packages and to pull base/runtime images (such as PostgreSQL and the registry) from public sources during initial setup; only the application's own images are required to come from the on-host registry.
- The on-host registry is reachable only from the VM itself; exposing it to the broader network is out of scope for this feature.
- TLS termination, public DNS, firewalling, and backup of the PostgreSQL data directory are out of scope for this feature; the deploy produces a working stack on a reachable address but does not configure those concerns. The reverse proxy is present and structured to accept TLS configuration later, but is not configured with certificates in this feature.
- The "production" compose configuration is allowed to diverge from the development compose file where needed (for example, to reference registry images instead of local build contexts), but the service topology (server + PostgreSQL) matches development.
- Provisioning application users (creating API keys) remains a separate step run against the deployed server, as in development; the deploy itself does not seed users.
