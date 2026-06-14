# Feature Specification: Responsive Header for Mobile

**Feature Branch**: `054-responsive-header`

**Created**: 2026-06-14

**Status**: Draft

**Input**: User description: "the header in the web app has so many items now that it's cut off on mobile (the primary platform for this app). 'Download' isn't useful on mobile, 'Account' doesn't need prime placement, 'Tasks' and 'Plan' need easy access, 'Sign Out' could move, and the icon is probably enough without 'Twig' next to it. Hiding 'Download', 'Account' and 'Sign Out' in a hamburger menu seems like the obvious choice."

## Clarifications

### Session 2026-06-14

- Q: On phone-width viewports, how should the Download item behave? → A: Keep it in the compact menu (reachable in two taps), not removed entirely.
- Q: Where should the "Twig" wordmark text next to the logo be hidden? → A: On small screens only; the wordmark returns on wide viewports.

## User Scenarios & Testing *(mandatory)*

### User Story 1 - Reach primary navigation on a phone (Priority: P1)

As a person using the web app on a phone (the primary platform), I can see and tap the two navigation destinations I use constantly — Tasks and Plan — without any item being clipped off the edge of the screen.

**Why this priority**: The reported bug is that the header overflows on mobile, hiding and clipping items. Restoring reliable access to the two most-used destinations on the primary platform is the core value of this feature; everything else is secondary.

**Independent Test**: Load the app on a narrow (phone-width) viewport and confirm that Tasks and Plan are both fully visible, tappable, and that the active destination is visually distinguishable — with no horizontal clipping or overflow of the header.

**Acceptance Scenarios**:

1. **Given** the app is open on a phone-width viewport, **When** the header renders, **Then** Tasks and Plan are both fully visible and tappable with no part of the header cut off horizontally.
2. **Given** the user is on the Tasks destination, **When** they view the header, **Then** Tasks is shown as the active destination.
3. **Given** the user taps Plan, **When** navigation completes, **Then** Plan becomes the active destination and the Plan view loads.

---

### User Story 2 - Reach secondary actions through a compact menu on a phone (Priority: P2)

As a person on a phone, I can still reach the less-frequent actions — Account, Sign out, and (where relevant) Download — through a compact menu, so they remain available without crowding the header.

**Why this priority**: These actions are important but infrequent. They must remain reachable, but moving them out of the always-visible header is what creates the room needed to fix the overflow. This depends on a place to put them, so it follows P1.

**Independent Test**: On a phone-width viewport, open the compact menu from the header and confirm Account and Sign out are present and functional, and that the menu can be opened and dismissed by touch.

**Acceptance Scenarios**:

1. **Given** the app is open on a phone-width viewport, **When** the user opens the compact menu, **Then** Account and Sign out are listed and selectable.
2. **Given** the compact menu is open, **When** the user selects Account, **Then** the Account view loads and the menu closes.
3. **Given** the compact menu is open, **When** the user selects Sign out, **Then** the user is signed out and returned to the login view.
4. **Given** the compact menu is open, **When** the user taps outside the menu or selects an item, **Then** the menu closes.

---

### User Story 3 - Keep the full header on wider screens (Priority: P3)

As a person using the app on a tablet or desktop, where there is room, I continue to see the full set of header items directly without needing to open a menu.

**Why this priority**: Wider screens do not have the overflow problem, so collapsing items there would be a regression in convenience. Preserving the existing experience on large screens protects current desktop/tablet users while the mobile experience is improved.

**Independent Test**: Load the app on a wide (desktop-width) viewport and confirm all destinations and actions remain directly visible and usable, matching today's behavior.

**Acceptance Scenarios**:

1. **Given** the app is open on a wide viewport, **When** the header renders, **Then** Tasks, Plan, Download, and Account are directly visible along with the Sign out action.
2. **Given** the viewport is resized from wide to narrow, **When** it crosses the breakpoint, **Then** the header transitions to the compact mobile layout without clipping or broken layout, and back again when widened.

---

### Edge Cases

- **Very narrow viewports** (e.g., small phones ~320px wide): the header must still fit — logo plus the primary destinations plus the menu control — with no horizontal clipping or scrolling.
- **Menu open during navigation**: if the compact menu is open and the user navigates (via menu item, browser back, or deep link), the menu closes rather than persisting over the new view.
- **Brand affordance**: with the "Twig" wordmark hidden on small screens, the logo alone must still clearly read as the app/home affordance.
- **Active destination inside the menu**: if a destination that lives in the compact menu corresponds to the current view, that state should be discoverable (e.g., reflected when the menu is open) rather than misleading the user about where they are.
- **Download on mobile**: Download leads to a desktop-only CLI/TUI binary that does not run on mobile; the design must avoid leading mobile users to an action that cannot help them while still keeping it reachable for those who want it.

## Requirements *(mandatory)*

### Functional Requirements

- **FR-001**: The header MUST fit within the available width on phone-sized viewports with no horizontal clipping or overflow of any item.
- **FR-002**: Tasks and Plan MUST remain directly visible and selectable in the header on all supported viewport sizes, including phones.
- **FR-003**: The currently active destination MUST be visually distinguishable on all viewport sizes.
- **FR-004**: On small viewports, Account and Sign out MUST be relocated from the always-visible header into a compact menu that the user can open and dismiss by touch.
- **FR-005**: On small viewports, Download MUST NOT occupy always-visible header space; it MUST instead remain reachable from the compact menu rather than being removed entirely.
- **FR-006**: All actions and destinations available before this change (Tasks, Plan, Download, Account, Sign out) MUST remain reachable on every supported viewport size after the change.
- **FR-007**: On wide viewports, the header MUST continue to present Tasks, Plan, Download, Account, and Sign out directly, preserving the current experience.
- **FR-008**: The header MUST adapt automatically to viewport width, switching between the full and compact layouts at a defined breakpoint without requiring a page reload.
- **FR-009**: The "Twig" wordmark MAY be hidden on small viewports to save space; when hidden, the logo alone MUST remain as a recognizable brand/home affordance.
- **FR-010**: The compact menu MUST close when the user selects an item, navigates away, or dismisses it (e.g., taps outside).
- **FR-011**: Selecting Sign out from the compact menu MUST perform the same sign-out behavior as the existing Sign out control (ending the session and returning the user to login).
- **FR-012**: The compact menu control MUST be operable by touch and reachable via keyboard/assistive technology, with an accessible name indicating it opens a menu.

### Key Entities

*Not applicable — this feature changes presentation and navigation layout only; it introduces no new persisted data.*

## Success Criteria *(mandatory)*

### Measurable Outcomes

- **SC-001**: On viewports as narrow as 320px wide, no header item is clipped or pushed off-screen, and the header introduces no horizontal scrolling.
- **SC-002**: On a phone-width viewport, a user can reach Tasks and Plan in a single tap each, with no intermediate menu.
- **SC-003**: On a phone-width viewport, a user can reach Account, Sign out, and Download in at most two taps (open menu, then select).
- **SC-004**: 100% of the actions/destinations available before the change remain reachable on phone-width viewports after the change.
- **SC-005**: On wide viewports, all header items remain directly accessible in a single tap/click, with no regression from the prior layout.
- **SC-006**: Resizing the browser across the breakpoint produces a clean layout in both directions with no clipped, overlapping, or broken header elements.

## Assumptions

- **Primary platform is mobile**: per the user, the web app is used primarily on phones, so the mobile layout is the priority and the desktop layout must merely be preserved.
- **Breakpoint-driven layout**: a single width breakpoint distinguishes "small" (compact menu) from "wide" (full header). The exact pixel value is an implementation detail to be chosen during planning to comfortably fit logo + Tasks + Plan + menu control on common phone widths.
- **Compact menu pattern**: the secondary items are collapsed into a single menu opened from a control in the header (the "hamburger menu" the user proposed). The user stated they are open to alternatives, but this spec assumes the menu approach.
- **Download stays reachable via the menu rather than fully removed**: confirmed in Clarifications (2026-06-14). Download is kept in the compact menu on small viewports (reachable, not prominent) rather than dropped entirely, so the binary stays discoverable for users on larger phones/tablets while never crowding the header.
- **Wordmark hidden on small screens only**: confirmed in Clarifications (2026-06-14). The "Twig" text next to the logo is hidden on small viewports (logo retained) and returns on wide viewports.
- **No backend changes**: this is a front-end presentation change only; existing auth and navigation endpoints/behavior are reused unchanged.
- **No new destinations**: the set of destinations/actions is exactly today's set; this feature only changes how they are arranged across viewport sizes.
