# Implementation Plan: TUI Theme Pass

**Branch**: `018-tui-theme-pass` | **Date**: 2026-05-28 | **Spec**: [spec.md](./spec.md)

**Input**: Feature specification from `/specs/018-tui-theme-pass/spec.md`

## Summary

Refresh the interactive TUI (`internal/tui`) so the two panes are clearly framed
and the app shares one consistent visual language, with **no behavior change**.
The approach: introduce a centralized `theme.go` palette of semantic
`lipgloss.AdaptiveColor` styles, replace the hand-rolled column stitching
(`splitLines` + `padRightAnsi` loops) with lipgloss `RoundedBorder()` boxes
joined by `lipgloss.JoinHorizontal`, render rows as
`{treePrefix}{chevron-or-space} {checkbox} {name}` (separating fold state from
completion state), polish the detail pane (header + dimmed aligned labels), and
style the help line as a full-width footer. The existing `styled` flag continues
to gate all styling so non-TTY output stays plain.

## Technical Context

**Language/Version**: Go (toolchain per `services/todo/go.mod`)

**Primary Dependencies**: `github.com/charmbracelet/lipgloss v1.1.0`,
`github.com/charmbracelet/bubbletea v1.3.10`,
`github.com/charmbracelet/bubbles v1.0.0` (the `help` model)

**Storage**: N/A (presentation-only; no persistence touched)

**Testing**: `go test ./internal/tui/...` with `export_test.go` shims; golden-ish
substring assertions on rendered strings (styled and unstyled paths)

**Target Platform**: Interactive terminal (TTY); styling gated by `styled` flag
(`cli.WantStyled(os.Stdout)`)

**Project Type**: Single Go module (`services/todo`) — CLI/TUI client tier

**Performance Goals**: Render is interactive/per-frame; no measurable budget
beyond "no perceptible lag" — unchanged from today

**Constraints**: Must not panic at degenerate terminal sizes (clamp inner
dimensions ≥ 0 width / ≥ 1 height); unstyled output must remain byte-plain (no
ANSI, no decorative glyphs); behavior, keybindings, and data unchanged

**Scale/Scope**: ~4 files changed/added in one package (`internal/tui`):
`theme.go` (new), `view.go`, `details.go`, `tree.go`; tests added alongside

## Constitution Check

*GATE: Must pass before Phase 0 research. Re-check after Phase 1 design.*

**Principle I — Simplicity / YAGNI**: ✅ PASS. The change *removes* complexity:
the manual `splitLines`/`padRightAnsi` line-stitching is replaced by lipgloss's
own border/join primitives. No new packages, no speculative theming
configuration (palette is fixed, adaptive only). One new file (`theme.go`)
centralizes styles that are otherwise scattered — this is consolidation, not a
new abstraction layer.

**Principle II — API-First Design**: ✅ PASS (not applicable in the
request/response sense). This feature changes no proto messages, no ConnectRPC
services, no DB queries, and no shared cross-tier types. There is no
frontend/backend contract to define. The only "contract" is the internal UI
rendering contract (glyph mapping, palette semantics, layout/width invariants,
styled-gate behavior), captured in `contracts/ui-rendering-contract.md` to keep
the renderer and its tests aligned.

**Quality Gates**:
1. Contracts exist in `contracts/` — ✅ the UI rendering contract is provided; no
   API contracts are required because no API surface changes.
2. No added abstractions without Complexity Tracking — ✅ none added.
3. Constitution Check completed and passing for I & II — ✅ above.
4. No placeholder tokens remain — ✅ verified in delivered artifacts.

No violations → Complexity Tracking table is empty.

## Project Structure

### Documentation (this feature)

```text
specs/018-tui-theme-pass/
├── plan.md              # This file (/speckit-plan command output)
├── research.md          # Phase 0 output (/speckit-plan command)
├── data-model.md        # Phase 1 output (/speckit-plan command)
├── quickstart.md        # Phase 1 output (/speckit-plan command)
├── contracts/           # Phase 1 output (/speckit-plan command)
│   └── ui-rendering-contract.md
└── tasks.md             # Phase 2 output (/speckit-tasks command - NOT created here)
```

### Source Code (repository root)

```text
services/todo/internal/tui/
├── theme.go             # NEW — semantic AdaptiveColor palette + shared styles
│                        #       (highlightStyle, errorStyle move here)
├── view.go              # MODIFIED — bordered panes via lipgloss + JoinHorizontal;
│                        #       new row layout (chevron + checkbox); softened
│                        #       cursor; footer styling; removes splitLines/
│                        #       padRightAnsi stitching where lipgloss replaces it
├── details.go           # MODIFIED — bold accent header + dimmed aligned labels;
│                        #       gains a styled gate parameter
├── tree.go              # MODIFIED — expose fold state (chevron vs old [-]/[+])
│                        #       to the renderer; expandable-children logic unchanged
├── theme_test.go        # NEW — palette/style sanity (adaptive, styled-gate)
├── view_test.go         # MODIFIED — glyph mapping, border width, cursor, footer
├── details_test.go      # MODIFIED — header + label alignment (unstyled path)
└── tree_test.go         # MODIFIED — chevron presence ⇔ expandable children
```

**Structure Decision**: No new packages (Constitution I). All work is confined to
the existing `internal/tui` package, matching the design doc's "Architecture"
section. Data flow is unchanged: `Model.View()` → per-mode view fn → pane
renderers.

## Complexity Tracking

> No Constitution Check violations. Table intentionally empty.

| Violation | Why Needed | Simpler Alternative Rejected Because |
|-----------|------------|-------------------------------------|
| (none)    | —          | —                                   |
