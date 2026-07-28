# Phase 0 Research: External Commands for Plan Entry Boundaries

**Feature**: 070-plan-entry-hooks
**Date**: 2026-07-28

No `NEEDS CLARIFICATION` markers survived the Technical Context — the feature sits entirely inside the existing CLI/TUI module and reuses mechanisms already in the codebase. The research below records the decisions that shaped the design.

---

## D1: Which process fires the hooks?

**Decision**: The TUI (`internal/tui`) only. No daemon, no CLI-side firing.

**Rationale**: Boundaries fire at wall-clock moments, which requires a process that is alive at that moment. `twig`'s CLI commands are one-shot and exit immediately; the TUI is the only long-lived process. This also matches the precedent set by the pomodoro hooks, which fire from whichever process is running the timer. The spec records this as an explicit assumption.

**Alternatives considered**:
- *A background daemon* — would fire hooks with no session open, but introduces process supervision, IPC, single-instance locking, and a lifecycle story. Far beyond the request and a direct Principle I violation.
- *An `at`/`cron`-style external scheduler written on plan save* — would leak scheduling state outside the app and desynchronise the moment the plan is edited elsewhere.

---

## D2: Where do "today's entries" come from?

**Decision**: A dedicated background fetch of *today's* plan, cached in its own model state, independent of the Plan tab's visible day.

**Rationale**: Three existing behaviours rule out reusing `m.plan.entries`:
1. The Plan tab supports day navigation (`internal/tui/update.go:2049-2065`) — the user can be looking at tomorrow while today's boundaries need watching.
2. `m.plan.loaded` stays false until the Plan tab is first visited — a user who lives on the Tasks tab would never arm any hooks.
3. Background plan loads are already discarded unless the Plan tab is active (`handlePlanEntriesMsg`, `internal/tui/plan_update.go:518-524`).

A separate cache keyed to the local date sidesteps all three without touching the existing plan reducer.

**Alternatives considered**:
- *Reuse `m.plan.entries` when `m.plan.day == today`* — simplest, but silently stops working whenever the user browses days or stays off the Plan tab. Correctness loss too large for the simplicity gained.
- *Widen `planEntriesMsg` with a "for hooks" flag* — entangles the hook path with the visible-plan reducer, which already has four early-return branches for background loads. A distinct message type is cheaper to reason about.

**Cost control**: the fetch is dispatched only when at least one of the four keys is non-empty, so an unconfigured user issues zero extra RPCs (SC-006).

---

## D3: Firing cadence and the "not before, within 60s" window

**Decision**: A dedicated 15-second ticker, started in `Init` only when hooks are configured. Entry data is re-fetched from the same ticker, throttled to once per 60 seconds.

**Rationale**: FR-004 caps lateness at 60 seconds. The existing per-minute `autoRefreshTickMsg` heartbeat could carry the evaluation for free, but its phase is set by app launch time, so a boundary landing just after a tick waits nearly the full minute — technically compliant, but a "switch tasks now" alert arriving 55 seconds late reads as broken. A 15-second tick caps lateness at ~15s for one extra ticker, and the ticker only exists when the feature is in use.

**Alternatives considered**:
- *Piggyback on `autoRefreshTickMsg`* — zero new infrastructure, but up to ~60s late and it couples an unrelated subsystem to hook firing.
- *`tea.Tick` scheduled to the exact next boundary* — perfect precision, but every plan edit must cancel and reschedule the pending timer, and bubbletea has no timer cancellation. Rejected as complexity for a sub-minute gain.

---

## D4: Dedup and "no retroactive firing" — watermark instead of a fired-set

**Decision**: A single monotonic `watermark time.Time`. A boundary fires when `watermark < boundaryTime <= now`, and the watermark advances to `now` after each evaluation.

**Rationale**: This one comparison satisfies four requirements at once:
- **FR-005 (fire at most once)** — the watermark advances past a fired boundary, so the next tick's `>` test fails.
- **FR-006 (nothing retroactive)** — the watermark is seeded at app start, so boundaries that already passed are behind it from the first tick.
- **FR-017 / moved entries** — the schedule is re-derived from the freshly fetched entries every tick; an entry moved to a future time is simply a boundary ahead of the watermark and fires normally.
- **Sleep / clock jump** — paired with a "not more than 2 minutes late" grace check, a forward clock jump skips the boundaries it flew over instead of firing a burst.

Crucially this removes the need for a `map[boundaryKey]bool` fired-set and the key-invalidation rules that would come with it (what happens to the entry's key when it is renamed? moved? deleted and re-added with the same id?). Deleting that entire question is a direct Principle I win.

**Alternatives considered**:
- *A `fired` set keyed by (entry id, edge, minute)* — the obvious approach, but requires deciding when to evict entries, and gets the moved-entry case wrong unless the boundary time is part of the key (at which point the watermark does the same job with one field).

**Guard**: the watermark only advances on ticks where today's entries are actually loaded. Otherwise an in-flight first fetch could let the watermark sail past a boundary, silently swallowing it.

---

## D5: Placeholder substitution and shell escaping

**Decision**: A single left-to-right pass over the command string writing into a `strings.Builder`. `%s` → name verbatim; `%q` → name wrapped in double quotes with `"`, `\`, `$`, and `` ` `` backslash-escaped; `%t` → `at.Format("15:04")`; `%%` → `%`; anything else → both characters emitted unchanged.

**Rationale**: A single forward pass is inherently non-recursive (FR-013) — substituted text is written to the output and never re-scanned — so a task named `100%s done` cannot inject a placeholder. The four escaped characters are exactly the set the POSIX shell reinterprets inside double quotes; escaping more would corrupt names, escaping fewer allows command injection through a task name.

**Alternatives considered**:
- *`strings.NewReplacer` / repeated `strings.ReplaceAll`* — concise, but `ReplaceAll` chains are order-dependent and re-scan substituted text, reopening the injection hole that FR-013 closes.
- *Single-quote wrapping for `%q`* — safer still (only `'` needs handling), but the user's spec says double quotes, and single quotes would suppress the deliberate `$VAR` expansion users may want elsewhere in their command.
- *Escaping `%s` too* — then `%s` and `%q` would differ only by the quote characters, making `%s` useless inside a command the user has already quoted their own way.

---

## D6: Process execution and failure reporting

**Decision**: Copy the pomodoro hook runner verbatim in shape — `exec.Command("sh", "-c", expanded)` inside a `tea.Cmd`, with stdio deliberately **not** wired to the TUI's terminal, returning a `planHookErrMsg` on failure.

**Rationale**: `internal/tui/pomodoro.go:58-70` already solved this problem, including the non-obvious part: leaving `Stdout`/`Stderr` unset so a chatty hook cannot scribble over the alt-screen. Running inside a `tea.Cmd` gives FR-015 (non-blocking) for free, since bubbletea runs each command on its own goroutine. Reusing the shape keeps the two hook systems behaving identically, which is what a user who already configured pomodoro hooks will expect.

**Failure surface**: `planHookErrMsg` sets `m.notice` rather than `m.err`. A hook that exits non-zero is a note about the user's own config, not a twig error, and `m.notice` is the transient status-bar channel used for exactly this class of message. The text names the offending key so the user knows which line of their config to fix (FR-014), in the warm tone Principle IV requires.

---

## D7: Package placement

**Decision**: New files inside `internal/tui` (`plan_hook.go`, `plan_hook_test.go`), not a new `internal/planhook` package.

**Rationale**: The TUI is the only consumer. The pure logic (boundary derivation, placeholder expansion) is directly unit-testable in place — `internal/tui` already has 20+ test files and an `export_test.go` shim convention for exactly this. A new package would be an abstraction with one caller, which Principle I prohibits.

**Revisit if**: a future feature needs boundary firing outside the TUI, at which point the extraction is mechanical.
