# Feature Specification: Alternate Profiles

**Feature Branch**: `033-alternate-profiles`

**Created**: 2026-06-02

**Status**: Draft

**Input**: User description: "Alternate profiles — some users have more than one account (e.g. work and home) and want to switch between them. The config file should support alternate named profiles. `twig --profile home` selects the `home` profile; with no flag (or `--profile default`) the root config is used. An explicit `--profile` flag takes precedence over environment variables, but environment variables take precedence over the implicit default profile."

## Clarifications

### Session 2026-06-02

- Q: How should a named profile's settings resolve against the root-level settings? → A: Hybrid — credentials (API URL and API key) are profile-scoped with no inheritance (an unset credential in the profile does NOT fall back to the root credential); all other settings (e.g. pomodoro lifecycle hooks) always come from the root and cannot be overridden per profile.
- Q: When env vars (TWIG_ADDR/TWIG_API_KEY) are set, do they override credentials for an explicitly selected profile too? → A: Yes — environment variables override the credentials of whichever profile is selected (named or default). The earlier "explicit --profile beats env vars" idea is dropped; env vars always win for credentials.
- Q: Should profile selection also be possible via an environment variable, not just the `--profile` flag? → A: Yes — a `TWIG_PROFILE` env var selects the profile. The `--profile` flag overrides `TWIG_PROFILE` when both are set; if neither is set, the root/default profile is used. (This selection axis is separate from the `TWIG_ADDR`/`TWIG_API_KEY` credential-override env vars.)

## User Scenarios & Testing *(mandatory)*

### User Story 1 - Switch to a named profile (Priority: P1)

A person who keeps a separate account for work and for home wants to point twig at the right account without editing files or juggling environment variables. They add a named profile block to their config file and select it at launch with `--profile <name>`. Everything they do in that session — every command and the interactive TUI — talks to the account belonging to that profile.

**Why this priority**: This is the core of the feature. Without the ability to select a named profile, none of the multi-account value exists. It is the minimum viable slice that delivers the whole point of the work.

**Independent Test**: Configure a root account and a second `home` profile in the config file. Run a command with `--profile home` and confirm it acts against the home account; run the same command without the flag and confirm it acts against the root account. Launch the TUI with `--profile home` and confirm it shows the home account's data.

**Acceptance Scenarios**:

1. **Given** a config file with root settings and a `[profile:home]` block with different connection settings, **When** the user runs a command with `--profile home`, **Then** the command uses the `home` profile's settings.
2. **Given** the same config, **When** the user launches the interactive TUI with `--profile home`, **Then** the TUI uses the `home` profile's settings.
3. **Given** the same config, **When** the user runs a command with no `--profile` flag and no `TWIG_PROFILE` set, **Then** the command uses the root (default) settings exactly as it does today.
4. **Given** the same config and `TWIG_PROFILE=home` exported, **When** the user runs a command with no `--profile` flag, **Then** the command uses the `home` profile's settings.
5. **Given** `TWIG_PROFILE=home` exported, **When** the user runs a command with `--profile default`, **Then** the `--profile` flag wins and the root (default) settings are used.

---

### User Story 2 - Explicit default and backward compatibility (Priority: P2)

A person who has only ever used a single account, or who wants to be explicit, expects existing behavior to be untouched. `--profile default` selects the root settings, and a config file with no profile blocks at all keeps working exactly as before.

**Why this priority**: Protects every existing user from breakage and gives a clear, explicit way to name the default. It is essential for adoption but secondary to the core switching capability.

**Independent Test**: With a config file that has no profile blocks, run any command and the TUI and confirm behavior is identical to today. Then run the same command with `--profile default` and confirm it produces the same result as running with no flag.

**Acceptance Scenarios**:

1. **Given** a config file with no profile blocks, **When** the user runs any command or launches the TUI without `--profile`, **Then** behavior is identical to the pre-feature behavior.
2. **Given** any config file, **When** the user passes `--profile default`, **Then** the root settings are selected (equivalent to selecting the implicit default profile).

---

### User Story 3 - Clear precedence between flag, environment, and config (Priority: P2)

A person who sometimes overrides connection settings with environment variables needs predictable rules for which value wins. Environment variables override the credentials of whichever profile is selected — they take precedence over both the root and any named profile — so a value exported in the shell is always in effect regardless of the chosen profile. The profile chooses the baseline credentials; environment variables, when present, override them.

**Why this priority**: Ambiguous precedence leads to "why is it talking to the wrong account?" surprises, which undermines trust in the whole feature. It is closely tied to P1 but is its own testable concern.

**Independent Test**: Set environment variables for the connection settings. Run with an explicit `--profile home` and confirm the environment values are used (overriding the home profile's credentials). Then run with no profile flag and confirm the environment values override the root config.

**Acceptance Scenarios**:

1. **Given** connection environment variables are set **and** a `home` profile is configured, **When** the user runs with `--profile home`, **Then** the environment variable values override the `home` profile's credentials.
2. **Given** connection environment variables are set **and** no `--profile` flag is passed, **When** the user runs a command, **Then** the environment variables override the root config values.

---

### User Story 4 - Helpful error for an unknown profile (Priority: P3)

A person who mistypes a profile name, or names a profile that was never configured, gets an immediate, clear error rather than silently falling back to the wrong account.

**Why this priority**: Improves safety and usability but is a refinement on top of the core switching behavior. Without it the feature still works for correctly-spelled profiles.

**Independent Test**: Run with `--profile nonexistent` against a config that has no such profile and confirm the program stops with an error that names the missing profile and points at the config file.

**Acceptance Scenarios**:

1. **Given** a config file with no `bogus` profile, **When** the user runs with `--profile bogus`, **Then** the program exits with a non-zero status and an error that names the missing profile and the config file location, and does not contact any account.

---

### Edge Cases

- **`--profile` with no value** (e.g. trailing flag with nothing after it): the program reports a usage error rather than treating the next token or empty string as a profile name.
- **Profile block present but empty**: a named profile that sets no credentials of its own has no API key (so the existing "no API key found" error applies) and uses the built-in default API URL; it still uses the root-level non-credential settings (e.g. pomodoro hooks). It does NOT inherit the root credentials.
- **Profile sets only some credentials**: a profile that sets, say, only the API key still uses the built-in default API URL (not the root's API URL); root credentials are never inherited. Non-credential settings continue to come from the root.
- **A `default` profile is explicitly defined in the config**: the name `default` always refers to the root settings; selecting `--profile default` (or `TWIG_PROFILE=default`) uses the root settings.
- **`TWIG_PROFILE` set to an empty string**: treated as unset, i.e. the root/default profile is used.
- **`TWIG_PROFILE` names an unknown profile**: same actionable error as an unknown `--profile` value (names the missing profile and config location; no account is contacted).
- **Missing credentials after selection**: if the selected profile has no API key from any source (profile value or `TWIG_API_KEY`), the existing "no API key found" error applies and names the relevant config source.
- **Profile selected but the named account is unreachable**: normal connection-error handling applies; profile selection does not change how connection failures are reported.

## Requirements *(mandatory)*

### Functional Requirements

- **FR-001**: The config file MUST support one or more named profile sections in addition to the existing root-level settings.
- **FR-002**: The system MUST accept a `--profile <name>` option at launch that selects which profile's settings to use.
- **FR-002a**: The system MUST accept a `TWIG_PROFILE` environment variable that selects which profile to use when the `--profile` option is absent.
- **FR-002b**: When both `--profile` and `TWIG_PROFILE` are present, the `--profile` option MUST take precedence for profile selection. This selection axis is independent of the `TWIG_ADDR`/`TWIG_API_KEY` credential-override variables.
- **FR-003**: Profile selection (via `--profile` or `TWIG_PROFILE`) MUST apply to both non-interactive commands and the interactive TUI.
- **FR-004**: When `--profile <name>` names a configured profile, the system MUST use that profile's settings for the session.
- **FR-005**: The name `default` MUST refer to the root-level settings; `--profile default` MUST be equivalent to selecting the implicit default profile.
- **FR-006**: When neither the `--profile` option nor the `TWIG_PROFILE` environment variable is set, the system MUST use the root (default) settings, preserving today's behavior.
- **FR-007**: Credentials (API URL and API key) MUST be profile-scoped: when a named profile is selected, its own API URL and API key are used, and a credential the profile does not set MUST NOT fall back to the root-level credential (it falls back only to the built-in default, where one exists — e.g. the default API URL).
- **FR-008**: A profile MUST be able to set its own account connection settings (at minimum the API URL and API key).
- **FR-008a**: Non-credential settings (e.g. pomodoro lifecycle hooks) MUST always come from the root-level settings and MUST NOT be overridable per profile; selecting a profile changes only the credentials, not these shared settings.
- **FR-009**: Environment variable credential overrides (API URL / API key) MUST take precedence over the credentials of whichever profile is selected — named or default. The selected profile supplies the baseline credentials; a set environment variable overrides the corresponding value.
- **FR-010**: When no `--profile` option is given, environment variable overrides MUST take precedence over the root (implicit default) config values, preserving today's behavior.
- **FR-011**: When the selected profile name (whether from `--profile` or `TWIG_PROFILE`) is not present in the config, the system MUST stop with a non-zero exit/error that names the missing profile and the config file location, and MUST NOT contact any account.
- **FR-012**: When `--profile` is supplied without a value, the system MUST report a usage error.
- **FR-013**: Existing config files that contain no profile sections MUST continue to work without modification.
- **FR-014**: Selecting a profile MUST NOT require any change to how credentials are provisioned or how the server is operated.

### Key Entities *(include if feature involves data)*

- **Profile**: A named set of account credentials (API URL and API key). A profile carries only credentials; non-credential settings are not part of a profile. Credentials are not inherited from the root.
- **Root / default settings**: The settings defined at the top level of the config file. The root credentials are used when no profile is selected or when the `default` profile is selected. The root is also the sole source of non-credential settings (e.g. pomodoro hooks) for every session, regardless of which profile is selected.
- **Profile selection**: The launch-time choice of which profile to use, expressed via the `--profile` option or the `TWIG_PROFILE` environment variable (flag wins), defaulting to the root settings when both are absent.

## Success Criteria *(mandatory)*

### Measurable Outcomes

- **SC-001**: A user with two accounts can switch which account twig uses by changing a single launch option, with no edits to the config file between uses.
- **SC-002**: 100% of commands and the TUI honor the selected profile within a single launch (no command ignores the selection).
- **SC-003**: Existing single-account config files produce identical behavior before and after this feature (zero breaking changes for users who do not adopt profiles).
- **SC-004**: Selecting an unknown profile produces an actionable error (names the profile and config location) and never silently uses a different account.
- **SC-005**: Precedence between an explicit profile, environment variables, and the implicit default is unambiguous and matches the documented rules in 100% of the precedence test scenarios.

## Assumptions

- **Profile selection is launch-time only.** The profile is chosen once when twig starts (via `--profile`) and applies for the whole session; there is no in-session profile switching in the TUI.
- **Hybrid resolution (see Clarifications).** Credentials (API URL and API key) are profile-scoped with no fallback to root credentials; all other settings (e.g. pomodoro lifecycle hooks) always come from the root and are not overridable per profile. This keeps shared workflow settings in one place while isolating accounts.
- **Two independent selection inputs.** A profile is selected by the `--profile` option or the `TWIG_PROFILE` environment variable (flag wins over env var; neither set means the root/default profile). This profile-selection axis is separate from the credential-override variables (`TWIG_ADDR`/`TWIG_API_KEY`), which override the credentials of whichever profile is selected.
- **Profile section naming follows the user's requested form.** Profiles are written as named sections keyed by `profile:<name>` as shown in the request; the exact config syntax is finalized during planning against the config-file format already in use.
- **The `default` name is reserved** to mean the root settings; a literal profile named `default` is not a distinct profile from the root.
- **Scope is the client only.** This affects how the CLI/TUI choose connection settings; the server, provisioning, and stored data are unchanged. Multiple profiles may point at the same or different servers/accounts.
- **Reuses the existing config file and discovery path.** Profiles live in the same single config file the tool already reads; no new file or directory is introduced.
