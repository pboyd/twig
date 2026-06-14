---

description: "Task list for Responsive Header for Mobile"
---

# Tasks: Responsive Header for Mobile

**Input**: Design documents from `/specs/054-responsive-header/`

**Prerequisites**: plan.md, spec.md, research.md, data-model.md, contracts/header-ui-contract.md, quickstart.md

**Tests**: Included — the project has strong front-end test conventions and the plan/quickstart call for a dedicated `AppHeader.test.tsx`. jsdom cannot evaluate CSS media queries, so tests assert behavior and DOM/ARIA structure; pixel-level breakpoint behavior is verified manually per `quickstart.md`.

**Organization**: Tasks are grouped by user story (US1=P1, US2=P2, US3=P3). All work is confined to `services/twig-web/`.

## Format: `[ID] [P?] [Story] Description`

- **[P]**: Can run in parallel (different files, no dependencies)
- **[Story]**: Which user story this task belongs to (US1, US2, US3)
- Exact file paths included in each description

## Path Conventions

Front-end SPA: all paths under `services/twig-web/src/`. No backend, proto, sqlc, or migration changes.

---

## Phase 1: Setup (Shared Infrastructure)

**Purpose**: Shared copy and test scaffolding used across stories

- [X] T001 [P] Add compact-menu copy (menu trigger accessible label, e.g. a light-toned "Menu") to `services/twig-web/src/theme/messages.ts`, following the existing playful-but-clear tone (Constitution IV)
- [X] T002 Create test scaffold `services/twig-web/src/components/AppHeader.test.tsx` that renders `<AppHeader>` wrapped in a router (`MemoryRouter`/`createMemoryRouter`) and a `QueryClientProvider`, mocking `fetch` for logout — `AppHeader` uses `useNavigate` and `useQueryClient`, so these providers are required

---

## Phase 2: Foundational (Blocking Prerequisites)

**Purpose**: The responsive container restructure that every story builds on

**⚠️ CRITICAL**: Blocks all user stories — they all render into this structure

- [X] T003 Restructure `services/twig-web/src/components/AppHeader.tsx` into a responsive `sm:`-breakpoint (640px) layout: wrap the logo in a `<NavLink to="/tasks">` with an accessible name (home affordance), and set up the two presentation slots (direct items vs. compact menu) that later tasks fill. No item moves yet; this establishes the skeleton.

**Checkpoint**: Header skeleton renders at all sizes; user-story work can begin.

---

## Phase 3: User Story 1 - Reach primary navigation on a phone (Priority: P1) 🎯 MVP

**Goal**: On phone-width viewports, Tasks and Plan stay fully visible and tappable with no clipping; the "Twig" wordmark is hidden below the breakpoint.

**Independent Test**: On a phone-width viewport, Tasks and Plan are both visible/tappable, active styling tracks the route, and the header does not overflow horizontally (verified manually at 320px per quickstart).

### Tests for User Story 1

- [X] T004 [US1] In `services/twig-web/src/components/AppHeader.test.tsx`, assert Tasks and Plan render as links at all sizes and that the active destination receives active styling (write before implementation; expect fail)

### Implementation for User Story 1

- [X] T005 [US1] In `services/twig-web/src/components/AppHeader.tsx`, render Tasks and Plan as always-visible nav links (every breakpoint) reusing the existing `navLinkClass` active styling
- [X] T006 [US1] In `services/twig-web/src/components/AppHeader.tsx`, wrap the "Twig" wordmark in `hidden sm:inline` so it is hidden below 640px while the logo stays visible (per Clarifications: small screens only)

**Checkpoint**: Tasks/Plan reachable and unclipped on phones; wordmark hidden on small screens. MVP deliverable.

---

## Phase 4: User Story 2 - Reach secondary actions through a compact menu (Priority: P2)

**Goal**: On small viewports, Download, Account, and Sign out are reachable from an accessible compact dropdown menu.

**Independent Test**: On a phone-width viewport, open the menu and confirm Account, Download, and Sign out are present and functional; the menu opens/closes by touch and keyboard; Sign out logs the user out.

**Note**: T010 edits the same file as US3's T013 (`AppHeader.tsx` item slots) — coordinate; not parallel with US3.

### Implementation for User Story 2

- [X] T007 [US2] Create `services/twig-web/src/components/HeaderMenu.tsx`: a trigger `<button>` (`aria-haspopup="menu"`, `aria-expanded`, accessible label from `messages.ts`, 44px touch target) plus a `role="menu"` dropdown panel listing Download, Account (as links/`menuitem`) and Sign out (as a `menuitem` button), styled with shared theme/dark-mode classes
- [X] T008 [US2] In `services/twig-web/src/components/HeaderMenu.tsx`, implement close behavior: close on item activation, outside pointer-down, `Escape` keypress, and route change (per contract C14)
- [X] T009 [US2] Implement the Sign out action in `services/twig-web/src/components/HeaderMenu.tsx` (or lift a shared handler from `AppHeader.tsx`): `POST /auth/logout` with `credentials: "include"`, `queryClient.clear()`, then navigate to `/login` — identical to existing behavior (contract C13)
- [X] T010 [US2] In `services/twig-web/src/components/AppHeader.tsx`, mount `<HeaderMenu>` so it shows only below the breakpoint (e.g. `sm:hidden`) and remove Download/Account/Sign out from the always-visible small-screen header

### Tests for User Story 2

- [X] T011 [US2] In `services/twig-web/src/components/AppHeader.test.tsx`, assert the menu trigger is present and labeled, opening it reveals Account, Download, and Sign out, selecting an item closes the menu, and Sign out calls the logout fetch + navigates to `/login`
- [X] T012 [US2] In `services/twig-web/src/components/AppHeader.test.tsx`, assert the active state is reflected for a menu destination when its route is current and the menu is open (contract C15)

**Checkpoint**: All secondary actions reachable via the menu on small screens; US1 still works.

---

## Phase 5: User Story 3 - Keep the full header on wider screens (Priority: P3)

**Goal**: On wide viewports (≥640px), Tasks, Plan, Download, Account, and Sign out all remain directly visible — no regression from today.

**Independent Test**: On a wide viewport, all five items are directly visible/clickable and the compact menu trigger is not shown; resizing across the breakpoint produces a clean layout both ways.

**Note**: T013 edits the same `AppHeader.tsx` item slots as US2's T010 — they are the two halves of one responsive duplication; do sequentially.

### Implementation for User Story 3

- [X] T013 [US3] In `services/twig-web/src/components/AppHeader.tsx`, render Download, Account, and Sign out directly at `sm:` and up (e.g. `hidden sm:flex`) preserving the current layout, and ensure the `HeaderMenu` trigger is hidden at `sm:` and up

### Tests for User Story 3

- [X] T014 [US3] In `services/twig-web/src/components/AppHeader.test.tsx`, assert the direct Download/Account/Sign out elements exist in the wide-layout markup and that the menu trigger is gated to small screens (structure-level, since jsdom can't measure breakpoints)

**Checkpoint**: All three stories functional; full header preserved on wide screens.

---

## Phase 6: Polish & Cross-Cutting Concerns

**Purpose**: Verification and quality gates across stories

- [X] T015 [P] Run `npm test` and `npm run build` in `services/twig-web` and confirm the full suite passes with no type/build regressions
- [X] T016 Manual viewport verification per `specs/054-responsive-header/quickstart.md`: 320px shows no clipping/horizontal scroll, and resizing across 640px transitions cleanly in both directions
- [X] T017 [P] Review new menu copy in `services/twig-web/src/theme/messages.ts` for tone (Constitution IV) and confirm reuse of shared theme/touch-target conventions (Constitution III)

---

## Dependencies & Execution Order

### Phase Dependencies

- **Setup (Phase 1)**: No dependencies — start immediately. T001 and T002 are independent.
- **Foundational (Phase 2)**: Depends on Setup — BLOCKS all user stories.
- **User Stories (Phase 3–5)**: All depend on Foundational (T003). They share `AppHeader.tsx`, so run **sequentially in priority order** (US1 → US2 → US3) rather than in parallel.
- **Polish (Phase 6)**: Depends on all targeted stories being complete.

### User Story Dependencies

- **US1 (P1)**: After Foundational. Independent — the MVP.
- **US2 (P2)**: After Foundational. Adds `HeaderMenu.tsx` (new file) + edits `AppHeader.tsx` item slots.
- **US3 (P3)**: After Foundational. Edits the same `AppHeader.tsx` item slots as US2 (T010 ↔ T013) — coordinate; not independent at the file level.

### Within Each User Story

- Tests written before implementation where listed; verify they fail first.
- Foundational skeleton (T003) before any item placement.

### Parallel Opportunities

- T001 ‖ T002 (Setup, different files).
- T015 ‖ T017 (Polish, independent checks).
- Cross-story parallelism is limited: US1/US2/US3 all modify `AppHeader.tsx`, and most tests live in `AppHeader.test.tsx`, so those are sequential. `HeaderMenu.tsx` (T007–T009) is the main isolated new file.

---

## Parallel Example: Setup

```bash
# These two touch different files and can run together:
Task: "Add compact-menu copy to services/twig-web/src/theme/messages.ts"   # T001
Task: "Create AppHeader.test.tsx scaffold with router + query providers"     # T002
```

---

## Implementation Strategy

### MVP First (User Story 1 Only)

1. Phase 1 Setup → 2. Phase 2 Foundational (T003) → 3. Phase 3 US1.
4. **STOP and VALIDATE**: phones show Tasks/Plan with no clipping and no wordmark; wide screens unchanged.
5. Ship if desired — the overflow bug's core is fixed.

### Incremental Delivery

1. Setup + Foundational → skeleton ready.
2. US1 → primary nav fixed (MVP).
3. US2 → secondary actions reachable via menu.
4. US3 → confirm/preserve full wide-screen header.
5. Polish → tests, build, manual viewport pass, tone/consistency review.

---

## Notes

- [P] = different files, no dependency.
- Most tasks edit `AppHeader.tsx` / `AppHeader.test.tsx`, so honest parallelism is low; the dependency notes call out the file conflicts.
- Front-end only — no `make proto`, `sqlc`, or migrations.
- Commit after each task or logical group.
- Verify tests fail before implementing where TDD tasks are listed.
