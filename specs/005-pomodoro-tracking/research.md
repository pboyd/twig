# Research — Pomodoro Tracking (005)

All decisions below are resolved. No `NEEDS CLARIFICATION` items remain.

## 1. Enforcing "at most one active pomodoro per user"

**Decision**: Postgres partial unique index — `CREATE UNIQUE INDEX pomodoros_one_active_per_user ON pomodoros (user_id) WHERE end_at IS NULL;`. The handler catches `pgx`'s `unique_violation` (SQLSTATE 23505, constraint name `pomodoros_one_active_per_user`) and returns Connect `AlreadyExists`.

**Rationale**: The invariant is a true database constraint, so it belongs in the database. The partial index is exactly the "at most one row matching a predicate" pattern Postgres provides for this case. It is race-free without any application-level locking, and it costs almost nothing (a single tiny index covering at most one row per user).

**Alternatives considered**:
- *Application-level lock or SELECT-then-INSERT inside a transaction.* Requires a `SERIALIZABLE` transaction or an advisory lock; more code, easier to get wrong, slower, and still funnels through the same constraint check on insert. Rejected as more complex with no upside.
- *Boolean `active_pomodoro_id` column on `users`.* Two writes per state change, two places to keep in sync, and the FK direction reverses naturally with the table model. Rejected.

## 2. Where 25-minute clock authority lives

**Decision**: The server is authoritative. `StartPomodoro` records `start_at = NOW()`. `CompletePomodoro` sets `end_at = start_at + 25 minutes` (using the shared `pomodoro.Length` constant) — *not* `NOW()` — so a completed pomodoro's duration is exactly 25 minutes regardless of when the client called complete. `CancelPomodoro` sets `end_at = NOW()`. The CLI never sends a timestamp; it only sends RPC calls.

**Rationale**: FR-012 mandates `end_at == start_at + 25 minutes` for completed pomodoros. Letting the server compute it from its own stored `start_at` plus a server-side constant means a stale or slow client can't poison the record. Cancellation, by contrast, is genuinely "right now" — the client triggers it interactively and `NOW()` is the right answer.

**Alternatives considered**:
- *Client sends desired `end_at`.* Lets a buggy or malicious client write impossible durations; requires the server to re-validate anyway. Rejected.
- *Server sets `end_at = NOW()` on complete and stores a separate `duration` column.* Splits the source of truth and complicates the "remaining time" computation. Rejected.

## 3. External-cancel detection in the countdown UI

**Decision**: The CLI polls `GetActivePomodoro` once per second on the same tick as the countdown redraw. If the response indicates "no active pomodoro" (or the active pomodoro's `id` differs from the one the UI is attached to), the UI exits cleanly with a message.

**Rationale**: 1Hz polling matches the countdown's own redraw cadence, so it adds zero perceivable latency and one trivial request per second per active user. Detection happens within 1–2 seconds — well under SC-005's 5-second bound — at a load (≤1 req/s per active CLI) that is irrelevant at this project's 1–2 user scale.

**Alternatives considered**:
- *Server-Sent Events or websockets.* Adds a new transport, a new dependency, and lifecycle complexity for a problem that 1Hz polling solves trivially. Rejected on Principle I.
- *Long-poll with a server-side wait.* Cheaper in steady state but requires the handler to grow a wait/notify path. Rejected for the same reason.
- *No detection (user just sees a stale countdown until it expires).* Violates FR-039 and SC-005. Rejected.

## 4. Raw terminal input for `c` / `q`

**Decision**: Use `golang.org/x/term` to put `stdin` in raw mode while the countdown UI is active, restore it on exit (including on signal/panic via `defer`). Read single bytes; treat `c` / `C` as cancel and `q` / `Q` as quit. Any other key is ignored. If `stdin` is not a TTY (`term.IsTerminal(int(os.Stdin.Fd())) == false`), skip raw mode and run a non-interactive countdown that still polls and still completes/cancels on the server but does not consume keystrokes.

**Rationale**: `x/term` is the smallest standard-adjacent option (it lives under `golang.org/x` and has been treated as standard-library-equivalent throughout this project's earlier features). Single-keystroke reads are exactly the input model the spec describes; a full TUI framework would dwarf the actual UI logic.

**Alternatives considered**:
- *Bubble Tea / tview.* A dependency 10× larger than the code that uses it; introduces an event-loop model the rest of the CLI does not need. Rejected.
- *Line-buffered reads (user must press Enter).* Doesn't match the spec ("pressing `c`", "pressing `q`"). Rejected.
- *Signal-based (SIGINT to cancel).* Conflates "I want to cancel the pomodoro" with "I want to abort the process" and gives no way to quit without canceling. Rejected.

## 5. Exposing pomodoro count/history

**Decision**: Extend `GetTaskResponse` with two extra fields:
- `int64 completed_pomodoro_count` — number of completed pomodoros for the task.
- `repeated Pomodoro pomodoros` — full list (completed and canceled), ordered by `start_at`.

No separate `ListPomodoros` RPC is introduced.

**Rationale**: The countdown UI needs the task name, the task's estimate, *and* the count of previously completed pomodoros — three pieces of data that all come from "this task". Bundling them into the existing `GetTask` response means the UI makes one call to render, not two; it also keeps the API surface smaller, in line with Principle I. The full list is cheap to include (small N) and removes the need to add another endpoint later for "history" features.

**Alternatives considered**:
- *Dedicated `ListPomodorosByTask` RPC.* Justifiable if pomodoro counts could grow into the thousands per task, but at this project's scale a task with even 100 pomodoros is far beyond realistic — and even at that size the payload is trivial. Rejected as speculative API surface.
- *Count only, no list.* Forced a follow-up feature for any history view (or for a CLI `history` subcommand). Rejected for negligible savings.

## 6. CLI `pom cancel` and `pom resume` signatures

**Decision**: Both take no positional arguments. Each operates on the user's single active pomodoro (which the server already knows). `cancel` errors with exit 1 if no pomodoro is active; `resume` does the same. `start <task_id>` still takes a task_id because it identifies which task the new pomodoro belongs to.

**Rationale**: This matches the clarification recorded in `spec.md` (Q3, Q5 follow-up). The single-active invariant means the user can never be ambiguous about *which* pomodoro `cancel`/`resume` refers to, so requiring a task_id only adds a footgun.

**Alternatives considered**:
- *Keep task_id on `cancel`/`resume` for symmetry with `start`.* Rejected per clarification — symmetry is not worth the footgun.

## 7. `--exec` execution model

**Decision**: Synchronous; CLI calls `os/exec.Command(<sh>, "-c", cmd)` (or directly if `cmd` parses as `argv` — keep it as `sh -c` for shell semantics), inheriting the CLI's stdout/stderr. On non-zero exit, print a one-line warning naming the command and the exit code; the CLI itself exits 0 (the pomodoro is recorded complete regardless).

**Rationale**: Matches the clarification recorded in `spec.md` (Q2). `sh -c` gives users the shell semantics they expect for one-liners like `notify-send "Break!"`; inheriting stdio means notifier/sound errors are visible.

**Alternatives considered**:
- *Detached / async.* The user often runs short notifier commands; detaching and discarding output makes failures invisible. Rejected per clarification.
- *Propagate the exec exit code.* Conflates two distinct outcomes (the pomodoro succeeded; the side-effect failed). Rejected per clarification.

## 8. Stale-active-pomodoro on `resume`

**Decision**: At the start of `runResume`, the CLI calls `GetActivePomodoro`. If `now - start_at > pomodoro.Length`, it immediately calls `CompletePomodoro` and exits without displaying a countdown.

**Rationale**: FR-038. The API does not run timers itself, so a pomodoro started >25 minutes ago and never explicitly completed is effectively "should have already finished". The resume path is the natural place to enforce that. Doing this on the *client* (rather than on every API call) avoids touching unrelated read paths and matches the spec's "API only tracks data, it doesn't time anything by itself".

**Alternatives considered**:
- *Server-side auto-completion on read.* Surprising side effects from `GET` calls. Rejected.
- *Background sweeper.* Adds a background process for what is effectively a no-op until a user explicitly resumes. Rejected on Principle I.
