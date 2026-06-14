# Quickstart: Markdown Rendering in the Web App

**Feature**: `055-markdown-web-rendering` | **Date**: 2026-06-14

All work is in `services/twig-web/`. No backend, proto, or `make proto` /
`sqlc generate` steps are involved.

## Prerequisites

```bash
cd services/twig-web
npm install            # then add the two runtime deps below
```

Add dependencies:

```bash
npm install react-markdown remark-gfm
```

> Do **not** add `rehype-raw` — leaving raw HTML disabled is what keeps rendering
> safe (no XSS). See `contracts/markdown-component.md` C-01.

## Run the app

```bash
# from services/twig-web
npm run dev            # http://localhost:5173 (proxies /task.v1 → :8080)
```

The backend stack must be running for real data:

```bash
# from repo root
make dev               # postgres + server on :8080
```

## Build / verify the component

```bash
# from services/twig-web
npm test               # Vitest — runs *.test.tsx interaction tests
npm run build          # tsc + vite production build (type-check gate)
```

## What you'll add

1. **`src/components/Markdown.tsx`** — the `Markdown` component
   (`mode="block" | "inline"`) per `contracts/markdown-component.md`. Built on
   `react-markdown` + `remark-gfm`, styled with existing theme tokens.
2. **`src/components/Markdown.test.tsx`** — interaction tests covering the test
   contract (constructs, inline vs block, link sanitization, raw-HTML inertness,
   malformed input, empty input).
3. **Render-site updates** (display-only):
   - `src/pages/TaskDetailPage.tsx` — description → `<Markdown mode="block">`,
     header name → `<Markdown mode="inline">`.
   - `src/components/TreeRow.tsx` — task name → `<Markdown mode="inline">`.
   - `src/components/PlanEntryRow.tsx` — `entry.displayName` →
     `<Markdown mode="inline">`.
4. **Leave `TaskForm` alone** — editing stays raw-source (round-trip safety).

## Manual smoke test

1. Open a task and Edit; paste a markdown sample into the description:
   ```markdown
   # Heading
   - [x] done item
   - [ ] todo item

   **bold** _italic_ ~~struck~~ `code` and a [link](https://example.com).

   > a quote

   | a | b |
   |---|---|
   | 1 | 2 |

   ```js
   const x = 1;
   ```
   ```
2. Save and view the task — every construct should render formatted, the link
   should be clickable, and no `*`/`#`/`|` syntax should remain.
3. Set a task name to `**Ship** the ~~old~~ report` — confirm it renders with
   emphasis on one line in the tree, the detail header, and the day planner.
4. Reopen Edit — confirm the raw markdown source is shown (not the rendered
   form), and saving without changes leaves it identical.
5. Paste `[click](javascript:alert(1))` and `<img src=x onerror=alert(1)>` into a
   description — confirm neither executes.

## Success checks (from spec)

- SC-001: every supported construct renders formatted, no leftover syntax.
- SC-002: same content reads equivalently in TUI and web.
- SC-003: injection payloads are neutralized.
- SC-004: edit-then-save-unchanged leaves content byte-for-byte identical.
- SC-005: plain text looks the same as before (line breaks preserved).
- SC-006: no perceptible delay versus the old plain-text view.
