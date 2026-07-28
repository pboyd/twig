package tui

import (
	"errors"
	"strings"
	"testing"
	"time"

	planv1 "github.com/pboyd/twig/api/gen/plan/v1"
	taskv1 "github.com/pboyd/twig/api/gen/task/v1"
	"github.com/pboyd/twig/internal/cli"
	"github.com/pboyd/twig/internal/config"
)

// ── helpers ─────────────────────────────────────────────────────────────────

func pint32Plan(v int32) *int32 { return &v }

// makeEntry constructs a PlanEntry for tests with sensible defaults: linked
// to the given task id, with the given start minute + duration, and the given
// name override (empty string clears it).
func makeEntry(taskID int64, id int32, startMin *int32, dur int32, name string) *planv1.PlanEntry {
	return &planv1.PlanEntry{
		Id:             id,
		TaskId:         taskID,
		Name:           name,
		StartMinute:    startMin,
		DurationMinute: dur,
	}
}

// ── T015: dueBoundaries ─────────────────────────────────────────────────────

func TestDueBoundaries_FiresOnceAcrossConsecutiveEvaluations(t *testing.T) {
	at := time.Date(2026, 7, 28, 14, 0, 0, 0, time.Local)
	bs := []planBoundary{{at: at, edge: edgeStart, task: true, name: "Task A"}}
	watermark := at.Add(-time.Minute)

	first := dueBoundaries(bs, watermark, at.Add(15*time.Second))
	if len(first) != 1 {
		t.Fatalf("first evaluation: want 1 due boundary, got %d", len(first))
	}

	// On the next tick the watermark has advanced past it, so it must NOT fire again.
	advanced := dueBoundaries(bs, at.Add(15*time.Second), at.Add(30*time.Second))
	if len(advanced) != 0 {
		t.Fatalf("second evaluation: want 0 due boundaries, got %d", len(advanced))
	}
}

func TestDueBoundaries_BoundaryAlreadyPastAtStartupNeverFires(t *testing.T) {
	// A boundary whose fire time is at-or-before the seeded watermark: an
	// entry that was already due before the TUI was launched.
	at := time.Date(2026, 7, 28, 14, 0, 0, 0, time.Local)
	bs := []planBoundary{{at: at, edge: edgeStart, task: true, name: "Already past"}}
	watermark := at // boundary at watermark is NOT strictly after, so excluded

	got := dueBoundaries(bs, watermark, at.Add(time.Hour))
	if len(got) != 0 {
		t.Fatalf("boundary at watermark should not fire, got %d", len(got))
	}
}

func TestDueBoundaries_BoundaryMoreThan2MinutesLateIsSkipped(t *testing.T) {
	// Clock jumps forward: a boundary 5 minutes in the past should be
	// skipped — we do not burst-fire things a clock jump flew over.
	at := time.Date(2026, 7, 28, 14, 0, 0, 0, time.Local)
	bs := []planBoundary{{at: at, edge: edgeStart, task: true, name: "Slipped"}}
	watermark := at.Add(-time.Minute)

	got := dueBoundaries(bs, watermark, at.Add(5*time.Minute))
	if len(got) != 0 {
		t.Fatalf("boundary 5m late should be skipped, got %d", len(got))
	}
}

func TestDueBoundaries_EndBeforeStartOnCollision(t *testing.T) {
	// Two boundaries land on the same minute: end must come back first.
	at := time.Date(2026, 7, 28, 14, 0, 0, 0, time.Local)
	bs := []planBoundary{
		{at: at, edge: edgeStart, task: true, name: "next"},
		{at: at, edge: edgeEnd, task: true, name: "previous"},
	}
	watermark := at.Add(-time.Minute)

	got := dueBoundaries(bs, watermark, at.Add(15*time.Second))
	if len(got) != 2 {
		t.Fatalf("want 2 due boundaries, got %d", len(got))
	}
	if got[0].edge != edgeEnd || got[1].edge != edgeStart {
		t.Errorf("ordering = [%d, %d], want [edgeEnd, edgeStart]", got[0].edge, got[1].edge)
	}
}

// ── T019: derivation + hookFor (US1) ────────────────────────────────────────

func TestPlanBoundaries_UntimedEntryProducesNoBoundaries(t *testing.T) {
	day := "2026-07-28"
	entries := []*planv1.PlanEntry{
		makeEntry(1, 10, nil, 30, "Untimed task"), // nil StartMinute
	}

	got := planBoundaries(entries, day, nil)
	if len(got) != 0 {
		t.Errorf("untimed entry: want 0 boundaries, got %d", len(got))
	}
}

func TestPlanBoundaries_TimedTaskEntryProducesStartAndEnd(t *testing.T) {
	day := "2026-07-28"
	startMin := int32(14 * 60) // 14:00
	dur := int32(30)           // 30 minutes → ends at 14:30
	entries := []*planv1.PlanEntry{
		makeEntry(1, 10, &startMin, dur, "Write report"),
	}

	got := planBoundaries(entries, day, nil)
	if len(got) != 2 {
		t.Fatalf("want 2 boundaries, got %d", len(got))
	}
	wantStart := time.Date(2026, 7, 28, 14, 0, 0, 0, time.Local)
	wantEnd := time.Date(2026, 7, 28, 14, 30, 0, 0, time.Local)

	if got[0].at != wantEnd || got[0].edge != edgeEnd || !got[0].task {
		t.Errorf("end boundary: at=%v edge=%d task=%v", got[0].at, got[0].edge, got[0].task)
	}
	if got[1].at != wantStart || got[1].edge != edgeStart || !got[1].task {
		t.Errorf("start boundary: at=%v edge=%d task=%v", got[1].at, got[1].edge, got[1].task)
	}
}

func TestPlanBoundaries_ZeroDurationYieldsCoincidentStartAndEnd(t *testing.T) {
	day := "2026-07-28"
	startMin := int32(9 * 60)
	dur := int32(0)
	entries := []*planv1.PlanEntry{
		makeEntry(1, 10, &startMin, dur, ""),
	}

	got := planBoundaries(entries, day, nil)
	if len(got) != 2 {
		t.Fatalf("zero duration: want 2 boundaries, got %d", len(got))
	}
	if !got[0].at.Equal(got[1].at) {
		t.Errorf("start==end expected, got %v and %v", got[0].at, got[1].at)
	}
	if got[0].edge != edgeEnd || got[1].edge != edgeStart {
		t.Errorf("ordering = [%d, %d], want [end, start]", got[0].edge, got[1].edge)
	}
}

func TestPlanBoundaries_NameFallsBackToLinkedTaskName(t *testing.T) {
	day := "2026-07-28"
	startMin := int32(15 * 60)
	taskID := int64(42)
	taskName := "Parent task name"

	// Make the task tree contain the linked task under the given id.
	tree := []*cli.TreeNode{{Task: &taskv1.Task{Id: taskID, Name: taskName}}}

	entries := []*planv1.PlanEntry{
		makeEntry(taskID, 10, &startMin, 30, ""), // no name override
	}

	got := planBoundaries(entries, day, tree)
	if len(got) != 2 {
		t.Fatalf("want 2 boundaries, got %d", len(got))
	}
	for _, b := range got {
		if b.name != taskName {
			t.Errorf("name = %q, want fallback to %q", b.name, taskName)
		}
	}
}

func TestHookFor_TaskBoundaries(t *testing.T) {
	cfg := config.PlanConfig{
		OnTaskStart: "task-start.sh",
		OnTaskEnd:   "task-end.sh",
		// Event keys deliberately set so we can prove they are ignored for tasks.
		OnEventStart: "event-start.sh",
		OnEventEnd:   "event-end.sh",
	}

	startCmd, startKey := hookFor(cfg, planBoundary{task: true, edge: edgeStart})
	if startCmd != "task-start.sh" || startKey != "on_task_start" {
		t.Errorf("task start: got (%q, %q), want (task-start.sh, on_task_start)", startCmd, startKey)
	}

	endCmd, endKey := hookFor(cfg, planBoundary{task: true, edge: edgeEnd})
	if endCmd != "task-end.sh" || endKey != "on_task_end" {
		t.Errorf("task end: got (%q, %q), want (task-end.sh, on_task_end)", endCmd, endKey)
	}
}

func TestHookFor_EventBoundariesRouteToEventKeys(t *testing.T) {
	// FR-003: an entry that does not link a task (an event) routes to the
	// on_event_* keys, independent of any task keys that may also be set.
	cfg := config.PlanConfig{
		OnTaskStart:  "task-start.sh",
		OnTaskEnd:    "task-end.sh",
		OnEventStart: "event-start.sh",
		OnEventEnd:   "event-end.sh",
	}

	startCmd, startKey := hookFor(cfg, planBoundary{task: false, edge: edgeStart})
	if startCmd != "event-start.sh" || startKey != "on_event_start" {
		t.Errorf("event start: got (%q, %q), want (event-start.sh, on_event_start)", startCmd, startKey)
	}

	endCmd, endKey := hookFor(cfg, planBoundary{task: false, edge: edgeEnd})
	if endCmd != "event-end.sh" || endKey != "on_event_end" {
		t.Errorf("event end: got (%q, %q), want (event-end.sh, on_event_end)", endCmd, endKey)
	}
}

func TestHookFor_TaskBoundaryNeverFiresEventKey(t *testing.T) {
	// A task entry must never route to an event key, regardless of what
	// event keys are configured.
	cfg := config.PlanConfig{
		OnTaskStart:  "task-start.sh",
		OnEventStart: "event-start.sh",
	}

	cmd, key := hookFor(cfg, planBoundary{task: true, edge: edgeStart})
	if cmd != "task-start.sh" || key != "on_task_start" {
		t.Errorf("task boundary must not pick event key: got (%q, %q)", cmd, key)
	}
}

func TestHookFor_EventBoundaryNeverFiresTaskKey(t *testing.T) {
	// An event entry must never route to a task key, regardless of what
	// task keys are configured.
	cfg := config.PlanConfig{
		OnTaskStart:  "task-start.sh",
		OnEventStart: "event-start.sh",
	}

	cmd, key := hookFor(cfg, planBoundary{task: false, edge: edgeStart})
	if cmd != "event-start.sh" || key != "on_event_start" {
		t.Errorf("event boundary must not pick task key: got (%q, %q)", cmd, key)
	}
}

func TestHookFor_OnlyEventKeysConfiguredTaskBoundaryDispatchesNothing(t *testing.T) {
	// With only event keys configured, a task boundary returns ("", "") so
	// the tick reducer skips dispatch (data-model.md §1: empty = do nothing).
	cfg := config.PlanConfig{OnEventStart: "event-start.sh"}

	cmd, key := hookFor(cfg, planBoundary{task: true, edge: edgeStart})
	if cmd != "" || key != "" {
		t.Errorf("only event keys + task boundary: got (%q, %q), want (\"\", \"\")", cmd, key)
	}
}

// ── T025: placeholder expansion (US3) ───────────────────────────────────────

func TestExpandHookCmd_PercentSInsertsNameVerbatim(t *testing.T) {
	got := expandHookCmd(`echo %s`, `Write report`, time.Date(2026, 7, 28, 14, 0, 0, 0, time.Local))
	if got != `echo Write report` {
		t.Errorf("%%s: got %q, want %q", got, `echo Write report`)
	}
}

func TestExpandHookCmd_PercentQWrapsAndEscapes(t *testing.T) {
	got := expandHookCmd(`echo %q`, `Write report`, time.Date(2026, 7, 28, 14, 0, 0, 0, time.Local))
	if got != `echo "Write report"` {
		t.Errorf("%%q: got %q, want %q", got, `echo "Write report"`)
	}
}

func TestExpandHookCmd_PercentTFormatsScheduledTime(t *testing.T) {
	at := time.Date(2026, 7, 28, 9, 5, 0, 0, time.Local)
	got := expandHookCmd(`echo %t`, ``, at)
	if got != `echo 09:05` {
		t.Errorf("%%t zero-padding: got %q, want %q", got, `echo 09:05`)
	}
}

func TestExpandHookCmd_PercentTReportsScheduledNotExecution(t *testing.T) {
	// FR-011: %t uses the boundary's scheduled time even when now is later.
	at := time.Date(2026, 7, 28, 14, 0, 0, 0, time.Local)
	got := expandHookCmd(`echo %t`, ``, at)
	if got != `echo 14:00` {
		t.Errorf("%%t scheduled: got %q, want %q", got, `echo 14:00`)
	}
}

func TestExpandHookCmd_PercentPercentEmitsLiteralPercent(t *testing.T) {
	got := expandHookCmd(`100%% done`, ``, time.Date(2026, 7, 28, 14, 0, 0, 0, time.Local))
	if got != `100% done` {
		t.Errorf("%%%%: got %q, want %q", got, `100% done`)
	}
}

func TestExpandHookCmd_UnknownPercentXPassesThrough(t *testing.T) {
	got := expandHookCmd(`save %z today`, ``, time.Date(2026, 7, 28, 14, 0, 0, 0, time.Local))
	if got != `save %z today` {
		t.Errorf("unknown %%z passthrough: got %q, want %q", got, `save %z today`)
	}
}

func TestExpandHookCmd_TrailingBarePercentEmittedAsIs(t *testing.T) {
	got := expandHookCmd(`100%`, ``, time.Date(2026, 7, 28, 14, 0, 0, 0, time.Local))
	if got != `100%` {
		t.Errorf("trailing bare %%: got %q, want %q", got, `100%`)
	}
}

// FR-013: substitution is not re-scanned, so a name containing %s cannot
// inject a placeholder. Only one expand pass over the original command.
func TestExpandHookCmd_NonRecursiveOnNameContainingPlaceholder(t *testing.T) {
	got := expandHookCmd(`echo %s`, `100%s done`, time.Date(2026, 7, 28, 14, 0, 0, 0, time.Local))
	if got != `echo 100%s done` {
		t.Errorf("non-recursion: got %q, want %q", got, `echo 100%s done`)
	}
}

// FR-010: the four POSIX-shell-reinterpreted characters inside double quotes
// must all be escaped by %q.
func TestExpandHookCmd_PercentQEscapesAllFourChars(t *testing.T) {
	// name = a"b\c$d`e — all four escapable chars present.
	name := "a\"b\\c$d`e"
	got := expandHookCmd("echo %q", name, time.Date(2026, 7, 28, 14, 0, 0, 0, time.Local))
	// expected literal:
	//   echo "a\"b\\c\$d\`e"
	want := "echo " + `"` + `a\"b\\c\$d\` + "`" + `e"`
	if got != want {
		t.Errorf("%%q escape four chars: got %q, want %q", got, want)
	}
}

func TestExpandHookCmd_PercentQWithEmptyNameYieldsEmptyQuotedString(t *testing.T) {
	got := expandHookCmd(`echo %q`, ``, time.Date(2026, 7, 28, 14, 0, 0, 0, time.Local))
	if got != `echo ""` {
		t.Errorf("%%q empty name: got %q, want %q", got, `echo ""`)
	}
}

// ── T026: contracts/config-schema.md worked example ─────────────────────────

// The worked example from contracts/config-schema.md:
//
//	on_task_start = 'notify-send "%t: start %q"'
//	task named:    Fix "auth" $bug
//	scheduled at:  14:00
//
// expandHookCmd renders %t as the scheduled HH:MM and %q as the name wrapped
// in double quotes with " $ (and \ and `) escaped — once. The literal outer
// string is therefore:
//
//	notify-send "14:00: start "Fix \"auth\" \$bug""
//
// (outer-quote closes after the time string; %q opens its own outer quote
// around the name; the shell sees a balanced pair).
func TestExpandHookCmd_WorkedExampleFromConfigSchema(t *testing.T) {
	at := time.Date(2026, 7, 28, 14, 0, 0, 0, time.Local)
	got := expandHookCmd(`notify-send "%t: start %q"`, `Fix "auth" $bug`, at)
	want := `notify-send "14:00: start "Fix \"auth\" \$bug""`
	if got != want {
		t.Errorf("worked example: got %q, want %q", got, want)
	}
}

// ── T015/T019: tick reducer ────────────────────────────────────────────────

func TestHandlePlanHookTick_WatermarkDoesNotAdvanceWhileNotLoaded(t *testing.T) {
	// The single most likely bug per research D4: advancing watermark
	// without a successful first fetch sails it past boundaries.
	now := time.Date(2026, 7, 28, 14, 0, 30, 0, time.Local)
	m := Model{
		nowFunc: func() time.Time { return now },
	}
	m.planHooks.cfg = config.PlanConfig{OnTaskStart: "x.sh"}
	m.planHooks.watermark = now.Add(-time.Minute)
	m.planHooks.day = now.Format("2006-01-02")
	m.planHooks.loaded = false
	m.planHooks.lastFetch = now // skip refetch for this test

	m, _ = m.handlePlanHookTick()

	if !m.planHooks.watermark.Equal(now.Add(-time.Minute)) {
		t.Errorf("watermark advanced while unloaded: was %v, now %v",
			now.Add(-time.Minute), m.planHooks.watermark)
	}
}

func TestHandlePlanHookTick_DayRolloverClearsLoaded(t *testing.T) {
	t0 := time.Date(2026, 7, 28, 23, 59, 45, 0, time.Local)
	t1 := t0.Add(45 * time.Second)
	m := Model{
		nowFunc: func() time.Time { return t1 },
	}
	m.planHooks.cfg = config.PlanConfig{OnTaskStart: "x.sh"}
	m.planHooks.day = "2026-07-28"
	m.planHooks.loaded = true
	m.planHooks.watermark = t0.Add(-time.Minute)

	m, _ = m.handlePlanHookTick()

	if m.planHooks.day != "2026-07-29" {
		t.Errorf("day did not roll over: got %q, want 2026-07-29", m.planHooks.day)
	}
	if m.planHooks.loaded {
		t.Error("day rollover: expected loaded=false after reset")
	}
	if len(m.planHooks.entries) != 0 {
		t.Errorf("day rollover: entries not cleared: %d", len(m.planHooks.entries))
	}
}

func TestHandlePlanHookEntriesMsg_StaleDayResponseDropped(t *testing.T) {
	now := time.Date(2026, 7, 28, 14, 0, 0, 0, time.Local)
	m := Model{nowFunc: func() time.Time { return now }}
	m.planHooks.day = "2026-07-29" // fetch was for an older day

	m = m.handlePlanHookEntriesMsg(planHookEntriesMsg{
		day:     "2026-07-28",
		entries: []*planv1.PlanEntry{},
	})

	if m.planHooks.loaded {
		t.Error("stale response: loaded should remain false")
	}
}

// ── misc: hook error message text (T027-shaped expectation) ────────────────

func TestPlanHookErrMsg_RoutesToNotice(t *testing.T) {
	now := time.Date(2026, 7, 28, 14, 0, 0, 0, time.Local)
	m := Model{nowFunc: func() time.Time { return now }}

	next, _ := m.Update(planHookErrMsg{key: "on_task_start", err: errors.New("exit 127")})
	nm := next.(Model)

	if nm.err != nil {
		t.Errorf("err should be nil for hook failure, got %v", nm.err)
	}
	if !strings.Contains(nm.notice, "on_task_start") {
		t.Errorf("notice should name the offending key, got %q", nm.notice)
	}
	if !strings.Contains(nm.notice, "exit 127") {
		t.Errorf("notice should carry the error, got %q", nm.notice)
	}
}

// ── code-review fixes ──────────────────────────────────────────────────────

// A DST-transition day must not shift boundaries. StartMinute is a wall-clock
// minute-of-day, so an entry the plan grid renders as 14:00 must fire at 14:00
// wall clock even when the day is only 23 hours long.
func TestPlanBoundaries_DSTSpringForwardKeepsWallClock(t *testing.T) {
	nyc, err := time.LoadLocation("America/New_York")
	if err != nil {
		t.Skipf("tzdata unavailable: %v", err)
	}
	orig := time.Local
	time.Local = nyc
	defer func() { time.Local = orig }()

	// 2026-03-08 is the US spring-forward date: 02:00 EST jumps to 03:00 EDT.
	start := int32(14 * 60)
	entries := []*planv1.PlanEntry{makeEntry(7, 1, &start, 30, "Afternoon block")}

	got := planBoundaries(entries, "2026-03-08", nil)
	if len(got) != 2 {
		t.Fatalf("want 2 boundaries, got %d", len(got))
	}
	for _, b := range got {
		wantHour, wantMin := 14, 0
		if b.edge == edgeEnd {
			wantMin = 30
		}
		if b.at.Hour() != wantHour || b.at.Minute() != wantMin {
			t.Errorf("edge %v: got %02d:%02d, want %02d:%02d (adding absolute duration to midnight shifts across the DST gap)",
				b.edge, b.at.Hour(), b.at.Minute(), wantHour, wantMin)
		}
	}
}

// Rolling over to a new day must zero lastFetch so the new day's entries are
// fetched immediately, not up to a full fetch interval later.
func TestHandlePlanHookTick_DayRolloverForcesImmediateRefetch(t *testing.T) {
	t0 := time.Date(2026, 7, 28, 23, 59, 45, 0, time.Local)
	t1 := t0.Add(45 * time.Second) // 2026-07-29 00:00:30
	m := Model{nowFunc: func() time.Time { return t1 }}
	m.planHooks.cfg = config.PlanConfig{OnTaskStart: "x.sh"}
	m.planHooks.day = "2026-07-28"
	m.planHooks.loaded = true
	m.planHooks.watermark = t0.Add(-time.Minute)
	m.planHooks.lastFetch = t0 // recent: the throttle would otherwise skip

	m, _ = m.handlePlanHookTick()

	if !m.planHooks.lastFetch.Equal(t1) {
		t.Errorf("rollover did not force a refetch: lastFetch=%v, want %v", m.planHooks.lastFetch, t1)
	}
}

// A plan mutation must invalidate the hook cache, or an entry added within the
// fetch interval of its own start time is discovered after the watermark has
// already moved past it and never fires (FR-017).
func TestPlanMutatedMsg_InvalidatesHookCache(t *testing.T) {
	now := time.Date(2026, 7, 28, 13, 59, 20, 0, time.Local)
	m := Model{nowFunc: func() time.Time { return now }}
	m.planHooks.lastFetch = now.Add(-10 * time.Second)

	next, _ := m.Update(planMutatedMsg{highlightID: 3})
	nm := next.(Model)

	if !nm.planHooks.lastFetch.IsZero() {
		t.Errorf("plan mutation did not invalidate the hook cache: lastFetch=%v", nm.planHooks.lastFetch)
	}
}
