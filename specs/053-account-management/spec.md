# Feature Specification: Self-Serve Account Management

**Feature Branch**: `053-account-management`

**Created**: 2026-06-13

**Status**: Draft

**Input**: User description: "self-serve account management — The web app needs basic account management functions: Change a password; Add/Revoke API keys. Account creation will continue to be through the `provision` command to the server, but once the account is created, users should be able to take care of the rest on their own."

## Clarifications

### Session 2026-06-13

- Q: When changing a password, how should the account's other active web sessions be treated? → A: Keep all sessions valid — a password change does not invalidate any existing sessions.
- Q: Is a label required when creating an API key, or optional? → A: Optional with a sensible default when omitted.
- Q: What minimum password strength should be enforced? → A: 8+ characters, no character-composition rules.
- Q: Should there be a limit on how many active API keys one account can hold? → A: No limit.

## User Scenarios & Testing *(mandatory)*

### User Story 1 - Change my password (Priority: P1)

A signed-in user wants to change their account password — for example, after suspecting it was exposed, or on a routine security rotation — without contacting an administrator or re-running the provisioning command.

**Why this priority**: Password rotation is the most fundamental self-service security need. Today it requires an administrator to re-provision the account, which is operationally painful and resets credentials the user may not want reset. This is the highest-value, most-requested capability and is independently shippable.

**Independent Test**: Sign in, open account settings, enter the current password and a new password, submit, sign out, and confirm the new password works while the old one is rejected.

**Acceptance Scenarios**:

1. **Given** a signed-in user on the account settings screen, **When** they enter their correct current password and a valid new password (and confirmation), **Then** the password is changed and they see a clear success confirmation.
2. **Given** a signed-in user, **When** they enter an incorrect current password, **Then** the change is rejected with a message indicating the current password is wrong, and the password is unchanged.
3. **Given** a signed-in user, **When** the new password and its confirmation do not match, **Then** the change is rejected before submission with a clear validation message.
4. **Given** a signed-in user, **When** the new password does not meet minimum strength requirements, **Then** the change is rejected with guidance on the requirements.

---

### User Story 2 - Create a new API key (Priority: P2)

A signed-in user wants to generate a new API key so they can use the CLI on a new machine or rotate an existing key, giving each key a recognizable label.

**Why this priority**: API key creation unblocks CLI usage from new devices without administrator involvement. It depends on the account-management surface introduced in P1 but delivers distinct value and can ship independently after it.

**Independent Test**: Sign in, open account settings, create a new key with a label, and confirm the full key value is displayed exactly once and that the new key then authenticates a CLI/API request.

**Acceptance Scenarios**:

1. **Given** a signed-in user on the account settings screen, **When** they create a new API key with a label, **Then** the full key value is displayed exactly once with a clear warning that it cannot be retrieved again, and the key appears in their list of keys.
2. **Given** a newly created API key, **When** it is used to authenticate an API/CLI request, **Then** the request succeeds.
3. **Given** a signed-in user, **When** they create a key without providing a label, **Then** the system accepts the creation and stores a sensible default label, and the key is still usable.
4. **Given** a user who has navigated away after creating a key, **When** they return to the key list, **Then** the full key value is no longer shown (only non-secret metadata).

---

### User Story 3 - View and revoke API keys (Priority: P2)

A signed-in user wants to see the API keys associated with their account and revoke any key they no longer trust or use, so a lost or compromised key can be disabled.

**Why this priority**: Revocation is the security counterpart to creation; without it, a leaked key cannot be disabled short of administrator re-provisioning. It is paired closely with P2 but is independently testable.

**Independent Test**: Sign in, view the list of API keys (showing non-secret metadata), revoke one, and confirm that the revoked key no longer authenticates while remaining keys still work.

**Acceptance Scenarios**:

1. **Given** a signed-in user with one or more API keys, **When** they open the key list, **Then** they see each key's label and creation date (and other non-secret metadata), but never the secret key value.
2. **Given** a signed-in user viewing their keys, **When** they revoke a specific key and confirm the action, **Then** that key is removed from the list and can no longer authenticate API/CLI requests.
3. **Given** a user revoking the only remaining API key, **When** they confirm, **Then** the system warns that this will remove all CLI/API access and proceeds only on explicit confirmation.
4. **Given** a user revoking a key, **When** the revocation succeeds, **Then** other keys belonging to the user continue to authenticate normally.

---

### Edge Cases

- **Wrong current password**: Repeated incorrect current-password attempts during a password change are rejected and rate-limited consistently with the existing login protection.
- **Password reuse**: The new password being identical to the current one is permitted but does not break the session; the active session remains valid.
- **Session handling on password change**: Changing the password does not invalidate any existing web sessions; the current session and all other active sessions for the account remain valid.
- **API keys unaffected by password change**: Changing the password does not revoke existing API keys, since they are independent credentials.
- **Revoking the in-use key**: A user may revoke the very key currently used by their CLI; subsequent CLI requests with that key fail until a new key is issued. The web session (cookie-based) is unaffected.
- **Last key revoked**: A user can revoke their final API key, leaving zero keys; this is allowed after an explicit warning, and they can create a new one later.
- **Duplicate labels**: Two keys may share the same label; labels are descriptive only and are not required to be unique.
- **Concurrent revocation**: Revoking a key that was already revoked (e.g., from another tab) results in a clear, idempotent outcome rather than an error.
- **Unauthenticated access**: Account-management functions are unavailable to users who are not signed in.

## Requirements *(mandatory)*

### Functional Requirements

#### Password management

- **FR-001**: The web app MUST provide an account-management area accessible only to authenticated users.
- **FR-002**: A signed-in user MUST be able to change their own password by supplying their current password and a new password.
- **FR-003**: The system MUST verify the supplied current password before applying any password change, and MUST reject the change if it is incorrect.
- **FR-004**: The system MUST require the new password to be entered twice (or otherwise confirmed) and MUST reject the change if the two entries do not match.
- **FR-005**: The system MUST require the new password to be at least 8 characters long and reject shorter passwords with user-facing guidance. No character-composition (upper/lower/digit/symbol) rules are imposed.
- **FR-006**: A password change MUST NOT invalidate any existing web sessions; the current session and all other active sessions for the account remain valid afterward.
- **FR-007**: A password change MUST NOT revoke or alter the user's existing API keys.

#### API key management

- **FR-008**: A signed-in user MUST be able to view a list of their own API keys, showing non-secret metadata (at minimum a label and creation date).
- **FR-009**: The system MUST NEVER display or expose the secret value of an existing API key after its initial creation.
- **FR-010**: A signed-in user MUST be able to create a new API key associated with their account, optionally providing a descriptive label. When no label is provided, the system MUST store a sensible default label rather than rejecting the request.
- **FR-011**: On creation, the system MUST display the full secret key value exactly once, accompanied by a clear warning that it cannot be retrieved later.
- **FR-012**: A newly created API key MUST be immediately usable to authenticate API/CLI requests for that user.
- **FR-013**: A signed-in user MUST be able to revoke any of their own API keys, and the system MUST require explicit confirmation before revoking.
- **FR-014**: A revoked API key MUST immediately stop authenticating API/CLI requests, while all of the user's non-revoked keys continue to work.
- **FR-015**: The system MUST allow a user to hold multiple concurrent API keys per account, with no enforced upper limit on the number of active keys.
- **FR-016**: The system MUST allow a user to revoke their last remaining key, after warning that doing so removes all CLI/API access.

#### Scope and authorization

- **FR-017**: All account-management actions MUST operate only on the currently authenticated user's own account; a user MUST NOT be able to view, change, or revoke another user's password or keys.
- **FR-018**: Account creation MUST remain exclusively an administrator action via the existing server-side provisioning command; the self-serve surface MUST NOT create new accounts.
- **FR-019**: Repeated failed current-password verifications MUST be subject to the same abuse protections (e.g., rate limiting) as the existing sign-in flow.

### Key Entities *(include if feature involves data)*

- **User account**: Represents an individual's credentials. Has a username (set at provisioning, not editable here) and a password (changeable via this feature). Owns zero or more API keys and zero or more active sessions.
- **API key**: A long-lived credential belonging to a user, used to authenticate CLI/API requests. Has a descriptive label and a creation timestamp; its secret value is shown only at creation time and is never recoverable afterward. Can be individually revoked.
- **Session**: A web sign-in session belonging to a user. Affected by password changes (other sessions invalidated) but independent of API keys.

## Success Criteria *(mandatory)*

### Measurable Outcomes

- **SC-001**: A signed-in user can change their password end-to-end in under 1 minute without administrator involvement.
- **SC-002**: After a successful password change, the old password is rejected and the new password is accepted on the next sign-in, 100% of the time.
- **SC-003**: A signed-in user can create a new API key and successfully authenticate a CLI/API request with it within 2 minutes, without administrator involvement.
- **SC-004**: A revoked API key fails authentication on its next use, 100% of the time, while the user's other keys continue to succeed.
- **SC-005**: The secret value of an API key is visible only during the single creation step and is never displayed again, verified across every account-management screen.
- **SC-006**: Account-management requests that target another user's account or that arrive unauthenticated are rejected 100% of the time.
- **SC-007**: After launch, password changes and API-key rotation no longer require administrator action (target: zero support tickets needing an administrator for these tasks).

## Assumptions

- The existing session-cookie authentication for the web app and API-key authentication for the CLI remain in place; this feature builds on them rather than replacing them.
- The existing data model already supports multiple labeled API keys per user and independent password hashing, so no fundamental credential redesign is required.
- Minimum password strength is a length-only rule of 8+ characters with no character-composition requirements (see FR-005).
- Password changes do not invalidate any existing sessions (see FR-006); session revocation on credential change is out of scope.
- API key revocation is immediate and permanent (no soft-delete/restore); a "revoked" key is simply removed.
- No email-based password reset / "forgot password" flow is in scope — changing a password requires knowing the current one. Users who have lost their password are recovered via administrator re-provisioning.
- Username changes, account deletion, and multi-factor authentication are out of scope for this feature.
- The web app remains a same-origin SPA talking to the existing server; no new third-party identity provider is introduced.
