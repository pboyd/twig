---

description: "Task list for Markdown Rendering in the Web App"
---

# Tasks: Markdown Rendering in the Web App

**Input**: Design documents from `/specs/055-markdown-web-rendering/`

**Prerequisites**: plan.md, spec.md, research.md, data-model.md, contracts/markdown-component.md, quickstart.md

**Tests**: Included. The component contract (`contracts/markdown-component.md`) defines a mandatory test contract (sanitization, inline-vs-block, malformed/empty handling), and the plan lists `Markdown.test.tsx` as a deliverable. Tests follow the existing Vitest + Testing Library `*.test.tsx` conventions.

**Organization**: Tasks are grouped by user story. The shared `Markdown` component is foundational (one file, used by every story); each story is then independently testable wiring at distinct render sites.

## Format: `[ID] [P?] [Story] Description`

- **[P]**: Can run in parallel (different files, no dependencies)
- **[Story]**: Which user story this task belongs to (US1, US2, US3)
- All paths are under `services/twig-web/` unless noted.

## Path Conventions

- Web frontend (React SPA): `services/twig-web/src/`
- No backend, proto, or DB changes in this feature.

---

## Phase 1: Setup (Shared Infrastructure)

**Purpose**: Add the rendering dependencies.

- [X] T001 Add `react-markdown` and `remark-gfm` as runtime dependencies in `services/twig-web/package.json` (run `npm install react-markdown remark-gfm` from `services/twig-web`). Do NOT add `rehype-raw` or `dompurify` — safety relies on react-markdown's React-element rendering (research.md Decision 1, contract C-01).

---

## Phase 2: Foundational (Blocking Prerequisites)

**Purpose**: The shared `Markdown` component that all user stories consume.

**⚠️ CRITICAL**: No user story wiring can begin until the component exists.

- [X] T002 [P] Write the component test suite first (expected to fail) in `services/twig-web/src/components/Markdown.test.tsx`, covering the test contract in `contracts/markdown-component.md`: block mode renders each construct to its expected element (heading→`h*`, list→`li`, GFM table→`table`, link→`a[href]`, fenced code→`code` in `pre`, strikethrough→`del`, task checkbox→checkbox input, blockquote→`blockquote`, hr→`hr`); inline mode renders emphasis but emits NO block element and stays single-line with block syntax (leading `# `, `- `) degraded to inline text; a `javascript:` link href is neutralized; raw `<script>` / `<img onerror=...>` in source produces no executable element; malformed markdown renders without throwing; empty/whitespace renders nothing; single newlines render as line breaks.
- [X] T003 Implement `Markdown` component in `services/twig-web/src/components/Markdown.tsx` per `contracts/markdown-component.md` (props: `children: string`, `mode?: "block" | "inline"` default `"block"`, `className?: string`). Use `react-markdown` + `remark-gfm`; omit `rehype-raw`. Block mode: full GFM `components` map styled with existing theme tokens / Tailwind dark-mode conventions (`src/theme/tokens.ts`), code blocks `overflow-x-auto`, tables in a horizontally scrollable wrapper, links styled like in-app links (`text-blue-700 dark:text-blue-400 hover:underline`) with `target="_blank"` + `rel="noopener noreferrer"`, soft line breaks enabled. Inline mode: restrict to inline elements via `allowedElements` (`strong`, `em`, `del`, `code`, `a`) + `unwrapDisallowed`, map wrapping `p`→fragment/`span`. Wrap export in `React.memo` (stable on `children`/`mode`/`className`) per contract C-11. Make T002 pass.

**Checkpoint**: `Markdown` component exists, all component tests green (`npm test`), type-checks (`npm run build`).

---

## Phase 3: User Story 1 - Read faithfully rendered long-text fields (Priority: P1) 🎯 MVP

**Goal**: Task descriptions authored elsewhere render as faithfully formatted GFM (headings, lists, task lists, emphasis, code, blockquotes, rules, tables, clickable links) on the task detail view.

**Independent Test**: Open a task whose description contains a representative construct mix; confirm each renders formatted with no leftover syntax and the link is clickable (spec US1 acceptance scenarios 1–6).

### Tests for User Story 1

- [X] T004 [US1] Add an integration test in `services/twig-web/src/pages/TaskDetailPage.test.tsx` asserting a task description containing a heading, bullet + numbered + task lists, bold/italic/strikethrough/inline code, fenced code block, blockquote, horizontal rule, GFM table, and a link renders the corresponding elements (no raw `*`/`#`/`|` syntax shown; link has correct `href`). Ensure it fails before T005.

### Implementation for User Story 1

- [X] T005 [US1] In `services/twig-web/src/pages/TaskDetailPage.tsx`, replace the description `<p className="… whitespace-pre-wrap">{task.description}</p>` (around line 167–171) with `<Markdown mode="block">{task.description}</Markdown>` (keep the existing wrapper container/margins; render only when `task.description` is non-empty). Import the new component.
- [X] T006 [US1] Update any existing assertions in `services/twig-web/src/pages/TaskDetailPage.test.tsx` that relied on the plain `<p>` description so they pass with the new rendered output; run `npm test` for this file.

**Checkpoint**: Task descriptions render full markdown; US1 independently demoable (the MVP).

---

## Phase 4: User Story 2 - Inline formatting in short fields (Priority: P2)

**Goal**: Task names render light inline markdown (bold, italic, strikethrough, inline code, links) on a single line everywhere they appear — tree rows, detail header, and the day planner.

**Independent Test**: Set a task name to `**Ship** the ~~old~~ report`; confirm it renders with emphasis on one line in the tree, the detail header, and the planner, with no leftover syntax (spec US2 acceptance scenarios 1–2).

### Tests for User Story 2

- [X] T007 [US2] Add interaction tests asserting inline name rendering and single-line/no-block behavior: in `services/twig-web/src/pages/TaskTreePage.test.tsx` (or a `TreeRow` test) for the tree row, in `services/twig-web/src/pages/TaskDetailPage.test.tsx` for the header `<h1>`, and in `services/twig-web/src/pages/PlanPage.test.tsx` for the plan entry name — each: emphasis renders (`strong`/`em`/`del`), no block element is emitted, and leading block syntax (`# `) degrades to inline text. Ensure they fail before T008–T010.

### Implementation for User Story 2

- [X] T008 [P] [US2] In `services/twig-web/src/components/TreeRow.tsx`, render the task name (around line 155) via `<Markdown mode="inline">{task.name}</Markdown>`, preserving the existing row styling/click target. Import the component.
- [X] T009 [P] [US2] In `services/twig-web/src/components/PlanEntryRow.tsx`, render `entry.displayName` (the `nameContent` span, around line 21–27) via `<Markdown mode="inline">{entry.displayName}</Markdown>`, keeping the existing completed/strikethrough styling on the wrapper.
- [X] T010 [US2] In `services/twig-web/src/pages/TaskDetailPage.tsx`, render the header name inside the `<h1>` (around line 165) via `<Markdown mode="inline">{task.name}</Markdown>`, keeping the existing heading classes. (Same file as T005 — sequence after T005.)
- [X] T011 [P] [US2] Update any existing name assertions in `services/twig-web/src/pages/TaskTreePage.test.tsx` and `services/twig-web/src/pages/PlanPage.test.tsx` affected by inline rendering; run `npm test` for those files.

**Checkpoint**: Names render inline emphasis consistently across tree, detail header, and planner; US1 + US2 both work.

---

## Phase 5: User Story 3 - Author and edit markdown without content loss (Priority: P3)

**Goal**: Editing shows the raw markdown source (not rendered output); saving preserves stored content byte-for-byte; rendering is display-only.

**Independent Test**: Enter markdown in the description, save, view it rendered, reopen edit and confirm raw source is shown, save unchanged and confirm content is byte-for-byte identical (spec US3 acceptance scenarios 1–3).

### Tests for User Story 3

- [X] T012 [US3] Add a test in `services/twig-web/src/components/TaskForm.test.tsx` (create if absent, following existing `*.test.tsx` conventions) asserting that opening a task with a markdown description for editing shows the raw markdown source in the `<textarea>` (not rendered HTML), and that submitting without edits produces an unchanged payload (verify against `src/lib/updatePayload.ts` output). Ensure it fails if any rendering leaks into the editor.

### Implementation for User Story 3

- [X] T013 [US3] Verify and, if needed, guard that `services/twig-web/src/components/TaskForm.tsx` continues to bind the raw `task.description` / `task.name` strings into its input/`<textarea>` (no `Markdown` rendering in edit mode), and confirm no render site introduced in US1/US2 mutates the stored value (display-only). Make T012 pass; no functional change expected if the editor is already raw.

**Checkpoint**: Round-trip safety confirmed; all three stories independently functional.

---

## Phase 6: Polish & Cross-Cutting Concerns

**Purpose**: Validation, accessibility, performance, and tone across the feature.

- [X] T014 [P] Verify accessibility of rendered output: external links carry `rel="noopener noreferrer"`, headings/lists/tables use semantic elements, and code blocks remain readable in dark mode — adjust the `components` map in `services/twig-web/src/components/Markdown.tsx` if any gap is found.
- [X] T015 [P] Confirm contained overflow (FR-011): add/verify a test or manual check that a very wide table and a long unbroken code line scroll within their region in `Markdown.tsx` and do not break page layout.
- [X] T016 [P] Review any new user-facing copy (e.g. image alt fallback / placeholder text) for warm-tone compliance (Constitution Principle IV), reusing `services/twig-web/src/theme/messages.ts` where appropriate.
- [X] T017 Run the full gate from `services/twig-web`: `npm test` (all green) and `npm run build` (type-check + production build pass), then execute the manual smoke test in `quickstart.md` (constructs render, link clickable, name inline, raw edit shows source, `javascript:` and `<img onerror>` payloads do not execute).

---

## Dependencies & Execution Order

### Phase Dependencies

- **Setup (Phase 1)**: No dependencies — start immediately.
- **Foundational (Phase 2)**: Depends on Setup (deps installed). BLOCKS all user stories.
- **User Stories (Phase 3–5)**: All depend on Foundational (`Markdown` component). US1 is the MVP.
- **Polish (Phase 6)**: Depends on the stories being implemented.

### User Story Dependencies

- **US1 (P1)**: After Foundational. Independent. Edits `TaskDetailPage.tsx` (description).
- **US2 (P2)**: After Foundational. Independent of US1, except T010 edits the same file as T005 (`TaskDetailPage.tsx`) so sequence T010 after T005.
- **US3 (P3)**: After Foundational. Independent (editor/`TaskForm` files); validates display-only behavior of US1/US2 sites.

### Within Each User Story

- Write the story's test(s) first and confirm they fail, then implement, then update affected existing tests.

### Parallel Opportunities

- T002 (component tests) can be authored in parallel with reading the contract; T003 implements against it.
- US2: T008 (`TreeRow.tsx`) and T009 (`PlanEntryRow.tsx`) are different files → run in parallel; T010 (`TaskDetailPage.tsx`) sequences after US1's T005.
- Polish: T014, T015, T016 are independent → parallel.
- With multiple developers, after Phase 2 completes US1/US2/US3 can proceed largely in parallel (mind the `TaskDetailPage.tsx` overlap between T005 and T010).

---

## Parallel Example: User Story 2

```bash
# After the Markdown component (Phase 2) is done, wire the two independent name sites together:
Task: "Render task name via <Markdown mode=\"inline\"> in services/twig-web/src/components/TreeRow.tsx"
Task: "Render entry.displayName via <Markdown mode=\"inline\"> in services/twig-web/src/components/PlanEntryRow.tsx"
# Then sequence the detail header (same file as the US1 description change):
Task: "Render header name via <Markdown mode=\"inline\"> in services/twig-web/src/pages/TaskDetailPage.tsx"
```

---

## Implementation Strategy

### MVP First (User Story 1 Only)

1. Phase 1: Setup (install deps).
2. Phase 2: Foundational (`Markdown` component + tests). CRITICAL — blocks all stories.
3. Phase 3: US1 (block-render task descriptions).
4. **STOP and VALIDATE**: faithfully rendered descriptions — demoable MVP.

### Incremental Delivery

1. Setup + Foundational → component ready.
2. US1 → descriptions render → demo (MVP).
3. US2 → names render inline everywhere → demo.
4. US3 → round-trip safety confirmed → demo.
5. Polish → a11y, overflow, tone, full gate.

---

## Notes

- [P] = different files, no dependencies.
- This feature is frontend-only: no proto/`make proto`, no `sqlc`, no server changes.
- Safety (FR-009/SC-003) comes from omitting `rehype-raw`/`dangerouslySetInnerHTML` — never add raw-HTML rendering.
- Editing stays raw-source; rendering is display-only (FR-007/FR-008/SC-004).
- Commit after each task or logical group; stop at any checkpoint to validate a story independently.
