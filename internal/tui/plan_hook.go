package tui

import (
	"context"
	"os/exec"
	"sort"
	"strings"
	"time"

	"connectrpc.com/connect"
	tea "charm.land/bubbletea/v2"
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
// started in Init only when cfg.enabled() (research.md D3).
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
// deliberately separate from listPlanCmd so the visible-plan reducer does
// not need to learn about this subscriber (research.md D2).
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

// runPlanHook runs `sh -c <cmd>` with stdio detached from the TUI terminal so
// it cannot corrupt the alt-screen. Returns nil for an empty command.
// Mirrors runPomHook at internal/tui/pomodoro.go.
func runPlanHook(cmd, key string) tea.Cmd {
	if cmd == "" {
		return nil
	}
	return func() tea.Msg {
		c := exec.Command("sh", "-c", cmd)
		// Do not assign Stdout/Stderr — leave stdio detached from the
		// alt-screen so a chatty hook cannot scribble over the UI.
		if err := c.Run(); err != nil {
			return planHookErrMsg{key: key, err: err}
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
	var out []planBoundary
	for _, e := range entries {
		if e == nil || e.StartMinute == nil {
			continue
		}
		start := midnight.Add(time.Duration(*e.StartMinute) * time.Minute)
		end := start.Add(time.Duration(e.DurationMinute) * time.Minute)
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
// task-vs-event routing in data-model.md §3. US1 implements the task keys;
// US2 fills in the event keys. Returns ("", "") for event boundaries until
// US2, or when the appropriate task key is empty (caller skips dispatch).
func hookFor(cfg config.PlanConfig, b planBoundary) (cmd, key string) {
	switch {
	case b.task && b.edge == edgeStart:
		return cfg.OnTaskStart, "on_task_start"
	case b.task && b.edge == edgeEnd:
		return cfg.OnTaskEnd, "on_task_end"
	}
	// Event hooks: deferred to US2.
	return "", ""
}

// dueBoundaries applies the watermark filter and sort. Returns the boundaries
// in (at, edge) order — ends before starts at the same instant — that should
// fire this tick. The whole FR-004/005/006 contract plus the clock-jump
// guard lives in one predicate;

// research.md D4.
func dueBoundaries(bs []planBoundary, watermark, now time.Time) []planBoundary {
	var due []planBoundary
	for _, b := range bs {
		if watermark.Before(b.at) && !b.at.After(now) && now.Sub(b.at) <= 2*time.Minute {
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
			b.WriteByte('"')
			b.WriteString(shellEscapeDoubleQuoted(name))
			b.WriteByte('"')
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

// handlePlanHookTick runs the 15s evaluation. It rolls the watched day over
// when the local date changed, refetches when stale, fires due boundaries,
// and finally advances the watermark — but only on ticks where today's
// entries are actually loaded (research.md D4 guard).
func (m Model) handlePlanHookTick() (Model, tea.Cmd) {
	now := m.nowOrDefault()
	today := now.Format("2006-01-02")

	// Day boundary: clear cached entries and force a refetch.
	if m.planHooks.day != today {
		m.planHooks.day = today
		m.planHooks.entries = nil
		m.planHooks.loaded = false
	}

	var cmds []tea.Cmd

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
		if len(cmds) > 0 {
			return m, tea.Batch(append(cmds, planHookTickCmd())...)
		}
		return m, planHookTickCmd()
	}

	boundaries := planBoundaries(m.planHooks.entries, m.planHooks.day, m.tree)
	due := dueBoundaries(boundaries, m.planHooks.watermark, now)
	for _, b := range due {
		cmd, key := hookFor(m.planHooks.cfg, b)
		if cmd == "" {
			continue
		}
		expanded := expandHookCmd(cmd, b.name, b.at)
		cmds = append(cmds, runPlanHook(expanded, key))
	}
	m.planHooks.watermark = now

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
