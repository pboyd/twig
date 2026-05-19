# Feature Specification: User Authentication

**Feature Branch**: `003-user-auth`

**Created**: 2026-05-19

**Status**: Draft

**Input**: User description: "Read the plan in `docs/plans/2026-05-19-auth-design.md`"

## Clarifications

### Session 2026-05-19

- Q: Should sign-in have brute-force protection (rate limiting / lockout)? → A: No — explicitly out of scope; rely on strong provisioned passwords and the trusted-LAN deployment.
- Q: Should authentication events be logged? → A: Yes — log key auth events (sign-in success/failure, sign-out, rejected credentials) to the service log.
- Q: What happens when provisioning is given an already-existing username? → A: Overwrite — update that account's password and issue a fresh API key.

## User Scenarios & Testing *(mandatory)*

### User Story 1 - Browser user signs in to manage their tasks (Priority: P1)

A person opens the todo service in a web browser, enters their username and password, and gains access to their task list. While signed in, they can read and change their own tasks without re-entering credentials. When finished, they sign out and their access ends.

**Why this priority**: Browser sign-in is the primary way people interact with the service. Without it, no one can reach their tasks through the frontend, so it delivers the core value of the feature on its own.

**Independent Test**: Provision one account, sign in with the correct credentials, confirm the task list loads and stays reachable across multiple requests, then sign out and confirm task access is refused afterward.

**Acceptance Scenarios**:

1. **Given** a provisioned account, **When** the user submits the correct username and password, **Then** the system grants access and remembers them for subsequent requests.
2. **Given** a provisioned account, **When** the user submits an incorrect username or password, **Then** the system refuses access and does not grant a session.
3. **Given** a signed-in user, **When** they perform task operations, **Then** they succeed without re-entering credentials.
4. **Given** a signed-in user, **When** they sign out, **Then** their session ends and further task requests are refused until they sign in again.
5. **Given** a session that has passed its expiry, **When** the user makes a request, **Then** the system refuses access and the user must sign in again.

---

### User Story 2 - CLI tool authenticates with an API key (Priority: P2)

A person uses a command-line tool to manage their tasks. The tool is configured once with a long-lived API key. On every request, the tool presents that key and the service resolves it to the same user, granting access to that user's tasks.

**Why this priority**: The CLI is a secondary but supported client. It depends on the same per-user access model as the browser flow but uses a credential suited to unattended, scripted use rather than an interactive login.

**Independent Test**: Provision an account with an API key, configure the CLI with that key, run task commands, and confirm they operate on the correct user's tasks; then present an invalid or revoked key and confirm access is refused.

**Acceptance Scenarios**:

1. **Given** a valid API key, **When** the CLI makes a task request carrying that key, **Then** the system resolves it to the owning user and grants access.
2. **Given** an invalid or unrecognized API key, **When** a request carries it, **Then** the system refuses access.
3. **Given** an API key that has been revoked, **When** a request carries it, **Then** the system refuses access.
4. **Given** a request that carries no credential of either kind, **When** it reaches a task operation, **Then** the system refuses access.

---

### User Story 3 - Operator provisions accounts at deploy time (Priority: P3)

An operator deploying the service supplies the set of user accounts and their passwords. On startup in provisioning mode, the service creates each account, generates one API key per account, displays each raw API key exactly once, and then stops so the operator can record the keys and start the service normally.

**Why this priority**: Provisioning is how accounts and keys come to exist, so the other stories depend on it. It is ranked lower only because it is an infrequent, operator-facing setup step rather than an everyday user journey.

**Independent Test**: Run the service in provisioning mode with two accounts, confirm two accounts exist with one API key each, and confirm each raw key is shown exactly once during that run.

**Acceptance Scenarios**:

1. **Given** the service is started in provisioning mode with a list of accounts, **When** startup completes, **Then** each account exists with a stored password and exactly one API key.
2. **Given** provisioning has run, **When** the operator inspects the output, **Then** each account's raw API key was displayed exactly once and is not retrievable afterward.
3. **Given** an operator needs to replace a key, **When** they revoke the existing key and re-run provisioning, **Then** a new key is issued and the old key no longer grants access.

---

### Edge Cases

- A request presents both a browser session and an API key: the system resolves the request to a user using a defined, deterministic precedence rather than failing.
- A user attempts to read or modify a task that belongs to another user by referencing its identifier directly: the system refuses, behaving as if the task does not exist for that user.
- A session reaches its expiry mid-use: the next request is refused and the user is required to sign in again.
- An API key is revoked while a CLI tool still has it configured: the next request using that key is refused.
- The sign-in and sign-out actions themselves are reachable without an existing credential; all other task operations require one.
- Provisioning is given a username that already exists: the system updates that account's password and issues a fresh API key rather than duplicating the account.

## Requirements *(mandatory)*

### Functional Requirements

- **FR-001**: System MUST require a valid credential for every task operation; requests without a recognized credential MUST be refused.
- **FR-002**: System MUST support two credential types — an interactive session for browser clients and a long-lived API key for CLI clients — and resolve both to a single owning user.
- **FR-003**: System MUST provide a sign-in action that accepts a username and password, grants a session on success, and refuses access on failure without revealing whether the username or the password was wrong.
- **FR-004**: System MUST provide a sign-out action that ends the current session so it can no longer be used.
- **FR-005**: Sessions MUST expire after a configurable duration, after which they no longer grant access.
- **FR-006**: System MUST keep a browser session credential inaccessible to client-side scripts and MUST only transmit it over a secure connection.
- **FR-007**: System MUST resist cross-site request forgery for session-authenticated requests.
- **FR-008**: System MUST store passwords only in a non-reversible hashed form; plaintext passwords MUST NOT be persisted.
- **FR-009**: System MUST store API keys only in a non-reversible hashed form; the raw API key MUST be displayed exactly once at provisioning time and never retrievable afterward.
- **FR-010**: API keys and session tokens MUST be generated from cryptographically strong randomness so they cannot be guessed.
- **FR-011**: System MUST allow an API key to be revoked, after which requests carrying that key are refused.
- **FR-012**: System MUST associate every task with exactly one owning user.
- **FR-013**: System MUST ensure each user can read and modify only their own tasks; tasks owned by other users MUST NOT be visible or modifiable, including when referenced by direct identifier.
- **FR-014**: Credential checking MUST be applied uniformly to all task operations, independent of which credential type authenticated the request.
- **FR-015**: System MUST provide a deploy-time provisioning mode that creates the configured user accounts, issues one API key per account, displays each raw key once, and then stops.
- **FR-016**: When a request carries more than one credential type, the system MUST resolve the user using a defined, deterministic precedence.
- **FR-017**: The sign-in and sign-out actions MUST be reachable without an existing credential.
- **FR-018**: System MUST record key authentication events — sign-in successes, sign-in failures, sign-out, and rejected credentials — to the service log.
- **FR-019**: When provisioning encounters a username that already exists, the system MUST update that account's password and issue a fresh API key for it, without duplicating the account.

### Key Entities *(include if feature involves data)*

- **User account**: Represents a person authorized to use the service. Identified by a unique username; holds a stored (hashed) password. Owns tasks, sessions, and API keys.
- **Session**: A time-limited credential created by an interactive sign-in, belonging to one user. Has a creation time and an expiry time, after which it no longer grants access.
- **API key**: A long-lived credential belonging to one user, intended for CLI use. Stored only in hashed form; carries an optional human-readable label and a creation time. Can be revoked.
- **Task**: An existing entity, now owned by exactly one user. All task reads and writes are scoped to the owning user.

## Success Criteria *(mandatory)*

### Measurable Outcomes

- **SC-001**: 100% of task operations that lack a valid credential are refused.
- **SC-002**: A user never sees or modifies another user's task — zero cross-user task access in testing, including attempts that reference another user's task by identifier.
- **SC-003**: Both client types (browser and CLI) can reach the same user's tasks using their respective credentials, with identical task-access behavior regardless of credential type.
- **SC-004**: A signed-in user can complete sign-in and reach their task list in under 30 seconds and without entering credentials more than once per session.
- **SC-005**: Sessions stop granting access once their configured lifetime elapses, verified against the configured duration.
- **SC-006**: After provisioning, no raw API key or plaintext password can be retrieved from stored data.
- **SC-007**: A revoked API key is refused on the first request following revocation.

## Assumptions

- The approved design document `docs/plans/2026-05-19-auth-design.md` is the source of truth; this specification restates its intent in user and requirement terms.
- User accounts are created only at deploy time by an operator; in-app self-service registration and user management are out of scope.
- The service is deployed on a trusted LAN over a secure connection, serving roughly one or two users per instance, so scale and public-internet hardening beyond the listed properties are not in scope.
- The default session lifetime is 30 days unless an operator configures a different value.
- Each user is provisioned with exactly one API key initially; additional keys per user are not required, though the model does not forbid them.
- The existing task functionality is unchanged except for becoming owned by, and scoped to, a single user.

### Out of Scope

- In-app user management or self-service account creation.
- Password reset or recovery flows.
- Multi-factor authentication.
- External identity providers (OAuth / OIDC / SSO).
- A user-facing interface for API key rotation; rotation is done by revoking a key and re-running provisioning.
- Brute-force protection on sign-in — rate limiting, throttling, and account lockout are out of scope; the trusted-LAN deployment and strong provisioned passwords are relied on instead.
