package handler_test

import (
	"context"
	"fmt"
	"strings"
	"sync"
	"testing"
	"time"

	"connectrpc.com/connect"
	"github.com/jackc/pgx/v5/pgtype"
	"github.com/jackc/pgx/v5/pgxpool"

	planv1 "github.com/pboyd/todo/services/todo/gen/plan/v1"
	taskv1 "github.com/pboyd/todo/services/todo/gen/task/v1"
	"github.com/pboyd/todo/services/todo/internal/auth"
	"github.com/pboyd/todo/services/todo/internal/db"
	"github.com/pboyd/todo/services/todo/internal/handler"
)

func newTestPlanHandler(t *testing.T) (*handler.Plan, *handler.Task, int64) {
	t.Helper()
	taskH, userID := newTestHandler(t)

	pool := testPool(t)
	// testPool already registers cleanup; we just use the same DSN.
	planH := &handler.Plan{
		Queries: taskH.Queries,
		Pool:    pool,
	}
	t.Cleanup(func() {
		pool.Exec(context.Background(), "DELETE FROM plan_entries WHERE user_id = $1", userID)
	})
	return planH, taskH, userID
}

func newTestPlanPool(t *testing.T) (*handler.Plan, *handler.Task, int64, *pgxpool.Pool) {
	t.Helper()
	taskH, userID := newTestHandler(t)
	pool := testPool(t)
	planH := &handler.Plan{
		Queries: taskH.Queries,
		Pool:    pool,
	}
	t.Cleanup(func() {
		pool.Exec(context.Background(), "DELETE FROM plan_entries WHERE user_id = $1", userID)
	})
	return planH, taskH, userID, pool
}

func todayStr() string { return time.Now().Format("2006-01-02") }

func insertPlanEntry(t *testing.T, q *db.Queries, userID int64, day string, taskID int64, name string, start, dur int) db.PlanEntry {
	t.Helper()
	d, err := time.Parse("2006-01-02", day)
	if err != nil {
		t.Fatalf("parse day: %v", err)
	}
	pgDay := pgtype.Date{Time: d, Valid: true}

	nextID, err := q.NextPlanEntryId(context.Background(), db.NextPlanEntryIdParams{UserID: userID, Day: pgDay})
	if err != nil {
		t.Fatalf("NextPlanEntryId: %v", err)
	}

	params := db.InsertPlanEntryParams{
		UserID:         userID,
		Day:            pgDay,
		ID:             nextID,
		StartMinute:    int16(start),
		DurationMinute: int16(dur),
	}
	if taskID != 0 {
		params.TaskID = pgtype.Int8{Int64: taskID, Valid: true}
	}
	if name != "" {
		params.Name = pgtype.Text{String: name, Valid: true}
	}

	entry, err := q.InsertPlanEntry(context.Background(), params)
	if err != nil {
		t.Fatalf("InsertPlanEntry: %v", err)
	}
	return entry
}

// ---- T013: ListPlanEntries ----

func TestListPlanEntries(t *testing.T) {
	planH, _, userID := newTestPlanHandler(t)
	ctx := ctxWithUser(userID)
	day := todayStr()

	insertPlanEntry(t, planH.Queries, userID, day, 0, "Lunch", 720, 60)
	insertPlanEntry(t, planH.Queries, userID, day, 0, "Meeting", 540, 30)

	resp, err := planH.ListPlanEntries(ctx, connect.NewRequest(&planv1.ListPlanEntriesRequest{Day: day}))
	if err != nil {
		t.Fatalf("ListPlanEntries: %v", err)
	}
	if len(resp.Msg.Entries) != 2 {
		t.Fatalf("expected 2 entries, got %d", len(resp.Msg.Entries))
	}
	if resp.Msg.Entries[0].StartMinute >= resp.Msg.Entries[1].StartMinute {
		t.Errorf("entries not sorted by start_minute: %d, %d",
			resp.Msg.Entries[0].StartMinute, resp.Msg.Entries[1].StartMinute)
	}
}

func TestListPlanEntries_Isolation(t *testing.T) {
	planH, _, userID := newTestPlanHandler(t)
	taskH2, userID2 := newTestHandler(t)
	pool2 := testPool(t)
	planH2 := &handler.Plan{Queries: taskH2.Queries, Pool: pool2}
	t.Cleanup(func() {
		pool2.Exec(context.Background(), "DELETE FROM plan_entries WHERE user_id = $1", userID2)
	})

	day := todayStr()
	// Insert entry for user2.
	insertPlanEntry(t, planH2.Queries, userID2, day, 0, "Other user entry", 480, 30)

	// User1 should see nothing.
	resp, err := planH.ListPlanEntries(ctxWithUser(userID), connect.NewRequest(&planv1.ListPlanEntriesRequest{Day: day}))
	if err != nil {
		t.Fatalf("ListPlanEntries: %v", err)
	}
	if len(resp.Msg.Entries) != 0 {
		t.Errorf("expected 0 entries for user1, got %d", len(resp.Msg.Entries))
	}
}

// ---- T025: ListPlanEntries name fallback ----

func TestListPlanEntries_NameFallback(t *testing.T) {
	planH, taskH, userID := newTestPlanHandler(t)
	ctx := ctxWithUser(userID)
	day := todayStr()

	taskResp, err := taskH.CreateTask(ctx, connect.NewRequest(&taskv1.CreateTaskRequest{Name: "Deep Work"}))
	if err != nil {
		t.Fatalf("CreateTask: %v", err)
	}
	taskID := taskResp.Msg.Task.Id

	// Entry with task_id but no name (should fall back to task name).
	insertPlanEntry(t, planH.Queries, userID, day, taskID, "", 480, 60)
	// Entry with explicit name and no task_id.
	insertPlanEntry(t, planH.Queries, userID, day, 0, "Lunch", 720, 60)

	resp, err := planH.ListPlanEntries(ctx, connect.NewRequest(&planv1.ListPlanEntriesRequest{Day: day}))
	if err != nil {
		t.Fatalf("ListPlanEntries: %v", err)
	}
	if len(resp.Msg.Entries) != 2 {
		t.Fatalf("expected 2 entries, got %d", len(resp.Msg.Entries))
	}
	for _, e := range resp.Msg.Entries {
		if e.StartMinute == 480 && e.Name != "Deep Work" {
			t.Errorf("task-linked entry: name = %q, want %q", e.Name, "Deep Work")
		}
		if e.StartMinute == 720 && e.Name != "Lunch" {
			t.Errorf("event entry: name = %q, want %q", e.Name, "Lunch")
		}
	}
}

// ---- T018: AddPlanEvent tests ----

func TestAddPlanEvent(t *testing.T) {
	planH, _, userID := newTestPlanHandler(t)
	ctx := ctxWithUser(userID)
	day := todayStr()

	t.Run("happy path explicit duration", func(t *testing.T) {
		resp, err := planH.AddPlanEvent(ctx, connect.NewRequest(&planv1.AddPlanEventRequest{
			Day:            day,
			Name:           "Standup",
			StartMinute:    540,
			DurationMinute: 15,
		}))
		if err != nil {
			t.Fatalf("AddPlanEvent: %v", err)
		}
		e := resp.Msg.Entry
		if e.Name != "Standup" {
			t.Errorf("name = %q, want %q", e.Name, "Standup")
		}
		if e.DurationMinute != 15 {
			t.Errorf("duration = %d, want 15", e.DurationMinute)
		}
		if e.Id != 1 {
			t.Errorf("id = %d, want 1", e.Id)
		}
	})

	t.Run("default 30 when duration 0", func(t *testing.T) {
		resp, err := planH.AddPlanEvent(ctx, connect.NewRequest(&planv1.AddPlanEventRequest{
			Day:            day,
			Name:           "Coffee",
			StartMinute:    600,
			DurationMinute: 0,
		}))
		if err != nil {
			t.Fatalf("AddPlanEvent: %v", err)
		}
		if resp.Msg.Entry.DurationMinute != 30 {
			t.Errorf("duration = %d, want 30", resp.Msg.Entry.DurationMinute)
		}
	})

	t.Run("empty name rejected", func(t *testing.T) {
		_, err := planH.AddPlanEvent(ctx, connect.NewRequest(&planv1.AddPlanEventRequest{
			Day:         day,
			Name:        "",
			StartMinute: 700,
		}))
		if connect.CodeOf(err) != connect.CodeInvalidArgument {
			t.Errorf("expected InvalidArgument, got %v", err)
		}
	})

	t.Run("out of range start_minute rejected", func(t *testing.T) {
		_, err := planH.AddPlanEvent(ctx, connect.NewRequest(&planv1.AddPlanEventRequest{
			Day:         day,
			Name:        "Bad",
			StartMinute: 1440,
		}))
		if connect.CodeOf(err) != connect.CodeInvalidArgument {
			t.Errorf("expected InvalidArgument, got %v", err)
		}
	})

	t.Run("midnight crossing rejected", func(t *testing.T) {
		_, err := planH.AddPlanEvent(ctx, connect.NewRequest(&planv1.AddPlanEventRequest{
			Day:            day,
			Name:           "Late",
			StartMinute:    1400,
			DurationMinute: 60,
		}))
		if connect.CodeOf(err) != connect.CodeInvalidArgument {
			t.Errorf("expected InvalidArgument, got %v", err)
		}
	})

	t.Run("overlap rejected with conflicting id", func(t *testing.T) {
		// Standup at 540–555 already exists from above.
		_, err := planH.AddPlanEvent(ctx, connect.NewRequest(&planv1.AddPlanEventRequest{
			Day:            day,
			Name:           "Overlap",
			StartMinute:    542,
			DurationMinute: 15,
		}))
		if connect.CodeOf(err) != connect.CodeFailedPrecondition {
			t.Fatalf("expected FailedPrecondition, got %v", err)
		}
		if !strings.Contains(err.Error(), "1") {
			t.Errorf("error message should name conflicting id 1, got: %v", err)
		}
	})

	t.Run("touching boundary allowed", func(t *testing.T) {
		// Standup ends at 555, so starting at 555 should be fine.
		_, err := planH.AddPlanEvent(ctx, connect.NewRequest(&planv1.AddPlanEventRequest{
			Day:            day,
			Name:           "After standup",
			StartMinute:    555,
			DurationMinute: 30,
		}))
		if err != nil {
			t.Errorf("touching boundary should be allowed, got: %v", err)
		}
	})

	t.Run("id allocation no renumbering after delete", func(t *testing.T) {
		dayFr := "2099-01-01"
		// Insert A(1), B(2), C(3); delete B(2); existing [1,3]; next insert gets 4.
		resp1, _ := planH.AddPlanEvent(ctx, connect.NewRequest(&planv1.AddPlanEventRequest{
			Day: dayFr, Name: "A", StartMinute: 60, DurationMinute: 30,
		}))
		resp2, _ := planH.AddPlanEvent(ctx, connect.NewRequest(&planv1.AddPlanEventRequest{
			Day: dayFr, Name: "B", StartMinute: 120, DurationMinute: 30,
		}))
		resp3, _ := planH.AddPlanEvent(ctx, connect.NewRequest(&planv1.AddPlanEventRequest{
			Day: dayFr, Name: "C", StartMinute: 180, DurationMinute: 30,
		}))
		if resp1 == nil || resp2 == nil || resp3 == nil {
			t.Fatal("setup inserts failed")
		}
		// Remove the middle entry; remaining ids are [1, 3].
		planH.RemovePlanEntry(ctx, connect.NewRequest(&planv1.RemovePlanEntryRequest{
			Day: dayFr, Id: resp2.Msg.Entry.Id,
		}))
		resp4, err := planH.AddPlanEvent(ctx, connect.NewRequest(&planv1.AddPlanEventRequest{
			Day: dayFr, Name: "D", StartMinute: 240, DurationMinute: 30,
		}))
		if err != nil {
			t.Fatalf("fourth insert: %v", err)
		}
		// MAX(1,3)+1 = 4, not 2 (no reuse of deleted id 2, and no renumbering of id 3).
		if resp4.Msg.Entry.Id != 4 {
			t.Errorf("id = %d, want 4 (MAX(1,3)+1)", resp4.Msg.Entry.Id)
		}
	})
}

// ---- T019: AddPlanTask tests ----

func TestAddPlanTask(t *testing.T) {
	planH, taskH, userID := newTestPlanHandler(t)
	ctx := ctxWithUser(userID)
	day := "2099-02-01"

	taskResp, err := taskH.CreateTask(ctx, connect.NewRequest(&taskv1.CreateTaskRequest{Name: "Feature X"}))
	if err != nil {
		t.Fatalf("CreateTask: %v", err)
	}
	taskID := taskResp.Msg.Task.Id

	t.Run("happy path explicit duration", func(t *testing.T) {
		resp, err := planH.AddPlanTask(ctx, connect.NewRequest(&planv1.AddPlanTaskRequest{
			Day:            day,
			TaskId:         taskID,
			StartMinute:    480,
			DurationMinute: 90,
		}))
		if err != nil {
			t.Fatalf("AddPlanTask: %v", err)
		}
		if resp.Msg.Entry.DurationMinute != 90 {
			t.Errorf("duration = %d, want 90", resp.Msg.Entry.DurationMinute)
		}
		if resp.Msg.Entry.TaskId != taskID {
			t.Errorf("task_id = %d, want %d", resp.Msg.Entry.TaskId, taskID)
		}
	})

	t.Run("default duration from estimate minus completed", func(t *testing.T) {
		// Set estimate to 3.
		_, err := taskH.SetEstimate(ctx, connect.NewRequest(&taskv1.SetEstimateRequest{
			TaskId:   taskID,
			Estimate: 3,
		}))
		if err != nil {
			t.Fatalf("SetEstimate: %v", err)
		}
		// Start one pomodoro to simulate 1 completed.
		_, err = taskH.StartPomodoro(ctx, connect.NewRequest(&taskv1.StartPomodoroRequest{TaskId: taskID}))
		if err != nil {
			t.Fatalf("StartPomodoro: %v", err)
		}
		_, err = taskH.CompletePomodoro(ctx, connect.NewRequest(&taskv1.CompletePomodoroRequest{}))
		if err != nil {
			t.Fatalf("CompletePomodoro: %v", err)
		}
		// estimate=3, completed=1 → remaining=2 → duration=60.
		day2 := "2099-02-02"
		resp, err := planH.AddPlanTask(ctx, connect.NewRequest(&planv1.AddPlanTaskRequest{
			Day:         day2,
			TaskId:      taskID,
			StartMinute: 480,
		}))
		if err != nil {
			t.Fatalf("AddPlanTask: %v", err)
		}
		if resp.Msg.Entry.DurationMinute != 60 {
			t.Errorf("duration = %d, want 60 (2 remaining * 30)", resp.Msg.Entry.DurationMinute)
		}
	})

	t.Run("default 30 when estimate is 0", func(t *testing.T) {
		taskR, _ := taskH.CreateTask(ctx, connect.NewRequest(&taskv1.CreateTaskRequest{Name: "No estimate"}))
		day3 := "2099-02-03"
		resp, err := planH.AddPlanTask(ctx, connect.NewRequest(&planv1.AddPlanTaskRequest{
			Day:         day3,
			TaskId:      taskR.Msg.Task.Id,
			StartMinute: 480,
		}))
		if err != nil {
			t.Fatalf("AddPlanTask: %v", err)
		}
		if resp.Msg.Entry.DurationMinute != 30 {
			t.Errorf("duration = %d, want 30", resp.Msg.Entry.DurationMinute)
		}
	})

	t.Run("not found for unknown task", func(t *testing.T) {
		_, err := planH.AddPlanTask(ctx, connect.NewRequest(&planv1.AddPlanTaskRequest{
			Day:         day,
			TaskId:      9999999,
			StartMinute: 600,
		}))
		if connect.CodeOf(err) != connect.CodeNotFound {
			t.Errorf("expected NotFound, got %v", err)
		}
	})

	t.Run("not found when task belongs to another user", func(t *testing.T) {
		_, userID2 := newTestHandler(t)
		ctx2 := auth.WithUserID(context.Background(), userID2)
		_, err := planH.AddPlanTask(ctx2, connect.NewRequest(&planv1.AddPlanTaskRequest{
			Day:         day,
			TaskId:      taskID,
			StartMinute: 600,
		}))
		if connect.CodeOf(err) != connect.CodeNotFound {
			t.Errorf("expected NotFound, got %v", err)
		}
	})

	t.Run("overlap rejected", func(t *testing.T) {
		day4 := "2099-02-04"
		planH.AddPlanTask(ctx, connect.NewRequest(&planv1.AddPlanTaskRequest{
			Day: day4, TaskId: taskID, StartMinute: 480, DurationMinute: 60,
		}))
		_, err := planH.AddPlanTask(ctx, connect.NewRequest(&planv1.AddPlanTaskRequest{
			Day: day4, TaskId: taskID, StartMinute: 500, DurationMinute: 30,
		}))
		if connect.CodeOf(err) != connect.CodeFailedPrecondition {
			t.Errorf("expected FailedPrecondition, got %v", err)
		}
	})
}

// ---- T020: Concurrency test ----

func TestAddPlanEvent_Concurrency(t *testing.T) {
	planH1, _, userID := newTestPlanHandler(t)
	// Use a second pool instance so both goroutines get separate connections.
	pool2 := testPool(t)
	planH2 := &handler.Plan{Queries: db.New(pool2), Pool: pool2}

	ctx := ctxWithUser(userID)
	day := "2099-03-01"

	var wg sync.WaitGroup
	errs := make([]error, 2)
	wg.Add(2)
	go func() {
		defer wg.Done()
		_, errs[0] = planH1.AddPlanEvent(ctx, connect.NewRequest(&planv1.AddPlanEventRequest{
			Day: day, Name: "Race A", StartMinute: 480, DurationMinute: 30,
		}))
	}()
	go func() {
		defer wg.Done()
		_, errs[1] = planH2.AddPlanEvent(ctx, connect.NewRequest(&planv1.AddPlanEventRequest{
			Day: day, Name: "Race B", StartMinute: 480, DurationMinute: 30,
		}))
	}()
	wg.Wait()

	successes := 0
	failures := 0
	for _, err := range errs {
		if err == nil {
			successes++
		} else {
			failures++
		}
	}
	if successes != 1 {
		t.Errorf("expected exactly 1 success, got %d (errs: %v, %v)", successes, errs[0], errs[1])
	}

	// Verify exactly one row in DB.
	pool3 := testPool(t)
	var count int
	pool3.QueryRow(context.Background(),
		"SELECT COUNT(*) FROM plan_entries WHERE user_id = $1 AND day = $2", userID, day).Scan(&count)
	if count != 1 {
		t.Errorf("expected 1 row in DB, got %d", count)
	}
}

// ---- T033: RemovePlanEntry tests ----

func TestRemovePlanEntry(t *testing.T) {
	planH, _, userID := newTestPlanHandler(t)
	ctx := ctxWithUser(userID)
	day := "2099-04-01"

	e := insertPlanEntry(t, planH.Queries, userID, day, 0, "To remove", 480, 30)

	t.Run("happy path", func(t *testing.T) {
		_, err := planH.RemovePlanEntry(ctx, connect.NewRequest(&planv1.RemovePlanEntryRequest{
			Day: day, Id: e.ID,
		}))
		if err != nil {
			t.Fatalf("RemovePlanEntry: %v", err)
		}
	})

	t.Run("not found after removal", func(t *testing.T) {
		_, err := planH.RemovePlanEntry(ctx, connect.NewRequest(&planv1.RemovePlanEntryRequest{
			Day: day, Id: e.ID,
		}))
		if connect.CodeOf(err) != connect.CodeNotFound {
			t.Errorf("expected NotFound, got %v", err)
		}
	})

	t.Run("isolation another user cannot remove", func(t *testing.T) {
		_, userID2 := newTestHandler(t)
		ctx2 := auth.WithUserID(context.Background(), userID2)
		e2 := insertPlanEntry(t, planH.Queries, userID, day, 0, "Mine", 600, 30)
		_, err := planH.RemovePlanEntry(ctx2, connect.NewRequest(&planv1.RemovePlanEntryRequest{
			Day: day, Id: e2.ID,
		}))
		if connect.CodeOf(err) != connect.CodeNotFound {
			t.Errorf("expected NotFound for wrong user, got %v", err)
		}
	})

	t.Run("no renumbering after delete", func(t *testing.T) {
		day2 := "2099-04-02"
		e1 := insertPlanEntry(t, planH.Queries, userID, day2, 0, "First", 480, 30)
		e2 := insertPlanEntry(t, planH.Queries, userID, day2, 0, "Second", 540, 30)
		insertPlanEntry(t, planH.Queries, userID, day2, 0, "Third", 600, 30)

		planH.RemovePlanEntry(ctx, connect.NewRequest(&planv1.RemovePlanEntryRequest{
			Day: day2, Id: e2.ID,
		}))

		resp, err := planH.AddPlanEvent(ctx, connect.NewRequest(&planv1.AddPlanEventRequest{
			Day: day2, Name: "Fourth", StartMinute: 660, DurationMinute: 30,
		}))
		if err != nil {
			t.Fatalf("AddPlanEvent: %v", err)
		}
		if resp.Msg.Entry.Id != e1.ID+3 {
			t.Logf("ids: e1=%d, new=%d", e1.ID, resp.Msg.Entry.Id)
		}
		// Specifically, must NOT be e2.ID (no renumbering).
		if resp.Msg.Entry.Id == e2.ID {
			t.Errorf("new entry got renumbered id %d; expected id > max existing", resp.Msg.Entry.Id)
		}
	})
}

// ---- T034: RenamePlanEntry tests ----

func TestRenamePlanEntry(t *testing.T) {
	planH, taskH, userID := newTestPlanHandler(t)
	ctx := ctxWithUser(userID)
	day := "2099-05-01"

	e := insertPlanEntry(t, planH.Queries, userID, day, 0, "Old name", 480, 30)

	t.Run("happy path", func(t *testing.T) {
		resp, err := planH.RenamePlanEntry(ctx, connect.NewRequest(&planv1.RenamePlanEntryRequest{
			Day: day, Id: e.ID, Name: "New name",
		}))
		if err != nil {
			t.Fatalf("RenamePlanEntry: %v", err)
		}
		if resp.Msg.Entry.Name != "New name" {
			t.Errorf("name = %q, want %q", resp.Msg.Entry.Name, "New name")
		}
	})

	t.Run("empty name rejected", func(t *testing.T) {
		_, err := planH.RenamePlanEntry(ctx, connect.NewRequest(&planv1.RenamePlanEntryRequest{
			Day: day, Id: e.ID, Name: "",
		}))
		if connect.CodeOf(err) != connect.CodeInvalidArgument {
			t.Errorf("expected InvalidArgument, got %v", err)
		}
	})

	t.Run("whitespace-only name rejected", func(t *testing.T) {
		_, err := planH.RenamePlanEntry(ctx, connect.NewRequest(&planv1.RenamePlanEntryRequest{
			Day: day, Id: e.ID, Name: "   ",
		}))
		if connect.CodeOf(err) != connect.CodeInvalidArgument {
			t.Errorf("expected InvalidArgument, got %v", err)
		}
	})

	t.Run("name too long rejected", func(t *testing.T) {
		_, err := planH.RenamePlanEntry(ctx, connect.NewRequest(&planv1.RenamePlanEntryRequest{
			Day: day, Id: e.ID, Name: strings.Repeat("x", 256),
		}))
		if connect.CodeOf(err) != connect.CodeInvalidArgument {
			t.Errorf("expected InvalidArgument, got %v", err)
		}
	})

	t.Run("not found for unknown id", func(t *testing.T) {
		_, err := planH.RenamePlanEntry(ctx, connect.NewRequest(&planv1.RenamePlanEntryRequest{
			Day: day, Id: 9999, Name: "Whatever",
		}))
		if connect.CodeOf(err) != connect.CodeNotFound {
			t.Errorf("expected NotFound, got %v", err)
		}
	})

	t.Run("rename task-linked overrides task name on list", func(t *testing.T) {
		taskResp, _ := taskH.CreateTask(ctx, connect.NewRequest(&taskv1.CreateTaskRequest{Name: "Task Z"}))
		taskID := taskResp.Msg.Task.Id
		day2 := "2099-05-02"
		te := insertPlanEntry(t, planH.Queries, userID, day2, taskID, "", 480, 30)

		planH.RenamePlanEntry(ctx, connect.NewRequest(&planv1.RenamePlanEntryRequest{
			Day: day2, Id: te.ID, Name: "Overridden",
		}))

		listResp, err := planH.ListPlanEntries(ctx, connect.NewRequest(&planv1.ListPlanEntriesRequest{Day: day2}))
		if err != nil {
			t.Fatalf("ListPlanEntries: %v", err)
		}
		if len(listResp.Msg.Entries) != 1 {
			t.Fatalf("expected 1 entry, got %d", len(listResp.Msg.Entries))
		}
		if listResp.Msg.Entries[0].Name != "Overridden" {
			t.Errorf("name = %q, want %q", listResp.Msg.Entries[0].Name, "Overridden")
		}
	})
}

// ---- T035: MovePlanEntry tests ----

func TestMovePlanEntry(t *testing.T) {
	planH, _, userID := newTestPlanHandler(t)
	ctx := ctxWithUser(userID)
	day := "2099-06-01"

	e := insertPlanEntry(t, planH.Queries, userID, day, 0, "Movable", 480, 60)

	t.Run("happy path start and duration updated", func(t *testing.T) {
		resp, err := planH.MovePlanEntry(ctx, connect.NewRequest(&planv1.MovePlanEntryRequest{
			Day: day, Id: e.ID, StartMinute: 600, DurationMinute: 45,
		}))
		if err != nil {
			t.Fatalf("MovePlanEntry: %v", err)
		}
		if resp.Msg.Entry.StartMinute != 600 {
			t.Errorf("start = %d, want 600", resp.Msg.Entry.StartMinute)
		}
		if resp.Msg.Entry.DurationMinute != 45 {
			t.Errorf("duration = %d, want 45", resp.Msg.Entry.DurationMinute)
		}
	})

	t.Run("duration 0 preserves prior duration", func(t *testing.T) {
		day2 := "2099-06-02"
		e2 := insertPlanEntry(t, planH.Queries, userID, day2, 0, "Fixed dur", 480, 90)
		resp, err := planH.MovePlanEntry(ctx, connect.NewRequest(&planv1.MovePlanEntryRequest{
			Day: day2, Id: e2.ID, StartMinute: 600, DurationMinute: 0,
		}))
		if err != nil {
			t.Fatalf("MovePlanEntry: %v", err)
		}
		if resp.Msg.Entry.DurationMinute != 90 {
			t.Errorf("duration = %d, want 90 (preserved)", resp.Msg.Entry.DurationMinute)
		}
	})

	t.Run("overlap with another entry rejected", func(t *testing.T) {
		day3 := "2099-06-03"
		e1 := insertPlanEntry(t, planH.Queries, userID, day3, 0, "Block A", 480, 60)
		insertPlanEntry(t, planH.Queries, userID, day3, 0, "Block B", 600, 60)
		// Try to move e1 to overlap Block B.
		_, err := planH.MovePlanEntry(ctx, connect.NewRequest(&planv1.MovePlanEntryRequest{
			Day: day3, Id: e1.ID, StartMinute: 620, DurationMinute: 60,
		}))
		if connect.CodeOf(err) != connect.CodeFailedPrecondition {
			t.Errorf("expected FailedPrecondition, got %v", err)
		}
	})

	t.Run("move to touching boundary allowed", func(t *testing.T) {
		day4 := "2099-06-04"
		e1 := insertPlanEntry(t, planH.Queries, userID, day4, 0, "A", 480, 60)
		insertPlanEntry(t, planH.Queries, userID, day4, 0, "B", 600, 60)
		// Move e1 to end exactly at 600 (touching B's start).
		_, err := planH.MovePlanEntry(ctx, connect.NewRequest(&planv1.MovePlanEntryRequest{
			Day: day4, Id: e1.ID, StartMinute: 540, DurationMinute: 60,
		}))
		if err != nil {
			t.Errorf("touching boundary should be allowed, got: %v", err)
		}
	})

	t.Run("past midnight rejected", func(t *testing.T) {
		_, err := planH.MovePlanEntry(ctx, connect.NewRequest(&planv1.MovePlanEntryRequest{
			Day: day, Id: e.ID, StartMinute: 1400, DurationMinute: 60,
		}))
		if connect.CodeOf(err) != connect.CodeInvalidArgument {
			t.Errorf("expected InvalidArgument, got %v", err)
		}
	})

	t.Run("not found for unknown id", func(t *testing.T) {
		_, err := planH.MovePlanEntry(ctx, connect.NewRequest(&planv1.MovePlanEntryRequest{
			Day: day, Id: 9999, StartMinute: 300, DurationMinute: 30,
		}))
		if connect.CodeOf(err) != connect.CodeNotFound {
			t.Errorf("expected NotFound, got %v", err)
		}
	})

	t.Run("isolation another user cannot move", func(t *testing.T) {
		_, userID2 := newTestHandler(t)
		ctx2 := auth.WithUserID(context.Background(), userID2)
		_, err := planH.MovePlanEntry(ctx2, connect.NewRequest(&planv1.MovePlanEntryRequest{
			Day: day, Id: e.ID, StartMinute: 300, DurationMinute: 30,
		}))
		if connect.CodeOf(err) != connect.CodeNotFound {
			t.Errorf("expected NotFound for wrong user, got %v", err)
		}
	})
}

// ---- T041: ClearPlan tests ----

func TestClearPlan(t *testing.T) {
	planH, _, userID := newTestPlanHandler(t)
	ctx := ctxWithUser(userID)

	t.Run("empty day no-op", func(t *testing.T) {
		resp, err := planH.ClearPlan(ctx, connect.NewRequest(&planv1.ClearPlanRequest{
			Day: "2099-07-01", StartMinute: 480,
		}))
		if err != nil {
			t.Fatalf("ClearPlan: %v", err)
		}
		if resp.Msg.DeletedCount != 0 || resp.Msg.TrimmedStraddlingEntry {
			t.Errorf("expected {0, false}, got {%d, %v}", resp.Msg.DeletedCount, resp.Msg.TrimmedStraddlingEntry)
		}
	})

	t.Run("cutoff before all entries removes everything", func(t *testing.T) {
		day := "2099-07-02"
		insertPlanEntry(t, planH.Queries, userID, day, 0, "A", 480, 30)
		insertPlanEntry(t, planH.Queries, userID, day, 0, "B", 540, 30)
		resp, err := planH.ClearPlan(ctx, connect.NewRequest(&planv1.ClearPlanRequest{
			Day: day, StartMinute: 400,
		}))
		if err != nil {
			t.Fatalf("ClearPlan: %v", err)
		}
		if resp.Msg.DeletedCount != 2 || resp.Msg.TrimmedStraddlingEntry {
			t.Errorf("expected {2, false}, got {%d, %v}", resp.Msg.DeletedCount, resp.Msg.TrimmedStraddlingEntry)
		}
	})

	t.Run("cutoff after all entries removes nothing", func(t *testing.T) {
		day := "2099-07-03"
		insertPlanEntry(t, planH.Queries, userID, day, 0, "A", 480, 30)
		resp, err := planH.ClearPlan(ctx, connect.NewRequest(&planv1.ClearPlanRequest{
			Day: day, StartMinute: 600,
		}))
		if err != nil {
			t.Fatalf("ClearPlan: %v", err)
		}
		if resp.Msg.DeletedCount != 0 || resp.Msg.TrimmedStraddlingEntry {
			t.Errorf("expected {0, false}, got {%d, %v}", resp.Msg.DeletedCount, resp.Msg.TrimmedStraddlingEntry)
		}
	})

	t.Run("exact start is deleted not trimmed", func(t *testing.T) {
		day := "2099-07-04"
		insertPlanEntry(t, planH.Queries, userID, day, 0, "Exact", 480, 60)
		resp, err := planH.ClearPlan(ctx, connect.NewRequest(&planv1.ClearPlanRequest{
			Day: day, StartMinute: 480,
		}))
		if err != nil {
			t.Fatalf("ClearPlan: %v", err)
		}
		if resp.Msg.DeletedCount != 1 || resp.Msg.TrimmedStraddlingEntry {
			t.Errorf("expected {1, false}, got {%d, %v}", resp.Msg.DeletedCount, resp.Msg.TrimmedStraddlingEntry)
		}
	})

	t.Run("straddle trims and removes later", func(t *testing.T) {
		day := "2099-07-05"
		insertPlanEntry(t, planH.Queries, userID, day, 0, "Straddle", 480, 90) // 8:00–9:30
		insertPlanEntry(t, planH.Queries, userID, day, 0, "After", 600, 30)    // 10:00–10:30
		// Cutoff at 510 (8:30) — straddles the first entry, deletes the second.
		resp, err := planH.ClearPlan(ctx, connect.NewRequest(&planv1.ClearPlanRequest{
			Day: day, StartMinute: 510,
		}))
		if err != nil {
			t.Fatalf("ClearPlan: %v", err)
		}
		if resp.Msg.DeletedCount != 1 || !resp.Msg.TrimmedStraddlingEntry {
			t.Errorf("expected {1, true}, got {%d, %v}", resp.Msg.DeletedCount, resp.Msg.TrimmedStraddlingEntry)
		}
		// Verify trimmed entry remains with shortened duration.
		listResp, _ := planH.ListPlanEntries(ctx, connect.NewRequest(&planv1.ListPlanEntriesRequest{Day: day}))
		if len(listResp.Msg.Entries) != 1 {
			t.Fatalf("expected 1 remaining entry, got %d", len(listResp.Msg.Entries))
		}
		if listResp.Msg.Entries[0].DurationMinute != 30 {
			t.Errorf("trimmed duration = %d, want 30", listResp.Msg.Entries[0].DurationMinute)
		}
	})
}

// ---- T006: ListPlanEntries.Completed field ----

func TestListPlanEntries_Completed(t *testing.T) {
	planH, taskH, userID := newTestPlanHandler(t)
	ctx := ctxWithUser(userID)
	day := "2099-08-01"

	// Create a task that we'll mark completed.
	taskResp, err := taskH.CreateTask(ctx, connect.NewRequest(&taskv1.CreateTaskRequest{Name: "Finish it"}))
	if err != nil {
		t.Fatalf("CreateTask: %v", err)
	}
	taskID := taskResp.Msg.Task.Id

	// Create a second task that remains incomplete.
	taskResp2, err := taskH.CreateTask(ctx, connect.NewRequest(&taskv1.CreateTaskRequest{Name: "Keep going"}))
	if err != nil {
		t.Fatalf("CreateTask: %v", err)
	}
	taskID2 := taskResp2.Msg.Task.Id

	// Event entry (no task_id) → always completed=false.
	insertPlanEntry(t, planH.Queries, userID, day, 0, "Standup", 540, 15)
	// Task entry, task not yet completed → completed=false.
	insertPlanEntry(t, planH.Queries, userID, day, taskID2, "", 600, 30)
	// Task entry, task completed → completed=true.
	insertPlanEntry(t, planH.Queries, userID, day, taskID, "", 660, 30)

	// Complete taskID.
	_, err = taskH.CompleteTask(ctx, connect.NewRequest(&taskv1.CompleteTaskRequest{Id: taskID}))
	if err != nil {
		t.Fatalf("CompleteTask: %v", err)
	}

	resp, err := planH.ListPlanEntries(ctx, connect.NewRequest(&planv1.ListPlanEntriesRequest{Day: day}))
	if err != nil {
		t.Fatalf("ListPlanEntries: %v", err)
	}
	if len(resp.Msg.Entries) != 3 {
		t.Fatalf("expected 3 entries, got %d", len(resp.Msg.Entries))
	}

	// entries are sorted by start_minute: 540 (event), 600 (incomplete task), 660 (completed task).
	event := resp.Msg.Entries[0]
	incompleteTask := resp.Msg.Entries[1]
	completedTask := resp.Msg.Entries[2]

	if event.Completed {
		t.Errorf("event entry: Completed = true, want false")
	}
	if incompleteTask.Completed {
		t.Errorf("incomplete task entry: Completed = true, want false")
	}
	if !completedTask.Completed {
		t.Errorf("completed task entry: Completed = false, want true")
	}
}

// Ensure fmt is used.
var _ = fmt.Sprintf
var _ = pgxpool.Pool{}
