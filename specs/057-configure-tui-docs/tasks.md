---
description: "Task list for TUI Configuration Instructions on Download Page"
---

# Tasks: TUI Configuration Instructions on Download Page

**Input**: Design documents from `/specs/057-configure-tui-docs/`

**Prerequisites**: plan.md, spec.md, research.md, data-model.md, contracts/download-page-config-section.md

**Tests**: Included — the UI contract (`contracts/download-page-config-section.md`) defines explicit test expectations and the page already has `DownloadPage.test.tsx`.

**Organization**: Tasks grouped by user story. Note: both stories edit the same two files (`src/theme/messages.ts`, `src/pages/DownloadPage.tsx`), so they are sequential, not parallel, across stories.

## Format: `[ID] [P?] [Story] Description`

- **[P]**: Can run in parallel (different files, no dependencies)
- **[Story]**: Which user story this task belongs to (US1, US2)

## Path Conventions

Web frontend at `services/twig-web/`. All paths below are relative to repo root.

---

## Phase 1: Setup (Shared Infrastructure)

**Purpose**: Confirm the authoritative configuration identifiers before writing user-facing copy.

- [ ] T001 Confirm the authoritative env-var and config-key names to use in copy — `TWIG_API_KEY`, `TWIG_ADDR`, config path `~/.config/twig/config.toml`, and TOML keys `api_url`/`api_key` — by checking `CLAUDE.md` and `specs/015-pomodoro-config-file/contracts/config-schema.md`; note any discrepancy in `specs/057-configure-tui-docs/research.md` (the live `TWIG_`-prefixed names are authoritative over the legacy `TODO_` names).

---

## Phase 2: Foundational (Blocking Prerequisites)

**Purpose**: Create the shared `downloadConfig` message group that both stories render from.

**⚠️ CRITICAL**: No user story rendering work can begin until this exists.

- [ ] T002 Add a `downloadConfig` message group to `services/twig-web/src/theme/messages.ts` with the shared keys: section `configHeading`, `configIntro` (states the two required settings), `serverAddressLabel`, `credentialStep`, and `credentialLinkLabel` (link text for the Account page). Author in the warm/playful tone consistent with existing `download*` and `account.*` copy.

**Checkpoint**: Shared copy scaffold ready — user story phases can begin.

---

## Phase 3: User Story 1 - Configure and launch the TUI for the first time (Priority: P1) 🎯 MVP

**Goal**: A signed-in user on `/download` sees instructions that name the two required settings, show the server address (current origin), link to `/account` to create an API key, give an env-var quick-start command, and show the `./twig` launch command — enough to get a connected TUI.

**Independent Test**: Render `/download` with a stubbed `window.location.origin`; verify the origin, the `/account` link, the env-var command (`TWIG_API_KEY`/`TWIG_ADDR`), and the `./twig` launch command all appear (contract items C1–C5, C8, C9).

### Tests for User Story 1

> Write these first and confirm they FAIL before implementation.

- [ ] T003 [US1] Extend `services/twig-web/src/pages/DownloadPage.test.tsx` with assertions: rendered server address equals a stubbed `window.location.origin` (C3); a link resolving to `/account` is present (C4); the env-var command text contains `TWIG_API_KEY` and `TWIG_ADDR` (C5); the `./twig` launch command is present (C8).

### Implementation for User Story 1

- [ ] T004 [US1] Add the US1 strings to the `downloadConfig` group in `services/twig-web/src/theme/messages.ts`: `quickStartHeading`, `quickStartCommand(origin: string)` (env-var one-liner with `TWIG_API_KEY`/`TWIG_ADDR=<origin>`), `launchHeading`, and `launchCommand` (`./twig`, noting it opens the interactive TUI). (depends on T002)
- [ ] T005 [US1] Render the configuration section in `services/twig-web/src/pages/DownloadPage.tsx` after the existing download card / `downloadPostHint`: heading + intro, the credential step with a `react-router` `Link` to `/account`, the server address read from `window.location.origin`, the env-var quick-start in a code block, and the launch command in a code block. Use only `messages.downloadConfig.*` strings. (depends on T002, T004)
- [ ] T006 [US1] Ensure the new section reuses the page's existing card styling, spacing, and dark-mode token classes, and that all commands render inside copy-paste-friendly `<pre>`/`<code>` blocks (C9) in `services/twig-web/src/pages/DownloadPage.tsx`. (depends on T005)

**Checkpoint**: US1 fully functional — a first-time user can configure via env vars and launch the TUI from on-page instructions. MVP complete.

---

## Phase 4: User Story 2 - Choose between a quick start and a persistent setup (Priority: P2)

**Goal**: Add a persistent setup path (config file at `~/.config/twig/config.toml`) alongside the quick start, and state that environment variables take precedence.

**Independent Test**: Render `/download`; verify the config-file path and TOML snippet appear (C6) and that the precedence note (env vars win) is present (C7), in addition to the US1 quick start.

### Tests for User Story 2

- [ ] T007 [US2] Extend `services/twig-web/src/pages/DownloadPage.test.tsx` with assertions: the config-file path `~/.config/twig/config.toml` is shown (C6); the precedence note stating environment variables take precedence is present (C7).

### Implementation for User Story 2

- [ ] T008 [US2] Add the US2 strings to the `downloadConfig` group in `services/twig-web/src/theme/messages.ts`: `persistHeading`, `persistIntro` (mentions `~/.config/twig/config.toml`), `persistFileContents(origin: string)` (TOML with `api_url`/`api_key`), and `precedenceNote` (env vars win over the file). (depends on T002)
- [ ] T009 [US2] Render the persistent-setup block in `services/twig-web/src/pages/DownloadPage.tsx`: the config-file path, the TOML snippet in a code block (using the current origin for `api_url`), and the precedence note. Place it after the US1 quick-start within the same section. (depends on T005, T008)

**Checkpoint**: Both US1 and US2 work — quick start and persistent setup documented with precedence stated.

---

## Phase 5: Polish & Cross-Cutting Concerns

**Purpose**: Verify tone, tests, and build.

- [ ] T010 [P] Review all new `downloadConfig` copy in `services/twig-web/src/theme/messages.ts` for the warm/playful tone (Constitution Principle IV) and clarity/accuracy.
- [ ] T011 Run `npm test`, `npm run build`, and `npm run dev` from `services/twig-web/`, then walk the `specs/057-configure-tui-docs/quickstart.md` manual verification steps.

---

## Dependencies & Execution Order

### Phase Dependencies

- **Setup (Phase 1)**: No dependencies — start immediately.
- **Foundational (Phase 2 / T002)**: Depends on T001 — BLOCKS both stories.
- **User Story 1 (Phase 3)**: Depends on T002. Delivers the MVP.
- **User Story 2 (Phase 4)**: Depends on T002 and on US1's rendered section (T005) since it extends the same section/files.
- **Polish (Phase 5)**: Depends on all desired stories complete.

### Within Each User Story

- Test task (T003 / T007) written and failing before implementation.
- Messages (T004 / T008) before rendering (T005 / T009).

### Parallel Opportunities

- Cross-story parallelism is **not** available: US1 and US2 both edit `messages.ts` and `DownloadPage.tsx`.
- T010 is [P] (read-only review of a file no longer being edited once stories are done).
- T003 (test file) and T004 (messages file) touch different files and could overlap, but T003 should be authored to fail first per TDD.

---

## Implementation Strategy

### MVP First (User Story 1 Only)

1. Phase 1: Setup (T001)
2. Phase 2: Foundational (T002)
3. Phase 3: User Story 1 (T003–T006)
4. **STOP and VALIDATE**: A first-time user can configure via env vars and launch the TUI from `/download`.
5. Ship.

### Incremental Delivery

1. Setup + Foundational → shared copy ready.
2. US1 → first-run quick start + launch (MVP).
3. US2 → persistent config-file path + precedence.
4. Polish → tone review, tests, build, manual verification.

---

## Notes

- [P] tasks = different files, no dependencies.
- [Story] label maps each task to a user story for traceability.
- This is a frontend-only, additive change: no backend, proto, or CLI work.
- Commit after each task or logical group.
- Authoritative identifiers: `TWIG_API_KEY`, `TWIG_ADDR`, `~/.config/twig/config.toml` (keys `api_url`/`api_key`).
