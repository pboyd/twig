# Implementation Plan: Plan Objectives and Notes

**Branch**: `074-plan-objective-notes` | **Date**: 2026-08-06 | **Spec**: [spec.md](./spec.md)

**Input**: Feature specification from `/specs/074-plan-objective-notes/spec.md`

## Summary

Give each plan day two optional markdown fields of its own — a short **objective** and
long-form **notes** — and surface them on the TUI planning tab and, for the objective, on
the CLI.

The day itself has no record today: a plan day exists only as the set of `plan_entries`
rows sharing a `(user_id, day)`. So the work starts with a new `plan_days` table keyed on
that same pair, reached through two focused upsert RPCs (`SetPlanObjective`, `SetPlanNotes`)
and read by piggybacking a `PlanDay` message on the `ListPlanEntriesResponse` the TUI and
CLI already fetch for every day — no extra round trip on the auto-refresh path.

On the planning tab, the objective gets a full-width band above the existing grid/details
row, omitted entirely when unset so days without one look exactly as they do now; `o` opens
a single-field editor in that same slot where Enter saves and Esc cancels. The right column
splits, with a Notes pane under Details; `n` turns the whole right column into a textarea
that saves with `ctrl+s` and reaches `$EDITOR` with `ctrl+g`, reusing the existing helper
untouched. Both fields render through the same markdown renderer as task descriptions.
On the CLI, `objective` joins the `plan` subcommand switch after `--date` parsing, so it
inherits the flag and the today-default for free.

## Technical Context

**Language/Version**: Go 1.24 (three modules: root CLI/TUI, `api/`, `services/twig/`)

**Primary Dependencies**: ConnectRPC + protobuf (`api/gen/`), Bubble Tea v2 / Bubbles v2 /
Lipgloss v2 (TUI), pgx/v5 + sqlc (server), golang-migrate, the in-repo
`internal/markdown` renderer

**Storage**: PostgreSQL. One new table, `plan_days (user_id, day, objective, notes)`,
migration `000015`

**Testing**: `go test ./...` at the repo root (CLI/TUI unit tests) and in `services/twig/`
(handler integration tests, which require `DATABASE_URL` and skip silently without it)

**Target Platform**: Linux; terminal (TUI/CLI) plus the HTTP/2 server

**Project Type**: Multi-module Go monorepo — CLI/TUI client, shared API module, server.
The React SPA is out of scope for this feature

**Performance Goals**: No perceptible delay opening the planning tab or moving between days
(SC-008). The read path adds zero round trips — the new fields ride on the existing
`ListPlanEntries` call

**Constraints**: Layout must survive any content length and the narrowest supported
terminal (SC-006); days with no objective must consume no extra vertical space (SC-003);
generated code (`api/gen/`, `services/twig/internal/db/`) is never hand-edited

**Scale/Scope**: One migration, one proto message + two RPCs, one handler file touched,
one CLI subcommand, and the planning-tab view/update path. At most one `plan_days` row per
user per touched day

## Constitution Check

*GATE: Must pass before Phase 0 research. Re-check after Phase 1 design.*

| Principle | Status | Notes |
|-----------|--------|-------|
| I. Simplicity / YAGNI | ✅ | One table, `NOT NULL DEFAULT ''` columns so "absent" has exactly one spelling, no row-deletion logic, no new abstraction layers. The two setters are separate rather than one optional-field updater precisely to avoid presence-tracking. Read piggybacks an existing call instead of adding a `GetPlanDay`. No Complexity Tracking entries needed |
| II. API-First Design | ✅ | `contracts/plan-day.md` (proto + CLI) and `contracts/tui-planning-layout.md` (UI) are committed before any implementation task. Proto changes are additive; field numbers and RPC names are fixed by the contract |
| III. UI/UX Consistency | ✅ | Both panes use the existing `paneBox`/row-join layout and the shared markdown renderer and theme; no ad-hoc colors. `ctrl+s`/`esc`/`ctrl+g` keep their program-wide meanings. The objective editor's Enter-to-save departs from the buttons-and-tab form pattern — justified because the field is single-line, matching the filter input and date prompt, and requested explicitly in the spec. The CLI subcommand inherits `--date` and the today-default from the plan command, so it behaves as a user would predict |
| IV. Playful User Messages | ✅ | Save confirmations, empty states, and errors carry the warm tone, with error text still actionable (register documented in `contracts/tui-planning-layout.md` and `research.md` D8). One deliberate exception: the CLI objective **read** prints bare text with no decoration so it can be piped or used in a shell prompt — noted in the contract |

**Post-Phase 1 re-check**: unchanged. The design added no abstraction beyond the table,
the message, and two RPCs; contracts exist for both the wire and UI surfaces; no new
styling or key semantics were introduced.

## Project Structure

### Documentation (this feature)

```text
specs/074-plan-objective-notes/
├── plan.md                          # This file
├── research.md                      # Phase 0: D1–D8 decisions + confirmed codebase facts
├── data-model.md                    # Phase 1: plan_days table, queries, client state
├── quickstart.md                    # Phase 1: build loop and manual verification
├── contracts/
│   ├── plan-day.md                  # Phase 1: proto messages, RPCs, CLI contract
│   └── tui-planning-layout.md       # Phase 1: planning tab layout, modes, keys
├── checklists/
│   └── requirements.md              # Spec quality checklist (all passing)
├── spec.md
└── tasks.md                         # Phase 2 (/speckit-tasks — not created here)
```

### Source Code (repository root)

```text
api/
├── proto/plan/v1/plan.proto              # + PlanDay, + 2 RPCs, + ListPlanEntriesResponse.day
└── gen/plan/v1/                          # regenerated: make proto

services/twig/
├── db/
│   ├── migrations/
│   │   ├── 000015_plan_days.up.sql       # new
│   │   └── 000015_plan_days.down.sql     # new
│   └── queries/plan.sql                  # + GetPlanDay, + 2 upserts
├── internal/db/                          # regenerated: sqlc generate
└── internal/handler/
    ├── plan.go                           # + SetPlanObjective, + SetPlanNotes, + day on List
    └── plan_test.go                      # integration tests (need DATABASE_URL)

internal/                                 # root module: CLI + TUI
├── cli/
│   ├── plan.go                           # + objective subcommand, + usage line
│   └── plan_test.go
└── tui/
    ├── model.go                          # + planState fields, + 2 planMode values
    ├── keymap.go                          # + PlanObjective (o), + PlanNotes (n), + help
    ├── view.go                           # objective band + right-column split (both branches)
    ├── plan_view.go                      # objective/notes pane rendering
    ├── plan_update.go                    # editor modes, save cmds, planEntriesMsg.day
    ├── update.go                         # editorFinishedMsg routing for planNotesEdit
    └── *_test.go                         # plan_view_test, plan_update_test, autorefresh_apply_test
```

**Structure Decision**: The existing three-module layout is unchanged. Work lands in the
established places for each tier — proto in `api/`, migration/queries/handler in
`services/twig/`, CLI and TUI in the root module's `internal/`. No new package is created:
the objective and notes are plan-tab concerns and belong with the plan code that already
exists. `services/twig-web/` is untouched (FR-030).

## Implementation Order

Contract-first, then bottom-up, so each tier is testable before the one above it lands:

1. **Proto** — `api/proto/plan/v1/plan.proto` per `contracts/plan-day.md`, then `make proto`.
2. **Storage** — migration `000015`, queries in `plan.sql`, `sqlc generate`.
3. **Handler** — `SetPlanObjective`, `SetPlanNotes`, populate `ListPlanEntriesResponse.day`;
   integration tests for upsert, clear, trim, the 255-rune limit, field isolation, and
   per-user isolation.
4. **CLI** — `objective` subcommand and usage; tests for read, write, clear, silent-empty,
   and bare-text output.
5. **TUI state** — `planState` fields, the two `planMode` values, `planEntriesMsg.day`, and
   the background-refresh rule that an open draft is never overwritten.
6. **TUI keys and editors** — `o`/`n` bindings gated on `planList`, the two editors, save and
   cancel commands, `ctrl+g` routing, help entries.
7. **TUI layout** — objective band and right-column split in both the styled and unstyled
   branches of `viewPlanning`.
8. **Docs** — update `AGENTS.md` if the planning tab's documented behaviour needs it.

Steps 4 and 5–7 are independent of each other once step 3 lands.

## Risks

| Risk | Mitigation |
|---|---|
| Handler tests skip silently without `DATABASE_URL`, so a green `go test ./...` proves nothing about the new RPCs | `quickstart.md` makes exporting it a prerequisite; the definition of done requires the server suite to run with it set |
| The objective band changes height arithmetic feeding both `paneBox` calls, risking off-by-one layout breakage | Height is subtracted once, before the existing computation; tests assert byte-identical output on days with no objective (SC-003) and intact borders at small sizes |
| The unstyled branch of `viewPlanning` is a separate code path and is easy to forget | Both branches are named explicitly in `contracts/tui-planning-layout.md` and each gets a test |
| Auto-refresh could overwrite an in-progress objective or notes draft | The apply rule is stated in the contract and covered in `autorefresh_apply_test.go`, alongside the existing wrong-day-discard test |
| Adding `n` to the planning tab could shadow `NewSub` on the Tasks tab | The planning tab dispatches through its own key switch; a regression test pins `n` on Tasks |

## Complexity Tracking

No Constitution Check violations. Table intentionally empty.
