# Research: Pomodoro Config File

## R1 — Config file location

**Decision**: `$XDG_CONFIG_HOME/todo/config.toml`, falling back to `~/.config/todo/config.toml` when `XDG_CONFIG_HOME` is unset.

**Rationale**: Matches the XDG Base Directory Specification, which Linux users expect and macOS users accept. The Go standard library exposes this directly via `os.UserConfigDir()`, which already implements the XDG fallback chain. No new dependency, no homegrown path logic.

**Alternatives considered**:
- `~/.todorc` — single-file dotfile is simpler but pollutes `$HOME` and conflicts with the project's existing minimal footprint there.
- `~/.config/todo/config.yaml` — YAML adds a heavier parser dependency than TOML and is more error-prone for users hand-editing it (significant whitespace).
- A `--config` CLI flag — out of scope for v1; can be added later without breaking the default path.

## R2 — Config file format

**Decision**: TOML, parsed by `github.com/BurntSushi/toml`.

**Rationale**: TOML is unambiguous for hand-editing, has clear section semantics that fit `[pomodoro]` hooks cleanly, and `BurntSushi/toml` is the de-facto Go TOML library (BSD-licensed, single file's worth of public API, used by `cargo` ecosystem tooling). YAML is heavier and indentation-sensitive; JSON disallows comments, which we want for an example-driven config.

**Alternatives considered**:
- YAML (`gopkg.in/yaml.v3`) — bigger dependency surface, whitespace sensitivity.
- JSON — no comments, awkward to hand-edit.
- Env-file (`.env`) — flat, but can't naturally group pomodoro hooks under a section.

## R3 — Precedence between env vars and config

**Decision**: Environment variable wins when both are set. Specifically: `TODO_API_KEY` overrides `api_key`; `TODO_ADDR` overrides `api_url`.

**Rationale**: FR-009 mandates this for backward compatibility. The existing test suite (`cli_test.go`) already exercises env-var-driven setup; keeping env vars authoritative when present means those tests stay green and existing scripted deployments don't break.

**Alternatives considered**:
- Config wins over env — would silently change behavior for users with existing env-var setups.
- First-write-wins / error on conflict — surprising and unhelpful.

## R4 — Hook execution semantics

**Decision**: Hooks run synchronously via `exec.Command("sh", "-c", configuredString)`. Failure (non-zero exit, command not found) prints a warning to stderr and continues; the pomodoro is unaffected.

**Rationale**:
- Reuses the existing `execHook` function already in `internal/cli/pom.go`. No new abstraction.
- Synchronous execution keeps ordering predictable: start hook completes before the countdown UI starts; complete/cancel hook completes before the user sees a prompt. This matches the spec's Assumptions.
- The "warn but don't fail" policy comes directly from FR-008.

**Alternatives considered**:
- Async/background hook execution — invites race conditions where a DND-off hook hasn't completed before the next pomodoro starts.
- Aborting the pomodoro on hook failure — violates FR-008 and is a poor UX for transient hook problems.

## R5 — Start-hook ordering relative to server start

**Decision**: The start hook runs *after* the server `StartPomodoro` RPC succeeds, before the countdown UI renders. If the RPC fails, the start hook does NOT run.

**Rationale**: Edge case in the spec: "A start hook is configured but the pomodoro server-side start call fails. The start hook must NOT run, because no pomodoro actually began." Running the hook only after a confirmed start guarantees no spurious DND toggles when the server is unreachable or the task is invalid.

**Alternatives considered**:
- Fire the start hook eagerly before the RPC — fails the spec edge case.
- Fire the start hook after the *first* countdown tick — gratuitously delayed; offers no benefit.

## R6 — TUI hook delivery

**Decision**: The TUI's pomodoro launch path (`internal/tui/pomodoro.go::execPomodoroStart` / `execPomodoroResume`) delegates to the same CLI runner that handles `todo pom start` / `resume`. That runner already calls `runCountdownAndComplete`, where the hook calls will live, so the TUI inherits the behavior for free.

**Rationale**: One code path, one set of tests. FR-012 (identical CLI/TUI behavior) becomes a structural guarantee rather than a runtime promise.

**Alternatives considered**:
- Re-implement hook firing in the TUI — duplicates code, risks drift.

## R7 — Removal of `--exec` flag

**Decision**: Hard removal. Passing `--exec` to `todo pom start` or `todo pom resume` produces an "unknown flag" error.

**Rationale**: FR-003 mandates this. Keeping a deprecated alias means two ways to specify the same thing — extra code, extra docs, extra surface to test. The user explicitly chose removal in the original request.

**Alternatives considered**:
- Soft-deprecate with a warning — adds maintenance burden and a flag-parsing path that just funnels to the new code anyway.
