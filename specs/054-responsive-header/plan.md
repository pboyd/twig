# Implementation Plan: Responsive Header for Mobile

**Branch**: `054-responsive-header` | **Date**: 2026-06-14 | **Spec**: [spec.md](./spec.md)

**Input**: Feature specification from `/specs/054-responsive-header/spec.md`

## Summary

The web app header (`AppHeader.tsx`) renders logo + "Twig" wordmark plus five always-visible items (Tasks, Plan, Download, Account, Sign out), which overflow and clip on phone-width viewports — the app's primary platform. This plan keeps **Tasks** and **Plan** always visible, hides the **"Twig"** wordmark below the breakpoint (logo retained), and collapses **Download**, **Account**, and **Sign out** into a compact dropdown menu on small viewports while preserving today's full header on wide viewports. The change is front-end only (no backend, no proto/contract changes) and is delivered via Tailwind responsive utilities plus one small accessible dropdown component.

## Technical Context

**Language/Version**: TypeScript 5.8, React 19

**Primary Dependencies**: React Router 7, Tailwind CSS v4 (`@tailwindcss/vite`), TanStack Query 5. No new dependencies.

**Storage**: N/A — no persisted state. The menu open/closed flag is ephemeral component state.

**Testing**: Vitest 3 + Testing Library (`@testing-library/react`, `jest-dom`), jsdom environment.

**Target Platform**: Mobile-first web (phones primary), also tablet/desktop browsers.

**Project Type**: Web SPA front-end (`services/twig-web/`).

**Performance Goals**: No measurable runtime cost; layout switches via CSS at the breakpoint with no reflow jank. No added network calls.

**Constraints**: Must fit viewports down to 320px wide with no horizontal clipping/scroll; 44px minimum touch targets (already enforced by `Button`); keyboard- and screen-reader-operable menu.

**Scale/Scope**: One component file (`AppHeader.tsx`), one new small component (`HeaderMenu.tsx`), one new test file, minor `messages.ts` additions. ~5 destinations/actions total; no new routes.

## Constitution Check

*GATE: Must pass before Phase 0 research. Re-check after Phase 1 design.*

| Principle | Status | Notes |
|-----------|--------|-------|
| I. Simplicity / YAGNI | ✅ | Tailwind responsive utilities + one minimal dropdown built from existing primitives. No new dependencies, no abstraction layers, no library for the menu. |
| II. API-First Design | ✅ | No backend, proto, or data-contract changes. The relevant contract is a UI behavior contract, documented in `contracts/header-ui-contract.md`. |
| III. UI/UX Consistency | ✅ | Reuses existing nav-link styling, the shared `Button`, theme tokens, dark-mode classes, and the 44px touch-target convention. No ad-hoc colors. |
| IV. Playful User Messages | ✅ | Minimal new copy (a menu accessible label) routed through `theme/messages.ts`; tone kept light and consistent. No dry one-off strings introduced inline. |

Post-Phase 1 re-check: **PASS** — design introduces no new dependencies, no backend contract, and reuses shared UI primitives. Complexity Tracking table remains empty.

## Project Structure

### Documentation (this feature)

```text
specs/054-responsive-header/
├── plan.md              # This file
├── research.md          # Phase 0 output
├── data-model.md        # Phase 1 output
├── quickstart.md        # Phase 1 output
├── contracts/
│   └── header-ui-contract.md   # Phase 1 output (UI behavior contract)
├── checklists/
│   └── requirements.md  # From /speckit-specify
└── tasks.md             # Phase 2 output (/speckit-tasks — NOT created here)
```

### Source Code (repository root)

```text
services/twig-web/
├── src/
│   ├── components/
│   │   ├── AppHeader.tsx          # MODIFY: responsive layout; wordmark hidden < breakpoint;
│   │   │                          #         secondary items move into HeaderMenu on small screens
│   │   ├── AppHeader.test.tsx     # NEW: responsive behavior + menu interaction tests
│   │   ├── HeaderMenu.tsx         # NEW: small accessible dropdown (Download, Account, Sign out)
│   │   └── Button.tsx             # REUSE (unchanged) — Sign out action / menu trigger styling
│   └── theme/
│       └── messages.ts           # MODIFY: add menu label(s)
└── (no backend / proto / db changes)
```

**Structure Decision**: This is a front-end-only change confined to `services/twig-web/src/components/`. The existing single-SPA structure is used as-is. `AppHeader.tsx` is modified to drive a responsive layout via Tailwind's `sm:` breakpoint; the secondary actions are extracted into a new `HeaderMenu.tsx` so the dropdown logic (toggle, outside-click/Escape close, ARIA) stays self-contained and `AppHeader` stays readable. A new `AppHeader.test.tsx` covers behavior; every page test already mocks `AppHeader`, so page tests are unaffected.

## Complexity Tracking

> No Constitution Check violations. No entries required.
