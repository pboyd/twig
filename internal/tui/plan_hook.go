package tui

import (
	"context"
	"os/exec"
	"sort"
	"strings"
	"time"

	tea "charm.land/bubbletea/v2"
	"connectrpc.com/connect"
	planv1 "github.com/pboyd/twig/api/gen/plan/v1"
	planv1connect "github.com/pboyd/twig/api/gen/plan/v1/planv1connect"
	"github.com/pboyd/twig/internal/cli"
	"github.com/pboyd/twig/internal/config"
)

// ── state ──────────────────────────────────────────────────────────────────

// planHookState holds the entire mutable feature state for the plan entry
// boundary hooks. Lives as one field on Model so the reducers touch a single
// named struct (see data-model.md §4).
type planHookState struct {
	cfg       config.PlanConfig
	day       string
	entries   []*planv1.PlanEntry
	loaded    bool
	lastFetch time.Time
	watermark time.Time
}

// ── message types ──────────────────────────────────────────────────────────

// planHookTickMsg drives evaluation. Fired every 15s by its own chain;
// started in Init only when cfg.Enabled() (research.md D3).
type planHookTickMsg struct{}

// planHookEntriesMsg carries today's entries for the hook watcher. Distinct
// from planEntriesMsg so the visible-plan reducer is untouched (research.md D2).
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

// ── boundary types ─────────────────────────────────────────────────────────

// boundaryEdge is which end of an entry a boundary sits at. edgeEnd sorts
// first (FR-016) so ends run before starts when they collide; using iota
// keeps edgeEnd == 0, which means natural ascending order puts ends first.
type boundaryEdge int

const (
	edgeEnd boundaryEdge = iota
	edgeStart
)

// planBoundary is one scheduled moment derived from one timed plan entry.
// Purely derived — never persisted, rebuilt from scratch on every tick.
type planBoundary struct {
	at   time.Time
	edge boundaryEdge
	task bool
	name string
}

// ── ticker / fetch ─────────────────────────────────────────────────────────

// planHookTickInterval is how often the hook watcher evaluates.
const planHookTickInterval = 15 * time.Second

// planHookFetchInterval caps how often we re-fetch today's plan entries from
// the server.
const planHookFetchInterval = 60 * time.Second

// planHookTickCmd schedules the next planHookTickMsg after planHookTickInterval.
func planHookTickCmd() tea.Cmd {
	return tea.Tick(planHookTickInterval, func(_ time.Time) tea.Msg {
		return planHookTickMsg{}
	})
}

// listPlanHooksCmd fetches today's plan entries for the hook watcher. Kept
// deliberately separate from listPlanCmd (research.md D2):
//   - the visible plan refetches every autoRefreshInterval (10 min) and only
//     while the Plan tab is active, while hooks need today's entries every
//     60s regardless of the active tab;
//   - m.plan.day follows day navigation, planHooks.day is always today.
func listPlanHooksCmd(client planv1connect.PlanServiceClient, day string) tea.Cmd {
	return func() tea.Msg {
		resp, err := client.ListPlanEntries(context.Background(), connect.NewRequest(&planv1.ListPlanEntriesRequest{Day: day}))
		if err != nil {
			return planHookEntriesMsg{day: day, err: err}
		}
		return planHookEntriesMsg{day: day, entries: resp.Msg.Entries}
	}
}

// ── hook runner ────────────────────────────────────────────────────────────

// hookCommand builds the exec.Cmd for a user-configured hook: always via
// `sh -c` so pipes and redirects work, with stdio left unassigned so a chatty
// hook cannot scribble over the alt-screen. Both the pomodoro and plan-entry
// hook runners must go through this — it is the one place the subprocess
// policy is defined.
func hookCommand(cmd string) *exec.Cmd { return exec.Command("sh", "-c", cmd) }

// plannedHook pairs an expanded shell command with the config key that
// produced it (used to name the key in a failure notice), in the order the
// boundary evaluation says it must launch.
type plannedHook struct {
	cmd string
	key string
}

// runPlanHooks starts every hook in hooks — `sh -c <cmd>`, stdio detached
// from the TUI terminal so a chatty hook cannot corrupt the alt-screen — in
// order, then waits for each to exit in that same order. FR-016 ("ends
// before starts" when a boundary collision is due the same tick) only needs
// launch order preserved, not completion order, so starting is separated
// from waiting: a hook that blocks on user input or a hung dbus delays only
// the msg this command eventually returns, never the launch of a later hook
// in the same batch (FR-015). This whole function runs as a single tea.Cmd,
// off the Update goroutine, so the *next* tick's boundaries — a separate
// command — are never blocked by it either.
//
// Only the first failure (to start or to exit clean) is reported; the
// status bar shows one notice at a time regardless.
func runPlanHooks(hooks []plannedHook) tea.Cmd {
	var pending []plannedHook
	for _, h := range hooks {
		if h.cmd != "" {
			pending = append(pending, h)
		}
	}
	if len(pending) == 0 {
		return nil
	}
	return func() tea.Msg {
		type started struct {
			proc *exec.Cmd
			key  string
		}
		var running []started
		var firstErr *planHookErrMsg
		for _, h := range pending {
			c := hookCommand(h.cmd)
			// Do not assign Stdout/Stderr — leave stdio detached from the
			// alt-screen so a chatty hook cannot scribble over the UI.
			if err := c.Start(); err != nil {
				if firstErr == nil {
					firstErr = &planHookErrMsg{key: h.key, err: err}
				}
				continue
			}
			running = append(running, started{proc: c, key: h.key})
		}
		for _, r := range running {
			if err := r.proc.Wait(); err != nil && firstErr == nil {
				firstErr = &planHookErrMsg{key: r.key, err: err}
			}
		}
		if firstErr != nil {
			return *firstErr
		}
		return nil
	}
}

// ── pure logic: derivation + selection + firing ────────────────────────────

// planBoundaries derives the watched boundary instants from a day's plan
// entries. Skips entries with nil StartMinute (FR-007); start = local
// midnight of day + StartMinute; end = start + DurationMinute.
// Resolves the display name from entry.Name, falling back to the linked
// task's name when present in the task tree (FR-009).
func planBoundaries(entries []*planv1.PlanEntry, day string, tree []*cli.TreeNode) []planBoundary {
	midnight, err := time.ParseInLocation("2006-01-02", day, time.Local)
	if err != nil {
		return nil
	}
	y, mo, d := midnight.Date()
	var out []planBoundary
	for _, e := range entries {
		if e == nil || e.StartMinute == nil {
			continue
		}
		// StartMinute/DurationMinute are wall-clock minutes-of-day, which is
		// how the plan grid renders them. Build the instants with time.Date
		// rather than adding a Duration to midnight: on a DST-transition day
		// adding absolute time would fire an entry the plan shows as 14:00 at
		// 15:00 (spring forward) or 13:00 (fall back).
		start := time.Date(y, mo, d, 0, int(*e.StartMinute), 0, 0, time.Local)
		end := time.Date(y, mo, d, 0, int(*e.StartMinute)+int(e.DurationMinute), 0, 0, time.Local)
		name := e.Name
		if name == "" && e.TaskId != 0 {
			name = findTaskName(tree, e.TaskId)
		}
		out = append(out,
			planBoundary{at: end, edge: edgeEnd, task: e.TaskId != 0, name: name},
			planBoundary{at: start, edge: edgeStart, task: e.TaskId != 0, name: name},
		)
	}
	return out
}

// hookFor picks the (cmd, key) pair for a single boundary according to the
// task-vs-event routing in data-model.md §3. A task boundary never fires an
// event key (and vice versa). When the routed-to command string is empty
// (the user did not configure the corresponding key), the key is reported
// as empty too — the caller skips dispatch on cmd == "".
func hookFor(cfg config.PlanConfig, b planBoundary) (cmd, key string) {
	switch {
	case b.task && b.edge == edgeStart:
		cmd, key = cfg.OnTaskStart, "on_task_start"
	case b.task && b.edge == edgeEnd:
		cmd, key = cfg.OnTaskEnd, "on_task_end"
	case !b.task && b.edge == edgeStart:
		cmd, key = cfg.OnEventStart, "on_event_start"
	case !b.task && b.edge == edgeEnd:
		cmd, key = cfg.OnEventEnd, "on_event_end"
	}
	if cmd == "" {
		key = ""
	}
	return cmd, key
}

// planHookMaxLateness is how far past its scheduled instant a boundary may
// still fire. Matches the contract's Timing row and FR-004/SC-001 ("within
// 60 seconds"); a boundary later than this is treated as missed, not fired
// late.
const planHookMaxLateness = 60 * time.Second

// dueBoundaries applies the watermark filter and sort. Returns the boundaries
// in (at, edge) order — ends before starts at the same instant — that should
// fire this tick. The whole FR-004/005/006 contract plus the clock-jump
// guard lives in one predicate; see research.md D4.
func dueBoundaries(bs []planBoundary, watermark, now time.Time) []planBoundary {
	var due []planBoundary
	for _, b := range bs {
		if watermark.Before(b.at) && !b.at.After(now) && now.Sub(b.at) <= planHookMaxLateness {
			due = append(due, b)
		}
	}
	sort.Slice(due, func(i, j int) bool {
		if !due[i].at.Equal(due[j].at) {
			return due[i].at.Before(due[j].at)
		}
		return due[i].edge < due[j].edge
	})
	return due
}

// ── placeholder expansion ─────────────────────────────────────────────────

// shellEscapeDoubleQuoted backslash-escapes the exact set of characters the
// POSIX shell reinterprets inside double quotes. Adding more would corrupt
// legitimate names; adding fewer reopens the injection hole (research.md D5).
func shellEscapeDoubleQuoted(name string) string {
	var b strings.Builder
	b.Grow(len(name) + 2)
	for _, r := range name {
		switch r {
		case '"', '\\', '$', '`':
			b.WriteByte('\\')
		}
		b.WriteRune(r)
	}
	return b.String()
}

// expandHookCmd performs a single left-to-right pass over the command, writing
// into a strings.Builder. Substituted text is never re-scanned, so a name that
// contains `%s` cannot inject a placeholder (FR-013). The `%t` format is
// `at.Format("15:04")` — the boundary's *scheduled* minute, not the
// wall-clock time of execution (FR-011).
func expandHookCmd(cmd, name string, at time.Time) string {
	var b strings.Builder
	b.Grow(len(cmd))
	for i := 0; i < len(cmd); i++ {
		c := cmd[i]
		if c != '%' {
			b.WriteByte(c)
			continue
		}
		// Bare trailing `%` — emit as-is.
		if i+1 >= len(cmd) {
			b.WriteByte('%')
			continue
		}
		switch cmd[i+1] {
		case 's':
			b.WriteString(name)
			i++
		case 'q':
			// Escaped for embedding inside quotes the user supplies themselves
			// (`"%q"` in the command string) — %q does not add its own
			// delimiters, so used bare it is not shell-safe (that is %s's job).
			b.WriteString(shellEscapeDoubleQuoted(name))
			i++
		case 't':
			b.WriteString(at.Format("15:04"))
			i++
		case '%':
			b.WriteByte('%')
			i++
		default:
			// Unknown `%X` — emit both characters unchanged.
			b.WriteByte('%')
			b.WriteByte(cmd[i+1])
			i++
		}
	}
	return b.String()
}

// ── reducers ───────────────────────────────────────────────────────────────

// dueHooksCmd derives boundaries from entries/day, selects the ones due
// between watermark and now, and returns one command that launches them all
// in order (nil if none are due). Shared by the normal-tick path and the
// day-rollover path in handlePlanHookTick so evaluation logic exists once.
func dueHooksCmd(cfg config.PlanConfig, entries []*planv1.PlanEntry, day string, tree []*cli.TreeNode, watermark, now time.Time) tea.Cmd {
	boundaries := planBoundaries(entries, day, tree)
	due := dueBoundaries(boundaries, watermark, now)
	var hooks []plannedHook
	for _, b := range due {
		cmd, key := hookFor(cfg, b)
		if cmd == "" {
			continue
		}
		hooks = append(hooks, plannedHook{cmd: expandHookCmd(cmd, b.name, b.at), key: key})
	}
	// One runPlanHooks command, not one tea.Cmd per hook: dueBoundaries
	// returns ends before starts (FR-016), and runPlanHooks launches them in
	// that order before waiting on any of them, so "start B" cannot notify
	// before "end A" for back-to-back entries — without coupling this tick's
	// completion to how long any hook takes to exit (FR-015).
	return runPlanHooks(hooks)
}

// handlePlanHookTick runs the 15s evaluation. It rolls the watched day over
// when the local date changed, refetches when stale, fires due boundaries,
// and finally advances the watermark — but only on ticks where today's
// entries are actually loaded (research.md D4 guard).
func (m Model) handlePlanHookTick() (Model, tea.Cmd) {
	now := m.nowOrDefault()
	today := now.Format("2006-01-02")

	var cmds []tea.Cmd

	// Day boundary: evaluate the outgoing day's remaining boundaries before
	// clearing cached entries and forcing a refetch. Without this, an entry
	// ending exactly at 24:00 (the server allows start_minute+duration_minute
	// == 1440) has its end boundary normalized to 00:00 of the next day,
	// which the rollover below would otherwise discard unevaluated — the
	// entries slice is replaced by the new day's fetch before that boundary
	// is ever considered.
	if m.planHooks.day != today {
		if m.planHooks.loaded {
			if hooksCmd := dueHooksCmd(m.planHooks.cfg, m.planHooks.entries, m.planHooks.day, m.tree, m.planHooks.watermark, now); hooksCmd != nil {
				cmds = append(cmds, hooksCmd)
			}
		}
		m.planHooks.day = today
		m.planHooks.entries = nil
		m.planHooks.loaded = false
		// Zero lastFetch so the new day's entries are fetched on this tick
		// rather than up to a full fetch interval later. Combined with the
		// lateness guard in dueBoundaries, a throttled refetch could
		// otherwise drop a boundary sitting just after midnight.
		m.planHooks.lastFetch = time.Time{}
	}

	// Throttled fetch.
	if m.planHooks.lastFetch.IsZero() || now.Sub(m.planHooks.lastFetch) >= planHookFetchInterval {
		if m.planClient != nil {
			cmds = append(cmds, listPlanHooksCmd(m.planClient, today))
		}
		m.planHooks.lastFetch = now
	}

	// Only fire + advance watermark when loaded; otherwise boundaries could
	// sail past the watermark in flight (research.md D4).
	if !m.planHooks.loaded {
		return m, tea.Batch(append(cmds, planHookTickCmd())...)
	}

	if hooksCmd := dueHooksCmd(m.planHooks.cfg, m.planHooks.entries, m.planHooks.day, m.tree, m.planHooks.watermark, now); hooksCmd != nil {
		cmds = append(cmds, hooksCmd)
	}
	// Guard against a backward clock step (e.g. an NTP correction):
	// assigning watermark = now unconditionally would regress it, making
	// already-fired boundaries due again on a later tick (violates FR-005,
	// "at most once per session").
	if now.After(m.planHooks.watermark) {
		m.planHooks.watermark = now
	}

	cmds = append(cmds, planHookTickCmd())
	return m, tea.Batch(cmds...)
}

// handlePlanHookEntriesMsg applies today's entries from the watcher fetch.
// On success it replaces m.planHooks.entries and sets loaded + lastFetch.
// On error it only advances lastFetch — a transient RPC failure must not
// disarm the hooks or surface noise (data-model.md §4).
func (m Model) handlePlanHookEntriesMsg(msg planHookEntriesMsg) Model {
	if msg.err != nil {
		m.planHooks.lastFetch = m.nowOrDefault()
		return m
	}
	// Stale response (day rolled over while fetch was in flight): drop it.
	if msg.day != m.planHooks.day {
		return m
	}
	m.planHooks.entries = msg.entries
	m.planHooks.loaded = true
	m.planHooks.lastFetch = m.nowOrDefault()
	return m
}
