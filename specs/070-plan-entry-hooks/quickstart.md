# Quickstart: External Commands for Plan Entry Boundaries

**Feature**: 070-plan-entry-hooks

## For users

Add a `[plan]` section to `~/.config/twig/config.toml`:

```toml
[plan]
on_task_start  = 'notify-send "%t: start %q"'
on_task_end    = 'notify-send "%t: wrap up %q"'
on_event_start = 'notify-send "%t: %q is starting"'
on_event_end   = 'notify-send "%t: %q is over"'
```

Restart the TUI. Any timed entry on today's plan now announces itself at its start and end.

Placeholders: `%s` name (raw), `%q` name (quoted and shell-safe), `%t` time as `HH:MM`, `%%` a literal `%`. Full details in [contracts/config-schema.md](contracts/config-schema.md).

**Two things to know**: hooks fire only while the TUI is running, and a boundary that has already passed when you launch never fires retroactively.

### Try it in two minutes

```bash
# 1. Configure a hook that writes to a file (works headless, unlike notify-send)
cat >> ~/.config/twig/config.toml <<'EOF'

[plan]
on_task_start = 'echo "%t START %s" >> /tmp/twig-hooks.log'
on_task_end   = 'echo "%t END %s"   >> /tmp/twig-hooks.log'
EOF

# 2. Launch the TUI, go to the Plan tab, add a task entry starting 1-2 minutes
#    out with a 1-minute duration. Leave the TUI open.

# 3. Watch it fire
tail -f /tmp/twig-hooks.log
```

---

## For implementers

### Files touched

| File | Change |
|---|---|
| `internal/config/config.go` | Add `PlanConfig` struct + `Plan` field on `Config`; `enabled()` predicate |
| `internal/config/config_test.go` | Parsing, empty-section, unknown-key, profile-isolation cases |
| `internal/tui/plan_hook.go` | **New.** Boundary derivation, watermark evaluation, placeholder expansion, ticker, fetch cmd, runner |
| `internal/tui/plan_hook_test.go` | **New.** Unit tests for the pure logic |
| `internal/tui/model.go` | Add `planHookState` field; wire `cfg` in `newModel` |
| `internal/tui/tui.go` | Pass `cfg.Plan` into `newModel` |
| `internal/tui/update.go` | `Init` starts the ticker when enabled; handle the three new messages |
| `internal/tui/export_test.go` | Shims for the unexported helpers under test |
| `CLAUDE.md` | Point the SPECKIT plan reference at this feature |

### Key implementation notes

**The whole firing rule is one predicate.** No fired-set, no eviction policy:

```go
watermark.Before(b.at) && !b.at.After(now) && now.Sub(b.at) <= 2*time.Minute
```

`watermark` is seeded to now at model init (so nothing retroactive fires), and advances to `now` after each evaluation — but **only on ticks where today's entries are loaded**. Advancing it while the first fetch is still in flight would silently swallow a boundary.

**Placeholder expansion is one forward pass** into a `strings.Builder`. Do not reach for `strings.ReplaceAll` chains — they re-scan substituted text and reopen the injection hole that FR-013 closes.

**Copy the pomodoro runner's shape** (`internal/tui/pomodoro.go:58-70`), including the non-obvious part: leave `Stdout`/`Stderr` unset so a chatty hook cannot scribble over the alt-screen.

**Do not reuse `m.plan.entries`.** The Plan tab has day navigation and stays unloaded until first visited; the hook watcher needs its own cache of *today*. See research D2.

### Testing

Everything worth testing is pure. `nowOrDefault()`/`m.nowFunc` already exists for clock injection (`internal/tui/update.go:2887`).

```bash
go test ./internal/config/... ./internal/tui/...
```

Cases that must be covered:

| Area | Cases |
|---|---|
| Expansion | each of `%s` `%q` `%t` `%%`; unknown `%z` passthrough; trailing bare `%`; non-recursion (`100%s done`); escaping of `"` `\` `$` `` ` `` |
| Derivation | untimed → no boundaries; task vs event routing; end = start + duration; name falls back to the task's; zero duration |
| Watermark | fires once and only once; nothing retroactive at startup; skips a >2min-late boundary (clock jump); does not advance while unloaded |
| Ordering | end before start on a collision |
| Config | absent `[plan]` → disabled, no ticker, no fetch; unknown key ignored; profile does not override |

### Manual verification

```bash
go build -o twig ./cmd/twig && ./twig
```

Confirm with **no** `[plan]` section configured that no extra `ListPlanEntries` traffic appears and nothing is executed (SC-006).
