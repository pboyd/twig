# Quickstart: TUI Configuration Instructions on Download Page

## What this feature does

Adds a "configure the TUI" section to the web app's `/download` page so a freshly-downloaded binary becomes usable without external docs.

## Where the work happens

All changes are in `services/twig-web/`:

| File | Change |
|------|--------|
| `src/theme/messages.ts` | Add a `downloadConfig` message group (warm/playful tone). |
| `src/pages/DownloadPage.tsx` | Render the new section after the download card; read `window.location.origin`; link to `/account` via `react-router` `Link`. |
| `src/pages/DownloadPage.test.tsx` | Add assertions per the contract (origin, `/account` link, env vars, config path, launch command). |

## Dev loop

```bash
cd services/twig-web
npm install            # if not already
npm run dev            # http://localhost:5173 — view /download
npm test               # run Vitest (DownloadPage.test.tsx)
npm run build          # production build sanity check
```

## Manual verification (maps to spec User Story 1)

1. Sign in and open `/download`.
2. Confirm the configuration section appears below the download button/hint.
3. Confirm the server address shown matches the browser's current origin.
4. Click the "create an API key" link → lands on `/account` (no full reload).
5. Confirm both the env-var quick start and the `~/.config/twig/config.toml` persistent setup are shown, with a note that env vars win.
6. Confirm the launch command `./twig` is shown with a note that it opens the TUI.

## Reference

- Contract: `contracts/download-page-config-section.md`
- Decisions: `research.md`
- Config schema background: `specs/015-pomodoro-config-file/contracts/config-schema.md` (note: live env-var names are `TWIG_API_KEY` / `TWIG_ADDR` per CLAUDE.md)
