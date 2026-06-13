# Research: Goal Status Updates

**Feature**: 051-goal-status-updates | **Date**: 2026-06-12

No open `NEEDS CLARIFICATION` items: the spec's `/speckit-clarify` session and a
UI brainstorming session resolved every material unknown. This file records the
decisions and their rationale.

## Decision 1 — Status modeled as a history log

- **Decision**: A goal has zero-or-more timestamped status updates; the newest
  is the "latest status". Not a single overwritten field.
- **Rationale**: "Timestamped entries" + "read it again later" (spec input) and
  the clarification answer both point to an append-style history.
- **Alternatives**: Single overwritable status field — rejected; loses history
  and the "read it again later" value.

## Decision 2 — Surface: TUI only

- **Decision**: Manage status updates only in the TUI Goals tab. No `twig goal`
  CLI subcommand, no web app changes.
- **Rationale**: The user called out the TUI specifically; clarification
  confirmed. Goals have no web surface today.
- **Alternatives**: TUI + CLI parity — deferred to a possible later feature to
  keep this one small (Principle I).

## Decision 3 — Editable + deletable updates

- **Decision**: Users can edit an update's text and delete an update. Edited text
  replaces the original under its original `created_at`; no separate "edited at".
- **Rationale**: Consistent with other text fields being editable; clarification
  confirmed. A separate edited timestamp is YAGNI for a personal tool.
- **Alternatives**: Append-only / immutable — rejected; fixing typos would force
  delete-and-re-add.

## Decision 4 — Reading model: latest inline + dedicated history view

- **Decision**: The goal detail pane shows a "Latest status" section; pressing
  `s` opens a dedicated, scrollable status-history view.
- **Rationale**: The detail pane does not scroll today and updates can be long
  (5000+ chars, SC-004). A dedicated view gives history real room without
  bloating the detail pane or pushing the Tasks section off-screen.
- **Alternatives**: All-inline scrollable detail (long histories bury tasks);
  modal overlay (less room than a full view). Both rejected.

## Decision 5 — History view shape: master list → full reader

- **Decision**: History is a master list (timestamp + first line, newest first)
  with a cursor; `enter` opens a full, scrollable reader for the selected update.
- **Rationale**: A compact list scans well across many updates; the reader gives
  long markdown entries a proper scroll surface. The list cursor is also the
  selection target for edit/delete (US3).
- **Alternatives**: Single scrolling stream with a per-update cursor — simpler
  but unwieldy when several long entries are concatenated.

## Decision 6 — Compose via `$EDITOR`

- **Decision**: New and edit both shell out to `$EDITOR` (temp `.md`) via the
  existing `openEditorCmd`. Quick-add is `S` from the detail pane; `n`/`e` add
  and edit from inside the history view.
- **Rationale**: This is exactly how goal/task descriptions are edited today —
  the established pattern for long markdown text. Reuse beats a bespoke inline
  multiline editor (Principle I/III).
- **Alternatives**: Inline TUI multiline editor — more code, inconsistent with
  descriptions; rejected.

## Decision 7 — API: embed latest, lazy history

- **Decision**: Add read-only `Goal.latest_status_update`, populated by
  `ListGoals`/`GetGoal`; fetch full history lazily via `ListGoalStatusUpdates`
  when the history view opens. New RPCs live on the existing `GoalService`.
- **Rationale**: Goals already load once; embedding the latest keeps the detail
  pane render at zero extra round-trips (Performance Goals). Status updates are
  sub-resources of a goal, so a separate service is unwarranted (Principle I/II).
- **Alternatives**: Fully lazy (fetch per goal on cursor move) — extra RPC churn
  and more state to keep the "latest" in sync; rejected.

## Decision 8 — Persistence & scoping

- **Decision**: `goal_status_updates` table, FK `goal_id → goals(id)` `ON DELETE
  CASCADE`. No denormalized `user_id`; every query scopes by joining to `goals`
  on `user_id`.
- **Rationale**: Cascade satisfies "delete goal deletes its updates" (FR-012) at
  the DB level. Scoping via join avoids a redundant `user_id` column that could
  drift (Principle I).
- **Alternatives**: Denormalized `user_id` column — simpler single-table queries
  but redundant; rejected. Application-level cascade — more code, weaker
  guarantee; rejected.
