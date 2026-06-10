# Feature Specification: Download CLI Binary from Web App

**Feature Branch**: `047-download-cli-binary`

**Created**: 2026-06-10

**Status**: Draft

**Input**: User description: "Provide an option to download the CLI binary from the webapp."

## Clarifications

### Session 2026-06-10

- Q: How should the binary be hosted and served? → A: Server-hosted — the `linux/amd64` CLI is built into the server's container image and served by a new authenticated server endpoint on the same origin as the web app.
- Q: Where in the web app should the download option live? → A: A dedicated download page/route, reached via a link from the app header.

## User Scenarios & Testing *(mandatory)*

### User Story 1 - Download the CLI binary (Priority: P1)

A signed-in user wants to start using the command-line tool. From the web app they open a clearly-labeled download option, click it, receive the Linux (x86-64) binary, and can run the tool.

**Why this priority**: This is the entire core of the feature. Without the ability to obtain a runnable binary from the web app, nothing else matters. It is a complete, demonstrable slice of value on its own.

**Independent Test**: From a logged-in session, locate the download option, activate it, and verify a runnable Linux x86-64 binary is received.

**Acceptance Scenarios**:

1. **Given** a signed-in user, **When** they open the download option, **Then** a single, clearly-labeled download for Linux (x86-64) is presented.
2. **Given** the download option, **When** the user activates it, **Then** the binary is delivered to their device.
3. **Given** the downloaded binary, **When** the user runs it on a Linux x86-64 machine with valid credentials, **Then** it behaves identically to a binary obtained by building from source.

---

### User Story 2 - Verify what I'm downloading (Priority: P2)

A security-conscious user wants to confirm the version and integrity of the binary before running it. The download option displays the version being offered and a checksum so the user can verify the file after downloading.

**Why this priority**: Improves trust and supports careful users, but the binary is usable without it.

**Independent Test**: Open the download option, read the displayed version and checksum, download the binary, and confirm the computed checksum of the file matches the displayed value.

**Acceptance Scenarios**:

1. **Given** the download option, **When** the user views it, **Then** the version of the offered tool is clearly displayed.
2. **Given** the download option, **When** the user views it, **Then** a checksum for the binary is shown.
3. **Given** a downloaded binary, **When** the user computes its checksum, **Then** it matches the value displayed in the web app.

---

### Edge Cases

- What happens if the binary artifact is missing or temporarily unavailable? The user sees a clear, friendly error rather than a silent failure or a corrupt/empty file.
- How does the system behave for a user browsing from a non-Linux or non-x86-64 device? The download is still offered, with a clear label stating it targets Linux (x86-64) so the user understands what they are getting; it is the user's responsibility to run it on a compatible machine.
- How does the system handle a very large binary on a slow connection? The download proceeds as a normal file transfer; the user can cancel and retry.
- What happens when a new version is released? The download option reflects the current published version without requiring the user to clear caches or take manual action.

## Requirements *(mandatory)*

### Functional Requirements

- **FR-001**: The web app MUST provide a dedicated download page, reachable via a link from the app header within the signed-in experience, that presents the command-line binary download.
- **FR-002**: The download option MUST offer the Linux (x86-64) build and label it clearly so the user knows the target platform.
- **FR-003**: Activating the download MUST deliver the unmodified binary artifact to the user's device.
- **FR-004**: The system MUST display the version of the tool being offered for download.
- **FR-005**: The system MUST display a checksum for the binary so users can verify file integrity.
- **FR-006**: When the binary cannot be served, the system MUST surface a clear, user-friendly error instead of delivering a corrupt or empty file.
- **FR-007**: The download option MUST reflect the currently published version of the tool without requiring manual user intervention to see updates.
- **FR-008**: Access to the download MUST follow the same access rules as the rest of the signed-in web app (available to authenticated users).
- **FR-009**: The binary MUST be served by the server on the same origin as the web app, behind authentication; the file MUST NOT be reachable by unauthenticated requests.

### Key Entities *(include if feature involves data)*

- **CLI Binary Artifact**: A runnable build of the command-line tool for Linux (x86-64). Key attributes: version, file size, integrity checksum, human-readable platform label.

## Success Criteria *(mandatory)*

### Measurable Outcomes

- **SC-001**: A signed-in user can locate the download option and obtain the binary in under 30 seconds and with no more than two interactions.
- **SC-002**: 100% of delivered binaries match the checksum displayed in the web app (no corrupt or altered deliveries).
- **SC-003**: A binary downloaded through the web app behaves identically to one obtained by building from source for the same version.
- **SC-004**: The download option offered in the web app always yields a working download (zero broken or dead download links).

## Assumptions

- The only supported target is Linux (x86-64); no other operating systems or architectures are offered. Additional targets may be added later without changing the user-facing flow.
- The web app is the existing authenticated single-page app; the download option lives inside that signed-in experience, so it inherits the app's authentication and is available to logged-in users. Public (pre-login) downloading is out of scope for this feature.
- The Linux x86-64 binary is built into the server's container image during the image build and served by an authenticated server endpoint on the same origin as the web app (no external hosting, no separate release pipeline).
- A single "current" version of the tool is offered at a time; serving historical/older versions is out of scope for v1.
- The checksum is provided for user-side verification; the web app is not responsible for performing the verification on the user's behalf.
