package cli

import (
	"bytes"
	"context"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"connectrpc.com/connect"
	taskv1 "github.com/pboyd/twig/api/gen/task/v1"
	taskv1connect "github.com/pboyd/twig/api/gen/task/v1/taskv1connect"
	"github.com/pboyd/twig/internal/report"
	"google.golang.org/protobuf/types/known/timestamppb"
)

// fakeServiceWithCount extends fakeTaskService to handle CountCompletedPomodoros.
type fakeServiceWithCount struct {
	*fakeTaskService
	pomCount int64
}

func (s *fakeServiceWithCount) CountCompletedPomodoros(
	_ context.Context,
	_ *connect.Request[taskv1.CountCompletedPomodorosRequest],
) (*connect.Response[taskv1.CountCompletedPomodorosResponse], error) {
	return connect.NewResponse(&taskv1.CountCompletedPomodorosResponse{Count: s.pomCount}), nil
}

// newReportHarness serves a fakeServiceWithCount over httptest and returns a
// connected client for exercising doReport end-to-end.
func newReportHarness(t *testing.T, pomCount int64) (taskv1connect.TaskServiceClient, *fakeServiceWithCount, *httptest.Server) {
	t.Helper()
	svc := &fakeServiceWithCount{fakeTaskService: newFakeTaskService(), pomCount: pomCount}
	mux := http.NewServeMux()
	path, handler := taskv1connect.NewTaskServiceHandler(svc)
	mux.Handle(path, handler)
	srv := httptest.NewServer(mux)
	t.Cleanup(srv.Close)
	return taskv1connect.NewTaskServiceClient(srv.Client(), srv.URL), svc, srv
}

// reportNowForTest is the fixed reference time for CLI report tests.
var reportNowForTest = time.Date(2025, 6, 11, 14, 0, 0, 0, time.Local)

// makeCompletedTask creates a task with a completedAt time.
func makeCompletedTask(id int64, name string, completedAt time.Time, parentID *int64) *taskv1.Task {
	t := &taskv1.Task{
		Id:          id,
		Name:        name,
		CompletedAt: timestamppb.New(completedAt),
	}
	if parentID != nil {
		t.ParentId = parentID
	}
	return t
}

func int64p(v int64) *int64 { return &v }

// TestParseReportArgs_DefaultPreset verifies that no args → recent preset.
func TestParseReportArgs_DefaultPreset(t *testing.T) {
	p, code := parseReportArgs([]string{})
	if code != 0 {
		t.Fatalf("expected exit 0, got %d", code)
	}
	if p.Label != "Yesterday & Today" {
		t.Errorf("Label = %q, want 'Yesterday & Today'", p.Label)
	}
}

// TestParseReportArgs_ExplicitPreset checks named presets.
func TestParseReportArgs_ExplicitPreset(t *testing.T) {
	for _, preset := range report.PresetOrder() {
		p, code := parseReportArgs([]string{preset})
		if code != 0 {
			t.Errorf("preset %q: expected exit 0, got %d", preset, code)
			continue
		}
		if p.Label == "" {
			t.Errorf("preset %q: empty label", preset)
		}
	}
}

// TestParseReportArgs_ExplicitRange tests --from/--to.
func TestParseReportArgs_ExplicitRange(t *testing.T) {
	p, code := parseReportArgs([]string{"--from", "2025-03-01", "--to", "2025-03-31"})
	if code != 0 {
		t.Fatalf("expected exit 0, got %d", code)
	}
	if !strings.Contains(p.Label, "Mar") {
		t.Errorf("Label = %q, expected March dates", p.Label)
	}
}

// TestParseReportArgs_ValidationErrors tests invalid argument combinations.
func TestParseReportArgs_ValidationErrors(t *testing.T) {
	cases := []struct {
		args []string
		desc string
	}{
		{[]string{"week", "--from", "2025-03-01", "--to", "2025-03-31"}, "preset + --from/--to"},
		{[]string{"--from", "2025-03-01"}, "--from without --to"},
		{[]string{"--to", "2025-03-31"}, "--to without --from"},
		{[]string{"--from", "not-a-date", "--to", "2025-03-31"}, "bad --from"},
		{[]string{"--from", "2025-03-31", "--to", "2025-03-01"}, "inverted range"},
		{[]string{"fortnight"}, "unknown preset"},
	}
	for _, tc := range cases {
		_, code := parseReportArgs(tc.args)
		if code == 0 {
			t.Errorf("case %q: expected non-zero exit code", tc.desc)
		}
	}
}

// TestRenderReport_DayGrouped verifies day-grouped output contains expected text.
func TestRenderReport_DayGrouped(t *testing.T) {
	now := reportNowForTest
	p, _ := report.ParsePeriod("recent", "", "", now)

	jun11 := time.Date(2025, 6, 11, 12, 0, 0, 0, time.Local).UTC()
	jun10 := time.Date(2025, 6, 10, 14, 0, 0, 0, time.Local).UTC()

	tasks := []*taskv1.Task{
		makeCompletedTask(1, "Write summary", jun11, nil),
		makeCompletedTask(2, "Fix proxy", jun10, nil),
		{Id: 3, Name: "Web app polish"}, // parent, not completed in range
		makeCompletedTask(4, "Wire download page", jun10, int64p(3)),
	}

	entries := report.BuildEntries(tasks, p)
	totals := report.Totals{TasksCompleted: len(entries), PomodorosCompleted: 5}

	var buf bytes.Buffer
	renderReport(&buf, p, entries, tasks, totals, false) // unstyled (plain)
	out := buf.String()

	// Header
	if !strings.Contains(out, "Yesterday & Today") {
		t.Errorf("expected period label in header, got:\n%s", out)
	}
	// Task names
	if !strings.Contains(out, "Write summary") {
		t.Errorf("expected 'Write summary', got:\n%s", out)
	}
	if !strings.Contains(out, "Fix proxy") {
		t.Errorf("expected 'Fix proxy', got:\n%s", out)
	}
	// Parent context
	if !strings.Contains(out, "Web app polish") {
		t.Errorf("expected parent context, got:\n%s", out)
	}
	if !strings.Contains(out, "under:") {
		t.Errorf("expected '(under: ...)' context, got:\n%s", out)
	}
	// Totals
	if !strings.Contains(out, "3 tasks done") {
		t.Errorf("expected '3 tasks done', got:\n%s", out)
	}
	if !strings.Contains(out, "5 pomodoros burned") {
		t.Errorf("expected '5 pomodoros burned', got:\n%s", out)
	}
}

// TestRenderReport_DayOrdering checks most-recent-day-first ordering.
func TestRenderReport_DayOrdering(t *testing.T) {
	now := reportNowForTest
	p, _ := report.ParsePeriod("recent", "", "", now)

	jun11 := time.Date(2025, 6, 11, 12, 0, 0, 0, time.Local).UTC()
	jun10 := time.Date(2025, 6, 10, 14, 0, 0, 0, time.Local).UTC()

	tasks := []*taskv1.Task{
		makeCompletedTask(1, "Jun11 task", jun11, nil),
		makeCompletedTask(2, "Jun10 task", jun10, nil),
	}

	entries := report.BuildEntries(tasks, p)
	totals := report.Totals{TasksCompleted: len(entries)}

	var buf bytes.Buffer
	renderReport(&buf, p, entries, tasks, totals, false)
	out := buf.String()

	idx11 := strings.Index(out, "Jun 11")
	idx10 := strings.Index(out, "Jun 10")
	if idx11 < 0 || idx10 < 0 {
		t.Fatalf("missing day headers in output:\n%s", out)
	}
	if idx11 > idx10 {
		t.Errorf("Jun 11 should appear before Jun 10 (most recent first):\n%s", out)
	}
}

// TestRenderReport_EmptyState checks the empty state output.
func TestRenderReport_EmptyState(t *testing.T) {
	now := reportNowForTest
	p, _ := report.ParsePeriod("today", "", "", now)

	var buf bytes.Buffer
	renderReport(&buf, p, nil, nil, report.Totals{}, false)
	out := buf.String()

	if !strings.Contains(out, "Nothing checked off") {
		t.Errorf("expected empty state message, got:\n%s", out)
	}
}

// TestRenderReport_PlainWhenNoTTY checks that no ANSI escapes appear in plain output.
func TestRenderReport_PlainWhenNoTTY(t *testing.T) {
	now := reportNowForTest
	p, _ := report.ParsePeriod("recent", "", "", now)

	jun11 := time.Date(2025, 6, 11, 12, 0, 0, 0, time.Local).UTC()
	tasks := []*taskv1.Task{makeCompletedTask(1, "task1", jun11, nil)}
	entries := report.BuildEntries(tasks, p)

	var buf bytes.Buffer
	renderReport(&buf, p, entries, tasks, report.Totals{TasksCompleted: 1}, false)
	out := buf.String()

	if strings.Contains(out, "\x1b[") {
		t.Errorf("expected no ANSI escapes in plain output, got:\n%s", out)
	}
}

// TestRenderReport_AccomplishmentLayout checks that periods > 14 days render
// the accomplishment-grouped layout with Finished and ongoing sections.
func TestRenderReport_AccomplishmentLayout(t *testing.T) {
	now := reportNowForTest
	p, _ := report.ParsePeriod("quarter", "", "", now)
	if p.ReportLayout() != report.AccomplishmentGrouped {
		t.Skip("quarter unexpectedly short; skip")
	}

	jun2 := time.Date(2025, 6, 2, 12, 0, 0, 0, time.Local).UTC()
	mayDeadline := time.Date(2025, 5, 20, 12, 0, 0, 0, time.Local).UTC()

	tasks := []*taskv1.Task{
		makeCompletedTask(1, "Ship alternate profiles", jun2, nil),
		makeCompletedTask(2, "Config schema", mayDeadline, int64p(1)),
		{Id: 3, Name: "Web app polish"},
		makeCompletedTask(4, "Wire download page", jun2, int64p(3)),
	}

	entries := report.BuildEntries(tasks, p)
	totals := report.Totals{TasksCompleted: len(entries), PomodorosCompleted: 10}

	var buf bytes.Buffer
	renderReport(&buf, p, entries, tasks, totals, false)
	out := buf.String()

	if !strings.Contains(out, "Finished") {
		t.Errorf("expected 'Finished' section, got:\n%s", out)
	}
	if !strings.Contains(out, "Progress on ongoing work") {
		t.Errorf("expected 'Progress on ongoing work' section, got:\n%s", out)
	}
	if !strings.Contains(out, "Ship alternate profiles") {
		t.Errorf("expected finished top-level task, got:\n%s", out)
	}
	// Finished headlines carry their completion date (contract: "✓ <name>  (done <date>)").
	if !strings.Contains(out, "Ship alternate profiles  (done Jun 2)") {
		t.Errorf("expected '(done Jun 2)' on the finished headline, got:\n%s", out)
	}
	if !strings.Contains(out, "Web app polish") {
		t.Errorf("expected ongoing group, got:\n%s", out)
	}
	// Totals line at top for accomplishment layout.
	if !strings.Contains(out, "tasks done") {
		t.Errorf("expected totals line, got:\n%s", out)
	}
}

// TestRenderReport_StyledHasANSI checks that ANSI codes appear in styled output.
func TestRenderReport_StyledHasANSI(t *testing.T) {
	now := reportNowForTest
	p, _ := report.ParsePeriod("recent", "", "", now)

	jun11 := time.Date(2025, 6, 11, 12, 0, 0, 0, time.Local).UTC()
	tasks := []*taskv1.Task{makeCompletedTask(1, "task1", jun11, nil)}
	entries := report.BuildEntries(tasks, p)

	var buf bytes.Buffer
	renderReport(&buf, p, entries, tasks, report.Totals{TasksCompleted: 1}, true)
	out := buf.String()

	if !strings.Contains(out, "\x1b[") {
		t.Errorf("expected ANSI escapes in styled output, got:\n%s", out)
	}
}

// TestDoReport_EndToEnd exercises the full fetch-and-render path: ListTasks +
// CountCompletedPomodoros over the wire, then day-grouped output and exit 0.
func TestDoReport_EndToEnd(t *testing.T) {
	client, svc, srv := newReportHarness(t, 7)

	completed := time.Date(2025, 6, 11, 10, 0, 0, 0, time.Local).UTC()
	svc.mu.Lock()
	svc.tasks[1] = makeCompletedTask(1, "Write summary", completed, nil)
	svc.tasks[2] = &taskv1.Task{Id: 2, Name: "Still open"}
	svc.nextID = 3
	svc.mu.Unlock()

	p, _ := report.ParsePeriod("recent", "", "", reportNowForTest)

	var buf bytes.Buffer
	code := doReport(client, srv.URL, p, &buf)
	if code != 0 {
		t.Fatalf("doReport exit = %d, want 0", code)
	}
	out := buf.String()
	if !strings.Contains(out, "Write summary") {
		t.Errorf("expected completed task in output, got:\n%s", out)
	}
	if strings.Contains(out, "Still open") {
		t.Errorf("incomplete task must not appear, got:\n%s", out)
	}
	if !strings.Contains(out, "7 pomodoros burned") {
		t.Errorf("expected pomodoro count from RPC, got:\n%s", out)
	}
}

// TestDoReport_ServerError verifies that an unreachable server yields exit 1
// and no partial report output (FR-013).
func TestDoReport_ServerError(t *testing.T) {
	client, _, srv := newReportHarness(t, 0)
	srv.Close() // make every RPC fail

	p, _ := report.ParsePeriod("recent", "", "", reportNowForTest)

	var buf bytes.Buffer
	code := doReport(client, srv.URL, p, &buf)
	if code != 1 {
		t.Fatalf("doReport exit = %d, want 1", code)
	}
	if buf.Len() != 0 {
		t.Errorf("expected no partial report output on error, got:\n%s", buf.String())
	}
}
