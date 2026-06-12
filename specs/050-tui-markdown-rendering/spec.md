# Feature Specification: Markdown Rendering in TUI Text Fields

**Feature Branch**: `050-tui-markdown-rendering`

**Created**: 2026-06-12

**Status**: Draft

**Input**: User description: "Text fields in the TUI should render markdown as well as possible in a terminal. GitHub-flavored markdown is preferred. Bold, underline, italic can be done with ANSI codes. Tables and lists can use Unicode glyphs. There will, of course, be limits to what's possible (hyperlinks and headings come to mind), but where possible we should try to faithfully represent markdown. This is especially important for the longer text fields (e.g. task details), but it would be helpful to at least have basic formatting for shorter fields too (e.g. task name)."

## Clarifications

### Session 2026-06-12

- Q: When rendered content is wider than the field/terminal (wide tables, long code lines, long URLs), how should overflow be handled? → A: Soft-wrap to the available width — long lines wrap and table columns reflow/shrink to fit; all content stays visible (no horizontal scroll, no truncation in display mode).
- Q: How should a markdown link surface its destination in display mode? → A: Emit terminal hyperlink escape sequences (OSC 8) where the terminal supports them (link text is clickable); on unsupported terminals, fall back to showing `text (url)` so the destination is always visible.
- Q: Which TUI text fields should be treated as markdown? → A: All user-authored fields across Goals/Tasks/Plan — task name & description, goal name & description, and plan-entry notes/text — with long fields getting full block rendering and short fields inline-only. Generated text (Report tab) is rendered as-is.

## User Scenarios & Testing *(mandatory)*

### User Story 1 - Read formatted long-text fields (Priority: P1)

A user has written a task description (or goal description, or plan entry note) using GitHub-flavored markdown — headings, bullet and numbered lists, bold/italic emphasis, inline and fenced code, blockquotes, tables, and links. When they view that field in the TUI, the content is displayed as formatted, structured text rather than raw markup symbols, so it is easy to scan and read inside the terminal.

**Why this priority**: Long-text fields are where markdown delivers the most value and where raw markup is most distracting. Rendering these is the core of the feature and on its own makes descriptions substantially more readable — a viable MVP.

**Independent Test**: Open a task whose description contains a representative mix of markdown (heading, list, bold, italic, code block, table, blockquote, link) and confirm each construct is displayed in a recognizably formatted way within the terminal, with no leftover markup symbols for supported inline styles.

**Acceptance Scenarios**:

1. **Given** a task description containing `**bold**`, `*italic*`, and `` `code` ``, **When** the user views the task in the TUI, **Then** the words render with bold, italic, and monospace/highlighted styling and the surrounding markup characters (`*`, `` ` ``) are not shown.
2. **Given** a description containing a bulleted list and a numbered list, **When** the user views it, **Then** each list renders with aligned item markers (e.g. Unicode bullets and numbers) and preserved nesting/indentation.
3. **Given** a description containing a GFM table, **When** the user views it, **Then** the table renders with aligned columns using box-drawing/Unicode glyphs and is readable within the available width.
4. **Given** a description containing a heading and a blockquote, **When** the user views it, **Then** the heading is visually distinguished from body text and the blockquote is visually set off (e.g. with a leading marker/indent).
5. **Given** a description containing a link, **When** the user views it, **Then** the link's display text is shown in a distinguished style and the destination remains discoverable.

---

### User Story 2 - Inline formatting in short fields (Priority: P2)

A user includes light inline markdown in a short, single-line field such as a task name or goal name (for example `**Ship** the report` or `~~old plan~~`). When that name appears in lists, headers, tree rows, and detail views, the inline emphasis is rendered so the formatting carries through wherever the name is displayed.

**Why this priority**: Short fields benefit from basic emphasis but cannot host block-level layout (lists, tables) on a single line. This extends the feature's value to the most frequently seen text in the UI, but is secondary to getting long-form rendering right.

**Independent Test**: Set a task name to include bold, italic, strikethrough, and inline code, then confirm the name renders with those inline styles everywhere it is displayed (tree row, detail header) while remaining on a single line.

**Acceptance Scenarios**:

1. **Given** a task name containing `**bold**` and `*italic*`, **When** the task appears in a list/tree row, **Then** the name renders with bold and italic styling on one line with no markup symbols.
2. **Given** a short field containing block-level markdown syntax (e.g. a leading `# ` or `- `), **When** it is displayed in a single-line context, **Then** the content renders legibly on one line without breaking the layout (block syntax is rendered inline or neutralized rather than producing a broken multi-line element).

---

### User Story 3 - Edit raw source without content loss (Priority: P3)

A user opens a text field for editing in the TUI. They see and edit the original markdown source (not the rendered output), make changes, and save. The stored text is exactly what they typed, and viewing it again shows the updated rendering.

**Why this priority**: Rendering must not interfere with authoring. Round-trip safety (display-only rendering) is essential for trust, but it follows naturally once viewing is in place, so it is grouped as a guardrail rather than the headline value.

**Independent Test**: Edit a field containing markdown, confirm the editor shows raw markdown characters, save without changes, and confirm the stored content is byte-for-byte identical and the rendered view is unchanged.

**Acceptance Scenarios**:

1. **Given** a field containing markdown, **When** the user enters edit mode, **Then** the raw markdown source is shown for editing (not the rendered form).
2. **Given** a field is rendered for display, **When** the user saves without edits, **Then** the underlying stored text is unchanged (rendering is display-only and never rewrites content).

---

### Edge Cases

- **Width overflow**: Wide tables, long code lines, and long URLs that exceed the terminal width soft-wrap to the available width (table columns reflow/shrink) rather than bleeding into other UI; all content stays visible without horizontal scroll or truncation.
- **Malformed/incomplete markdown**: Unclosed emphasis, a half-written table, or stray markup must render as readable text (as close to source as possible) and never crash or hang the TUI.
- **Deeply nested lists/quotes**: Nesting beyond a reasonable depth should still render legibly, with indentation capped if needed to fit the width.
- **Limited terminals**: On a terminal without ANSI color/style support (or where styling is disabled), output degrades to plain, readable text with no visible escape codes.
- **Unicode-incapable terminals**: Where box-drawing/Unicode glyphs are unavailable, tables/lists fall back to ASCII-equivalent markers rather than mojibake.
- **Content not intended as markdown**: Literal characters such as `*`, `_`, or `#` that are not valid markup should be preserved as typed rather than silently consumed.
- **Empty or whitespace-only field**: Renders as empty without error.
- **Non-ASCII text and emoji**: Render correctly and do not break column alignment more than the terminal itself does.
- **Unsupported constructs (e.g. images, raw HTML)**: Represented gracefully (e.g. alt text / placeholder / passed-through text) rather than dropped silently or shown as broken markup.

## Requirements *(mandatory)*

### Functional Requirements

- **FR-001**: The TUI MUST render GitHub-flavored markdown for multi-line / long-text display fields (task description/details, goal description, and other user-authored long-text fields), in display (read) mode.
- **FR-002**: Rendering MUST support inline constructs: bold, italic, bold-italic, strikethrough, and inline code.
- **FR-003**: Rendering MUST support block constructs: headings, unordered lists, ordered lists, nested lists, task-list checkboxes, blockquotes, fenced/indented code blocks, horizontal rules, and tables.
- **FR-004**: Bold, italic, and similar emphasis MUST be conveyed using terminal styling (ANSI), and for supported inline styles the surrounding markup characters MUST NOT appear in display mode.
- **FR-005**: Lists and tables MUST use Unicode glyphs (e.g. bullets, box-drawing characters) for markers and structure where the terminal supports them.
- **FR-006**: Headings MUST be visually distinguished from body text using available terminal styling (since true heading semantics cannot be reproduced in a terminal).
- **FR-007**: Links MUST render their display text in a distinguished style. On terminals that support hyperlink escape sequences (OSC 8), the link MUST be emitted as a clickable terminal hyperlink; on terminals without that support, the destination MUST be surfaced inline as `text (url)` so it is always visible.
- **FR-008**: Short, single-line fields (e.g. task name, goal name) MUST render inline markdown only; block-level syntax appearing in such fields MUST be rendered inline or neutralized so it does not break single-line layout.
- **FR-009**: Rendered short-field text MUST be applied consistently wherever the field is displayed (list rows, tree rows, detail headers, and any other display context).
- **FR-010**: Markdown rendering MUST be display-only and MUST NOT modify the stored text; editing a field MUST present the raw markdown source.
- **FR-011**: Rendering MUST stay within the available field/terminal width and MUST NOT corrupt or overflow into surrounding TUI elements; overly wide content MUST soft-wrap to the available width (long lines wrap; table columns reflow/shrink to fit) so all content stays visible in display mode — without horizontal scrolling or truncation.
- **FR-012**: Malformed, incomplete, or unsupported markdown MUST render as readable text approximating the source and MUST NOT cause the TUI to crash, hang, or visibly garble unrelated UI.
- **FR-013**: When terminal styling is unavailable or disabled, rendering MUST degrade gracefully to plain, readable text with no visible escape sequences; when Unicode glyphs are unavailable, structure MUST fall back to ASCII-equivalent markers.
- **FR-014**: Constructs that cannot be faithfully represented in a terminal (e.g. images, raw HTML) MUST be handled gracefully (placeholder, alt text, or pass-through) rather than dropped silently or shown as broken markup.

### Key Entities

- **Text field**: A user-authored text value displayed in the TUI. Has a length character (single-line/short vs. multi-line/long) that determines whether full block rendering or inline-only rendering applies. Carries raw markdown source (stored, edited) and a rendered form (display-only, derived).

## Success Criteria *(mandatory)*

### Measurable Outcomes

- **SC-001**: For a representative sample of GFM constructs (bold, italic, strikethrough, inline code, headings, ordered/unordered/nested lists, task lists, blockquotes, fenced code, horizontal rules, tables, links), each is displayed in a recognizably formatted way in the TUI — verifiable by visual inspection against a reference document.
- **SC-002**: In display mode, no markup characters for supported inline styles (e.g. `*`, `_`, `` ` ``, `~`) remain visible in the rendered output.
- **SC-003**: Editing any text field and saving without changes leaves the stored content byte-for-byte identical 100% of the time (no rendering-induced mutation).
- **SC-004**: Rendered content stays within the terminal width and does not corrupt adjacent UI for all edge-case inputs in the test set (wide tables, long URLs, deeply nested lists, malformed markup) — zero layout-corruption failures.
- **SC-005**: With styling disabled / on a non-color terminal, every field still displays as readable plain text with zero visible escape sequences.
- **SC-006**: Rendering a long field does not introduce perceptible lag (the view updates without a noticeable pause during normal navigation).
- **SC-007**: Short-field inline formatting renders consistently in 100% of the display contexts where that field appears.

## Assumptions

- **Scope is the interactive TUI only.** Plain-CLI (non-TTY) output and the React web app are out of scope; their text handling is unchanged by this feature.
- **GitHub-flavored markdown (GFM)** is the target dialect, consistent with the user's stated preference.
- **Editing shows raw source.** Text input/editing modes display the underlying markdown characters; rendering is a read/display concern only. There is no in-place "rich text" editing.
- **Fields treated as markdown** are the user-authored text fields across the Goals, Tasks, and Plan tabs (names/titles and descriptions/notes). Generated/computed text (e.g. the Report tab output) is rendered as-is unless it already contains user-authored markdown.
- **Length character drives rendering depth**: long/multi-line fields get full block + inline rendering; short/single-line fields get inline-only rendering.
- **Terminal capability detection** reuses the TUI's existing TTY/ANSI gating; where color/Unicode is unavailable, the documented fallbacks apply.
- **Link destinations** are surfaced via OSC 8 terminal hyperlink escape sequences where the terminal supports them (clickable), otherwise inline as `text (url)`; true clickable headings/anchors are not attempted.
- **No new stored data**: this feature does not change what is persisted on the server; the same raw text is stored and retrieved as today.
