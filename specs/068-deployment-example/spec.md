# Feature Specification: Real-World Deployment Example

**Feature Branch**: `068-deployment-example`

**Created**: 2026-07-25

**Status**: Draft

**Input**: User description: "real-world deployment example — example deployment scripts users can base their own production deployments on: Ansible, bare Fedora machine over SSH, configurable username and sudo password, podman-compose, PostgreSQL in podman with a persistent volume, Caddy reverse proxy with configurable domain, optional Let's Encrypt certificate, optional Cloudflare tunnel for NAT traversal. The reference implementation does most of this today and can serve as the basis. Since these are meant primarily as a starting point for a user to copy and own, document that process."

## Overview

Twig currently documents only a local development workflow (`make dev`). Anyone who wants to run Twig
for real — on a home server, a VPS, a spare Fedora box — has to invent their own deployment from
scratch: host packages, container orchestration, database persistence, TLS, and remote access.

This feature ships a **reference deployment** in the Twig repository: an automated, repeatable recipe
that turns a freshly installed Fedora machine reachable over SSH into a working Twig instance served
over a domain name, plus the documentation a user needs to copy that recipe into their own
infrastructure repository and take ownership of it.

The example is explicitly a *starting point*, not a product surface. Its success is measured by how
quickly a new operator gets a working instance and how confidently they can then modify it.

## User Scenarios & Testing *(mandatory)*

### User Story 1 - Deploy Twig to a bare Fedora machine (Priority: P1)

An operator has a freshly installed Fedora machine on their network. They can reach it over SSH with a
regular login account that has `sudo` access (protected by a password). They obtain the Twig
deployment example, fill in three things — the host address, the login account, and the domain name
they want to serve Twig on — supply the sudo password when prompted, and run the deployment. When it
finishes, Twig's web UI answers on the configured domain and the CLI can talk to the same instance.

**Why this priority**: This is the whole point of the feature. Without a working baseline deploy, the
optional TLS and tunnel layers have nothing to attach to. On its own it already delivers a usable
self-hosted Twig.

**Independent Test**: Provision a clean Fedora VM, fill in the host/account/domain configuration, run
the deployment, and confirm the Twig web UI loads at the configured domain and a provisioned API key
authenticates a CLI request.

**Acceptance Scenarios**:

1. **Given** a clean Fedora machine reachable over SSH and a filled-in configuration, **When** the
   operator runs the deployment, **Then** it completes without manual intervention beyond supplying
   the sudo password, and reports success.
2. **Given** a completed deployment, **When** the operator opens the configured domain in a browser,
   **Then** the Twig login page is served and a successful login reaches the task list.
3. **Given** a completed deployment, **When** the operator points the Twig CLI at the deployed
   instance with a provisioned API key, **Then** task and plan commands succeed against it.
4. **Given** a completed deployment, **When** the operator reboots the host, **Then** Twig, its
   database, and the reverse proxy come back up on their own with all previously created tasks intact.
5. **Given** a configuration that omits or malformats a required value (host, account, or domain),
   **When** the operator runs the deployment, **Then** it stops before changing anything on the host
   and names the missing or invalid value and where to set it.

---

### User Story 2 - Copy the example and make it your own (Priority: P2)

An operator wants to run Twig long-term and treat their deployment as their own infrastructure code.
They read the deployment documentation, copy the example out of the Twig repository into their own
private repository, learn which files are theirs to edit, where secrets belong, and what to expect
when Twig itself changes upstream. They diverge from the example freely without losing the ability to
see what changed upstream.

**Why this priority**: The user identified copy-and-own as the primary purpose. An example that only
works verbatim, or whose customization points are undiscoverable, fails even when it deploys
correctly.

**Independent Test**: Following only the written documentation, copy the example into a fresh
directory outside the Twig repo, change the domain and the database credentials, deploy it, and
confirm success — without reading the example's internals to figure out where those values live.

**Acceptance Scenarios**:

1. **Given** the deployment documentation, **When** a new operator reads it start to finish, **Then**
   it states the copy-and-own workflow, the prerequisites on their workstation and on the target host,
   and every value they must supply before the first run.
2. **Given** the copied example, **When** the operator looks for the values they are expected to
   change, **Then** those values are collected in a small number of clearly labeled configuration
   files, separate from the deployment logic.
3. **Given** the copied example, **When** the operator looks for where secrets go, **Then** the
   documentation names one designated encrypted location and the example contains no secret values in
   plain text, only placeholders.
4. **Given** the documentation, **When** the operator asks what support the example carries, **Then**
   it states plainly that the example is a starting point they own after copying, which parts are
   expected to change as Twig evolves, and how to re-check the upstream example later.
5. **Given** the documentation, **When** the operator needs day-two operations, **Then** it covers
   backing up and restoring the database, viewing logs, restarting the stack, provisioning additional
   users, and upgrading to a newer Twig release.

---

### User Story 3 - Redeploy and upgrade without losing data (Priority: P3)

An operator who already has a running instance changes a configuration value, or moves to a newer
Twig release, and re-runs the deployment. Only what needs to change changes; the task database
survives untouched. If the deployment detects that the database storage is not where it expects it to
be, it refuses to finish rather than silently proceeding.

**Why this priority**: A deployment example that people re-run is only trustworthy if re-running is
boring. Data loss on the second run would make the example actively harmful, but it depends on the
baseline deploy existing first.

**Independent Test**: Deploy, create tasks, re-run the deployment unchanged, confirm no service
disruption and no data change; then bump the Twig version, re-run, and confirm the tasks are still
there on the new version.

**Acceptance Scenarios**:

1. **Given** a deployed instance and an unchanged configuration, **When** the operator re-runs the
   deployment, **Then** it reports no changes made and does not restart the running services.
2. **Given** a deployed instance with existing tasks, **When** the operator changes the Twig version
   and re-runs the deployment, **Then** the new version is running, schema migrations have been
   applied, and the pre-existing tasks are unchanged.
3. **Given** a deployed instance whose database storage has gone missing or moved, **When** the
   operator re-runs the deployment, **Then** it fails loudly and explains what it expected to find,
   instead of starting an empty database.

---

### User Story 4 - Serve a trusted certificate for a public domain (Priority: P4)

An operator whose domain resolves publicly to the host turns on automatic certificate issuance in the
configuration and re-deploys. Browsers and the CLI then reach Twig over HTTPS with a publicly trusted
certificate, and renewal happens without the operator doing anything. With the option turned off, the
deployment still works over HTTPS using a locally generated certificate, and the documentation
explains the trust trade-off.

**Why this priority**: Explicitly optional in the request. Valuable for real-world use but not
required for a working instance, and it depends on public DNS the operator may not have.

**Independent Test**: Deploy with automatic certificates enabled against a publicly resolvable
domain, confirm a browser reports a trusted connection, then confirm renewal state persists across a
redeploy.

**Acceptance Scenarios**:

1. **Given** automatic certificates enabled and a publicly resolvable domain, **When** the deployment
   runs, **Then** a publicly trusted certificate is issued for that domain and Twig is served over
   HTTPS without browser warnings.
2. **Given** automatic certificates enabled, **When** the deployment is re-run later, **Then** the
   existing certificate and renewal state are reused rather than re-issued.
3. **Given** automatic certificates disabled, **When** the deployment runs, **Then** Twig is still
   served over HTTPS with a locally generated certificate, and the documentation states that clients
   will not trust it by default and how to proceed.
4. **Given** automatic certificate issuance fails (for example the domain does not resolve to the
   host), **When** the deployment runs, **Then** it reports the failure with the likely cause rather
   than reporting success on a broken site.

---

### User Story 5 - Reach a NAT-bound instance from anywhere (Priority: P5)

An operator's host sits behind a home router with no port forwarding. They enable the outbound tunnel
option, supply the tunnel credential their tunnel provider issued, and re-deploy. Twig then answers on
their domain from outside the network, with no inbound ports opened. With the option off, nothing
tunnel-related is installed.

**Why this priority**: Explicitly optional, applies to a subset of operators, and layers on top of
everything else.

**Independent Test**: Deploy with the tunnel enabled on a host with no inbound reachability, then load
the configured domain from a network outside the host's LAN.

**Acceptance Scenarios**:

1. **Given** the tunnel option enabled and a valid tunnel credential, **When** the deployment runs,
   **Then** the tunnel connector runs on the host and the configured domain serves Twig from outside
   the local network with no inbound ports opened.
2. **Given** the tunnel option enabled but no credential supplied, **When** the deployment runs,
   **Then** it stops before changing the host and explains where to put the credential.
3. **Given** the tunnel option disabled, **When** the deployment runs, **Then** no tunnel connector is
   installed or run, and local access is unaffected.
4. **Given** the tunnel option enabled, **When** the operator consults the documentation, **Then** it
   names the steps that must be completed in the tunnel provider's own console and cannot be
   automated.

---

### Edge Cases

- **Reused host**: the target already runs something on the HTTP/HTTPS ports, or already has a
  conflicting container stack. The deployment should surface the conflict rather than half-apply.
- **Wrong sudo password or no sudo rights**: must fail on connection or privilege escalation with a
  clear message, before any host change.
- **Host firewall enabled or absent**: the deployment must handle both without failing.
- **SELinux enforcing** (the Fedora default): container access to persistent storage must work without
  the operator disabling SELinux.
- **Domain not resolving for the operator's own clients**: documentation must explain that local
  clients need the domain to resolve to the host, and how (local DNS or hosts file) when public DNS
  does not point there.
- **Both optional layers enabled at once**: trusted certificates and the tunnel must coexist without
  a redirect loop or a certificate served for the wrong hostname.
- **Interrupted deployment**: re-running after an interruption must converge rather than requiring
  manual cleanup of the host.
- **Database credentials changed after first deploy**: the documentation must state that changing them
  post-install does not re-initialize an existing database, and what to do instead.
- **Low-memory or not-yet-updated fresh machine**: the deployment should install what it needs and
  report clearly if a prerequisite is unmet.

## Requirements *(mandatory)*

### Functional Requirements

**Deliverable and scope**

- **FR-001**: The repository MUST contain a self-contained example deployment for Twig, located in a
  single directory that can be copied out wholesale.
- **FR-002**: The example MUST deploy a working Twig instance — API, web UI, and database — to a
  Fedora host that starts with nothing but a base OS install and SSH access.
- **FR-003**: The example MUST be runnable from an operator's workstation against a remote host over
  SSH, with no manual steps performed on the host itself.
- **FR-004**: The example MUST be usable unmodified apart from its configuration values; deploying
  MUST NOT require editing deployment logic.

**Configuration**

- **FR-005**: The example MUST let the operator configure, at minimum: target host address, remote
  login account, the domain Twig is served on, the Twig version to deploy, and the database
  credentials.
- **FR-006**: The example MUST support a remote account whose privilege escalation requires a
  password, prompting for it at run time rather than requiring passwordless escalation.
- **FR-007**: The example MUST validate every required configuration value before making any change to
  the target host, and MUST fail with a message naming the offending value and the file to set it in.
- **FR-008**: The example MUST NOT contain real secrets; every secret MUST be a documented placeholder
  with one designated encrypted-at-rest location for the real value.
- **FR-009**: The example MUST keep secret values out of its own console output and out of any file it
  writes to the host with world-readable permissions.

**Runtime shape on the host**

- **FR-010**: Twig, its database, and the reverse proxy MUST run as containers on the host, managed as
  one stack.
- **FR-011**: Database storage MUST persist independently of container lifetime, survive redeploys and
  host reboots, and be reachable with ordinary host-level backup tools.
- **FR-012**: The whole stack MUST start automatically after a host reboot without operator action.
- **FR-013**: The reverse proxy MUST be the only component exposed on the host's network ports; Twig
  and the database MUST NOT be reachable directly from outside the host.
- **FR-014**: Database schema migrations MUST be applied as part of bringing the stack up.
- **FR-015**: The example MUST open only the host firewall ports the reverse proxy needs, and MUST
  succeed whether or not a host firewall is active.

**Optional layers**

- **FR-016**: Automatic issuance of a publicly trusted certificate MUST be an opt-in toggle, off by
  default, requiring no change beyond configuration values to enable.
- **FR-017**: With automatic issuance off, Twig MUST still be served over HTTPS using a locally
  generated certificate.
- **FR-018**: Certificate and renewal state MUST persist across redeploys so that redeploying does not
  trigger re-issuance.
- **FR-019**: An outbound tunnel for reaching the instance from outside a NAT'd network MUST be an
  opt-in toggle, off by default, and MUST require no inbound ports when enabled.
- **FR-020**: When a toggle is off, the components it governs MUST NOT be installed or run on the host.
- **FR-021**: Enabling both optional layers together MUST produce a working instance reachable both
  locally and from outside.

**Verification and safety**

- **FR-022**: The deployment MUST verify the deployed instance actually answers over the configured
  domain before reporting success, and MUST fail if it does not.
- **FR-023**: Re-running the deployment with unchanged configuration MUST report no changes and MUST
  NOT restart running services.
- **FR-024**: The deployment MUST refuse to complete if it detects that the database's persistent
  storage is missing, relocated, or was re-initialized during the run.
- **FR-025**: The example MUST offer a way to preview what a run would change without changing
  anything.

**Documentation**

- **FR-026**: The example MUST include documentation covering prerequisites (workstation and host),
  first-run walkthrough, every configuration value, and each optional layer's requirements.
- **FR-027**: The documentation MUST describe the copy-and-own workflow: how to copy the example into
  the operator's own repository, what is theirs to change, and how to compare against the upstream
  example later.
- **FR-028**: The documentation MUST state the example's support posture — a starting point, not a
  supported product surface — and what may change as Twig evolves.
- **FR-029**: The documentation MUST cover day-two operations: backup and restore of the task
  database, log inspection, restarting the stack, provisioning users, and upgrading Twig.
- **FR-030**: The documentation MUST explain how operators' own clients resolve the configured domain
  to the host, including the case where public DNS does not point at it.
- **FR-031**: Twig's top-level README MUST point from its existing local-development quick start to
  the deployment example, so operators can find it.
- **FR-032**: All operator-facing text the example emits — validation failures, progress messages, and
  its documentation prose — MUST follow the project's playful-but-clear voice, staying accurate and
  actionable.

### Key Entities

- **Target host**: the Fedora machine being deployed to. Identified by an address and a login account;
  holds all persistent state.
- **Deployment configuration**: the operator-owned values (host, account, domain, Twig version,
  database credentials, optional-layer toggles) kept separately from deployment logic.
- **Secret store**: the single designated encrypted location for credentials — database password,
  certificate-provider credential, tunnel credential.
- **Service stack**: the set of containers managed as one unit on the host — Twig server, database,
  reverse proxy, and optionally the tunnel connector.
- **Persistent data**: the task database's storage on the host, the only state whose loss is
  unrecoverable, plus certificate/renewal state whose loss is recoverable but costly.
- **Twig release**: the identifier of the published Twig version being deployed; the primary value
  changed when upgrading.

## Success Criteria *(mandatory)*

### Measurable Outcomes

- **SC-001**: An operator with SSH access to a fresh Fedora machine reaches a working Twig instance on
  their chosen domain in under 30 minutes of wall-clock time, of which under 10 minutes is hands-on.
- **SC-002**: First-run configuration requires the operator to supply no more than 5 values before the
  deployment succeeds.
- **SC-003**: A second, unchanged run of the deployment reports zero changes and causes zero seconds
  of service interruption.
- **SC-004**: Upgrading to a newer Twig release is a one-value change plus a re-run, and preserves
  100% of existing tasks, goals, and plans.
- **SC-005**: 100% of missing or malformed required configuration values are reported before any change
  is made to the target host.
- **SC-006**: The example contains zero real secrets; every secret is a placeholder resolved from the
  designated encrypted location.
- **SC-007**: An operator who has never seen the example can copy it, change the domain and database
  credentials, and deploy successfully using the documentation alone, without reading the deployment
  logic.
- **SC-008**: A deployed instance survives a host reboot with all services back and all data intact,
  with no operator action.
- **SC-009**: Each optional layer can be toggled on or off independently, and all four combinations
  produce a working instance for the access pattern each combination targets.
- **SC-010**: Following the documented backup procedure and restoring onto a freshly deployed host
  recovers all tasks, goals, and plans.

## Assumptions

- **Published image**: the example deploys the already-published Twig server image (which bundles the
  web UI) rather than building from source on the operator's workstation. A source-build path is out
  of scope for the example.
- **Target platform**: current Fedora (Server or an equivalent minimal install) on x86-64, with
  SELinux enforcing and `dnf` available. RHEL-family compatibility is welcome but not verified.
- **Operator baseline**: the operator can install Ansible on their workstation, has SSH key access to
  the host, knows the account's sudo password, and controls DNS for the domain they choose.
- **Single host, single instance**: one Twig instance on one machine. Multi-host, high-availability,
  and horizontal scaling are out of scope.
- **Certificate issuance path**: the opt-in trusted-certificate layer uses Let's Encrypt through the
  reverse proxy's built-in automation. The default validation path assumes the domain resolves
  publicly to the host; the tunnel scenario, where it does not, is documented as a separate
  configuration.
- **Tunnel provider**: the opt-in tunnel is Cloudflare Tunnel, driven by a connector credential the
  operator creates in Cloudflare's console. Creating the tunnel and mapping its public hostname stay
  manual, since they happen outside the host.
- **User provisioning**: creating the first Twig account remains a documented post-deploy command
  rather than something the deployment does silently, so the operator sees and keeps the API key.
- **Testing**: verification is manual against a throwaway VM. Automated deployment testing (CI
  provisioning a VM per commit) is out of scope.

## Out of Scope

- Distributions other than Fedora / RHEL-family.
- Rootless containers, Kubernetes, systemd Quadlet, or any orchestrator other than the podman-based
  stack the example uses.
- Monitoring, alerting, log shipping, and off-host backup automation. Backup is documented as a
  procedure, not automated.
- Reverse proxies, tunnel providers, and certificate authorities beyond the one of each the example
  supports.
- Automated database restore tooling, or migrating an existing local development database onto a
  deployed host.
- Provisioning the machine itself (cloud instance creation, OS installation, disk layout) and DNS
  record management.
- Any change to Twig's own application code, configuration surface, or container image.
