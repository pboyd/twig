# Quickstart: Responsive Header for Mobile

## Scope

Front-end only, in `services/twig-web/`. No backend, proto, or database changes — no `make proto`, `sqlc`, or migrations needed.

## Develop

```bash
cd services/twig-web
npm install          # if not already
npm run dev          # http://localhost:5173 (proxies API to :8080)
```

To see the responsive behavior, open the app and either:
- resize the browser window across the 640px boundary, or
- use browser devtools device emulation (e.g., iPhone ~390px) to view the compact layout.

You'll need a running server (`make dev` from repo root) and to be logged in to see the authenticated header.

## Verify the change

| Check | Expected |
|-------|----------|
| Wide window (≥640px) | Logo + "Twig" + Tasks, Plan, Download, Account, Sign out all directly visible (today's layout). |
| Narrow window (<640px) | Logo only (no "Twig" text), Tasks + Plan visible, a menu button present; Download/Account/Sign out moved into the menu. |
| Open the menu | Account, Download, Sign out listed; tap one → it acts and the menu closes. |
| Sign out from menu | Logged out, returned to `/login`. |
| Keyboard | Menu trigger is focusable; `Escape` closes the open menu. |
| 320px width | No horizontal scroll or clipped items. |

## Test

```bash
cd services/twig-web
npm test                       # full suite
npm test AppHeader             # the new header tests
```

Note: jsdom does not evaluate CSS media queries, so tests assert behavior and DOM/ARIA structure (presence of the menu trigger and secondary items, open/close, sign-out flow) rather than measured pixel layout. Visual breakpoint behavior is verified manually per the table above.

## Build

```bash
cd services/twig-web
npm run build                  # tsc -b && vite build
```
