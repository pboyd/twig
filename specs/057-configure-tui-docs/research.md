# Phase 0 Research: TUI Configuration Instructions on Download Page

No `NEEDS CLARIFICATION` markers remained after `/speckit-clarify`. This document records the decisions that shape the implementation.

## Decision 1: Source of the server address

**Decision**: Display the web app's own current origin via `window.location.origin`, rendered into the instructions at runtime.

**Rationale**: The web app is served from the same origin as the API server (CLAUDE.md: the Vite dev proxy and `SameSite=Strict` cookie design require same-origin; the SPA never calls `:8080` cross-origin). Therefore the browser's current origin is exactly the address the CLI's `TWIG_ADDR` must point at. Reading it dynamically is correct across localhost, staging, and production with zero per-deployment maintenance. Confirmed in clarification Q1 → A.

**Alternatives considered**:
- Static `http://localhost:8080` example — would be wrong on any real deployment and require users to guess/substitute.
- New server metadata endpoint (extend `/cli/info`) — unnecessary backend work; the origin is already known client-side. Rejected per Simplicity/YAGNI.

## Decision 2: Where users obtain an access credential

**Decision**: Link to the in-app Account page (`/account`) using `react-router` `Link`, and instruct the user to create an API key there.

**Rationale**: `AccountPage` already provides API-key management (`CreateApiKey` mints and reveals the secret once, plus list/revoke). It is the canonical, self-service place to get a key. Confirmed in clarification Q2 → A. Existing copy (`messages.account.apiKeysHeading`, `keySecretWarning`, `noKeys` = "create one to use the CLI") already frames keys as the CLI credential, so the flow is coherent.

**Alternatives considered**:
- Describe server-side provisioning generically — not self-service; ignores the existing Account UI.
- Assume the user already has a key — leaves first-time users stranded, defeating the feature's purpose.

## Decision 3: Configuration methods to document

**Decision**: Document two paths and state precedence:
1. **Quick start (env vars)** — `TWIG_API_KEY` and `TWIG_ADDR`, e.g. `TWIG_API_KEY=... TWIG_ADDR=<origin> ./twig`.
2. **Persistent (config file)** — a TOML file at `~/.config/twig/config.toml` with `api_key` and `api_url` keys.

Note that **environment variables take precedence** over the config file.

**Rationale**: Matches the documented config model (specs/015 config-schema and CLAUDE.md). Env vars are the lowest-friction way to try the tool immediately; the config file persists across sessions. Precedence (env wins) is part of the existing contract and must be stated to avoid confusion (FR-007). The TUI launches when `twig` runs with no arguments (CLAUDE.md), so the launch command is simply `./twig`.

**Alternatives considered**:
- Only env vars — fails FR-006 (persistence).
- Only config file — higher friction for a first run; fails FR-005.

**Note on key naming**: The current CLI/config uses the `TWIG_`-prefixed env vars and `api_url`/`api_key` TOML keys. The generic schema doc in specs/015 uses `TODO_`-prefixed names from an earlier era; the live binary's names (`TWIG_API_KEY`, `TWIG_ADDR`) are authoritative and will be used in the copy. (Implementation should verify against the running binary's documented env vars in CLAUDE.md before finalizing copy.)

## Decision 4: Placement and presentation on the page

**Decision**: Render the instructions as an additional section on the existing `DownloadPage`, immediately after the current download card / `downloadPostHint`, reusing existing card styling, dark-mode token classes, and code-block formatting. All strings live in a new `downloadConfig` group in `messages.ts`.

**Rationale**: FR-001/FR-010 require the instructions to live with the existing download guidance and be visible to a signed-in user without extra navigation. Reusing the established layout satisfies Principle III (UI/UX Consistency). Centralizing copy satisfies Principle IV and the project convention that all user-facing text lives in `theme/messages.ts`.

**Alternatives considered**:
- Separate route/page — extra navigation, violates FR-010 intent and adds complexity.
- Collapsible/accordion — acceptable but adds interaction state for no clear benefit at this scope; deferred unless copy length proves unwieldy.
