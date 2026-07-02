# Research: Web Plan Tab Day-Planner Redesign

**Date**: 2026-07-02 | **Spec**: [spec.md](./spec.md)

No `NEEDS CLARIFICATION` markers remained after `/speckit-clarify`. The decisions below resolve the implementation-level unknowns.

## D1: Timeline rendering technique

**Decision**: CSS grid with one row per 15-minute slot at fixed 44px row height (`spacing.touchTarget`). Entry blocks are grid items spanning their snapped slot range; the left column is an hour-labeled time gutter; hour boundaries get a ruled line across the track. Entries snap outward to 15-minute boundaries (start rounded down, end rounded up), exactly like the CLI/TUI planner's `snapDown15`/`snapUp15`.

**Rationale**:
- Mirrors the existing terminal planner geometry (15-minute slots, hour dividers), satisfying Principle III across clients.
- A 15-minute entry occupies one 44px row, so the shortest possible block meets the app's touch-target convention with no special-casing (FR-011, SC-004); back-to-back short entries can never collide because the server forbids overlaps and every block is at least one full slot tall.
- Grid rows make gaps appear as empty ruled rows for free (FR-003), and duration→height proportionality is inherent (FR-001).
- Pure snapping/window math lives in `planView.ts` as unit-testable functions; the component stays declarative.

**Alternatives considered**:
- *Pixel-per-minute absolute positioning* (Google Calendar style): more positional precision, but requires min-height special-casing for short entries, which reintroduces collision handling the no-overlap guarantee otherwise eliminates. Rejected as more complex for no user-visible gain at 15-minute planning granularity (the add-to-plan control already works in 15-minute steps).
- *Denser rows (e.g., 24px/slot) with overflowing action buttons*: breaks the 44px touch-target convention or forces hover-only actions. Rejected.
- *A calendar library (e.g., FullCalendar)*: heavyweight dependency for a read-mostly single-day view; violates Principle I. Rejected.

## D2: Timeline window computation

**Decision**: `computeWindow(entries)` in `planView.ts` returns `{ startMinute, endMinute }`: the earliest snapped start floored to the hour and the latest snapped end ceiled to the hour. When there are no timed entries, and as a minimum span, the window is 8:00–17:00 (480–1020); entries outside the default simply extend it. (Clarified 2026-07-02: auto-fit, terminal-planner convention.)

**Rationale**: Matches the CLI grid's "sized to contain every entry's snapped span, expanded to hour boundaries" behavior; the fixed fallback keeps an empty day from rendering a zero-height or 24-hour track.

**Alternatives considered**: fixed working-hours window and full 24-hour scrollable day — both explicitly declined during clarification.

## D3: Shared done control

**Decision**: Extract the Tasks tab's inline circle toggle (currently markup inside `TreeRow.tsx`) into a `CompletionToggle` component (props: `completed`, `disabled`, `onToggle`, optional `aria-label`s). `TreeRow`, the timeline entry block, and the untimed row all render it. The Plan tab wires it to `completeTask`/`uncompleteTask` (two-way, per clarification), reusing the existing error mapping: `completeBlockedBySubtasks` on blocked complete, `reopenBlockedByParent` on blocked re-open.

**Rationale**: Pixel-identical control across tabs is the requirement (FR-004/FR-005, SC-002); extraction of three duplicate usages is the simplest way to guarantee they never drift. `uncompleteTask` and both error messages already exist — no new API or copy.

**Alternatives considered**: copy the markup into the plan components (drifts apart again — the exact bug being fixed); restyle the plan's existing check button to look similar (still a second implementation). Rejected.

## D4: Current-time indicator

**Decision**: When the viewed day is today and "now" falls inside the window, render a horizontal accent line (theme accent color) across the track, absolutely positioned at `(nowMinute − windowStart) / 15 × 44px`. Refresh with a 60-second interval effect; render nothing on other days or when now is outside the window.

**Rationale**: FR-009; one absolutely-positioned element over the grid avoids disturbing slot layout; minute resolution matches the data's granularity. The TUI planner already marks "now", so cross-client behavior stays aligned.

**Alternatives considered**: re-render on a 1-second timer (pointless churn); omit entirely (declined — P3 story kept in scope).

## D5: Completed-state presentation on the timeline

**Decision**: Completed timed entries stay on the timeline with the filled-circle toggle and struck-through name — the Tasks tab's exact completed styling — replacing the current separate green check badge. Untimed completed entries remain hidden (current `groupPlan` behavior, unchanged).

**Rationale**: FR-005 and the clarified two-way toggle require the completed block to remain interactive; deleting the badge removes the last inconsistent affordance.

**Alternatives considered**: dimming/removing completed blocks from the timeline — hides where the day went; rejected.

## D6: Page composition & section order

**Decision**: `PlanPage` renders: day navigation header (unchanged) → untimed checklist section (moved above) → `PlanTimeline`. Loading/error/empty states unchanged (`Spinner`, `ErrorBanner`, `messages.planEmpty`). `PlanEntryRow` is repurposed as the untimed row (circle toggle + name link + remove); the timeline block is part of `PlanTimeline`.

**Rationale**: Untimed-above-grid matches the TUI (clarified 2026-07-02). Keeping per-entry error banners and the remove (trash) action as-is limits the diff to presentation.

**Alternatives considered**: a single component rendering both sections (harder to test, no reuse benefit). Rejected.
