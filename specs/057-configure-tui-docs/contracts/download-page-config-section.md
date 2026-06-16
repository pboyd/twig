# UI Contract: Download Page — TUI Configuration Section

**Surface**: `services/twig-web/src/pages/DownloadPage.tsx`
**Type**: Frontend UI contract (no network/API contract — feature adds no endpoints).

This contract defines what the new configuration section MUST render. It is the source of truth for the implementation and its tests.

## Placement

- The section MUST appear on the existing `/download` page, after the existing download card and `downloadPostHint`.
- It MUST be visible to any signed-in user reaching `/download` with no further navigation, expansion, or interaction required.

## Required content

| ID | Requirement | Verifiable by |
|----|-------------|---------------|
| C1 | A heading introducing TUI configuration. | Heading text from `messages.downloadConfig.configHeading` is present. |
| C2 | Statement that two settings are required: server address and an access credential. | Intro copy present. |
| C3 | The server address displayed equals `window.location.origin`. | Render with a known origin; assert that exact origin string appears. |
| C4 | A link to obtain a credential pointing to the in-app `/account` route, instructing the user to create an API key. | An anchor/`Link` with `href`/`to` resolving to `/account` is present with credential copy. |
| C5 | A quick-start method using environment variables `TWIG_API_KEY` and `TWIG_ADDR`, with `TWIG_ADDR` set to the current origin. | Command text contains both var names and the origin. |
| C6 | A persistent method: a TOML config file at `~/.config/twig/config.toml` containing the api key and api url keys. | Config-file path and snippet present. |
| C7 | A statement that environment variables take precedence over the config file. | Precedence copy present. |
| C8 | The exact launch command (`./twig`) and a note that running it with no arguments opens the interactive TUI. | Launch command present. |
| C9 | Commands and file snippets are rendered in copy-paste-friendly code blocks (e.g. `<pre>`/`<code>`), not interrupted by inline prose. | Snippets rendered inside code/pre elements. |

## Styling / consistency constraints

- MUST reuse existing page card styling, spacing, and dark-mode token classes already used on `DownloadPage`.
- MUST use the shared `react-router` `Link` for the `/account` link (in-app navigation, not a full reload).
- All visible strings MUST come from `messages.downloadConfig.*` (no hard-coded literals in the component).
- Copy MUST follow the project's warm/playful tone (Constitution Principle IV).

## Out of scope (explicitly not required)

- Reading, displaying, or generating the user's actual API key on this page.
- Multiple named profiles, pomodoro lifecycle hooks, or other advanced config keys.
- Any backend, proto, or CLI change.
- Per-platform variants beyond the existing single-platform download framing.

## Test expectations (`DownloadPage.test.tsx`)

The page test MUST assert, at minimum:
1. C3 — the rendered origin matches a stubbed `window.location.origin`.
2. C4 — a link resolving to `/account` is present.
3. C5 — env-var command contains `TWIG_API_KEY` and `TWIG_ADDR`.
4. C6 — config-file path `~/.config/twig/config.toml` is shown.
5. C8 — the `./twig` launch command is shown.
