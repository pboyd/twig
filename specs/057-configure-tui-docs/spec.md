# Feature Specification: TUI Configuration Instructions on Download Page

**Feature Branch**: `057-configure-tui-docs`

**Created**: 2026-06-16

**Status**: Draft

**Input**: User description: "users don't know how to configure the TUI because it's not been documented. Let's add instructions to configure the TUI to the download page."

## Clarifications

### Session 2026-06-16

- Q: How should the download page determine the server address shown in the configuration instructions? → A: Display the web app's own current origin dynamically (the web app is served same-origin as the API server, so its origin is the address the CLI must connect to).
- Q: Where should the instructions direct users to obtain the access credential? → A: Link to the in-app Account page's API-key management and tell users to create a key there.

## User Scenarios & Testing *(mandatory)*

### User Story 1 - Configure and launch the TUI for the first time (Priority: P1)

A signed-in user has just downloaded the CLI binary from the download page. They do not yet know that the tool needs to be pointed at a server and given credentials before it will work. On the same download page, immediately after the download instructions, they find clear step-by-step guidance that tells them which settings the tool requires (server address and an access credential), where to obtain the credential, and how to supply both. They follow the steps and successfully launch the interactive TUI connected to their account.

**Why this priority**: This is the entire purpose of the feature. Today a user can download a binary but is left stranded because nothing tells them how to make it connect. Without configuration guidance the downloaded tool is unusable for a new user.

**Independent Test**: From a logged-in session, open the download page, follow only the on-page configuration instructions (no external knowledge), and verify the tool launches into the TUI connected to the user's account.

**Acceptance Scenarios**:

1. **Given** a signed-in user on the download page, **When** they read the page, **Then** instructions explain that the tool requires a server address and an access credential before it can be used.
2. **Given** the configuration instructions, **When** the user looks for the server address, **Then** the correct address for their environment is shown so they can copy it without guessing.
3. **Given** the configuration instructions, **When** the user needs an access credential, **Then** the page tells them where to obtain one (and links there if such a place exists in the app).
4. **Given** the configuration instructions, **When** the user has supplied the required settings, **Then** the page tells them the exact command to launch the interactive TUI.
5. **Given** a user who has completed all steps on the page, **When** they run the launch command, **Then** the TUI opens connected to their account with no further unexplained setup.

---

### User Story 2 - Choose between a quick start and a persistent setup (Priority: P2)

A user wants their configuration to persist across terminal sessions rather than re-entering it each time. The instructions present both a fast way to try the tool immediately and a way to save the configuration so it is remembered on future launches, and explain when each is appropriate.

**Why this priority**: Improves the experience for returning users and reduces repeat friction, but the tool is already usable after User Story 1.

**Independent Test**: Follow the persistent-setup path described on the page, open a new terminal session, run the launch command without re-entering settings, and verify the TUI still connects.

**Acceptance Scenarios**:

1. **Given** the configuration instructions, **When** the user reads them, **Then** both an immediate/temporary method and a persistent/saved method of supplying configuration are described.
2. **Given** the persistent method, **When** the user applies it and starts a new terminal session, **Then** the tool launches without requiring the settings to be re-entered.
3. **Given** both methods are documented, **When** the user reads them, **Then** the page makes clear which source takes precedence if both are set.

---

### Edge Cases

- A user who skips the configuration steps and runs the tool with no settings should, by the tool's existing behavior, receive a clear error; the page should set the expectation that configuration is required so this is not surprising.
- A user browsing from a deployment whose server address differs from the documentation's example must still be able to determine the correct address (hence showing the actual address rather than a hard-coded placeholder).
- A user copying commands should be able to copy them cleanly without surrounding prose breaking the command.
- The instructions must remain accurate if the access-credential management location in the app changes or is unavailable; the guidance should degrade gracefully (describe the source in words even if a direct link is not present).

## Requirements *(mandatory)*

### Functional Requirements

- **FR-001**: The download page MUST present instructions for configuring the interactive TUI, positioned together with the existing download/run guidance.
- **FR-002**: The instructions MUST identify the two settings required to use the tool: the server address and an access credential.
- **FR-003**: The instructions MUST display the server address appropriate to the current deployment so the user can use it without guessing or editing a placeholder. The address shown MUST be the web app's own current origin, determined dynamically, since the web app is served same-origin as the API server.
- **FR-004**: The instructions MUST tell the user where to obtain an access credential and MUST link to the in-app Account page's API-key management, directing the user to create a key there.
- **FR-005**: The instructions MUST describe at least one immediate method (no persistent file required) for supplying the required settings.
- **FR-006**: The instructions MUST describe a persistent method for saving the configuration so it is reused across future sessions, including the location where the configuration is stored.
- **FR-007**: The instructions MUST state which configuration source takes precedence when more than one is set.
- **FR-008**: The instructions MUST state the exact command used to launch the interactive TUI once configuration is in place.
- **FR-009**: Configuration commands and file contents shown to the user MUST be presented in a form that is easy to copy accurately.
- **FR-010**: The instructions MUST be visible to a signed-in user on the download page without requiring any additional navigation beyond reaching that page.
- **FR-011**: The instructions MUST use language consistent with the rest of the app's user-facing copy and centralized message conventions.

### Key Entities

- **TUI configuration settings**: The user-facing settings required to run the tool — the server address and an access credential — plus the optional persistent configuration store and its precedence relative to immediate/temporary settings.

## Success Criteria *(mandatory)*

### Measurable Outcomes

- **SC-001**: A new user can go from a freshly downloaded binary to a connected, running TUI using only the on-page instructions, with no external documentation.
- **SC-002**: 100% of the settings required to launch the TUI are documented on the page (server address, access credential, and the launch command).
- **SC-003**: The server address shown on the page matches the address the tool must actually connect to for the current deployment.
- **SC-004**: A user can locate where to obtain an access credential directly from the instructions in one step.
- **SC-005**: Support questions or confusion about how to start the TUI after downloading are eliminated, because every required step is on the page.

## Assumptions

- The configuration model matches the existing tool behavior: the TUI launches when the binary is run with no arguments, and it requires a server address and an access credential supplied either via environment variables or a TOML configuration file in the user's config directory, with environment variables taking precedence.
- The web app is served from the same origin as the API server, so the browser's current origin is the correct server address for the CLI to connect to.
- The web app's existing Account page (API-key management: create/list/revoke) is the canonical place for a user to obtain an access credential, and the instructions link there and direct the user to create a key.
- The instructions are documentation/copy added to the existing download page; no changes to the CLI/TUI configuration mechanism or to backend endpoints are required.
- The current downloadable binary targets a single platform; instructions assume that platform and reuse the existing download page's platform framing.
- Advanced configuration (e.g., multiple named profiles, pomodoro lifecycle hooks) is out of scope for the on-page instructions, which focus on first-run connection; such details remain in the existing specs/config schema documentation.
