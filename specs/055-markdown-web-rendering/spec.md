# Feature Specification: Markdown Rendering in the Web App

**Feature Branch**: `055-markdown-web-rendering`

**Created**: 2026-06-14

**Status**: Draft

**Input**: User description: "markdown rendering in the web app. The TUI now supports markdown, we need to extend markdown support to the web app. Users may enter markdown in the web app's text fields if they know about it, but the primary request is to render markdown faithfully that was created elsewhere."

## User Scenarios & Testing *(mandatory)*

<!--
  User stories are prioritized as independently testable slices.
  Story 1 alone is a viable MVP.
-->

### User Story 1 - Read faithfully rendered long-text fields (Priority: P1)

A user opens a task in the web app whose description was authored elsewhere (for example in the TUI/CLI) using GitHub-flavored markdown — headings, bullet and numbered lists, task-list checkboxes, bold/italic emphasis, strikethrough, inline and fenced code, blockquotes, horizontal rules, tables, and links. When they view that task, the description is displayed as formatted, structured content rather than raw markup symbols, taking full advantage of the browser to reproduce the markdown faithfully (real headings, clickable links, styled code blocks, ruled tables).

**Why this priority**: The stated primary goal is to render markdown faithfully that was created elsewhere. Long-text fields (task descriptions) are where markdown carries the most meaning and where raw markup is most distracting. Rendering these well, on its own, delivers the core value and is a viable MVP.

**Independent Test**: Create or load a task whose description contains a representative mix of markdown (heading, bullet list, numbered list, task-list checkbox, bold, italic, strikethrough, inline code, fenced code block, blockquote, horizontal rule, table, and link). Open the task in the web app and confirm each construct renders in a recognizable, correctly formatted way, with no leftover markup characters for supported constructs and with links being clickable.

**Acceptance Scenarios**:

1. **Given** a task description containing `**bold**`, `*italic*`, `~~struck~~`, and `` `code` ``, **When** the user views the task, **Then** each renders with the corresponding styling and the surrounding markup characters are not shown.
2. **Given** a description containing bulleted, numbered, and task-list (`- [ ]` / `- [x]`) lists, including nesting, **When** the user views it, **Then** each list renders with appropriate markers, checkbox state, and preserved nesting.
3. **Given** a description containing headings of multiple levels, **When** the user views it, **Then** each heading is visually distinguished from body text and from other heading levels.
4. **Given** a description containing a GFM table, **When** the user views it, **Then** the table renders with delimited, aligned columns and a distinguished header row.
5. **Given** a description containing a fenced code block and a blockquote, **When** the user views it, **Then** the code block renders as preformatted monospace text preserving whitespace and the blockquote is visually set off.
6. **Given** a description containing a link, **When** the user views it, **Then** the link's display text is shown as a clickable link pointing to the correct destination.

---

### User Story 2 - Inline formatting in short fields (Priority: P2)

A user has a task whose name includes light inline markdown (for example `**Ship** the report` or `~~old plan~~`). Wherever that name appears in the web app — task tree rows, the task detail header, and the day planner — the inline emphasis is rendered so the formatting carries through consistently while the name stays on a single line.

**Why this priority**: Short fields benefit from basic emphasis but cannot host block-level layout on a single line. This extends the feature's value to the most frequently seen text in the UI, but is secondary to getting long-form rendering right.

**Independent Test**: Set a task name to include bold, italic, strikethrough, and inline code, then confirm the name renders with those inline styles in the tree row, the detail header, and the day planner, while remaining on a single line with no leftover markup characters.

**Acceptance Scenarios**:

1. **Given** a task name containing `**bold**` and `*italic*`, **When** the task appears in a tree row, the detail header, or the planner, **Then** the name renders with bold and italic styling on a single line with no markup characters.
2. **Given** a short field containing block-level markdown syntax (e.g. a leading `# ` or `- `), **When** it is displayed in a single-line context, **Then** the content renders legibly on one line without breaking the layout (block syntax is rendered inline or neutralized rather than producing a broken block element).

---

### User Story 3 - Author and edit markdown without content loss (Priority: P3)

A user who knows markdown enters it into a web app text field (e.g. a task name or description) when creating or editing a task. When they save, the stored text is exactly what they typed, and viewing the task again shows the faithful rendering. When they reopen the field to edit, they see and edit the original markdown source, not the rendered output.

**Why this priority**: Rendering must not interfere with authoring. The primary request is rendering content created elsewhere; entry support is explicitly secondary ("if they know about it"). Round-trip safety (display-only rendering, raw-source editing) is essential for trust but follows naturally once viewing is in place.

**Independent Test**: Enter markdown into a task description field, save, confirm the rendered view reflects it, reopen the field for editing, confirm the raw markdown source (not the rendered form) is shown, save without changes, and confirm the stored content is byte-for-byte identical.

**Acceptance Scenarios**:

1. **Given** a user types markdown into a description field and saves, **When** they view the task, **Then** the markdown is rendered faithfully.
2. **Given** a field containing markdown, **When** the user enters edit mode, **Then** the raw markdown source is shown for editing (not the rendered form).
3. **Given** a field is rendered for display, **When** the user saves without edits, **Then** the underlying stored text is unchanged (rendering is display-only and never rewrites content).

---

### Edge Cases

- **Malicious or unsafe content**: Because markdown is rendered into rich web content, any embedded scripts, event handlers, dangerous URLs (e.g. `javascript:`), or raw HTML that could execute code MUST be neutralized so that viewing a task can never run untrusted code or otherwise compromise the user's session. This is critical because content may originate from anywhere the task data was created or edited.
- **Raw HTML in source**: HTML embedded in markdown is either rendered as safe, sanitized content or shown as inert text — never executed.
- **Malformed/incomplete markdown**: Unclosed emphasis, a half-written table, or stray markup renders as readable text (as close to source as possible) and never breaks the page or crashes the view.
- **Width overflow**: Wide tables, long code lines, and long unbroken URLs are contained within their region (e.g. wrap or scroll within the field) and never break the surrounding page layout.
- **Content not intended as markdown**: Literal characters such as `*`, `_`, or `#` that are not valid markup are preserved as typed rather than silently consumed.
- **Empty or whitespace-only field**: Renders as empty without error and without showing a stray rendered artifact.
- **Plain text with no markdown**: Content with no markup renders as ordinary text, visually equivalent to today's plain-text display (including preserved line breaks).
- **Non-ASCII text and emoji**: Render correctly without corrupting layout.
- **Images**: Image markdown is represented gracefully (rendered if safe, otherwise shown as alt text / placeholder) rather than producing a broken element.

## Requirements *(mandatory)*

### Functional Requirements

- **FR-001**: The web app MUST render GitHub-flavored markdown for long-text display fields (task description) in display (read) mode.
- **FR-002**: Rendering MUST support inline constructs: bold, italic, bold-italic, strikethrough, inline code, and links.
- **FR-003**: Rendering MUST support block constructs: headings (multiple levels), unordered lists, ordered lists, nested lists, task-list checkboxes, blockquotes, fenced and indented code blocks, horizontal rules, and tables.
- **FR-004**: For all supported constructs, the source markup characters MUST NOT appear in display mode (e.g. `**`, `` ` ``, `#`, `|` used as syntax are consumed by rendering).
- **FR-005**: Links MUST render as clickable links pointing to the correct destination; link targets that are unsafe (e.g. script-executing schemes) MUST be neutralized.
- **FR-006**: The web app MUST render light inline markdown (bold, italic, strikethrough, inline code) for short single-line fields — at minimum task names — in every place those fields are displayed (task tree rows, task detail header, and the day planner), keeping them on a single line.
- **FR-007**: Rendering MUST be display-only and MUST NOT alter the stored content; the underlying text is preserved byte-for-byte.
- **FR-008**: When a user edits a markdown-bearing field, the editor MUST present the raw markdown source for editing, not the rendered output.
- **FR-009**: Rendered output MUST be sanitized so that untrusted content (scripts, event handlers, dangerous URL schemes, executable raw HTML) cannot execute or compromise the user's session when a task is viewed.
- **FR-010**: Malformed, incomplete, or non-markdown content MUST render as readable text and MUST NOT break page layout or cause the view to error.
- **FR-011**: Rendered content MUST be contained within its display region so that wide or long content (tables, code lines, long URLs) does not disrupt the surrounding page layout.
- **FR-012**: Content with no markdown markup MUST render as ordinary text equivalent to the current plain-text presentation, including preserved line breaks.
- **FR-013**: The set of supported markdown constructs in the web app MUST be consistent with what was authored for and rendered in the TUI, so that the same stored content reads equivalently across both clients (the web app may render more faithfully where the browser allows).

### Key Entities

- **Task name**: Short, single-line user-authored field; supports inline markdown only. Displayed in the task tree, task detail header, and day planner.
- **Task description**: Long, multi-line user-authored field; supports the full set of block and inline markdown constructs. Displayed on the task detail view.

## Success Criteria *(mandatory)*

### Measurable Outcomes

- **SC-001**: For a reference document exercising every supported construct (headings, lists, task lists, emphasis, strikethrough, inline/fenced code, blockquote, horizontal rule, table, link), 100% of constructs render in their intended formatted form with no leftover syntax characters.
- **SC-002**: The same stored task content renders with equivalent meaning and structure in both the web app and the TUI (no construct that renders in one is shown as raw markup in the other).
- **SC-003**: No untrusted markdown input can execute code or scripts when a task is viewed — verified against a set of known injection payloads, all of which are neutralized.
- **SC-004**: Editing then saving a markdown field without changes leaves the stored content byte-for-byte identical in 100% of cases.
- **SC-005**: Plain-text (non-markdown) content remains readable and visually equivalent to today's presentation, so existing users notice no regression for content that uses no markup.
- **SC-006**: Viewing a task with a rendered description adds no perceptible delay compared with the current plain-text view under normal field sizes.

## Assumptions

- The web app currently displays task **name** and **description**; goals and plan-entry notes are not yet surfaced as standalone editable fields in the web app, so this feature targets task name (inline) and task description (full), plus task names wherever they appear (tree, detail, planner). If/when goals appear in the web app, the same rendering approach is expected to extend to them.
- The backend and stored data are unchanged; markdown is stored as plain text exactly as authored, consistent with the existing API. No new fields or endpoints are required.
- "Faithful rendering" leverages the browser, so the web app can reproduce constructs more completely than the terminal (true headings, clickable links, ruled tables); fidelity is bounded only by safe-rendering (sanitization) constraints.
- GitHub-flavored markdown is the target dialect, matching the TUI feature (`050-tui-markdown-rendering`).
- Editing uses a raw-source editor (consistent with the TUI's round-trip-safe approach); a live side-by-side preview is out of scope for this iteration unless trivially available.
- Markdown rendering is display-only; it never rewrites or normalizes stored content.

## Dependencies

- Builds on the TUI markdown feature (`050-tui-markdown-rendering`) for the definition of the supported GitHub-flavored markdown construct set and round-trip-safety expectations.
- Relies on the existing `task.v1.TaskService` data (task name/description); no backend changes.
