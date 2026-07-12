# Research: Task Filters

**Feature**: 062-task-filters | **Date**: 2026-07-10

All Technical Context unknowns resolved. Decisions below were made against the current codebase (proto contract in `api/proto/task/v1/task.proto`, TUI visibility logic in `internal/tui/tree.go` `buildVisible`, server list path `handler/task.go` → `Queries.ListTasks`).

## D1: API shape — new `FilterTasks` RPC returning matched task IDs

**Decision**: Add a `FilterTasks` RPC to the existing `task.v1.TaskService`. Request carries the filter `expression`, the client's `show_all` state, and the client's current local day (`today`, `YYYY-MM-DD`). Response carries the matched task IDs (`repeated int64`). Ancestor-context display (FR-003) is computed by the client from the task tree it already holds.

**Rationale**:
- The TUI (and web app) already fetch the full task list via `ListTasks` and build the tree client-side; per-keystroke filtering only needs to know *which* tasks match, not re-transfer full `Task` payloads. IDs keep responses tiny and the client's tree state (cursor, expansion) stable.
- Matching semantics — the part that must be identical across clients (FR-011, SC-005) — stay 100% server-side, including the show-all default rule (FR-008). What the client does with the matched set (show ancestors, indent) is presentation, specified once in `contracts/filter-grammar.md` so future clients render identically.
- `ListTasks` stays untouched; its documented contract ("returns every stored task") is depended on by the web app and CLI.

**Alternatives considered**:
- *Add a `filter` field to `ListTasksRequest`* — muddies a stable contract, forces full Task messages per keystroke, and conditionally changes documented semantics. Rejected.
- *Return full Task objects (matches + ancestors)* — duplicates data the client has, makes the server compute presentation (ancestor chains), and churns client tree state per keystroke. Rejected.

## D2: Evaluation strategy — in-memory over the user's tasks, no SQL generation

**Decision**: The handler loads the caller's tasks with the existing `ListTasks` sqlc query, then a new pure-Go package `services/twig/internal/filter` parses the expression and evaluates it in memory (building parent/child maps for `^` transitivity and effective-goal resolution).

**Rationale**:
- Per-user task counts are hundreds to low thousands; in-memory evaluation is microseconds and trivially correct. The 300 ms budget (SC-002) is spent on the HTTP round-trip, not evaluation.
- Translating user expressions to SQL means dynamic SQL against sqlc's static-query model, an injection surface, and recursive CTEs for `^` — large complexity for zero measurable benefit at this scale (Principle I).
- A pure package with no DB or ConnectRPC imports is table-driven-testable, satisfying the repo convention that tests need no running database.

**Alternatives considered**:
- *Dynamic SQL / recursive CTEs* — rejected per above.
- *Client-side evaluation in the TUI* — violates FR-011 (web app must reuse semantics). Rejected.

## D3: Parser — hand-written lexer + recursive-descent, no new dependency

**Decision**: Implement the grammar (see `contracts/filter-grammar.md`) with a hand-written tokenizer and recursive-descent parser producing a small AST (`AND` list of conditions; condition = text term | field-op-value, with optional `^` on relationship fields).

**Rationale**: The grammar has one binary operator (`AND`), six comparison operators, four fields, and three literal types. That is an afternoon of straightforward Go and ~zero maintenance. A parser-generator or combinator library (participle, goyacc) adds a dependency and a learning surface for a grammar this small (Principle I). The AST leaves room to add `OR`/`NOT`/parens later without re-architecting.

**Alternatives considered**: participle (new dep, reflection-driven magic), goyacc (build-step complexity). Both rejected as disproportionate.

## D4: Relationship semantics — `parent_id`, `goal_id`, and `^`

**Decision**:
- `parent_id=N` matches tasks whose direct parent is N. `^parent_id=N` matches all descendants of N at any depth (children, grandchildren, …), excluding N itself.
- `goal_id=N` matches tasks *directly associated* with goal N (the association root — the only tasks whose stored `goal_id` is set, per the proto contract). `^goal_id=N` matches every task in the subtree of any association root for goal N (root included).
- Transitivity is resolved by walking parent/child maps built from the loaded task set.

**Rationale**: Mirrors how `goal_id` is stored (only on the association root; descendants inherit client-side per `task.proto`) and matches the spec's examples: `^goal_id=1` → "the whole tree, not only direct subtasks". The non-`^` forms are the natural "direct" readings.

## D5: Status semantics — `completed`, `snoozed`, and the reference day

**Decision**:
- `completed=true` ⇔ `completed_at` is set; `completed=false` ⇔ unset.
- `completed <op> YYYY-MM-DD` compares `completed_at` against midnight UTC of the given day (timestamps are stored UTC); any date comparison implies the task is completed.
- `snoozed=true` ⇔ `snooze_until` is set **and** is strictly after the request's `today` (the client's current local day) — exactly the rule `buildVisible` and the proto comment use ("clients hide the task while this day is strictly after the client's local current day"). `snoozed=false` is the complement.
- The request carries `today` because the server cannot know the client's local day; passing it keeps snooze semantics identical to the existing unfiltered view and deterministic for tests.

**Rationale**: Reuses the one snooze rule the product already has rather than inventing a second one. UTC-midnight date comparison is the simplest defensible reading of `completed < 2026-01-01`; sub-day precision is out of scope for a day-granular syntax.

## D6: Show-all default — applied server-side, per attribute

**Decision**: The evaluator implements FR-008 itself: when the expression lacks a `completed` condition, eligibility defaults from `show_all` (off → only incomplete); same independently for `snoozed`. Explicit conditions replace only their own attribute's default (clarification Q1). While a filter is active the TUI does **not** apply its own `buildVisible` status logic — the matched-ID set is the whole truth about what matches.

**Rationale**: Splitting the default rule across tiers is exactly the "thorny" trap the spec flags; one implementation, one truth (SC-005).

## D7: TUI integration — per-keystroke request with generation counter; matched set drives display

**Decision**:
- `/` (currently unbound in the Tasks tab) opens a filter `textinput` (same `charm.land/bubbles/v2/textinput` widget as the plan-entry and edit forms), rendered as a bar in the Tasks tab styled from `theme.go`.
- Every input change fires a `FilterTasks` command tagged with an incrementing generation; responses with a stale generation are discarded. No debounce — localhost round-trips are far under the 300 ms budget, and the generation counter alone guarantees last-write-wins.
- On success, the model stores the matched-ID set; `buildVisible` gains a filtered mode that emits exactly: matching tasks + their ancestor chains, tree structure preserved, expansion state ignored (a match inside a collapsed subtree must be visible; FR-003).
- On `InvalidArgument` (parse error), the model keeps the previous matched set and flags the input invalid (FR-009). Other errors surface through the TUI's existing error display.
- Key handling per clarifications: input focused → `Enter` applies + returns focus to list, `Esc` clears; list focused with active filter → `/` reopens pre-filled, `Esc` clears. Mutations while filtered re-fire the current expression after the mutation's refresh (FR-013).

**Rationale**: Minimal moving parts (a set of IDs and an integer), reuses every existing widget/pattern, and keeps all semantic decisions on the server.

## D8: Invalid-expression signaling — ConnectRPC `InvalidArgument`

**Decision**: Parse/validation failures return `connect.CodeInvalidArgument` with a playful but actionable message (Principle IV). The TUI maps that code to the "invalid expression" indicator; it never renders the raw error mid-typing.

**Rationale**: Uses the transport's native error model — no custom "valid: false" envelope (Principle I). The web app later gets the same well-typed error for free.
