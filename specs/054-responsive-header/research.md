# Phase 0 Research: Responsive Header for Mobile

No `NEEDS CLARIFICATION` items remained after `/speckit-clarify`. The decisions below resolve the implementation choices the spec left open as planning-level details.

## Decision 1: Breakpoint between compact and full header

- **Decision**: Use Tailwind's `sm` breakpoint (640px). Below 640px → compact layout (wordmark hidden, secondary items in menu). At ≥640px → full header (current behavior).
- **Rationale**: Common phones are <640px CSS-wide (e.g., iPhone ~390px, most Androids 360–430px), so they reliably get the compact layout, while tablets/desktops (≥640px, typically 768px+ in portrait) keep the full header. `sm` is Tailwind's default and needs no config. Logo + Tasks + Plan + a menu button fit comfortably even at 320px.
- **Alternatives considered**:
  - `md` (768px): would push tablet-portrait into the compact layout unnecessarily — they have room for the full header.
  - Custom breakpoint / container queries: more machinery than warranted (YAGNI); no other component needs it and the app currently uses no breakpoints.

## Decision 2: Compact menu implementation

- **Decision**: Build a small custom dropdown (`HeaderMenu.tsx`) using a toggle button + an absolutely-positioned panel of links/actions. No third-party menu/popover library.
- **Rationale**: Constitution Principle I (Simplicity). The menu has ~3 items and simple needs (open/close, close on select/outside/Escape). A custom component avoids a new dependency and matches the project's existing hand-rolled component style (`Toast`, `AddToPlanControl`).
- **Alternatives considered**:
  - Headless UI / Radix: adds a dependency and bundle weight for a trivial menu — rejected.
  - CSS-only `<details>`/`:focus-within`: harder to get correct outside-click and keyboard semantics; less controllable — rejected.

## Decision 3: Accessibility approach for the menu

- **Decision**: Trigger is a `<button>` with `aria-haspopup="menu"`, `aria-expanded`, and an accessible name (e.g., "Menu"). The panel uses `role="menu"` with `role="menuitem"` entries (links rendered as menu items; Sign out as a button). Close on `Escape`, on outside pointer-down, and on item activation. Trigger meets the 44px touch target.
- **Rationale**: Satisfies FR-012 (operable by touch, keyboard, and assistive tech with an accessible name) and the constitution's UI/UX consistency without over-engineering a full roving-tabindex menu — links remain natural tab stops, which is acceptable and simpler.
- **Alternatives considered**: Full WAI-ARIA menu with arrow-key roving focus — heavier than needed for 3 links; deferred unless a future need arises.

## Decision 4: Wordmark and logo affordance

- **Decision**: Wrap the "Twig" text in `hidden sm:inline` so it shows only at ≥640px. The logo remains visible at all sizes and is wrapped in a link to `/tasks` (home affordance) with an accessible name.
- **Rationale**: Confirmed in Clarifications (wordmark hidden on small screens only). Keeping the logo as a labeled home link preserves the brand/home affordance noted in the spec edge cases.
- **Alternatives considered**: Hiding the logo too on the smallest screens — rejected; the logo is the brand anchor and costs little space.

## Decision 5: Testing strategy

- **Decision**: Add `AppHeader.test.tsx` (Vitest + Testing Library). Assert: Tasks/Plan always rendered; the menu trigger exists and is labeled; opening it reveals Account, Download, and Sign out; selecting an item closes the menu; Sign out calls the logout flow and navigates to `/login`. Verify the responsive intent via the presence of the `sm:`-gated structure (both the menu trigger and the inline secondary items are rendered in the DOM; CSS controls visibility), since jsdom does not evaluate media queries.
- **Rationale**: Matches existing test conventions (every page already mocks `AppHeader`; this is the first real test of it). jsdom can't measure layout, so tests target behavior and DOM/ARIA structure rather than pixel widths.
- **Alternatives considered**: Visual/viewport regression testing — no such infra exists in the repo; out of scope.
