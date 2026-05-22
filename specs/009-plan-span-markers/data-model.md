# Data Model — Plan Span Markers

## Scope

This feature introduces **no** new persistent entities, no protobuf message changes, and no database changes. Recording this explicitly here so it isn't an open question downstream.

## Existing entities used (unchanged)

- **`planv1.PlanEntry`** (from `services/todo/gen/plan/v1/`): the rendering input. Fields read by the rendering code: `Id`, `Name`, `StartMinute`, `DurationMinute`. No fields are added or repurposed.

## Render-time derived concepts (not stored)

- **Block** — a maximal run of consecutive 15-minute slots assigned to the same `PlanEntry`. In this implementation a Block is **not** a value or type; it is implicit in the per-slot loop. For each slot the rendering code asks two questions of the owning entry — "does the entry start here?" and "does the entry end here?" — and that pair selects one of four marker glyphs. There is no `Block` struct, slice, or function.

## Validation rules / state transitions

None added. The feature changes rendering of existing valid plan data; it does not introduce any new state.
