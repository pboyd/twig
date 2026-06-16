# Phase 1 Data Model: TUI Configuration Instructions on Download Page

This feature introduces **no persistent data, no new entities, and no schema changes**. It is presentational copy on an existing page.

The only "data" involved is the set of user-facing strings and one runtime-derived value:

## Conceptual elements

| Element | Source | Notes |
|---------|--------|-------|
| Server address | `window.location.origin` (runtime) | Displayed verbatim in the env-var and config-file examples. Not stored. |
| Access credential | User-supplied; obtained from `/account` | Never read or displayed by this page — the page only links to where it is created. |
| Instruction copy | `messages.ts` → `downloadConfig` group | Static strings; some are functions taking the origin (e.g. `quickStartCommand(origin)`). |

## Message group shape (`downloadConfig`)

Indicative keys to add under `messages` (final wording authored during implementation, in warm/playful tone):

- `configHeading` — section title (e.g. "Point twig at your server")
- `configIntro` — one line explaining the two required settings
- `credentialStep` — copy directing the user to create a key, paired with a `Link` to `/account`
- `credentialLinkLabel` — link text for the Account page
- `serverAddressLabel` — label preceding the displayed origin
- `quickStartHeading` / `quickStartCommand(origin: string)` — env-var one-liner
- `persistHeading` / `persistIntro` — config-file path + intro
- `persistFileContents(origin: string)` — TOML snippet body
- `precedenceNote` — states env vars win over the config file
- `launchHeading` / `launchCommand` — the `./twig` launch command + what it does (opens the TUI)

(Exact key names may be adjusted during implementation; the contract in `contracts/download-page-config-section.md` is authoritative for required content.)
