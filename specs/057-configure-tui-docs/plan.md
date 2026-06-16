# Implementation Plan: TUI Configuration Instructions on Download Page

**Branch**: `057-configure-tui-docs` | **Date**: 2026-06-16 | **Spec**: [spec.md](./spec.md)

**Input**: Feature specification from `/specs/057-configure-tui-docs/spec.md`

## Summary

Add on-page instructions to the web app's existing Download page that teach a new user how to configure and launch the interactive TUI after downloading the binary. The instructions cover the two required settings (server address + access credential), point users to the in-app Account page to mint an API key, show the server address dynamically from the web app's own origin, present both a quick (env-var) and persistent (config file) setup path with precedence noted, and give the exact launch command. This is a frontend-only change: new copy in `messages.ts` plus rendering and a link in `DownloadPage.tsx`. No backend, proto, or CLI changes.

## Technical Context

**Language/Version**: TypeScript 5.x, React 19

**Primary Dependencies**: React Router (`react-router`), existing shared components (`Button`, `AppHeader`), `react-router` `Link` for in-app navigation. No new dependencies.

**Storage**: N/A (no persisted state introduced)

**Testing**: Vitest + Testing Library (`services/twig-web/src/pages/DownloadPage.test.tsx`)

**Target Platform**: Web SPA (`services/twig-web/`), served same-origin with the API server

**Project Type**: Web frontend (additive, single page touched)

**Performance Goals**: N/A — static copy; no new network calls. Server address is read synchronously from `window.location.origin`.

**Constraints**: Must stay same-origin (no cross-origin assumptions); copy must follow the warm/playful tone and live in `src/theme/messages.ts`; commands/snippets must be copy-paste-clean (rendered in code blocks).

**Scale/Scope**: One page (`DownloadPage.tsx`), one message group added to `messages.ts`, accompanying tests. Estimated <120 lines of changes.

## Constitution Check

*GATE: Must pass before Phase 0 research. Re-check after Phase 1 design.*

| Principle | Status | Notes |
|-----------|--------|-------|
| I. Simplicity / YAGNI | ✅ | Pure additive copy + rendering on an existing page. No new abstractions, components, or endpoints. Server address derived from `window.location.origin` rather than a new metadata field. |
| II. API-First Design | ✅ | No new contracts/endpoints. Feature consumes only the existing same-origin relationship and the existing `/account` route. UI contract for the new page section documented in `contracts/`. |
| III. UI/UX Consistency | ✅ | Reuses existing page layout, `Button`, card styling, dark-mode token classes, and `react-router` `Link` for the Account link. All copy centralized in `messages.ts`. |
| IV. Playful User Messages | ✅ | New copy authored in the established warm/playful tone, consistent with existing `download*` and `account.*` messages. |

No violations — Complexity Tracking table omitted.

## Project Structure

### Documentation (this feature)

```text
specs/057-configure-tui-docs/
├── plan.md              # This file
├── research.md          # Phase 0 output
├── data-model.md        # Phase 1 output
├── quickstart.md        # Phase 1 output
├── contracts/
│   └── download-page-config-section.md   # UI contract for the new section
└── tasks.md             # Phase 2 output (/speckit-tasks — not created here)
```

### Source Code (repository root)

```text
services/twig-web/
├── src/
│   ├── pages/
│   │   ├── DownloadPage.tsx          # MODIFIED: render config instructions + Account link
│   │   └── DownloadPage.test.tsx     # MODIFIED: assert config copy, origin, /account link, launch cmd
│   └── theme/
│       └── messages.ts               # MODIFIED: add downloadConfig.* message group
```

**Structure Decision**: Web frontend, additive. Only the existing `DownloadPage` and the centralized `messages.ts` are touched, plus the page's test file. No changes to the server module, `api/` protos, or the CLI/TUI. This keeps the change inside the established web-app conventions described in CLAUDE.md (centralized copy in `theme/messages.ts`, shared components, same-origin proxy).

## Complexity Tracking

> No Constitution Check violations — section intentionally empty.
