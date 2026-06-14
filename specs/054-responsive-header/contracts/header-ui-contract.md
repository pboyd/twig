# UI Contract: Responsive App Header

This feature exposes no network/API contract (no backend, proto, or DB changes). The contract below specifies the **observable behavior** of the `AppHeader` component across viewport sizes — the interface between the app and its users — and is the reference for acceptance tests.

## Components

- `AppHeader` (modified) — the top navigation bar rendered on all authenticated pages.
- `HeaderMenu` (new) — the compact dropdown holding secondary actions on small viewports.

## Breakpoint

- **Small viewport**: width < 640px (Tailwind `sm` boundary). Compact layout.
- **Wide viewport**: width ≥ 640px. Full layout (current behavior).
- The transition is CSS-driven and requires no reload or remount.

## Behavioral contract

### Always (all viewport sizes)

- C1: The logo is visible and links to `/tasks`, with an accessible name identifying it as Twig/home.
- C2: **Tasks** and **Plan** are rendered directly in the header, are tappable, and show active styling when their route is current.
- C3: No header element is clipped or causes horizontal overflow at viewport widths ≥ 320px.

### Wide viewport (≥640px)

- C4: The "Twig" wordmark is visible next to the logo.
- C5: **Download**, **Account**, and **Sign out** are rendered directly in the header (no menu required), matching today's layout.
- C6: The compact menu trigger is not shown.

### Small viewport (<640px)

- C7: The "Twig" wordmark is hidden (logo remains).
- C8: **Download**, **Account**, and **Sign out** are not in the always-visible header; they live in the compact menu.
- C9: A menu trigger button is shown with an accessible name and `aria-haspopup="menu"` / `aria-expanded` reflecting state; it meets the 44px minimum touch target.

### Compact menu (`HeaderMenu`)

- C10: Activating the trigger toggles the menu open/closed; `aria-expanded` updates accordingly.
- C11: When open, the menu lists **Download**, **Account**, and **Sign out**, each operable by touch and keyboard.
- C12: Selecting **Account** or **Download** navigates to that route and closes the menu.
- C13: Selecting **Sign out** performs `POST /auth/logout` (credentials included), clears cached query data, navigates to `/login`, and closes the menu — identical to the existing Sign out behavior.
- C14: The menu closes on: item activation, outside pointer-down, `Escape` keypress, and route change.
- C15: When a menu destination matches the current route, that active state is discoverable while the menu is open.

## Out of scope

- No changes to routes, auth endpoints, or any proto/data contract.
- No new persisted preferences.
