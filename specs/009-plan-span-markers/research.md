# Research — Plan Span Markers

## Open questions from Technical Context

None. The Technical Context for this feature contained no `NEEDS CLARIFICATION` markers — language, dependencies, target platform, and testing approach are all already established by the existing project. The visual contract (which glyphs, which positions) was settled during the brainstorming session that produced spec.md.

## Glyph choice — confirmed during brainstorming

- **Decision**: Use Unicode box-drawing characters `┌` (start), `│` (middle), `└` (end), `─` (single-slot).
- **Rationale**: The plan view already uses `│` as the column separator (`HH:MM │ ...`), so the same UTF-8 / box-drawing capability is already required of the user's terminal. Reusing the same character family means zero new font / locale assumptions. The four chosen glyphs are visually distinct from each other and from the existing `│` separator (because the separator sits in a different column).
- **Alternatives considered**:
  - ASCII fallback (`+`, `|`, `+`, `-`): rejected — the project already requires UTF-8 for `│`, so an ASCII path would add a branch with no real-world benefit.
  - Heavier box-drawing (`┏`, `┃`, `┗`, `━`): rejected — visually noisier than the light variants and not perceptibly clearer at terminal font sizes.

## Block detection — derive at render time, do not introduce a "Block" type

- **Decision**: The existing per-slot loop in `RenderGrid` decides per slot whether it is the first, middle, or last slot of a contiguous block by inspecting the owning entry's `StartMinute` and `EndMinute`. No new type, no precomputed block list.
- **Rationale**: Principle I (Simplicity/YAGNI). The information needed per slot is local: "does the entry start in this slot?", "does the entry end in this slot?". Two booleans determined inline cover the four marker cases (start, middle, end, single).
  - `startsHere = (t <= e.StartMinute < t+15)`
  - `endsHere   = (t <= e.EndMinute   <= t+15)` where `EndMinute = e.StartMinute + e.DurationMinute`
  - start && end → single (`─`); start && !end → start (`┌`); !start && end → end (`└`); !start && !end → middle (`│`).
- **Alternatives considered**:
  - Introduce a `Block` struct and a pre-pass that groups entries into blocks: rejected — adds a layer the rendering loop doesn't need. The current loop already finds the owning entry per slot; we just add two cheap comparisons.

## Test strategy — extend the existing table-driven tests

- **Decision**: Update the existing snapshot expectations in `plan_grid_test.go` for cases that currently exercise multi-slot entries, and add new table rows for: (a) a single-slot entry, (b) a 2-slot entry (start + end, no middle), (c) two adjacent entries (end marker on one line, start marker on the next).
- **Rationale**: The existing test file is the canonical place to express the rendered-output contract. Snapshot-style table tests already in use give the clearest failure messages when the visual contract is broken.
- **Alternatives considered**:
  - Add a separate `plan_grid_blocks_test.go`: rejected — fragments coverage of the same function across two files for no benefit.
