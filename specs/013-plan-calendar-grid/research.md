# Phase 0: Research — Plan Calendar Grid View

All five clarification questions in the spec were resolved during `/speckit-clarify`. This document records the design-level research that backs the implementation choices, so the plan can be re-derived without rereading the conversation history.

## R1. How to obtain per-entry task completion status

**Decision**: Add `bool completed = 7;` to the `PlanEntry` proto message. Populate it server-side in `ListPlanEntries` via a LEFT JOIN against the `tasks` table on `task_id`, where `completed = (task_id != 0 AND tasks.completed_at IS NOT NULL)`. Event entries (no task_id) always get `completed = false`.

**Rationale**:

- Single round trip — the existing list call already returns all entries; pushing the join server-side keeps the CLI as a thin presentation layer.
- Trivial proto evolution — adding a new field number is backwards-compatible (old clients ignore unknown fields; old servers send a default `false`, which is the safe default for "treat as not completed").
- No new RPC, no new sqlc query file — just an updated `SELECT` in the existing `ListPlanEntries` query.

**Alternatives considered**:

- *Client fetches tasks in a second call* — doubles the round trip count and would require new TaskService machinery already absent from the CLI's plan flow. Rejected on simplicity grounds.
- *Compute on the client by listing all tasks once* — couples plan rendering to task pagination and adds complexity for an indirect benefit (no server change). Rejected because the server change is one line and the join is already cheap.
- *Encode completion state in entry name (e.g., "[x] foo")* — abuses display data as transport data. Rejected outright.

## R2. Calendar layout algorithm

**Decision**: Two-phase render driven by a single integer row index measured in 15-minute units from the window's start minute.

1. **Layout phase**: For each entry, compute `(startRow, endRow)` after snapping the entry's `start_minute` and `start_minute + duration_minute` to the nearest 15-minute boundary. Detect adjacency: two entries are adjacent when `entryA.endRow == entryB.startRow`. The window's start hour is `min(8*60, floor(minStart/60)*60)`; the end hour is `max(17*60, ceil(maxEnd/60)*60)`.
2. **Draw phase**: Iterate row indices `r` from 0 to `(endHour - startHour) * 4 - 1`. For each row, determine: (a) is this an hour-boundary row (i.e., `r % 4 == 0`)? (b) which entry, if any, contains this row? (c) is the row the top or bottom edge of that entry, the top edge while also being the bottom of an adjacent prior entry (shared border), or interior? Emit the corresponding character sequence.

**Rationale**: Working in 15-minute integer units removes all time arithmetic from the inner loop and makes the snapping rule (FR-005) a single integer round. The single-pass row loop matches the existing renderer's complexity profile.

**Alternatives considered**:

- *Build a 2D character grid then stringify* — uses more memory and adds complexity (rune width handling, terminal width recomputation) for no behavioral benefit. Rejected.
- *Recursive layout of nested boxes for overlap* — out of scope; the spec explicitly assumes single-track scheduling.

## R3. Determining terminal width

**Decision**: Use `golang.org/x/term.GetSize(int(os.Stdout.Fd()))` to detect terminal width. Cap at the available width; floor at 60 columns; default to 80 when stdout is not a TTY (e.g., piped to `less` or captured into a file).

**Rationale**: `x/term` is already a transitively-imported, well-known dependency; the same package gates ANSI-style rendering in `internal/cli/render.go`. Using a single source of truth for "is this a TTY" keeps the strikethrough/dim logic and the width logic in sync.

**Alternatives considered**:

- *Read `$COLUMNS`* — set by shells but not always exported to subprocesses; unreliable.
- *Always render at 80 columns* — wastes terminal real estate on modern displays.

## R4. Strikethrough rendering

**Decision**: Use ANSI SGR `\x1b[9m` for strikethrough and `\x1b[2m` for dim, reset with `\x1b[0m`. Apply both together when (a) the entry's `completed` is true AND (b) stdout is a TTY (per the existing TTY detection in `internal/cli/render.go`). When not a TTY, strip styling — the label still reads correctly without it.

**Rationale**: SGR 9 is widely supported in modern terminals (xterm, gnome-terminal, kitty, alacritty, wezterm, iTerm2, Windows Terminal). Where it is not supported (a few minimal TTYs), the label still renders — just without the line through it — and the box geometry remains intact.

**Alternatives considered**:

- *Replace the label text with `~text~`* — bleeds presentation into the textual data; awkward when piped through another tool. Rejected.
- *Suffix completed entries with a `[done]` tag* — eats label width that we already need for the name. Rejected.

## R5. Now-indicator placement

**Decision**: When the rendered day equals today in the server's local timezone (the CLI already passes today as the default `day` argument), place a single marker character (`▶`) in the left gutter on the 15-minute row containing the current local time. Do not span the row across the calendar — keeping the marker in the gutter avoids interfering with entry boxes.

**Rationale**: The spec's Q3 choice was explicitly Option A (gutter marker, not full-width line). A gutter marker also dodges the question "what happens if 'now' falls inside an entry box?" — the answer is "the marker sits in the gutter next to the box, the box is unchanged."

**Alternatives considered**:

- *Full-width "now" line* — was offered as Option B in clarification; user picked A. Recorded here only for traceability.
- *Color-based highlight of the row instead of a glyph* — fails on non-TTY output and on monochrome terminals.

## R6. Test strategy

**Decision**: Replace `plan_grid_test.go`'s current golden-string assertions with new goldens covering: empty day, single multi-hour entry, single sub-hour entry, two adjacent entries (touching), entry starting off-hour, label that wraps, label that overflows and truncates with ellipsis, completed-task entry (TTY-off, so no SGR noise in goldens), and now-marker (with an injected clock so the test is deterministic).

**Rationale**: The renderer is pure — input is `([]*PlanEntry, day, now, width, isTTY)`, output is a string. Golden assertions catch all five FR families with minimal scaffolding. An injected `now` allows the now-marker test to be deterministic without touching wall-clock time.

**Alternatives considered**:

- *Snapshot testing with a third-party library* — adds a dependency for no clear gain over plain string comparison.
- *Property-based testing for layout invariants* — overkill for a UI renderer with a small, well-defined input space.
