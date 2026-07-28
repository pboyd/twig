# Phase 1 Data Model: External Commands for Plan Entry Boundaries

**Feature**: 070-plan-entry-hooks
**Date**: 2026-07-28

No database tables, no protobuf messages, no API changes. Every entity below is in-memory, client-side, and lives in the root module (`github.com/pboyd/twig`).

---

## 1. `config.PlanConfig` — `internal/config/config.go`

The four configurable command strings, parsed from the `[plan]` TOML table.

```go
type PlanConfig struct {
    OnEventStart string `toml:"on_event_start"`
    OnEventEnd   string `toml:"on_event_end"`
    OnTaskStart  string `toml:"on_task_start"`
    OnTaskEnd    string `toml:"on_task_end"`
}
```

Added to the existing `Config` struct alongside `Pomodoro`:

```go
type Config struct {
    APIURL   string             `toml:"api_url"`
    APIKey   string             `toml:"api_key"`
    Pomodoro PomodoroConfig     `toml:"pomodoro"`
    Plan     PlanConfig         `toml:"plan"`     // NEW
    Profiles map[string]Profile `toml:"profile"`
}
```

**Validation rules**:
- All four fields are optional; the zero value (empty string) means "do nothing" (FR-001).
- No environment-variable override, matching `PomodoroConfig` (spec assumption).
- Unknown keys inside `[plan]` are ignored — inherited free from `BurntSushi/toml`'s default decode behaviour (FR-018).
- `Config.Profile()` continues to take `Plan` from the root config, never from the named profile — same rule already applied to `Pomodoro` (`internal/config/config.go:64-76`).

**Derived predicate**:
```go
// enabled reports whether any hook is configured. Gates the ticker and the
// background fetch so an unconfigured user pays nothing (SC-006).
func (p PlanConfig) enabled() bool
```

---

## 2. `boundaryEdge` — `internal/tui/plan_hook.go`

Which end of an entry a boundary sits at. The ordering matters: FR-016 requires ends to run before starts when they collide, so `edgeEnd` sorts first.

```go
type boundaryEdge int

const (
    edgeEnd   boundaryEdge = iota // sorts first — FR-016
    edgeStart
)
```

---

## 3. `planBoundary` — `internal/tui/plan_hook.go`

One scheduled moment derived from one timed plan entry. This is the unit that is watched and fired. Purely derived — never persisted, rebuilt from scratch on every evaluation tick (FR-017).

```go
type planBoundary struct {
    at   time.Time    // absolute local time of the boundary
    edge boundaryEdge // start or end
    task bool         // true when the entry links a task (entry.TaskId != 0)
    name string       // display name: entry.Name, falling back to the task's name
}
```

**Derivation rules** (from `*planv1.PlanEntry`, for a given local day):
| Rule | Detail |
|---|---|
| Skip untimed | `entry.StartMinute == nil` yields no boundaries (FR-007) |
| Start time | local midnight of `day` + `*entry.StartMinute` minutes |
| End time | start + `entry.DurationMinute` minutes (spec assumption: no stored end) |
| Kind | `entry.TaskId != 0` → task hooks; otherwise event hooks (FR-002, FR-003) |
| Name | `entry.Name` when non-empty, else the linked task's name (FR-009) |
| Zero duration | start == end; both boundaries derived, both fire, end first per FR-016 |

**Name resolution note**: `ListPlanEntries` returns `name` empty when the entry has no override, expecting the client to fall back to the linked task's name (`api/proto/plan/v1/plan.proto:50-52`). The TUI already holds the task tree in `m.tree`, so the fallback is a lookup by `entry.TaskId`. When the task is not in the tree (e.g. the tree has not loaded yet), the name resolves to empty and `%s`/`%q` substitute empty/`""` rather than failing the hook.

**Ordering**: due boundaries are sorted by `(at, edge)` before dispatch (FR-016).

---

## 4. `planHookState` — `internal/tui/model.go`

The feature's entire mutable state, held as one field on `Model`.

```go
type planHookState struct {
    cfg       config.PlanConfig     // the four commands
    day       string                // local date the cached entries cover, YYYY-MM-DD
    entries   []*planv1.PlanEntry   // today's entries
    loaded    bool                  // false until the first successful fetch for `day`
    lastFetch time.Time             // fetch throttle — refetch when older than 60s
    watermark time.Time             // boundaries at or before this have been handled
}
```

**State transitions**:

| Event | Effect |
|---|---|
| Model init | `cfg` set from config; `watermark` = now (FR-006: nothing already past ever fires); `loaded` = false |
| Tick, `day != today` | clear `entries`, set `loaded` = false, set `day` = today, force refetch (FR-008) |
| Tick, `lastFetch` older than 60s | dispatch background fetch of today's entries |
| Fetch succeeds | replace `entries`, `loaded` = true, `lastFetch` = now |
| Fetch fails | leave `entries` and `loaded` alone, `lastFetch` = now — a transient RPC error must not disarm the hooks or spam the user |
| Tick, `loaded` == true | derive boundaries, fire those in `(watermark, now]` that are ≤2min late, then `watermark` = now |
| Tick, `loaded` == false | **do not** advance `watermark` — otherwise an in-flight first fetch lets the watermark sail past a boundary and swallow it (research D4) |

**Firing predicate** (the whole of FR-004/005/006 plus the clock-jump edge case):

```go
watermark.Before(b.at) && !b.at.After(now) && now.Sub(b.at) <= 2*time.Minute
```

---

## 5. Message types — `internal/tui/plan_hook.go`

```go
// planHookTickMsg drives evaluation. Fired every 15s by its own chain,
// started in Init only when cfg.enabled().
type planHookTickMsg struct{}

// planHookEntriesMsg carries today's entries for the hook watcher. Distinct
// from planEntriesMsg so the visible-plan reducer is untouched (research D2).
type planHookEntriesMsg struct {
    day     string
    entries []*planv1.PlanEntry
    err     error
}

// planHookErrMsg reports a hook that failed to launch or exited non-zero.
// key is the config key name ("on_task_start"), used to build the notice.
type planHookErrMsg struct {
    key string
    err error
}
```

---

## 6. Entity relationships

```text
config.PlanConfig ──┐
                    ├──> planHookState ──(derive, every tick)──> []planBoundary
[]*planv1.PlanEntry ┘         │                                        │
  (today, fetched)            │ watermark filter + sort                │
                              ▼                                        ▼
                        due boundaries ──> expandHookCmd(cmd, name, at) ──> sh -c
```

Nothing in this diagram outlives the process.
