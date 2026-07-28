# Contract: `[plan]` Config Section

**File**: `$XDG_CONFIG_HOME/twig/config.toml` (falls back to `~/.config/twig/config.toml`)

**Format**: TOML 1.0

**Status**: Stable for this feature. Extends the base schema documented in `specs/015-pomodoro-config-file/contracts/config-schema.md` (note: that document predates the `todo` → `twig` rename; read `todo_*` as `twig_*`). This contract adds one new top-level table and changes nothing existing.

---

## Schema

```toml
# --- Plan entry lifecycle hooks -------------------------------------------
# All four keys are OPTIONAL. Absent or empty means "do nothing".
# These fire only while the interactive TUI is running.

[plan]

# Runs at the START of a timed plan entry that is linked to a task.
on_task_start = 'notify-send "%t: start %q"'

# Runs at the END of a timed plan entry that is linked to a task.
# End time = start time + duration.
on_task_end = 'notify-send "%t: time to wrap up %q"'

# Runs at the START of a timed plan entry NOT linked to a task (an event).
on_event_start = 'notify-send "%t: %q is starting"'

# Runs at the END of a timed plan entry NOT linked to a task (an event).
on_event_end = 'notify-send "%t: %q is over"'
```

---

## Placeholders

Substituted into the command string immediately before execution.

| Placeholder | Expands to | Example |
|---|---|---|
| `%s` | The entry's display name, **verbatim and unescaped** | `Write report` |
| `%q` | The entry's display name, wrapped in double quotes and shell-escaped | `"Write report"` |
| `%t` | The boundary's scheduled time, zero-padded 24-hour `HH:MM` | `09:05`, `14:00` |
| `%%` | A single literal `%` | `%` |
| `%` + anything else | Passed through unchanged | `%z` → `%z` |

**Display name resolution**: the plan entry's own name when set; otherwise the name of the task it links to.

**`%t` is the *scheduled* time, not the wall-clock time of execution.** A hook that fires a few seconds late still reports the minute the plan says.

**`%s` vs `%q`**:
- `%q` is the safe default. The name is wrapped in `"` and the characters `"`, `\`, `$`, and `` ` `` are backslash-escaped, so any entry name reaches the command verbatim.
- `%s` inserts the raw name with no quoting or escaping. Use it inside quoting you have supplied yourself; keeping it shell-safe is your responsibility.

**Substitution is not recursive.** Text introduced by a substitution is never re-scanned, so an entry named `100%s done` cannot inject a placeholder.

### Worked example

Config:
```toml
[plan]
on_task_start = 'notify-send "%t: start %q"'
```

Plan entry: task named `Fix "auth" $bug`, scheduled 14:00–14:30.

At 14:00 the shell receives:
```sh
notify-send "14:00: start \"Fix \\\"auth\\\" \\$bug\""
```
and the notification reads:
```text
14:00: start "Fix "auth" $bug"
```

---

## Execution semantics

| Aspect | Behaviour |
|---|---|
| Shell | Run as `sh -c <expanded command>`. Pipes, `&&`, redirects, and env expansion all work. |
| Timing | Fires at or after the boundary, never before, and within 60 seconds of it. |
| Frequency | Each boundary fires at most once per session. |
| Retroactive | Boundaries already past when the TUI started, or past when an entry was added/rescheduled, do **not** fire. |
| Session scope | Fires only while the TUI is running. There is no daemon; boundaries passing with no session open are missed and never replayed. |
| Day scope | Only entries on the current local day are watched. The watched set follows the local date across midnight. |
| Untimed entries | Never fire. A task entry with no start time has no boundaries. |
| Collision order | When an end and a start land on the same minute, the end hook runs first. |
| Blocking | Hooks run detached from the TUI's terminal and never block the interface or delay a later boundary. |
| Failure | A non-zero exit or a missing binary produces a non-fatal status-bar notice naming the key. The session, the plan, and every later boundary are unaffected. |
| Output | The hook's stdout/stderr are **not** connected to the terminal — they cannot corrupt the alt-screen. Redirect to a file if you need them. |

---

## Precedence

| Setting | Env var | Config key | Default |
|---|---|---|---|
| Plan hooks | *(none)* | `[plan].on_*` | empty (no-op) |

Plan hooks are read from the **root** config table, never from a `[profile.<name>]` table — identical to the existing `[pomodoro]` hooks.

---

## Reader contract

| Condition | Behaviour |
|---|---|
| No `[plan]` section | Treated as all-empty. No ticker started, no extra requests issued, no warnings. Behaviour identical to before this feature. |
| Unknown key inside `[plan]` | Ignored, no warning (forward compatibility). |
| Non-string value for a key | TOML decode error, reported by the existing `config: failed to parse <path>: <toml error>` path. |
| Same command in several keys | Allowed. Each key fires independently. |

---

## Compatibility

Purely additive. No existing key changes meaning, no server-side change, no API change, and an existing config file without a `[plan]` section behaves exactly as it does today.
