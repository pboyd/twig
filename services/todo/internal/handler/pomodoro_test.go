package handler_test

import (
	"context"
	"os"
	"sync"
	"testing"
	"time"

	"connectrpc.com/connect"
	"github.com/jackc/pgx/v5/pgxpool"

	taskv1 "github.com/pboyd/todo/services/todo/gen/task/v1"
)

// testPool returns a pgxpool.Pool connected to DATABASE_URL for use in
// tests that need raw SQL access.
func testPool(t *testing.T) *pgxpool.Pool {
	t.Helper()
	dsn := os.Getenv("DATABASE_URL")
	if dsn == "" {
		t.Skip("DATABASE_URL not set; skipping integration test")
	}
	pool, err := pgxpool.New(context.Background(), dsn)
	if err != nil {
		t.Fatalf("pgxpool.New: %v", err)
	}
	t.Cleanup(pool.Close)
	return pool
}

// ---- T015: GetActivePomodoro ----

func TestGetActivePomodoro(t *testing.T) {
	h, userID := newTestHandler(t)
	ctx := ctxWithUser(userID)

	taskResp, err := h.CreateTask(ctx, connect.NewRequest(&taskv1.CreateTaskRequest{Name: "pomo t015"}))
	if err != nil {
		t.Fatalf("CreateTask: %v", err)
	}
	taskID := taskResp.Msg.Task.Id

	t.Run("returns empty when no active pomodoro", func(t *testing.T) {
		resp, err := h.GetActivePomodoro(ctx, connect.NewRequest(&taskv1.GetActivePomodoroRequest{}))
		if err != nil {
			t.Fatalf("GetActivePomodoro: %v", err)
		}
		if resp.Msg.Pomodoro != nil {
			t.Errorf("expected nil pomodoro, got %v", resp.Msg.Pomodoro)
		}
	})

	t.Run("returns active pomodoro when one exists", func(t *testing.T) {
		_, err := h.StartPomodoro(ctx, connect.NewRequest(&taskv1.StartPomodoroRequest{TaskId: taskID}))
		if err != nil {
			t.Fatalf("StartPomodoro: %v", err)
		}
		t.Cleanup(func() {
			_, _ = h.CancelPomodoro(ctx, connect.NewRequest(&taskv1.CancelPomodoroRequest{}))
		})

		resp, err := h.GetActivePomodoro(ctx, connect.NewRequest(&taskv1.GetActivePomodoroRequest{}))
		if err != nil {
			t.Fatalf("GetActivePomodoro: %v", err)
		}
		if resp.Msg.Pomodoro == nil {
			t.Fatal("expected active pomodoro, got nil")
		}
		if resp.Msg.Pomodoro.TaskId != taskID {
			t.Errorf("task_id = %d, want %d", resp.Msg.Pomodoro.TaskId, taskID)
		}
	})

	t.Run("does not return another user's active pomodoro", func(t *testing.T) {
		_, userB := newTestHandler(t)
		ctxB := ctxWithUser(userB)

		taskRespB, err := h.CreateTask(ctxB, connect.NewRequest(&taskv1.CreateTaskRequest{Name: "user B task"}))
		if err != nil {
			t.Fatalf("CreateTask for user B: %v", err)
		}

		_, err = h.StartPomodoro(ctxB, connect.NewRequest(&taskv1.StartPomodoroRequest{TaskId: taskRespB.Msg.Task.Id}))
		if err != nil {
			t.Fatalf("StartPomodoro for user B: %v", err)
		}
		t.Cleanup(func() {
			_, _ = h.CancelPomodoro(ctxB, connect.NewRequest(&taskv1.CancelPomodoroRequest{}))
		})

		resp, err := h.GetActivePomodoro(ctx, connect.NewRequest(&taskv1.GetActivePomodoroRequest{}))
		if err != nil {
			t.Fatalf("GetActivePomodoro for user A: %v", err)
		}
		if resp.Msg.Pomodoro != nil {
			t.Errorf("user A should not see user B's pomodoro")
		}
	})
}

// ---- T020: StartPomodoro ----

func TestStartPomodoro(t *testing.T) {
	h, userID := newTestHandler(t)
	ctx := ctxWithUser(userID)

	taskResp, err := h.CreateTask(ctx, connect.NewRequest(&taskv1.CreateTaskRequest{Name: "start pomo task"}))
	if err != nil {
		t.Fatalf("CreateTask: %v", err)
	}
	taskID := taskResp.Msg.Task.Id

	t.Run("happy path", func(t *testing.T) {
		before := time.Now()
		resp, err := h.StartPomodoro(ctx, connect.NewRequest(&taskv1.StartPomodoroRequest{TaskId: taskID}))
		if err != nil {
			t.Fatalf("StartPomodoro: %v", err)
		}
		t.Cleanup(func() {
			_, _ = h.CancelPomodoro(ctx, connect.NewRequest(&taskv1.CancelPomodoroRequest{}))
		})
		p := resp.Msg.Pomodoro
		if p.Id <= 0 {
			t.Errorf("expected positive id")
		}
		if p.TaskId != taskID {
			t.Errorf("task_id = %d, want %d", p.TaskId, taskID)
		}
		if p.StartAt == nil || p.StartAt.AsTime().Before(before) {
			t.Errorf("start_at not set or in past")
		}
		if p.EndAt != nil {
			t.Errorf("end_at should be nil for active pomodoro")
		}
		if p.Complete {
			t.Errorf("complete should be false")
		}
	})

	t.Run("NotFound for unknown task", func(t *testing.T) {
		_, err := h.StartPomodoro(ctx, connect.NewRequest(&taskv1.StartPomodoroRequest{TaskId: 999999}))
		if err == nil {
			t.Fatal("expected error")
		}
		ce, ok := err.(*connect.Error)
		if !ok || ce.Code() != connect.CodeNotFound {
			t.Errorf("expected CodeNotFound, got %v", err)
		}
	})

	t.Run("NotFound when task belongs to another user", func(t *testing.T) {
		_, userB := newTestHandler(t)
		ctxB := ctxWithUser(userB)
		_, err := h.StartPomodoro(ctxB, connect.NewRequest(&taskv1.StartPomodoroRequest{TaskId: taskID}))
		if err == nil {
			t.Fatal("expected error")
		}
		ce, ok := err.(*connect.Error)
		if !ok || ce.Code() != connect.CodeNotFound {
			t.Errorf("expected CodeNotFound, got %v", err)
		}
	})

	t.Run("AlreadyExists when user has active pomodoro", func(t *testing.T) {
		_, err := h.StartPomodoro(ctx, connect.NewRequest(&taskv1.StartPomodoroRequest{TaskId: taskID}))
		if err != nil {
			t.Fatalf("first StartPomodoro: %v", err)
		}
		t.Cleanup(func() {
			_, _ = h.CancelPomodoro(ctx, connect.NewRequest(&taskv1.CancelPomodoroRequest{}))
		})

		_, err = h.StartPomodoro(ctx, connect.NewRequest(&taskv1.StartPomodoroRequest{TaskId: taskID}))
		if err == nil {
			t.Fatal("expected AlreadyExists")
		}
		ce, ok := err.(*connect.Error)
		if !ok || ce.Code() != connect.CodeAlreadyExists {
			t.Errorf("expected CodeAlreadyExists, got %v", err)
		}
	})
}

// ---- T021: CompletePomodoro ----

func TestCompletePomodoro(t *testing.T) {
	h, userID := newTestHandler(t)
	ctx := ctxWithUser(userID)

	taskResp, err := h.CreateTask(ctx, connect.NewRequest(&taskv1.CreateTaskRequest{Name: "complete pomo task"}))
	if err != nil {
		t.Fatalf("CreateTask: %v", err)
	}
	taskID := taskResp.Msg.Task.Id

	t.Run("happy path", func(t *testing.T) {
		startResp, err := h.StartPomodoro(ctx, connect.NewRequest(&taskv1.StartPomodoroRequest{TaskId: taskID}))
		if err != nil {
			t.Fatalf("StartPomodoro: %v", err)
		}
		startAt := startResp.Msg.Pomodoro.StartAt.AsTime()

		resp, err := h.CompletePomodoro(ctx, connect.NewRequest(&taskv1.CompletePomodoroRequest{}))
		if err != nil {
			t.Fatalf("CompletePomodoro: %v", err)
		}
		p := resp.Msg.Pomodoro
		if !p.Complete {
			t.Errorf("complete should be true")
		}
		if p.EndAt == nil {
			t.Fatal("end_at should be set")
		}
		expectedEnd := startAt.Add(25 * time.Minute)
		if !p.EndAt.AsTime().Equal(expectedEnd) {
			t.Errorf("end_at = %v, want %v (start_at + 25 min)", p.EndAt.AsTime(), expectedEnd)
		}
	})

	t.Run("FailedPrecondition when no active pomodoro", func(t *testing.T) {
		_, err := h.CompletePomodoro(ctx, connect.NewRequest(&taskv1.CompletePomodoroRequest{}))
		if err == nil {
			t.Fatal("expected error")
		}
		ce, ok := err.(*connect.Error)
		if !ok || ce.Code() != connect.CodeFailedPrecondition {
			t.Errorf("expected CodeFailedPrecondition, got %v", err)
		}
	})

	t.Run("completed pomodoro no longer active", func(t *testing.T) {
		_, err := h.StartPomodoro(ctx, connect.NewRequest(&taskv1.StartPomodoroRequest{TaskId: taskID}))
		if err != nil {
			t.Fatalf("StartPomodoro: %v", err)
		}
		_, err = h.CompletePomodoro(ctx, connect.NewRequest(&taskv1.CompletePomodoroRequest{}))
		if err != nil {
			t.Fatalf("CompletePomodoro: %v", err)
		}

		active, err := h.GetActivePomodoro(ctx, connect.NewRequest(&taskv1.GetActivePomodoroRequest{}))
		if err != nil {
			t.Fatalf("GetActivePomodoro: %v", err)
		}
		if active.Msg.Pomodoro != nil {
			t.Errorf("completed pomodoro should not be active")
		}
	})
}

// ---- T027 / T028: CancelPomodoro ----

func TestCancelPomodoro(t *testing.T) {
	h, userID := newTestHandler(t)
	ctx := ctxWithUser(userID)

	taskResp, err := h.CreateTask(ctx, connect.NewRequest(&taskv1.CreateTaskRequest{Name: "cancel pomo task"}))
	if err != nil {
		t.Fatalf("CreateTask: %v", err)
	}
	taskID := taskResp.Msg.Task.Id

	t.Run("happy path", func(t *testing.T) {
		_, err := h.StartPomodoro(ctx, connect.NewRequest(&taskv1.StartPomodoroRequest{TaskId: taskID}))
		if err != nil {
			t.Fatalf("StartPomodoro: %v", err)
		}

		before := time.Now()
		resp, err := h.CancelPomodoro(ctx, connect.NewRequest(&taskv1.CancelPomodoroRequest{}))
		if err != nil {
			t.Fatalf("CancelPomodoro: %v", err)
		}
		after := time.Now()
		p := resp.Msg.Pomodoro
		if p.Complete {
			t.Errorf("complete should be false for canceled pomodoro")
		}
		if p.EndAt == nil {
			t.Fatal("end_at should be set after cancel")
		}
		canceledAt := p.EndAt.AsTime()
		if canceledAt.Before(before) || canceledAt.After(after) {
			t.Errorf("end_at %v not in range [%v, %v]", canceledAt, before, after)
		}
	})

	t.Run("FailedPrecondition when no active pomodoro", func(t *testing.T) {
		_, err := h.CancelPomodoro(ctx, connect.NewRequest(&taskv1.CancelPomodoroRequest{}))
		if err == nil {
			t.Fatal("expected error")
		}
		ce, ok := err.(*connect.Error)
		if !ok || ce.Code() != connect.CodeFailedPrecondition {
			t.Errorf("expected CodeFailedPrecondition, got %v", err)
		}
	})

	t.Run("canceled pomodoro not in GetActivePomodoro but in GetTask pomodoros", func(t *testing.T) {
		_, err := h.StartPomodoro(ctx, connect.NewRequest(&taskv1.StartPomodoroRequest{TaskId: taskID}))
		if err != nil {
			t.Fatalf("StartPomodoro: %v", err)
		}
		cancelResp, err := h.CancelPomodoro(ctx, connect.NewRequest(&taskv1.CancelPomodoroRequest{}))
		if err != nil {
			t.Fatalf("CancelPomodoro: %v", err)
		}
		canceledID := cancelResp.Msg.Pomodoro.Id

		active, err := h.GetActivePomodoro(ctx, connect.NewRequest(&taskv1.GetActivePomodoroRequest{}))
		if err != nil {
			t.Fatalf("GetActivePomodoro: %v", err)
		}
		if active.Msg.Pomodoro != nil {
			t.Errorf("canceled pomodoro should not be active")
		}

		gr, err := h.GetTask(ctx, connect.NewRequest(&taskv1.GetTaskRequest{Id: taskID}))
		if err != nil {
			t.Fatalf("GetTask: %v", err)
		}
		found := false
		for _, p := range gr.Msg.Pomodoros {
			if p.Id == canceledID {
				if p.Complete {
					t.Errorf("canceled pomodoro has complete=true")
				}
				if p.EndAt == nil {
					t.Errorf("canceled pomodoro has nil end_at")
				}
				found = true
			}
		}
		if !found {
			t.Errorf("canceled pomodoro not found in GetTask pomodoros")
		}
	})
}

// ---- T029: cascade delete ----

func TestPomodoro_CascadeDelete(t *testing.T) {
	h, userID := newTestHandler(t)
	ctx := ctxWithUser(userID)
	pool := testPool(t)

	taskResp, err := h.CreateTask(ctx, connect.NewRequest(&taskv1.CreateTaskRequest{Name: "cascade task"}))
	if err != nil {
		t.Fatalf("CreateTask: %v", err)
	}
	taskID := taskResp.Msg.Task.Id

	_, err = h.StartPomodoro(ctx, connect.NewRequest(&taskv1.StartPomodoroRequest{TaskId: taskID}))
	if err != nil {
		t.Fatalf("StartPomodoro: %v", err)
	}

	_, err = h.DeleteTask(ctx, connect.NewRequest(&taskv1.DeleteTaskRequest{Id: taskID}))
	if err != nil {
		t.Fatalf("DeleteTask: %v", err)
	}

	var count int64
	err = pool.QueryRow(context.Background(),
		"SELECT count(*) FROM pomodoros WHERE task_id = $1", taskID).Scan(&count)
	if err != nil {
		t.Fatalf("count query: %v", err)
	}
	if count != 0 {
		t.Errorf("expected 0 pomodoros after task delete, got %d", count)
	}
}

// ---- T030: concurrency invariant ----

func TestPomodoro_SingleActiveInvariant(t *testing.T) {
	h, userID := newTestHandler(t)
	ctx := ctxWithUser(userID)
	pool := testPool(t)

	task1, err := h.CreateTask(ctx, connect.NewRequest(&taskv1.CreateTaskRequest{Name: "task1"}))
	if err != nil {
		t.Fatalf("CreateTask 1: %v", err)
	}
	task2, err := h.CreateTask(ctx, connect.NewRequest(&taskv1.CreateTaskRequest{Name: "task2"}))
	if err != nil {
		t.Fatalf("CreateTask 2: %v", err)
	}

	var wg sync.WaitGroup
	errs := make([]error, 2)
	wg.Add(2)
	go func() {
		defer wg.Done()
		_, errs[0] = h.StartPomodoro(ctx, connect.NewRequest(&taskv1.StartPomodoroRequest{TaskId: task1.Msg.Task.Id}))
	}()
	go func() {
		defer wg.Done()
		_, errs[1] = h.StartPomodoro(ctx, connect.NewRequest(&taskv1.StartPomodoroRequest{TaskId: task2.Msg.Task.Id}))
	}()
	wg.Wait()

	t.Cleanup(func() {
		_, _ = h.CancelPomodoro(ctx, connect.NewRequest(&taskv1.CancelPomodoroRequest{}))
	})

	successes := 0
	for _, err := range errs {
		if err == nil {
			successes++
		} else {
			ce, ok := err.(*connect.Error)
			if !ok || ce.Code() != connect.CodeAlreadyExists {
				t.Errorf("expected CodeAlreadyExists, got %v", err)
			}
		}
	}
	if successes != 1 {
		t.Errorf("expected exactly 1 success, got %d (errs: %v, %v)", successes, errs[0], errs[1])
	}

	var count int64
	err = pool.QueryRow(context.Background(),
		"SELECT count(*) FROM pomodoros WHERE user_id = $1 AND end_at IS NULL", userID).Scan(&count)
	if err != nil {
		t.Fatalf("count query: %v", err)
	}
	if count != 1 {
		t.Errorf("expected 1 active pomodoro, got %d", count)
	}
}
