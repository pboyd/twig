package handler_test

import (
	"context"
	"fmt"
	"os"
	"strings"
	"sync/atomic"
	"testing"
	"time"

	"connectrpc.com/connect"
	"github.com/golang-migrate/migrate/v4"
	_ "github.com/golang-migrate/migrate/v4/database/postgres"
	_ "github.com/golang-migrate/migrate/v4/source/file"
	"github.com/jackc/pgx/v5/pgtype"
	"github.com/jackc/pgx/v5/pgxpool"

	goalv1 "github.com/pboyd/twig/api/gen/goal/v1"
	taskv1 "github.com/pboyd/twig/api/gen/task/v1"
	"github.com/pboyd/twig/services/twig/internal/auth"
	"github.com/pboyd/twig/services/twig/internal/db"
	"github.com/pboyd/twig/services/twig/internal/handler"
	"google.golang.org/protobuf/types/known/timestamppb"
)

var testUserCounter int64

// newTestHandler creates a Task handler backed by a real PostgreSQL database.
// Tests that call this are integration tests and require DATABASE_URL.
func newTestHandler(t *testing.T) (*handler.Task, int64) {
	t.Helper()
	dsn := os.Getenv("DATABASE_URL")
	if dsn == "" {
		t.Skip("DATABASE_URL not set; skipping integration test")
	}

	m, err := migrate.New("file://../../db/migrations", dsn)
	if err != nil {
		t.Fatalf("migrate.New: %v", err)
	}
	if err := m.Up(); err != nil && err != migrate.ErrNoChange {
		t.Fatalf("migrate up: %v", err)
	}

	pool, err := pgxpool.New(context.Background(), dsn)
	if err != nil {
		t.Fatalf("pgxpool.New: %v", err)
	}
	t.Cleanup(pool.Close)

	queries := db.New(pool)

	// Create a test user for this handler.
	hash, err := auth.HashPassword("testpass")
	if err != nil {
		t.Fatalf("HashPassword: %v", err)
	}
	n := atomic.AddInt64(&testUserCounter, 1)
	user, err := queries.CreateUser(context.Background(), db.CreateUserParams{
		Username:     fmt.Sprintf("testuser_%d_%s", n, t.Name()),
		PasswordHash: hash,
	})
	if err != nil {
		t.Fatalf("CreateUser: %v", err)
	}
	userID := user.ID

	t.Cleanup(func() {
		_, _ = pool.Exec(context.Background(), "DELETE FROM tasks WHERE user_id = $1", userID)
		_, _ = pool.Exec(context.Background(), "DELETE FROM goals WHERE user_id = $1", userID)
		_, _ = pool.Exec(context.Background(), "DELETE FROM users WHERE id = $1", userID)
	})
	return &handler.Task{Queries: queries, Pool: pool}, userID
}

// linkGoalForTest creates a goal with the given name, links it to the
// supplied top-level task, and returns the goal id. Used by move/goal
// tests that need a fully wired goal association.
func linkGoalForTest(t *testing.T, ctx context.Context, h *handler.Task, taskID int64, goalName string) int64 {
	t.Helper()
	gh := &handler.Goal{Queries: h.Queries, Pool: h.Pool}
	resp, err := gh.CreateGoal(ctx, connect.NewRequest(&goalv1.CreateGoalRequest{Name: goalName}))
	if err != nil {
		t.Fatalf("CreateGoal %q: %v", goalName, err)
	}
	goalID := resp.Msg.Goal.Id
	if _, err := h.SetTaskGoal(ctx, connect.NewRequest(&taskv1.SetTaskGoalRequest{
		TaskId: taskID,
		GoalId: &goalID,
	})); err != nil {
		t.Fatalf("SetTaskGoal %q → task %d: %v", goalName, taskID, err)
	}
	return goalID
}

// mustCreateTask creates a task and fails the test immediately on error,
// instead of letting a discarded error surface later as a nil-pointer panic.
func mustCreateTask(t *testing.T, ctx context.Context, h *handler.Task, name string, parentID *int64) int64 {
	t.Helper()
	resp, err := h.CreateTask(ctx, connect.NewRequest(&taskv1.CreateTaskRequest{
		Name:     name,
		ParentId: parentID,
	}))
	if err != nil {
		t.Fatalf("CreateTask %q: %v", name, err)
	}
	return resp.Msg.Task.Id
}

// ctxWithUser returns a context carrying the given user_id and a dummy client IP,
// matching the context shape the auth middleware produces in production.
func ctxWithUser(userID int64) context.Context {
	ctx := auth.WithUserID(context.Background(), userID)
	return auth.WithClientIP(ctx, "127.0.0.1")
}

// newGoalTestHandlerWithPool creates a Goal handler that shares the given task
// handler's DB connection. Use this when a test needs both Task and Goal
// operations on the same user — the task handler was already set up by
// newTestHandler, so no new user needs to be created.
func newGoalTestHandlerWithPool(t *testing.T, taskH *handler.Task) (*handler.Goal, int64) {
	t.Helper()
	return &handler.Goal{Queries: taskH.Queries, Pool: taskH.Pool}, 0
}

// ---- Unit tests: dbTaskToProto ----

func TestDbTaskToProto_CompletedAt(t *testing.T) {
	t.Run("completed_at unset when Valid false", func(t *testing.T) {
		row := db.Task{ID: 1, Name: "task"}
		pt := handler.ExportDbTaskToProto(row)
		if pt.CompletedAt != nil {
			t.Errorf("expected nil completed_at, got %v", pt.CompletedAt)
		}
	})

	t.Run("completed_at set when Valid true", func(t *testing.T) {
		ts := time.Date(2026, 5, 20, 12, 0, 0, 0, time.UTC)
		row := db.Task{
			ID:          1,
			Name:        "task",
			CompletedAt: pgtype.Timestamptz{Time: ts, Valid: true},
		}
		pt := handler.ExportDbTaskToProto(row)
		if pt.CompletedAt == nil {
			t.Fatal("expected non-nil completed_at")
		}
		if !pt.CompletedAt.AsTime().Equal(ts) {
			t.Errorf("completed_at = %v, want %v", pt.CompletedAt.AsTime(), ts)
		}
	})
}

// ---- Unit tests: name validation ----

func TestCreateTask_NameValidation(t *testing.T) {
	cases := []struct {
		name    string
		input   string
		wantErr connect.Code
	}{
		{"empty string", "", connect.CodeInvalidArgument},
		{"whitespace only", "   ", connect.CodeInvalidArgument},
		{"tab only", "\t", connect.CodeInvalidArgument},
		{"too long", strings.Repeat("a", 256), connect.CodeInvalidArgument},
		{"exactly 255 chars", strings.Repeat("a", 255), 0},
		{"valid short name", "Buy milk", 0},
	}

	h := &handler.Task{Queries: nil}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			if tc.wantErr == 0 {
				defer func() {
					if r := recover(); r != nil {
						// Panic from nil Queries is expected for valid-name cases.
					}
				}()
				req := connect.NewRequest(&taskv1.CreateTaskRequest{Name: tc.input})
				_, err := h.CreateTask(context.Background(), req)
				if err != nil {
					ce, ok := err.(*connect.Error)
					if ok && ce.Code() == connect.CodeInvalidArgument {
						t.Errorf("unexpected InvalidArgument for valid name %q: %v", tc.input, err)
					}
				}
				return
			}
			req := connect.NewRequest(&taskv1.CreateTaskRequest{Name: tc.input})
			_, err := h.CreateTask(context.Background(), req)
			if err == nil {
				t.Fatalf("expected error code %v, got nil", tc.wantErr)
			}
			ce, ok := err.(*connect.Error)
			if !ok {
				t.Fatalf("expected *connect.Error, got %T: %v", err, err)
			}
			if ce.Code() != tc.wantErr {
				t.Errorf("code = %v, want %v", ce.Code(), tc.wantErr)
			}
		})
	}
}

func TestUpdateTask_NameValidation(t *testing.T) {
	cases := []struct {
		name    string
		input   string
		wantErr connect.Code
	}{
		{"empty string", "", connect.CodeInvalidArgument},
		{"whitespace only", "  ", connect.CodeInvalidArgument},
		{"too long", strings.Repeat("b", 256), connect.CodeInvalidArgument},
	}

	h := &handler.Task{Queries: nil}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			req := connect.NewRequest(&taskv1.UpdateTaskRequest{Id: 1, Name: tc.input})
			_, err := h.UpdateTask(context.Background(), req)
			if err == nil {
				t.Fatal("expected error, got nil")
			}
			ce, ok := err.(*connect.Error)
			if !ok {
				t.Fatalf("expected *connect.Error, got %T: %v", err, err)
			}
			if ce.Code() != tc.wantErr {
				t.Errorf("code = %v, want %v", ce.Code(), tc.wantErr)
			}
		})
	}
}

// ---- Integration tests: US1 ----

func TestCreateTask_Integration(t *testing.T) {
	h, userID := newTestHandler(t)
	ctx := ctxWithUser(userID)

	t.Run("name only", func(t *testing.T) {
		resp, err := h.CreateTask(ctx, connect.NewRequest(&taskv1.CreateTaskRequest{
			Name: "Write the report",
		}))
		if err != nil {
			t.Fatalf("CreateTask: %v", err)
		}
		task := resp.Msg.Task
		if task.Id <= 0 {
			t.Errorf("expected positive id, got %d", task.Id)
		}
		if task.Name != "Write the report" {
			t.Errorf("name = %q, want %q", task.Name, "Write the report")
		}
		if task.Description != "" {
			t.Errorf("description = %q, want empty", task.Description)
		}
		if task.Due != nil {
			t.Errorf("due should be nil")
		}
		if task.ParentId != nil {
			t.Errorf("parent_id should be nil")
		}
	})

	t.Run("all fields", func(t *testing.T) {
		resp, err := h.CreateTask(ctx, connect.NewRequest(&taskv1.CreateTaskRequest{
			Name:        "Draft section",
			Description: "Some details",
		}))
		if err != nil {
			t.Fatalf("CreateTask: %v", err)
		}
		if resp.Msg.Task.Description != "Some details" {
			t.Errorf("description = %q, want %q", resp.Msg.Task.Description, "Some details")
		}
	})

	t.Run("name trimmed is valid", func(t *testing.T) {
		resp, err := h.CreateTask(ctx, connect.NewRequest(&taskv1.CreateTaskRequest{
			Name: "  trimmed  ",
		}))
		if err != nil {
			t.Fatalf("CreateTask: %v", err)
		}
		if resp.Msg.Task.Name != "trimmed" {
			t.Errorf("name = %q, want %q", resp.Msg.Task.Name, "trimmed")
		}
	})
}

func TestGetTask_Integration(t *testing.T) {
	h, userID := newTestHandler(t)
	ctx := ctxWithUser(userID)

	resp, err := h.CreateTask(ctx, connect.NewRequest(&taskv1.CreateTaskRequest{Name: "Get me"}))
	if err != nil {
		t.Fatalf("setup CreateTask: %v", err)
	}
	id := resp.Msg.Task.Id

	t.Run("found", func(t *testing.T) {
		gr, err := h.GetTask(ctx, connect.NewRequest(&taskv1.GetTaskRequest{Id: id}))
		if err != nil {
			t.Fatalf("GetTask: %v", err)
		}
		if gr.Msg.Task.Id != id {
			t.Errorf("id = %d, want %d", gr.Msg.Task.Id, id)
		}
		if gr.Msg.Task.Name != "Get me" {
			t.Errorf("name = %q, want %q", gr.Msg.Task.Name, "Get me")
		}
	})

	t.Run("not found", func(t *testing.T) {
		_, err := h.GetTask(ctx, connect.NewRequest(&taskv1.GetTaskRequest{Id: 999999}))
		if err == nil {
			t.Fatal("expected error, got nil")
		}
		ce, ok := err.(*connect.Error)
		if !ok || ce.Code() != connect.CodeNotFound {
			t.Errorf("expected CodeNotFound, got %v", err)
		}
	})
}

func TestListTasks_Integration(t *testing.T) {
	h, userID := newTestHandler(t)
	ctx := ctxWithUser(userID)

	t.Run("empty list", func(t *testing.T) {
		resp, err := h.ListTasks(ctx, connect.NewRequest(&taskv1.ListTasksRequest{}))
		if err != nil {
			t.Fatalf("ListTasks: %v", err)
		}
		if len(resp.Msg.Tasks) != 0 {
			t.Errorf("expected empty list, got %d tasks", len(resp.Msg.Tasks))
		}
	})

	names := []string{"Alpha", "Beta", "Gamma"}
	for _, n := range names {
		if _, err := h.CreateTask(ctx, connect.NewRequest(&taskv1.CreateTaskRequest{Name: n})); err != nil {
			t.Fatalf("setup CreateTask %q: %v", n, err)
		}
	}

	t.Run("ordered by id", func(t *testing.T) {
		resp, err := h.ListTasks(ctx, connect.NewRequest(&taskv1.ListTasksRequest{}))
		if err != nil {
			t.Fatalf("ListTasks: %v", err)
		}
		tasks := resp.Msg.Tasks
		if len(tasks) != 3 {
			t.Fatalf("expected 3 tasks, got %d", len(tasks))
		}
		for i := 1; i < len(tasks); i++ {
			if tasks[i].Id <= tasks[i-1].Id {
				t.Errorf("tasks not ordered by id: %d <= %d", tasks[i].Id, tasks[i-1].Id)
			}
		}
	})
}

// ---- Integration tests: US2 ----

func TestUpdateTask_Integration(t *testing.T) {
	h, userID := newTestHandler(t)
	ctx := ctxWithUser(userID)

	resp, err := h.CreateTask(ctx, connect.NewRequest(&taskv1.CreateTaskRequest{
		Name:        "Original",
		Description: "Original desc",
	}))
	if err != nil {
		t.Fatalf("setup CreateTask: %v", err)
	}
	id := resp.Msg.Task.Id

	t.Run("update name", func(t *testing.T) {
		ur, err := h.UpdateTask(ctx, connect.NewRequest(&taskv1.UpdateTaskRequest{
			Id:   id,
			Name: "Updated",
		}))
		if err != nil {
			t.Fatalf("UpdateTask: %v", err)
		}
		if ur.Msg.Task.Name != "Updated" {
			t.Errorf("name = %q, want %q", ur.Msg.Task.Name, "Updated")
		}
		if ur.Msg.Task.Description != "" {
			t.Errorf("description = %q, want empty after clear", ur.Msg.Task.Description)
		}
		if ur.Msg.Task.Id != id {
			t.Errorf("id changed: %d != %d", ur.Msg.Task.Id, id)
		}
	})

	t.Run("unknown id", func(t *testing.T) {
		_, err := h.UpdateTask(ctx, connect.NewRequest(&taskv1.UpdateTaskRequest{
			Id:   999999,
			Name: "x",
		}))
		if err == nil {
			t.Fatal("expected error, got nil")
		}
		ce, ok := err.(*connect.Error)
		if !ok || ce.Code() != connect.CodeNotFound {
			t.Errorf("expected CodeNotFound, got %v", err)
		}
	})
}

// ---- Integration tests: US3 ----

func TestHierarchy_Integration(t *testing.T) {
	h, userID := newTestHandler(t)
	ctx := ctxWithUser(userID)

	rootResp, err := h.CreateTask(ctx, connect.NewRequest(&taskv1.CreateTaskRequest{Name: "root"}))
	if err != nil {
		t.Fatalf("create root: %v", err)
	}
	rootID := rootResp.Msg.Task.Id

	childResp, err := h.CreateTask(ctx, connect.NewRequest(&taskv1.CreateTaskRequest{
		Name:     "child",
		ParentId: &rootID,
	}))
	if err != nil {
		t.Fatalf("create child: %v", err)
	}
	childID := childResp.Msg.Task.Id
	if childResp.Msg.Task.ParentId == nil || *childResp.Msg.Task.ParentId != rootID {
		t.Errorf("child parent_id = %v, want %d", childResp.Msg.Task.ParentId, rootID)
	}

	grandchildResp, err := h.CreateTask(ctx, connect.NewRequest(&taskv1.CreateTaskRequest{
		Name:     "grandchild",
		ParentId: &childID,
	}))
	if err != nil {
		t.Fatalf("create grandchild: %v", err)
	}
	grandchildID := grandchildResp.Msg.Task.Id

	t.Run("unknown parent on create", func(t *testing.T) {
		bad := int64(999999)
		_, err := h.CreateTask(ctx, connect.NewRequest(&taskv1.CreateTaskRequest{
			Name:     "orphan",
			ParentId: &bad,
		}))
		if err == nil {
			t.Fatal("expected error, got nil")
		}
		ce, ok := err.(*connect.Error)
		if !ok || ce.Code() != connect.CodeInvalidArgument {
			t.Errorf("expected CodeInvalidArgument, got %v", err)
		}
	})

	t.Run("self-parent cycle", func(t *testing.T) {
		_, err := h.UpdateTask(ctx, connect.NewRequest(&taskv1.UpdateTaskRequest{
			Id:       rootID,
			Name:     "root",
			ParentId: &rootID,
		}))
		if err == nil {
			t.Fatal("expected cycle error, got nil")
		}
		ce, ok := err.(*connect.Error)
		if !ok || ce.Code() != connect.CodeInvalidArgument {
			t.Errorf("expected CodeInvalidArgument, got %v", err)
		}
	})

	t.Run("descendant-parent cycle", func(t *testing.T) {
		_, err := h.UpdateTask(ctx, connect.NewRequest(&taskv1.UpdateTaskRequest{
			Id:       rootID,
			Name:     "root",
			ParentId: &grandchildID,
		}))
		if err == nil {
			t.Fatal("expected cycle error, got nil")
		}
		ce, ok := err.(*connect.Error)
		if !ok || ce.Code() != connect.CodeInvalidArgument {
			t.Errorf("expected CodeInvalidArgument, got %v", err)
		}
	})

	t.Run("unknown parent on update", func(t *testing.T) {
		bad := int64(999999)
		_, err := h.UpdateTask(ctx, connect.NewRequest(&taskv1.UpdateTaskRequest{
			Id:       rootID,
			Name:     "root",
			ParentId: &bad,
		}))
		if err == nil {
			t.Fatal("expected error, got nil")
		}
		ce, ok := err.(*connect.Error)
		if !ok || ce.Code() != connect.CodeInvalidArgument {
			t.Errorf("expected CodeInvalidArgument, got %v", err)
		}
	})

	t.Run("re-parent task", func(t *testing.T) {
		_, err := h.UpdateTask(ctx, connect.NewRequest(&taskv1.UpdateTaskRequest{
			Id:       grandchildID,
			Name:     "grandchild",
			ParentId: &rootID,
		}))
		if err != nil {
			t.Fatalf("re-parent: %v", err)
		}
	})
}

// ---- Integration tests: US4 ----

func TestDeleteTask_Integration(t *testing.T) {
	h, userID := newTestHandler(t)
	ctx := ctxWithUser(userID)

	t.Run("delete leaf", func(t *testing.T) {
		resp, err := h.CreateTask(ctx, connect.NewRequest(&taskv1.CreateTaskRequest{Name: "leaf"}))
		if err != nil {
			t.Fatalf("create: %v", err)
		}
		id := resp.Msg.Task.Id

		_, err = h.DeleteTask(ctx, connect.NewRequest(&taskv1.DeleteTaskRequest{Id: id}))
		if err != nil {
			t.Fatalf("DeleteTask: %v", err)
		}

		_, err = h.GetTask(ctx, connect.NewRequest(&taskv1.GetTaskRequest{Id: id}))
		if err == nil {
			t.Fatal("expected not found after delete")
		}
		ce, ok := err.(*connect.Error)
		if !ok || ce.Code() != connect.CodeNotFound {
			t.Errorf("expected CodeNotFound, got %v", err)
		}
	})

	t.Run("cascade delete subtree", func(t *testing.T) {
		parentResp, err := h.CreateTask(ctx, connect.NewRequest(&taskv1.CreateTaskRequest{Name: "parent"}))
		if err != nil {
			t.Fatalf("create parent: %v", err)
		}
		parentID := parentResp.Msg.Task.Id

		childResp, err := h.CreateTask(ctx, connect.NewRequest(&taskv1.CreateTaskRequest{
			Name:     "child",
			ParentId: &parentID,
		}))
		if err != nil {
			t.Fatalf("create child: %v", err)
		}
		childID := childResp.Msg.Task.Id

		_, err = h.DeleteTask(ctx, connect.NewRequest(&taskv1.DeleteTaskRequest{Id: parentID}))
		if err != nil {
			t.Fatalf("DeleteTask parent: %v", err)
		}

		_, err = h.GetTask(ctx, connect.NewRequest(&taskv1.GetTaskRequest{Id: childID}))
		if err == nil {
			t.Fatal("expected child to be deleted via cascade")
		}
	})

	t.Run("unknown id", func(t *testing.T) {
		_, err := h.DeleteTask(ctx, connect.NewRequest(&taskv1.DeleteTaskRequest{Id: 999999}))
		if err == nil {
			t.Fatal("expected error, got nil")
		}
		ce, ok := err.(*connect.Error)
		if !ok || ce.Code() != connect.CodeNotFound {
			t.Errorf("expected CodeNotFound, got %v", err)
		}
	})
}

// ---- Integration tests: CompleteTask (US1 + US4) ----

func TestCompleteTask_Integration(t *testing.T) {
	h, userID := newTestHandler(t)
	ctx := ctxWithUser(userID)

	// Create a leaf task.
	resp, err := h.CreateTask(ctx, connect.NewRequest(&taskv1.CreateTaskRequest{Name: "leaf"}))
	if err != nil {
		t.Fatalf("setup CreateTask: %v", err)
	}
	leafID := resp.Msg.Task.Id

	t.Run("marks leaf complete and sets completed_at", func(t *testing.T) {
		before := time.Now()
		cr, err := h.CompleteTask(ctx, connect.NewRequest(&taskv1.CompleteTaskRequest{Id: leafID}))
		if err != nil {
			t.Fatalf("CompleteTask: %v", err)
		}
		task := cr.Msg.Task
		if task.CompletedAt == nil {
			t.Fatal("expected completed_at to be set")
		}
		completedTime := task.CompletedAt.AsTime()
		if completedTime.Before(before) || completedTime.After(time.Now()) {
			t.Errorf("completed_at %v not near now", completedTime)
		}
	})

	t.Run("idempotent: second call returns same completed_at", func(t *testing.T) {
		first, err := h.CompleteTask(ctx, connect.NewRequest(&taskv1.CompleteTaskRequest{Id: leafID}))
		if err != nil {
			t.Fatalf("first CompleteTask: %v", err)
		}
		second, err := h.CompleteTask(ctx, connect.NewRequest(&taskv1.CompleteTaskRequest{Id: leafID}))
		if err != nil {
			t.Fatalf("second CompleteTask: %v", err)
		}
		if !first.Msg.Task.CompletedAt.AsTime().Equal(second.Msg.Task.CompletedAt.AsTime()) {
			t.Errorf("completed_at changed: first=%v second=%v",
				first.Msg.Task.CompletedAt.AsTime(), second.Msg.Task.CompletedAt.AsTime())
		}
	})

	t.Run("not found for unknown id", func(t *testing.T) {
		_, err := h.CompleteTask(ctx, connect.NewRequest(&taskv1.CompleteTaskRequest{Id: 999999}))
		if err == nil {
			t.Fatal("expected error, got nil")
		}
		ce, ok := err.(*connect.Error)
		if !ok || ce.Code() != connect.CodeNotFound {
			t.Errorf("expected CodeNotFound, got %v", err)
		}
	})

	t.Run("cross-user: not found for another user's task", func(t *testing.T) {
		_, userB := newTestHandler(t)
		ctxB := ctxWithUser(userB)
		_, err := h.CompleteTask(ctxB, connect.NewRequest(&taskv1.CompleteTaskRequest{Id: leafID}))
		if err == nil {
			t.Fatal("expected error, got nil")
		}
		ce, ok := err.(*connect.Error)
		if !ok || ce.Code() != connect.CodeNotFound {
			t.Errorf("expected CodeNotFound, got %v", err)
		}
	})

	t.Run("blocked by incomplete direct child", func(t *testing.T) {
		parentResp, err := h.CreateTask(ctx, connect.NewRequest(&taskv1.CreateTaskRequest{Name: "parent"}))
		if err != nil {
			t.Fatalf("create parent: %v", err)
		}
		parentID := parentResp.Msg.Task.Id
		_, err = h.CreateTask(ctx, connect.NewRequest(&taskv1.CreateTaskRequest{
			Name:     "child",
			ParentId: &parentID,
		}))
		if err != nil {
			t.Fatalf("create child: %v", err)
		}

		_, err = h.CompleteTask(ctx, connect.NewRequest(&taskv1.CompleteTaskRequest{Id: parentID}))
		if err == nil {
			t.Fatal("expected FailedPrecondition, got nil")
		}
		ce, ok := err.(*connect.Error)
		if !ok || ce.Code() != connect.CodeFailedPrecondition {
			t.Errorf("expected CodeFailedPrecondition, got %v", err)
		}
	})

	t.Run("blocked by incomplete grandchild", func(t *testing.T) {
		topResp, err := h.CreateTask(ctx, connect.NewRequest(&taskv1.CreateTaskRequest{Name: "top"}))
		if err != nil {
			t.Fatalf("create top: %v", err)
		}
		topID := topResp.Msg.Task.Id
		midResp, err := h.CreateTask(ctx, connect.NewRequest(&taskv1.CreateTaskRequest{
			Name:     "mid",
			ParentId: &topID,
		}))
		if err != nil {
			t.Fatalf("create mid: %v", err)
		}
		midID := midResp.Msg.Task.Id
		_, err = h.CreateTask(ctx, connect.NewRequest(&taskv1.CreateTaskRequest{
			Name:     "leaf2",
			ParentId: &midID,
		}))
		if err != nil {
			t.Fatalf("create leaf2: %v", err)
		}

		_, err = h.CompleteTask(ctx, connect.NewRequest(&taskv1.CompleteTaskRequest{Id: topID}))
		if err == nil {
			t.Fatal("expected FailedPrecondition for grandchild, got nil")
		}
		ce, ok := err.(*connect.Error)
		if !ok || ce.Code() != connect.CodeFailedPrecondition {
			t.Errorf("expected CodeFailedPrecondition, got %v", err)
		}
	})

	t.Run("succeeds once all descendants are complete", func(t *testing.T) {
		parentResp, err := h.CreateTask(ctx, connect.NewRequest(&taskv1.CreateTaskRequest{Name: "parent2"}))
		if err != nil {
			t.Fatalf("create parent2: %v", err)
		}
		parentID := parentResp.Msg.Task.Id
		childResp, err := h.CreateTask(ctx, connect.NewRequest(&taskv1.CreateTaskRequest{
			Name:     "child2",
			ParentId: &parentID,
		}))
		if err != nil {
			t.Fatalf("create child2: %v", err)
		}
		childID := childResp.Msg.Task.Id

		_, err = h.CompleteTask(ctx, connect.NewRequest(&taskv1.CompleteTaskRequest{Id: childID}))
		if err != nil {
			t.Fatalf("complete child2: %v", err)
		}
		_, err = h.CompleteTask(ctx, connect.NewRequest(&taskv1.CompleteTaskRequest{Id: parentID}))
		if err != nil {
			t.Fatalf("complete parent2 after child done: %v", err)
		}
	})
}

func TestUncompleteTask_Integration(t *testing.T) {
	h, userID := newTestHandler(t)
	ctx := ctxWithUser(userID)

	// Create and complete a leaf task to use in sub-tests.
	resp, err := h.CreateTask(ctx, connect.NewRequest(&taskv1.CreateTaskRequest{Name: "leaf"}))
	if err != nil {
		t.Fatalf("setup CreateTask: %v", err)
	}
	leafID := resp.Msg.Task.Id
	if _, err := h.CompleteTask(ctx, connect.NewRequest(&taskv1.CompleteTaskRequest{Id: leafID})); err != nil {
		t.Fatalf("setup CompleteTask: %v", err)
	}

	t.Run("clears completed_at", func(t *testing.T) {
		ur, err := h.UncompleteTask(ctx, connect.NewRequest(&taskv1.UncompleteTaskRequest{Id: leafID}))
		if err != nil {
			t.Fatalf("UncompleteTask: %v", err)
		}
		if ur.Msg.Task.CompletedAt != nil {
			t.Errorf("expected completed_at nil, got %v", ur.Msg.Task.CompletedAt)
		}
	})

	t.Run("idempotent on already-incomplete task", func(t *testing.T) {
		// leaf is now incomplete from the previous sub-test.
		_, err := h.UncompleteTask(ctx, connect.NewRequest(&taskv1.UncompleteTaskRequest{Id: leafID}))
		if err != nil {
			t.Fatalf("UncompleteTask on incomplete task: %v", err)
		}
	})

	t.Run("not found for unknown id", func(t *testing.T) {
		_, err := h.UncompleteTask(ctx, connect.NewRequest(&taskv1.UncompleteTaskRequest{Id: 999999}))
		if err == nil {
			t.Fatal("expected error, got nil")
		}
		ce, ok := err.(*connect.Error)
		if !ok || ce.Code() != connect.CodeNotFound {
			t.Errorf("expected CodeNotFound, got %v", err)
		}
	})

	t.Run("FailedPrecondition when parent is complete", func(t *testing.T) {
		parentResp, err := h.CreateTask(ctx, connect.NewRequest(&taskv1.CreateTaskRequest{Name: "parent"}))
		if err != nil {
			t.Fatalf("create parent: %v", err)
		}
		parentID := parentResp.Msg.Task.Id
		childResp, err := h.CreateTask(ctx, connect.NewRequest(&taskv1.CreateTaskRequest{
			Name:     "child",
			ParentId: &parentID,
		}))
		if err != nil {
			t.Fatalf("create child: %v", err)
		}
		childID := childResp.Msg.Task.Id
		if _, err := h.CompleteTask(ctx, connect.NewRequest(&taskv1.CompleteTaskRequest{Id: childID})); err != nil {
			t.Fatalf("complete child: %v", err)
		}
		if _, err := h.CompleteTask(ctx, connect.NewRequest(&taskv1.CompleteTaskRequest{Id: parentID})); err != nil {
			t.Fatalf("complete parent: %v", err)
		}

		_, err = h.UncompleteTask(ctx, connect.NewRequest(&taskv1.UncompleteTaskRequest{Id: childID}))
		if err == nil {
			t.Fatal("expected FailedPrecondition, got nil")
		}
		ce, ok := err.(*connect.Error)
		if !ok || ce.Code() != connect.CodeFailedPrecondition {
			t.Errorf("expected CodeFailedPrecondition, got %v", err)
		}
	})
}

func TestCreateTask_CompleteParentRejected(t *testing.T) {
	h, userID := newTestHandler(t)
	ctx := ctxWithUser(userID)

	parentResp, err := h.CreateTask(ctx, connect.NewRequest(&taskv1.CreateTaskRequest{Name: "complete-parent"}))
	if err != nil {
		t.Fatalf("create parent: %v", err)
	}
	parentID := parentResp.Msg.Task.Id
	_, err = h.CompleteTask(ctx, connect.NewRequest(&taskv1.CompleteTaskRequest{Id: parentID}))
	if err != nil {
		t.Fatalf("complete parent: %v", err)
	}

	t.Run("rejected when parent is complete", func(t *testing.T) {
		_, err := h.CreateTask(ctx, connect.NewRequest(&taskv1.CreateTaskRequest{
			Name:     "late child",
			ParentId: &parentID,
		}))
		if err == nil {
			t.Fatal("expected FailedPrecondition, got nil")
		}
		ce, ok := err.(*connect.Error)
		if !ok || ce.Code() != connect.CodeFailedPrecondition {
			t.Errorf("expected CodeFailedPrecondition, got %v", err)
		}
	})

	t.Run("incomplete parent still succeeds", func(t *testing.T) {
		incompleteParentResp, err := h.CreateTask(ctx, connect.NewRequest(&taskv1.CreateTaskRequest{Name: "incomplete-parent"}))
		if err != nil {
			t.Fatalf("create incomplete parent: %v", err)
		}
		incompleteID := incompleteParentResp.Msg.Task.Id
		_, err = h.CreateTask(ctx, connect.NewRequest(&taskv1.CreateTaskRequest{
			Name:     "child under incomplete",
			ParentId: &incompleteID,
		}))
		if err != nil {
			t.Fatalf("CreateTask under incomplete parent: %v", err)
		}
	})
}

func TestUpdateTask_CompleteParentRejected(t *testing.T) {
	h, userID := newTestHandler(t)
	ctx := ctxWithUser(userID)

	parentResp, err := h.CreateTask(ctx, connect.NewRequest(&taskv1.CreateTaskRequest{Name: "complete-parent"}))
	if err != nil {
		t.Fatalf("create parent: %v", err)
	}
	parentID := parentResp.Msg.Task.Id
	_, err = h.CompleteTask(ctx, connect.NewRequest(&taskv1.CompleteTaskRequest{Id: parentID}))
	if err != nil {
		t.Fatalf("complete parent: %v", err)
	}

	taskResp, err := h.CreateTask(ctx, connect.NewRequest(&taskv1.CreateTaskRequest{Name: "standalone"}))
	if err != nil {
		t.Fatalf("create standalone: %v", err)
	}
	taskID := taskResp.Msg.Task.Id

	t.Run("rejected when re-parenting under complete parent", func(t *testing.T) {
		_, err := h.UpdateTask(ctx, connect.NewRequest(&taskv1.UpdateTaskRequest{
			Id:       taskID,
			Name:     "standalone",
			ParentId: &parentID,
		}))
		if err == nil {
			t.Fatal("expected FailedPrecondition, got nil")
		}
		ce, ok := err.(*connect.Error)
		if !ok || ce.Code() != connect.CodeFailedPrecondition {
			t.Errorf("expected CodeFailedPrecondition, got %v", err)
		}
	})

	t.Run("update not changing parent does not regress", func(t *testing.T) {
		_, err := h.UpdateTask(ctx, connect.NewRequest(&taskv1.UpdateTaskRequest{
			Id:   taskID,
			Name: "standalone-renamed",
		}))
		if err != nil {
			t.Fatalf("UpdateTask without parent change: %v", err)
		}
	})
}

// ---- Integration tests: GetTask pomodoro fields (T014) ----

func TestGetTask_PomoFields(t *testing.T) {
	h, userID := newTestHandler(t)
	ctx := ctxWithUser(userID)

	taskResp, err := h.CreateTask(ctx, connect.NewRequest(&taskv1.CreateTaskRequest{Name: "pomo task"}))
	if err != nil {
		t.Fatalf("CreateTask: %v", err)
	}
	taskID := taskResp.Msg.Task.Id

	queries := h.Queries

	// Insert one completed and one canceled pomodoro directly.
	completed, err := queries.StartPomodoro(ctx, db.StartPomodoroParams{UserID: userID, TaskID: taskID})
	if err != nil {
		t.Fatalf("StartPomodoro (completed): %v", err)
	}
	_, err = queries.CompleteActivePomodoro(ctx, userID)
	if err != nil {
		t.Fatalf("CompleteActivePomodoro: %v", err)
	}

	_, err = queries.StartPomodoro(ctx, db.StartPomodoroParams{UserID: userID, TaskID: taskID})
	if err != nil {
		t.Fatalf("StartPomodoro (canceled): %v", err)
	}
	_, err = queries.CancelActivePomodoro(ctx, userID)
	if err != nil {
		t.Fatalf("CancelActivePomodoro: %v", err)
	}
	_ = completed

	gr, err := h.GetTask(ctx, connect.NewRequest(&taskv1.GetTaskRequest{Id: taskID}))
	if err != nil {
		t.Fatalf("GetTask: %v", err)
	}

	if gr.Msg.CompletedPomodoroCount != 1 {
		t.Errorf("completed_pomodoro_count = %d, want 1", gr.Msg.CompletedPomodoroCount)
	}
	if len(gr.Msg.Pomodoros) != 2 {
		t.Fatalf("len(pomodoros) = %d, want 2", len(gr.Msg.Pomodoros))
	}
	// Ordered by start_at ascending.
	if !gr.Msg.Pomodoros[0].StartAt.AsTime().Before(gr.Msg.Pomodoros[1].StartAt.AsTime()) &&
		!gr.Msg.Pomodoros[0].StartAt.AsTime().Equal(gr.Msg.Pomodoros[1].StartAt.AsTime()) {
		t.Errorf("pomodoros not in start_at order")
	}
}

// ---- Integration tests: SetEstimate (T040) ----

func TestSetEstimate_Integration(t *testing.T) {
	h, userID := newTestHandler(t)
	ctx := ctxWithUser(userID)

	taskResp, err := h.CreateTask(ctx, connect.NewRequest(&taskv1.CreateTaskRequest{Name: "estimate task"}))
	if err != nil {
		t.Fatalf("CreateTask: %v", err)
	}
	taskID := taskResp.Msg.Task.Id

	cases := []struct {
		name     string
		estimate int32
		wantCode connect.Code
	}{
		{"happy path 0", 0, 0},
		{"happy path 10", 10, 0},
		{"reject -1", -1, connect.CodeInvalidArgument},
		{"reject 11", 11, connect.CodeInvalidArgument},
		{"reject 100", 100, connect.CodeInvalidArgument},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			resp, err := h.SetEstimate(ctx, connect.NewRequest(&taskv1.SetEstimateRequest{
				TaskId:   taskID,
				Estimate: tc.estimate,
			}))
			if tc.wantCode != 0 {
				if err == nil {
					t.Fatalf("expected error code %v, got nil", tc.wantCode)
				}
				ce, ok := err.(*connect.Error)
				if !ok || ce.Code() != tc.wantCode {
					t.Errorf("code = %v, want %v; msg = %v", err, tc.wantCode, err)
				}
				// Verify error message for InvalidArgument.
				if tc.wantCode == connect.CodeInvalidArgument {
					if !strings.Contains(ce.Message(), "broken down further") {
						t.Errorf("error message %q does not contain 'broken down further'", ce.Message())
					}
				}
			} else {
				if err != nil {
					t.Fatalf("unexpected error: %v", err)
				}
				if resp.Msg.Task.Estimate != tc.estimate {
					t.Errorf("estimate = %d, want %d", resp.Msg.Task.Estimate, tc.estimate)
				}
			}
		})
	}

	t.Run("not found for unknown task", func(t *testing.T) {
		_, err := h.SetEstimate(ctx, connect.NewRequest(&taskv1.SetEstimateRequest{
			TaskId: 999999, Estimate: 3,
		}))
		if err == nil {
			t.Fatal("expected NotFound")
		}
		ce, ok := err.(*connect.Error)
		if !ok || ce.Code() != connect.CodeNotFound {
			t.Errorf("expected CodeNotFound, got %v", err)
		}
	})

	t.Run("not found for another user's task", func(t *testing.T) {
		_, userB := newTestHandler(t)
		ctxB := ctxWithUser(userB)
		_, err := h.SetEstimate(ctxB, connect.NewRequest(&taskv1.SetEstimateRequest{
			TaskId: taskID, Estimate: 3,
		}))
		if err == nil {
			t.Fatal("expected NotFound")
		}
		ce, ok := err.(*connect.Error)
		if !ok || ce.Code() != connect.CodeNotFound {
			t.Errorf("expected CodeNotFound, got %v", err)
		}
	})

	t.Run("overwrite value", func(t *testing.T) {
		_, err := h.SetEstimate(ctx, connect.NewRequest(&taskv1.SetEstimateRequest{TaskId: taskID, Estimate: 3}))
		if err != nil {
			t.Fatalf("set to 3: %v", err)
		}
		resp, err := h.SetEstimate(ctx, connect.NewRequest(&taskv1.SetEstimateRequest{TaskId: taskID, Estimate: 7}))
		if err != nil {
			t.Fatalf("set to 7: %v", err)
		}
		if resp.Msg.Task.Estimate != 7 {
			t.Errorf("estimate = %d, want 7", resp.Msg.Task.Estimate)
		}
	})
}

// ---- Cross-user isolation tests ----

func TestCrossUserIsolation(t *testing.T) {
	h, userA := newTestHandler(t)
	_, userB := newTestHandler(t)

	ctxA := ctxWithUser(userA)
	ctxB := ctxWithUser(userB)

	// User A creates a task.
	resp, err := h.CreateTask(ctxA, connect.NewRequest(&taskv1.CreateTaskRequest{Name: "alice task"}))
	if err != nil {
		t.Fatalf("CreateTask as user A: %v", err)
	}
	taskID := resp.Msg.Task.Id

	t.Run("user B cannot get user A's task", func(t *testing.T) {
		_, err := h.GetTask(ctxB, connect.NewRequest(&taskv1.GetTaskRequest{Id: taskID}))
		if err == nil {
			t.Fatal("expected not found, got nil")
		}
		ce, ok := err.(*connect.Error)
		if !ok || ce.Code() != connect.CodeNotFound {
			t.Errorf("expected CodeNotFound, got %v", err)
		}
	})

	t.Run("user B cannot update user A's task", func(t *testing.T) {
		_, err := h.UpdateTask(ctxB, connect.NewRequest(&taskv1.UpdateTaskRequest{
			Id:   taskID,
			Name: "hacked",
		}))
		if err == nil {
			t.Fatal("expected error, got nil")
		}
		ce, ok := err.(*connect.Error)
		if !ok || ce.Code() != connect.CodeNotFound {
			t.Errorf("expected CodeNotFound, got %v", err)
		}
	})

	t.Run("user B cannot delete user A's task", func(t *testing.T) {
		_, err := h.DeleteTask(ctxB, connect.NewRequest(&taskv1.DeleteTaskRequest{Id: taskID}))
		if err == nil {
			t.Fatal("expected error, got nil")
		}
		ce, ok := err.(*connect.Error)
		if !ok || ce.Code() != connect.CodeNotFound {
			t.Errorf("expected CodeNotFound, got %v", err)
		}
	})

	t.Run("user B list does not include user A's tasks", func(t *testing.T) {
		listResp, err := h.ListTasks(ctxB, connect.NewRequest(&taskv1.ListTasksRequest{}))
		if err != nil {
			t.Fatalf("ListTasks as user B: %v", err)
		}
		for _, task := range listResp.Msg.Tasks {
			if task.Id == taskID {
				t.Error("user B's list should not include user A's task")
			}
		}
	})
}

// ---- Integration tests: ListTasks completed_pomodoro_count (T007) ----

func TestListTasks_CompletedPomodoroCount(t *testing.T) {
	h, userID := newTestHandler(t)
	ctx := ctxWithUser(userID)
	queries := h.Queries

	// Create two tasks.
	r1, err := h.CreateTask(ctx, connect.NewRequest(&taskv1.CreateTaskRequest{Name: "task with poms"}))
	if err != nil {
		t.Fatalf("CreateTask: %v", err)
	}
	taskID := r1.Msg.Task.Id

	r2, err := h.CreateTask(ctx, connect.NewRequest(&taskv1.CreateTaskRequest{Name: "task no poms"}))
	if err != nil {
		t.Fatalf("CreateTask: %v", err)
	}
	noPomTaskID := r2.Msg.Task.Id

	// Complete 2 pomodoros for taskID.
	for i := 0; i < 2; i++ {
		_, err = queries.StartPomodoro(ctx, db.StartPomodoroParams{UserID: userID, TaskID: taskID})
		if err != nil {
			t.Fatalf("StartPomodoro %d: %v", i, err)
		}
		_, err = queries.CompleteActivePomodoro(ctx, userID)
		if err != nil {
			t.Fatalf("CompleteActivePomodoro %d: %v", i, err)
		}
	}

	listResp, err := h.ListTasks(ctx, connect.NewRequest(&taskv1.ListTasksRequest{}))
	if err != nil {
		t.Fatalf("ListTasks: %v", err)
	}

	counts := make(map[int64]int32)
	for _, task := range listResp.Msg.Tasks {
		counts[task.Id] = task.CompletedPomodoroCount
	}

	if got := counts[taskID]; got != 2 {
		t.Errorf("task with 2 completed pomodoros: got completed_pomodoro_count=%d, want 2", got)
	}
	if got := counts[noPomTaskID]; got != 0 {
		t.Errorf("task with no pomodoros: got completed_pomodoro_count=%d, want 0", got)
	}
}

// ---- Unit tests: ReorderTask validation (no DB needed) ----

func TestReorderTask_NoAnchor(t *testing.T) {
	h := &handler.Task{Queries: nil}
	req := connect.NewRequest(&taskv1.ReorderTaskRequest{TaskId: 1})
	_, err := h.ReorderTask(context.Background(), req)
	if err == nil {
		t.Fatal("expected error, got nil")
	}
	ce, ok := err.(*connect.Error)
	if !ok || ce.Code() != connect.CodeInvalidArgument {
		t.Errorf("expected CodeInvalidArgument, got %v", err)
	}
}

func TestReorderTask_AnchorEqualsTask(t *testing.T) {
	h := &handler.Task{Queries: nil}
	req := connect.NewRequest(&taskv1.ReorderTaskRequest{
		TaskId: 1,
		Anchor: &taskv1.ReorderTaskRequest_BeforeTaskId{BeforeTaskId: 1},
	})
	_, err := h.ReorderTask(context.Background(), req)
	if err == nil {
		t.Fatal("expected error, got nil")
	}
	ce, ok := err.(*connect.Error)
	if !ok || ce.Code() != connect.CodeInvalidArgument {
		t.Errorf("expected CodeInvalidArgument, got %v", err)
	}
}

// ---- Integration tests: position and ReorderTask ----

func TestCreateTask_Position(t *testing.T) {
	h, userID := newTestHandler(t)
	ctx := ctxWithUser(userID)

	r1, err := h.CreateTask(ctx, connect.NewRequest(&taskv1.CreateTaskRequest{Name: "first"}))
	if err != nil {
		t.Fatalf("create first: %v", err)
	}
	if r1.Msg.Task.Position != 0 {
		t.Errorf("first root task position = %d, want 0", r1.Msg.Task.Position)
	}

	r2, err := h.CreateTask(ctx, connect.NewRequest(&taskv1.CreateTaskRequest{Name: "second"}))
	if err != nil {
		t.Fatalf("create second: %v", err)
	}
	if r2.Msg.Task.Position != 1 {
		t.Errorf("second root task position = %d, want 1", r2.Msg.Task.Position)
	}

	// Child tasks get their own position sequence.
	parentID := r1.Msg.Task.Id
	c1, err := h.CreateTask(ctx, connect.NewRequest(&taskv1.CreateTaskRequest{Name: "child1", ParentId: &parentID}))
	if err != nil {
		t.Fatalf("create child1: %v", err)
	}
	if c1.Msg.Task.Position != 0 {
		t.Errorf("first child position = %d, want 0", c1.Msg.Task.Position)
	}

	c2, err := h.CreateTask(ctx, connect.NewRequest(&taskv1.CreateTaskRequest{Name: "child2", ParentId: &parentID}))
	if err != nil {
		t.Fatalf("create child2: %v", err)
	}
	if c2.Msg.Task.Position != 1 {
		t.Errorf("second child position = %d, want 1", c2.Msg.Task.Position)
	}
}

func TestUpdateTask_ReparentPosition(t *testing.T) {
	h, userID := newTestHandler(t)
	ctx := ctxWithUser(userID)

	p1, _ := h.CreateTask(ctx, connect.NewRequest(&taskv1.CreateTaskRequest{Name: "parent1"}))
	p2, _ := h.CreateTask(ctx, connect.NewRequest(&taskv1.CreateTaskRequest{Name: "parent2"}))
	p2ID := p2.Msg.Task.Id

	// Add two children under parent2.
	h.CreateTask(ctx, connect.NewRequest(&taskv1.CreateTaskRequest{Name: "p2-child1", ParentId: &p2ID}))
	h.CreateTask(ctx, connect.NewRequest(&taskv1.CreateTaskRequest{Name: "p2-child2", ParentId: &p2ID}))

	// Move a task from parent1's group to parent2's group.
	p1ID := p1.Msg.Task.Id
	child, _ := h.CreateTask(ctx, connect.NewRequest(&taskv1.CreateTaskRequest{Name: "mover", ParentId: &p1ID}))
	moverID := child.Msg.Task.Id

	ur, err := h.UpdateTask(ctx, connect.NewRequest(&taskv1.UpdateTaskRequest{
		Id:       moverID,
		Name:     "mover",
		ParentId: &p2ID,
	}))
	if err != nil {
		t.Fatalf("UpdateTask reparent: %v", err)
	}
	// Should be at position 2 (end of a 2-item group → index 2).
	if ur.Msg.Task.Position != 2 {
		t.Errorf("reparented task position = %d, want 2", ur.Msg.Task.Position)
	}
}

func TestReorderTask_BeforeMove(t *testing.T) {
	h, userID := newTestHandler(t)
	ctx := ctxWithUser(userID)

	a, _ := h.CreateTask(ctx, connect.NewRequest(&taskv1.CreateTaskRequest{Name: "A"}))
	b, _ := h.CreateTask(ctx, connect.NewRequest(&taskv1.CreateTaskRequest{Name: "B"}))
	c, _ := h.CreateTask(ctx, connect.NewRequest(&taskv1.CreateTaskRequest{Name: "C"}))

	aID, bID, cID := a.Msg.Task.Id, b.Msg.Task.Id, c.Msg.Task.Id

	// Move C before A → order should be C, A, B.
	resp, err := h.ReorderTask(ctx, connect.NewRequest(&taskv1.ReorderTaskRequest{
		TaskId: cID,
		Anchor: &taskv1.ReorderTaskRequest_BeforeTaskId{BeforeTaskId: aID},
	}))
	if err != nil {
		t.Fatalf("ReorderTask: %v", err)
	}
	if len(resp.Msg.Siblings) != 3 {
		t.Fatalf("expected 3 siblings, got %d", len(resp.Msg.Siblings))
	}
	wantOrder := []int64{cID, aID, bID}
	for i, s := range resp.Msg.Siblings {
		if s.Id != wantOrder[i] {
			t.Errorf("siblings[%d].Id = %d, want %d", i, s.Id, wantOrder[i])
		}
		if s.Position != int64(i) {
			t.Errorf("siblings[%d].Position = %d, want %d", i, s.Position, i)
		}
	}
}

func TestReorderTask_AfterMove(t *testing.T) {
	h, userID := newTestHandler(t)
	ctx := ctxWithUser(userID)

	a, _ := h.CreateTask(ctx, connect.NewRequest(&taskv1.CreateTaskRequest{Name: "A"}))
	b, _ := h.CreateTask(ctx, connect.NewRequest(&taskv1.CreateTaskRequest{Name: "B"}))
	c, _ := h.CreateTask(ctx, connect.NewRequest(&taskv1.CreateTaskRequest{Name: "C"}))

	aID, bID, cID := a.Msg.Task.Id, b.Msg.Task.Id, c.Msg.Task.Id

	// Move A after C → order should be B, C, A.
	resp, err := h.ReorderTask(ctx, connect.NewRequest(&taskv1.ReorderTaskRequest{
		TaskId: aID,
		Anchor: &taskv1.ReorderTaskRequest_AfterTaskId{AfterTaskId: cID},
	}))
	if err != nil {
		t.Fatalf("ReorderTask: %v", err)
	}
	wantOrder := []int64{bID, cID, aID}
	for i, s := range resp.Msg.Siblings {
		if s.Id != wantOrder[i] {
			t.Errorf("siblings[%d].Id = %d, want %d", i, s.Id, wantOrder[i])
		}
	}
}

func TestReorderTask_AnchorNotSibling(t *testing.T) {
	h, userID := newTestHandler(t)
	ctx := ctxWithUser(userID)

	root1, _ := h.CreateTask(ctx, connect.NewRequest(&taskv1.CreateTaskRequest{Name: "root1"}))
	root2, _ := h.CreateTask(ctx, connect.NewRequest(&taskv1.CreateTaskRequest{Name: "root2"}))
	root1ID := root1.Msg.Task.Id

	child, _ := h.CreateTask(ctx, connect.NewRequest(&taskv1.CreateTaskRequest{Name: "child", ParentId: &root1ID}))

	// Try to reorder root2 before child (different groups).
	_, err := h.ReorderTask(ctx, connect.NewRequest(&taskv1.ReorderTaskRequest{
		TaskId: root2.Msg.Task.Id,
		Anchor: &taskv1.ReorderTaskRequest_BeforeTaskId{BeforeTaskId: child.Msg.Task.Id},
	}))
	if err == nil {
		t.Fatal("expected error for cross-group reorder, got nil")
	}
	ce, ok := err.(*connect.Error)
	if !ok || ce.Code() != connect.CodeInvalidArgument {
		t.Errorf("expected CodeInvalidArgument, got %v", err)
	}
}

func TestReorderTask_TaskNotFound(t *testing.T) {
	h, userID := newTestHandler(t)
	ctx := ctxWithUser(userID)

	other, _ := h.CreateTask(ctx, connect.NewRequest(&taskv1.CreateTaskRequest{Name: "other"}))

	_, err := h.ReorderTask(ctx, connect.NewRequest(&taskv1.ReorderTaskRequest{
		TaskId: 999999,
		Anchor: &taskv1.ReorderTaskRequest_BeforeTaskId{BeforeTaskId: other.Msg.Task.Id},
	}))
	if err == nil {
		t.Fatal("expected error, got nil")
	}
	ce, ok := err.(*connect.Error)
	if !ok || ce.Code() != connect.CodeNotFound {
		t.Errorf("expected CodeNotFound, got %v", err)
	}
}

func TestReorderTask_NoLoss(t *testing.T) {
	h, userID := newTestHandler(t)
	ctx := ctxWithUser(userID)

	var ids []int64
	for i := 0; i < 5; i++ {
		r, _ := h.CreateTask(ctx, connect.NewRequest(&taskv1.CreateTaskRequest{Name: fmt.Sprintf("task%d", i)}))
		ids = append(ids, r.Msg.Task.Id)
	}

	// Reorder last before first.
	resp, err := h.ReorderTask(ctx, connect.NewRequest(&taskv1.ReorderTaskRequest{
		TaskId: ids[4],
		Anchor: &taskv1.ReorderTaskRequest_BeforeTaskId{BeforeTaskId: ids[0]},
	}))
	if err != nil {
		t.Fatalf("ReorderTask: %v", err)
	}
	if len(resp.Msg.Siblings) != 5 {
		t.Errorf("expected 5 siblings after reorder, got %d", len(resp.Msg.Siblings))
	}
	// Verify no duplicates and contiguous positions.
	seen := make(map[int64]bool)
	for i, s := range resp.Msg.Siblings {
		if seen[s.Id] {
			t.Errorf("duplicate task id %d in siblings", s.Id)
		}
		seen[s.Id] = true
		if s.Position != int64(i) {
			t.Errorf("siblings[%d].Position = %d, want %d", i, s.Position, i)
		}
	}
}

func TestCompleteTask_PositionUnchanged(t *testing.T) {
	h, userID := newTestHandler(t)
	ctx := ctxWithUser(userID)

	a, _ := h.CreateTask(ctx, connect.NewRequest(&taskv1.CreateTaskRequest{Name: "A"}))
	b, _ := h.CreateTask(ctx, connect.NewRequest(&taskv1.CreateTaskRequest{Name: "B"}))
	c, _ := h.CreateTask(ctx, connect.NewRequest(&taskv1.CreateTaskRequest{Name: "C"}))

	aID, bID, cID := a.Msg.Task.Id, b.Msg.Task.Id, c.Msg.Task.Id

	// Reorder: C, A, B.
	h.ReorderTask(ctx, connect.NewRequest(&taskv1.ReorderTaskRequest{
		TaskId: cID,
		Anchor: &taskv1.ReorderTaskRequest_BeforeTaskId{BeforeTaskId: aID},
	}))

	// Complete B.
	_, err := h.CompleteTask(ctx, connect.NewRequest(&taskv1.CompleteTaskRequest{Id: bID}))
	if err != nil {
		t.Fatalf("CompleteTask: %v", err)
	}

	// Uncomplete B.
	_, err = h.UncompleteTask(ctx, connect.NewRequest(&taskv1.UncompleteTaskRequest{Id: bID}))
	if err != nil {
		t.Fatalf("UncompleteTask: %v", err)
	}

	// List tasks and verify positions: C=0, A=1, B=2.
	listResp, err := h.ListTasks(ctx, connect.NewRequest(&taskv1.ListTasksRequest{}))
	if err != nil {
		t.Fatalf("ListTasks: %v", err)
	}
	posMap := make(map[int64]int64)
	for _, task := range listResp.Msg.Tasks {
		posMap[task.Id] = task.Position
	}
	if posMap[cID] != 0 {
		t.Errorf("C position = %d, want 0", posMap[cID])
	}
	if posMap[aID] != 1 {
		t.Errorf("A position = %d, want 1", posMap[aID])
	}
	if posMap[bID] != 2 {
		t.Errorf("B position = %d, want 2", posMap[bID])
	}
}

// ---- Unit tests: snooze_until field ----

func TestDbTaskToProto_SnoozeUntil(t *testing.T) {
	t.Run("snooze_until unset when Valid false", func(t *testing.T) {
		row := db.Task{ID: 1, Name: "task"}
		pt := handler.ExportDbTaskToProto(row)
		if pt.SnoozeUntil != nil {
			t.Errorf("expected nil snooze_until, got %v", pt.SnoozeUntil)
		}
	})

	t.Run("snooze_until set when Valid true", func(t *testing.T) {
		ts := time.Date(2026, 9, 1, 0, 0, 0, 0, time.UTC)
		row := db.Task{
			ID:          1,
			Name:        "task",
			SnoozeUntil: pgtype.Timestamptz{Time: ts, Valid: true},
		}
		pt := handler.ExportDbTaskToProto(row)
		if pt.SnoozeUntil == nil {
			t.Fatal("expected non-nil snooze_until")
		}
		if !pt.SnoozeUntil.AsTime().Equal(ts) {
			t.Errorf("snooze_until = %v, want %v", pt.SnoozeUntil.AsTime(), ts)
		}
	})
}

func TestCreateTask_SnoozeUntil_Integration(t *testing.T) {
	h, userID := newTestHandler(t)
	ctx := ctxWithUser(userID)

	snoozeTime := time.Date(2026, 9, 1, 0, 0, 0, 0, time.UTC)
	snoozeTS := timestamppb.New(snoozeTime)

	resp, err := h.CreateTask(ctx, connect.NewRequest(&taskv1.CreateTaskRequest{
		Name:        "snoozed task",
		SnoozeUntil: snoozeTS,
	}))
	if err != nil {
		t.Fatalf("CreateTask: %v", err)
	}
	if resp.Msg.Task.SnoozeUntil == nil {
		t.Fatal("expected snooze_until to be set")
	}
	if !resp.Msg.Task.SnoozeUntil.AsTime().Equal(snoozeTime) {
		t.Errorf("snooze_until = %v, want %v", resp.Msg.Task.SnoozeUntil.AsTime(), snoozeTime)
	}
}

func TestUpdateTask_SnoozeUntil_Integration(t *testing.T) {
	h, userID := newTestHandler(t)
	ctx := ctxWithUser(userID)

	// Create task without snooze.
	createResp, err := h.CreateTask(ctx, connect.NewRequest(&taskv1.CreateTaskRequest{Name: "task"}))
	if err != nil {
		t.Fatalf("CreateTask: %v", err)
	}
	id := createResp.Msg.Task.Id

	snoozeTime := time.Date(2026, 9, 1, 0, 0, 0, 0, time.UTC)

	t.Run("set snooze_until", func(t *testing.T) {
		resp, err := h.UpdateTask(ctx, connect.NewRequest(&taskv1.UpdateTaskRequest{
			Id:          id,
			Name:        "task",
			SnoozeUntil: timestamppb.New(snoozeTime),
		}))
		if err != nil {
			t.Fatalf("UpdateTask: %v", err)
		}
		if resp.Msg.Task.SnoozeUntil == nil {
			t.Fatal("expected snooze_until to be set after update")
		}
		if !resp.Msg.Task.SnoozeUntil.AsTime().Equal(snoozeTime) {
			t.Errorf("snooze_until = %v, want %v", resp.Msg.Task.SnoozeUntil.AsTime(), snoozeTime)
		}
	})

	t.Run("clear snooze_until (full-replace)", func(t *testing.T) {
		resp, err := h.UpdateTask(ctx, connect.NewRequest(&taskv1.UpdateTaskRequest{
			Id:   id,
			Name: "task",
			// SnoozeUntil intentionally omitted — should clear it.
		}))
		if err != nil {
			t.Fatalf("UpdateTask: %v", err)
		}
		if resp.Msg.Task.SnoozeUntil != nil {
			t.Errorf("expected nil snooze_until after clear, got %v", resp.Msg.Task.SnoozeUntil)
		}
	})
}

// ---- Integration tests: T008 — SetTaskGoal + goal_id in responses ----

func TestSetTaskGoal_SetAndClear(t *testing.T) {
	h, userID := newTestHandler(t)
	ctx := ctxWithUser(userID)
	gh := &handler.Goal{Queries: h.Queries, Pool: h.Pool}

	taskResp, err := h.CreateTask(ctx, connect.NewRequest(&taskv1.CreateTaskRequest{Name: "linked task"}))
	if err != nil {
		t.Fatalf("CreateTask: %v", err)
	}
	taskID := taskResp.Msg.Task.Id

	goalResp, err := gh.CreateGoal(ctx, connect.NewRequest(&goalv1.CreateGoalRequest{Name: "My goal"}))
	if err != nil {
		t.Fatalf("CreateGoal: %v", err)
	}
	goalID := goalResp.Msg.Goal.Id

	t.Run("set goal_id", func(t *testing.T) {
		resp, err := h.SetTaskGoal(ctx, connect.NewRequest(&taskv1.SetTaskGoalRequest{
			TaskId: taskID,
			GoalId: &goalID,
		}))
		if err != nil {
			t.Fatalf("SetTaskGoal: %v", err)
		}
		if resp.Msg.Task.GoalId == nil || *resp.Msg.Task.GoalId != goalID {
			t.Errorf("SetTaskGoal response goal_id = %v, want %d", resp.Msg.Task.GoalId, goalID)
		}
	})

	t.Run("goal_id appears in GetTask", func(t *testing.T) {
		gr, err := h.GetTask(ctx, connect.NewRequest(&taskv1.GetTaskRequest{Id: taskID}))
		if err != nil {
			t.Fatalf("GetTask: %v", err)
		}
		if gr.Msg.Task.GoalId == nil || *gr.Msg.Task.GoalId != goalID {
			t.Errorf("GetTask goal_id = %v, want %d", gr.Msg.Task.GoalId, goalID)
		}
	})

	t.Run("goal_id appears in ListTasks", func(t *testing.T) {
		lr, err := h.ListTasks(ctx, connect.NewRequest(&taskv1.ListTasksRequest{}))
		if err != nil {
			t.Fatalf("ListTasks: %v", err)
		}
		found := false
		for _, task := range lr.Msg.Tasks {
			if task.Id == taskID {
				found = true
				if task.GoalId == nil || *task.GoalId != goalID {
					t.Errorf("ListTasks goal_id = %v, want %d", task.GoalId, goalID)
				}
			}
		}
		if !found {
			t.Error("task not found in ListTasks")
		}
	})

	t.Run("clear goal_id", func(t *testing.T) {
		resp, err := h.SetTaskGoal(ctx, connect.NewRequest(&taskv1.SetTaskGoalRequest{
			TaskId: taskID,
			// GoalId nil → clear
		}))
		if err != nil {
			t.Fatalf("SetTaskGoal clear: %v", err)
		}
		if resp.Msg.Task.GoalId != nil {
			t.Errorf("expected nil goal_id after clear, got %v", resp.Msg.Task.GoalId)
		}
	})

	t.Run("goal_id nil in GetTask after clear", func(t *testing.T) {
		gr, err := h.GetTask(ctx, connect.NewRequest(&taskv1.GetTaskRequest{Id: taskID}))
		if err != nil {
			t.Fatalf("GetTask after clear: %v", err)
		}
		if gr.Msg.Task.GoalId != nil {
			t.Errorf("GetTask goal_id = %v, want nil", gr.Msg.Task.GoalId)
		}
	})
}

func TestSetTaskGoal_AncestorNestingRejected(t *testing.T) {
	h, userID := newTestHandler(t)
	ctx := ctxWithUser(userID)
	gh := &handler.Goal{Queries: h.Queries, Pool: h.Pool}

	goalResp, err := gh.CreateGoal(ctx, connect.NewRequest(&goalv1.CreateGoalRequest{Name: "Ancestor goal"}))
	if err != nil {
		t.Fatalf("CreateGoal: %v", err)
	}
	goalID := goalResp.Msg.Goal.Id

	parentResp, err := h.CreateTask(ctx, connect.NewRequest(&taskv1.CreateTaskRequest{Name: "parent"}))
	if err != nil {
		t.Fatalf("CreateTask parent: %v", err)
	}
	parentID := parentResp.Msg.Task.Id

	childResp, err := h.CreateTask(ctx, connect.NewRequest(&taskv1.CreateTaskRequest{
		Name:     "child",
		ParentId: &parentID,
	}))
	if err != nil {
		t.Fatalf("CreateTask child: %v", err)
	}
	childID := childResp.Msg.Task.Id

	// Assign goal to parent.
	_, err = h.SetTaskGoal(ctx, connect.NewRequest(&taskv1.SetTaskGoalRequest{
		TaskId: parentID,
		GoalId: &goalID,
	}))
	if err != nil {
		t.Fatalf("SetTaskGoal on parent: %v", err)
	}

	// Assigning same (or any) goal to child should fail — ancestor has goal.
	_, err = h.SetTaskGoal(ctx, connect.NewRequest(&taskv1.SetTaskGoalRequest{
		TaskId: childID,
		GoalId: &goalID,
	}))
	if err == nil {
		t.Fatal("expected FailedPrecondition (ancestor has goal), got nil")
	}
	ce, ok := err.(*connect.Error)
	if !ok || ce.Code() != connect.CodeFailedPrecondition {
		t.Errorf("expected CodeFailedPrecondition, got %v", err)
	}
}

func TestSetTaskGoal_DescendantNestingRejected(t *testing.T) {
	h, userID := newTestHandler(t)
	ctx := ctxWithUser(userID)
	gh := &handler.Goal{Queries: h.Queries, Pool: h.Pool}

	goalResp, err := gh.CreateGoal(ctx, connect.NewRequest(&goalv1.CreateGoalRequest{Name: "Descendant goal"}))
	if err != nil {
		t.Fatalf("CreateGoal: %v", err)
	}
	goalID := goalResp.Msg.Goal.Id

	parentResp, err := h.CreateTask(ctx, connect.NewRequest(&taskv1.CreateTaskRequest{Name: "parent"}))
	if err != nil {
		t.Fatalf("CreateTask parent: %v", err)
	}
	parentID := parentResp.Msg.Task.Id

	childResp, err := h.CreateTask(ctx, connect.NewRequest(&taskv1.CreateTaskRequest{
		Name:     "child",
		ParentId: &parentID,
	}))
	if err != nil {
		t.Fatalf("CreateTask child: %v", err)
	}
	childID := childResp.Msg.Task.Id

	// Assign goal to child.
	_, err = h.SetTaskGoal(ctx, connect.NewRequest(&taskv1.SetTaskGoalRequest{
		TaskId: childID,
		GoalId: &goalID,
	}))
	if err != nil {
		t.Fatalf("SetTaskGoal on child: %v", err)
	}

	// Assigning goal to parent should fail — descendant has goal.
	_, err = h.SetTaskGoal(ctx, connect.NewRequest(&taskv1.SetTaskGoalRequest{
		TaskId: parentID,
		GoalId: &goalID,
	}))
	if err == nil {
		t.Fatal("expected FailedPrecondition (descendant has goal), got nil")
	}
	ce, ok := err.(*connect.Error)
	if !ok || ce.Code() != connect.CodeFailedPrecondition {
		t.Errorf("expected CodeFailedPrecondition, got %v", err)
	}
}

func TestSetTaskGoal_NotFound(t *testing.T) {
	h, userID := newTestHandler(t)
	ctx := ctxWithUser(userID)
	gh := &handler.Goal{Queries: h.Queries, Pool: h.Pool}

	goalResp, err := gh.CreateGoal(ctx, connect.NewRequest(&goalv1.CreateGoalRequest{Name: "real goal"}))
	if err != nil {
		t.Fatalf("CreateGoal: %v", err)
	}
	goalID := goalResp.Msg.Goal.Id

	taskResp, err := h.CreateTask(ctx, connect.NewRequest(&taskv1.CreateTaskRequest{Name: "real task"}))
	if err != nil {
		t.Fatalf("CreateTask: %v", err)
	}
	taskID := taskResp.Msg.Task.Id

	t.Run("unknown task", func(t *testing.T) {
		_, err := h.SetTaskGoal(ctx, connect.NewRequest(&taskv1.SetTaskGoalRequest{
			TaskId: 999999,
			GoalId: &goalID,
		}))
		if err == nil {
			t.Fatal("expected CodeNotFound, got nil")
		}
		ce, ok := err.(*connect.Error)
		if !ok || ce.Code() != connect.CodeNotFound {
			t.Errorf("expected CodeNotFound, got %v", err)
		}
	})

	t.Run("unknown goal", func(t *testing.T) {
		badGoalID := int64(999999)
		_, err := h.SetTaskGoal(ctx, connect.NewRequest(&taskv1.SetTaskGoalRequest{
			TaskId: taskID,
			GoalId: &badGoalID,
		}))
		if err == nil {
			t.Fatal("expected CodeNotFound, got nil")
		}
		ce, ok := err.(*connect.Error)
		if !ok || ce.Code() != connect.CodeNotFound {
			t.Errorf("expected CodeNotFound, got %v", err)
		}
	})
}

// ── Fix #4: SetTaskGoal rejects completed/archived goals ─────────────────────

// TestSetTaskGoal_RejectsCompletedGoal verifies that linking a task to a
// completed goal returns InvalidArgument — completed goals are closed and should
// not accept new task associations.
func TestSetTaskGoal_RejectsCompletedGoal(t *testing.T) {
	h, userID := newTestHandler(t)
	gh, _ := newGoalTestHandlerWithPool(t, h)
	ctx := ctxWithUser(userID)

	// Create a goal and transition it to completed.
	goalResp, err := gh.CreateGoal(ctx, connect.NewRequest(&goalv1.CreateGoalRequest{Name: "Done goal"}))
	if err != nil {
		t.Fatalf("CreateGoal: %v", err)
	}
	goalID := goalResp.Msg.Goal.Id
	_, err = gh.SetGoalState(ctx, connect.NewRequest(&goalv1.SetGoalStateRequest{
		Id:    goalID,
		State: goalv1.GoalState_GOAL_STATE_COMPLETED,
	}))
	if err != nil {
		t.Fatalf("SetGoalState completed: %v", err)
	}

	// Create a task.
	taskResp, err := h.CreateTask(ctx, connect.NewRequest(&taskv1.CreateTaskRequest{Name: "My task"}))
	if err != nil {
		t.Fatalf("CreateTask: %v", err)
	}
	taskID := taskResp.Msg.Task.Id

	// Linking to completed goal must be rejected.
	_, err = h.SetTaskGoal(ctx, connect.NewRequest(&taskv1.SetTaskGoalRequest{
		TaskId: taskID,
		GoalId: &goalID,
	}))
	if err == nil {
		t.Fatal("expected error linking task to completed goal, got nil")
	}
	ce, ok := err.(*connect.Error)
	if !ok || ce.Code() != connect.CodeInvalidArgument {
		t.Errorf("expected CodeInvalidArgument for completed goal, got %v", err)
	}
}

// TestSetTaskGoal_RejectsArchivedGoal verifies that linking a task to an
// archived goal returns InvalidArgument.
func TestSetTaskGoal_RejectsArchivedGoal(t *testing.T) {
	h, userID := newTestHandler(t)
	gh, _ := newGoalTestHandlerWithPool(t, h)
	ctx := ctxWithUser(userID)

	// Create a goal and archive it.
	goalResp, err := gh.CreateGoal(ctx, connect.NewRequest(&goalv1.CreateGoalRequest{Name: "Archived goal"}))
	if err != nil {
		t.Fatalf("CreateGoal: %v", err)
	}
	goalID := goalResp.Msg.Goal.Id
	_, err = gh.SetGoalState(ctx, connect.NewRequest(&goalv1.SetGoalStateRequest{
		Id:    goalID,
		State: goalv1.GoalState_GOAL_STATE_ARCHIVED,
	}))
	if err != nil {
		t.Fatalf("SetGoalState archived: %v", err)
	}

	// Create a task.
	taskResp, err := h.CreateTask(ctx, connect.NewRequest(&taskv1.CreateTaskRequest{Name: "My task"}))
	if err != nil {
		t.Fatalf("CreateTask: %v", err)
	}
	taskID := taskResp.Msg.Task.Id

	// Linking to archived goal must be rejected.
	_, err = h.SetTaskGoal(ctx, connect.NewRequest(&taskv1.SetTaskGoalRequest{
		TaskId: taskID,
		GoalId: &goalID,
	}))
	if err == nil {
		t.Fatal("expected error linking task to archived goal, got nil")
	}
	ce, ok := err.(*connect.Error)
	if !ok || ce.Code() != connect.CodeInvalidArgument {
		t.Errorf("expected CodeInvalidArgument for archived goal, got %v", err)
	}
}

// ---- Spec 072: goal handling on parent change ----

// requireTask queries the stored row for verification.
func requireTask(t *testing.T, pool *pgxpool.Pool, userID, taskID int64) db.Task {
	t.Helper()
	var row db.Task
	err := pool.QueryRow(context.Background(),
		`SELECT id, parent_id, user_id, goal_id, position FROM tasks WHERE id = $1 AND user_id = $2`,
		taskID, userID).Scan(&row.ID, &row.ParentID, &row.UserID, &row.GoalID, &row.Position)
	if err != nil {
		t.Fatalf("query task %d: %v", taskID, err)
	}
	return row
}

// countGoalLinksBelow runs the invariant query and returns how many rows
// in the user's tree have parent_id IS NOT NULL AND goal_id IS NOT NULL.
func countGoalLinksBelow(t *testing.T, pool *pgxpool.Pool, userID int64) int {
	t.Helper()
	var n int
	if err := pool.QueryRow(context.Background(),
		`SELECT COUNT(*) FROM tasks WHERE user_id = $1 AND parent_id IS NOT NULL AND goal_id IS NOT NULL`,
		userID).Scan(&n); err != nil {
		t.Fatalf("count invariant violations: %v", err)
	}
	return n
}

// ----- US1: descent clears goal link -----

func TestUpdateTask_MoveGoalLinkedUnderGoalLinked(t *testing.T) {
	h, userID := newTestHandler(t)
	ctx := ctxWithUser(userID)

	t1ID := mustCreateTask(t, ctx, h, "Fitness task", nil)
	fitnessID := linkGoalForTest(t, ctx, h, t1ID, "Fitness")

	t2ID := mustCreateTask(t, ctx, h, "Health task", nil)
	linkGoalForTest(t, ctx, h, t2ID, "Health")

	// Move Health (t2) under Fitness (t1). The two goals differ — used to fail.
	ur, err := h.UpdateTask(ctx, connect.NewRequest(&taskv1.UpdateTaskRequest{
		Id:       t2ID,
		Name:     "Health task",
		ParentId: &t1ID,
	}))
	if err != nil {
		t.Fatalf("UpdateTask reparent: %v", err)
	}
	if ur.Msg.Task.GoalId != nil {
		t.Errorf("moved task goal_id = %v, want nil", ur.Msg.Task.GoalId)
	}
	if ur.Msg.Task.ParentId == nil || *ur.Msg.Task.ParentId != t1ID {
		t.Errorf("moved task parent_id = %v, want %d", ur.Msg.Task.ParentId, t1ID)
	}
	// The destination's own goal link must be untouched by the move.
	t1Row := requireTask(t, h.Pool, userID, t1ID)
	if !t1Row.GoalID.Valid || t1Row.GoalID.Int64 != fitnessID {
		t.Errorf("t1 goal_id = %v, want %d (Fitness link must survive the move)", t1Row.GoalID, fitnessID)
	}
}

func TestUpdateTask_MoveUnderSameGoal(t *testing.T) {
	h, userID := newTestHandler(t)
	ctx := ctxWithUser(userID)

	t1ID := mustCreateTask(t, ctx, h, "first", nil)
	fitnessID := linkGoalForTest(t, ctx, h, t1ID, "Fitness")

	t2ID := mustCreateTask(t, ctx, h, "second", nil)
	linkGoalForTest(t, ctx, h, t2ID, "Fitness")

	if _, err := h.UpdateTask(ctx, connect.NewRequest(&taskv1.UpdateTaskRequest{
		Id:       t2ID,
		Name:     "second",
		ParentId: &t1ID,
	})); err != nil {
		t.Fatalf("UpdateTask (same goal on both sides): %v", err)
	}
	t2Row := requireTask(t, h.Pool, userID, t2ID)
	if t2Row.GoalID.Valid {
		t.Errorf("t2 goal_id = %v, want NULL — descent always clears the moved task's own link", t2Row.GoalID.Int64)
	}
	t1Row := requireTask(t, h.Pool, userID, t1ID)
	if !t1Row.GoalID.Valid || t1Row.GoalID.Int64 != fitnessID {
		t.Errorf("t1 goal_id = %v, want %d (effective goal for t2 should still resolve to Fitness)", t1Row.GoalID, fitnessID)
	}
}

func TestUpdateTask_MoveGoalLinkedUnderGoalFree(t *testing.T) {
	h, userID := newTestHandler(t)
	ctx := ctxWithUser(userID)

	fitnessID := mustCreateTask(t, ctx, h, "fitness", nil)
	linkGoalForTest(t, ctx, h, fitnessID, "Fitness")

	freeID := mustCreateTask(t, ctx, h, "free", nil)

	ur, err := h.UpdateTask(ctx, connect.NewRequest(&taskv1.UpdateTaskRequest{
		Id:       fitnessID,
		Name:     "fitness",
		ParentId: &freeID,
	}))
	if err != nil {
		t.Fatalf("UpdateTask: %v", err)
	}
	if ur.Msg.Task.GoalId != nil {
		t.Errorf("moved task goal_id = %v, want nil", ur.Msg.Task.GoalId)
	}
	row := requireTask(t, h.Pool, userID, fitnessID)
	if row.GoalID.Valid {
		t.Errorf("row goal_id = %v, want NULL — a regression in the SQL write would slip past the RPC response alone", row.GoalID)
	}
}

func TestUpdateTask_RenameDoesNotClearGoal(t *testing.T) {
	h, userID := newTestHandler(t)
	ctx := ctxWithUser(userID)

	topID := mustCreateTask(t, ctx, h, "Run a 5k", nil)
	goalID := linkGoalForTest(t, ctx, h, topID, "Fitness")

	// Rename the goal-linked top-level task while sending nil parent_id.
	// Top-level + nil parent_id → noChange → goal preserved.
	ur, err := h.UpdateTask(ctx, connect.NewRequest(&taskv1.UpdateTaskRequest{
		Id:   topID,
		Name: "Run a 10k",
	}))
	if err != nil {
		t.Fatalf("rename top: %v", err)
	}
	if ur.Msg.Task.GoalId == nil || *ur.Msg.Task.GoalId != goalID {
		t.Errorf("top goal_id = %v, want %d", ur.Msg.Task.GoalId, goalID)
	}

	// Now nested case: child under top, rename while sending its current parent_id.
	childID := mustCreateTask(t, ctx, h, "Buy shoes", &topID)

	// Hand-craft a stray grandchild holding its own goal_id — a violation of
	// the doc's invariant that no parented task holds its own goal_id — to
	// detect a spurious ClearSubtreeGoals firing on noChange. G5 says a
	// no-change update must leave descendant goal links exactly as they are.
	otherGoalID := linkGoalForTest(t, ctx, h, mustCreateTask(t, ctx, h, "temp root for goal", nil), "Side quest")
	grandchildID := mustCreateTask(t, ctx, h, "Tie laces", &childID)
	if _, err := h.Pool.Exec(context.Background(),
		`UPDATE tasks SET goal_id = $1 WHERE id = $2 AND user_id = $3`,
		otherGoalID, grandchildID, userID); err != nil {
		t.Fatalf("write stray goal_id on grandchild: %v", err)
	}

	if _, err := h.UpdateTask(ctx, connect.NewRequest(&taskv1.UpdateTaskRequest{
		Id:       childID,
		Name:     "Buy better shoes",
		ParentId: &topID,
	})); err != nil {
		t.Fatalf("rename child: %v", err)
	}
	grandchild := requireTask(t, h.Pool, userID, grandchildID)
	if !grandchild.GoalID.Valid || grandchild.GoalID.Int64 != otherGoalID {
		t.Errorf("grandchild goal_id = %v, want %d — noChange must not touch descendant goal links", grandchild.GoalID, otherGoalID)
	}
}

// ----- US2: promotion preserves the inherited goal -----

func TestUpdateTask_PromoteInheritsGoal(t *testing.T) {
	h, userID := newTestHandler(t)
	ctx := ctxWithUser(userID)

	parentID := mustCreateTask(t, ctx, h, "parent", nil)
	goalID := linkGoalForTest(t, ctx, h, parentID, "Fitness")

	childID := mustCreateTask(t, ctx, h, "child", &parentID)

	ur, err := h.UpdateTask(ctx, connect.NewRequest(&taskv1.UpdateTaskRequest{
		Id:   childID,
		Name: "child",
		// ParentId omitted → promotion to root.
	}))
	if err != nil {
		t.Fatalf("promote: %v", err)
	}
	if ur.Msg.Task.GoalId == nil || *ur.Msg.Task.GoalId != goalID {
		t.Errorf("promoted goal_id = %v, want %d", ur.Msg.Task.GoalId, goalID)
	}
	if ur.Msg.Task.ParentId != nil {
		t.Errorf("promoted parent_id = %v, want nil", ur.Msg.Task.ParentId)
	}
	row := requireTask(t, h.Pool, userID, childID)
	if row.ParentID.Valid {
		t.Errorf("row parent_id still valid after promote")
	}
	if !row.GoalID.Valid || row.GoalID.Int64 != goalID {
		t.Errorf("row goal_id = %v, want %d", row.GoalID.Int64, goalID)
	}
}

func TestUpdateTask_PromoteFromDepthTwo(t *testing.T) {
	h, userID := newTestHandler(t)
	ctx := ctxWithUser(userID)

	aID := mustCreateTask(t, ctx, h, "A", nil)
	goalID := linkGoalForTest(t, ctx, h, aID, "Fitness")

	bID := mustCreateTask(t, ctx, h, "B", &aID)
	cID := mustCreateTask(t, ctx, h, "C", &bID)

	ur, err := h.UpdateTask(ctx, connect.NewRequest(&taskv1.UpdateTaskRequest{
		Id:   cID,
		Name: "C",
		// ParentId omitted → promote
	}))
	if err != nil {
		t.Fatalf("promote C: %v", err)
	}
	if ur.Msg.Task.GoalId == nil || *ur.Msg.Task.GoalId != goalID {
		t.Errorf("promoted C goal_id = %v, want %d", ur.Msg.Task.GoalId, goalID)
	}
}

func TestUpdateTask_PromoteWithNoGoalAnywhere(t *testing.T) {
	h, userID := newTestHandler(t)
	ctx := ctxWithUser(userID)

	parentID := mustCreateTask(t, ctx, h, "parent (no goal)", nil)
	childID := mustCreateTask(t, ctx, h, "child", &parentID)

	ur, err := h.UpdateTask(ctx, connect.NewRequest(&taskv1.UpdateTaskRequest{
		Id:   childID,
		Name: "child",
	}))
	if err != nil {
		t.Fatalf("promote (no ancestor goal): %v", err)
	}
	if ur.Msg.Task.GoalId != nil {
		t.Errorf("goal_id = %v, want nil", ur.Msg.Task.GoalId)
	}
}

func TestUpdateTask_NoChangeKeepsGoal(t *testing.T) {
	h, userID := newTestHandler(t)
	ctx := ctxWithUser(userID)

	topID := mustCreateTask(t, ctx, h, "top", nil)
	goalID := linkGoalForTest(t, ctx, h, topID, "Fitness")

	// Already top-level: send again with nil parent_id — must be noChange,
	// not a promotion (classifyParentChange: !storeHasParent && !reqHasParent).
	ur, err := h.UpdateTask(ctx, connect.NewRequest(&taskv1.UpdateTaskRequest{
		Id:   topID,
		Name: "top",
	}))
	if err != nil {
		t.Fatalf("UpdateTask (no-op promote): %v", err)
	}
	if ur.Msg.Task.GoalId == nil || *ur.Msg.Task.GoalId != goalID {
		t.Errorf("goal_id = %v, want %d", ur.Msg.Task.GoalId, goalID)
	}
}

// TestUpdateTask_PromoteKeepsOwnGoal exercises an actual promotion
// (parentPromotion, not parentNoChange): a child that already holds its own
// goal_id — a legacy row, since SetTaskGoal refuses this on a parented task —
// must keep that goal rather than inherit its ancestor's on promotion. This
// is the `stored.GoalID.Valid` branch of the promotion path.
func TestUpdateTask_PromoteKeepsOwnGoal(t *testing.T) {
	h, userID := newTestHandler(t)
	ctx := ctxWithUser(userID)

	parentID := mustCreateTask(t, ctx, h, "parent", nil)
	linkGoalForTest(t, ctx, h, parentID, "Fitness")

	childID := mustCreateTask(t, ctx, h, "child", &parentID)
	ownGoalID := linkGoalForTest(t, ctx, h, mustCreateTask(t, ctx, h, "temp root for own goal", nil), "Own goal")
	if _, err := h.Pool.Exec(context.Background(),
		`UPDATE tasks SET goal_id = $1 WHERE id = $2 AND user_id = $3`,
		ownGoalID, childID, userID); err != nil {
		t.Fatalf("write own goal_id on child: %v", err)
	}

	ur, err := h.UpdateTask(ctx, connect.NewRequest(&taskv1.UpdateTaskRequest{
		Id:   childID,
		Name: "child",
		// ParentId omitted → promotion to root.
	}))
	if err != nil {
		t.Fatalf("promote (own goal): %v", err)
	}
	if ur.Msg.Task.GoalId == nil || *ur.Msg.Task.GoalId != ownGoalID {
		t.Errorf("promoted goal_id = %v, want %d (own goal kept, not parent's Fitness)", ur.Msg.Task.GoalId, ownGoalID)
	}
}

// TestUpdateTask_PromoteRepositions covers G7 for a promotion: the task
// lands at the end of the (populated) root sibling group. Before the fix
// promotions kept a stale position; nothing previously read Position after
// a promotion, so this arm of task.go's repositioning block had no coverage.
func TestUpdateTask_PromoteRepositions(t *testing.T) {
	h, userID := newTestHandler(t)
	ctx := ctxWithUser(userID)

	// Populate the root group with two siblings ahead of the promoted task.
	mustCreateTask(t, ctx, h, "root sibling 1", nil)
	mustCreateTask(t, ctx, h, "root sibling 2", nil)

	parentID := mustCreateTask(t, ctx, h, "parent", nil)
	childID := mustCreateTask(t, ctx, h, "child", &parentID)

	ur, err := h.UpdateTask(ctx, connect.NewRequest(&taskv1.UpdateTaskRequest{
		Id:   childID,
		Name: "child",
		// ParentId omitted → promotion to root.
	}))
	if err != nil {
		t.Fatalf("promote: %v", err)
	}
	// Root group at promote time: sibling1, sibling2, parent → max position 2.
	if ur.Msg.Task.Position != 3 {
		t.Errorf("promoted task position = %d, want 3 (end of root group)", ur.Msg.Task.Position)
	}
	row := requireTask(t, h.Pool, userID, childID)
	if row.Position != 3 {
		t.Errorf("row position = %d, want 3", row.Position)
	}
}

// TestUpdateTask_PromoteRepairsDescendantGoals covers the promotion half of
// G3's symmetry with G2: promoting T must clear stray goal links on T's
// descendants too, the same way ClearSubtreeGoals already runs on descent.
// Without this, a subtree can end up with two competing goal links — a state
// SetTaskGoal refuses to create.
func TestUpdateTask_PromoteRepairsDescendantGoals(t *testing.T) {
	h, userID := newTestHandler(t)
	ctx := ctxWithUser(userID)

	parentID := mustCreateTask(t, ctx, h, "parent", nil)
	fitnessID := linkGoalForTest(t, ctx, h, parentID, "Fitness")

	movedID := mustCreateTask(t, ctx, h, "moved", &parentID)
	grandchildID := mustCreateTask(t, ctx, h, "grandchild", &movedID)

	// Hand-craft a legacy stray goal_id on the grandchild — no API path
	// reaches this state today, but old rows may already be in it.
	strayGoalID := linkGoalForTest(t, ctx, h, mustCreateTask(t, ctx, h, "temp root for stray goal", nil), "Stray")
	if _, err := h.Pool.Exec(context.Background(),
		`UPDATE tasks SET goal_id = $1 WHERE id = $2 AND user_id = $3`,
		strayGoalID, grandchildID, userID); err != nil {
		t.Fatalf("write stray goal_id on grandchild: %v", err)
	}

	if _, err := h.UpdateTask(ctx, connect.NewRequest(&taskv1.UpdateTaskRequest{
		Id:   movedID,
		Name: "moved",
		// ParentId omitted → promotion to root.
	})); err != nil {
		t.Fatalf("promote: %v", err)
	}

	moved := requireTask(t, h.Pool, userID, movedID)
	if !moved.GoalID.Valid || moved.GoalID.Int64 != fitnessID {
		t.Errorf("moved goal_id = %v, want %d (inherited from parent)", moved.GoalID, fitnessID)
	}
	grandchild := requireTask(t, h.Pool, userID, grandchildID)
	if grandchild.GoalID.Valid {
		t.Errorf("grandchild goal_id = %v, want NULL — promotion must clear descendant goal links same as descent", grandchild.GoalID.Int64)
	}
}

func TestUpdateTask_PromoteCarriesClosedGoal(t *testing.T) {
	for _, tc := range []struct {
		name  string
		state goalv1.GoalState
	}{
		{"completed", goalv1.GoalState_GOAL_STATE_COMPLETED},
		{"archived", goalv1.GoalState_GOAL_STATE_ARCHIVED},
	} {
		t.Run(tc.name, func(t *testing.T) {
			h, userID := newTestHandler(t)
			ctx := ctxWithUser(userID)
			gh := &handler.Goal{Queries: h.Queries, Pool: h.Pool}

			parentID := mustCreateTask(t, ctx, h, "parent", nil)
			goalResp, err := gh.CreateGoal(ctx, connect.NewRequest(&goalv1.CreateGoalRequest{Name: "Closed goal"}))
			if err != nil {
				t.Fatalf("CreateGoal: %v", err)
			}
			goalID := goalResp.Msg.Goal.Id
			if _, err := h.SetTaskGoal(ctx, connect.NewRequest(&taskv1.SetTaskGoalRequest{
				TaskId: parentID, GoalId: &goalID,
			})); err != nil {
				t.Fatalf("SetTaskGoal parent: %v", err)
			}

			childID := mustCreateTask(t, ctx, h, "child", &parentID)

			// Close the goal — direct SetTaskGoal must now refuse.
			if _, err := gh.SetGoalState(ctx, connect.NewRequest(&goalv1.SetGoalStateRequest{
				Id: goalID, State: tc.state,
			})); err != nil {
				t.Fatalf("SetGoalState: %v", err)
			}
			freshTaskID := mustCreateTask(t, ctx, h, "fresh task", nil)
			if _, err := h.SetTaskGoal(ctx, connect.NewRequest(&taskv1.SetTaskGoalRequest{
				TaskId: freshTaskID, GoalId: &goalID,
			})); err == nil {
				t.Fatal("SetTaskGoal to closed goal should fail")
			}

			// But promotion still carries it.
			ur, err := h.UpdateTask(ctx, connect.NewRequest(&taskv1.UpdateTaskRequest{
				Id:   childID,
				Name: "child",
			}))
			if err != nil {
				t.Fatalf("promote under closed-goal parent: %v", err)
			}
			if ur.Msg.Task.GoalId == nil || *ur.Msg.Task.GoalId != goalID {
				t.Errorf("promoted child goal_id = %v, want %d (closed, still inherited)", ur.Msg.Task.GoalId, goalID)
			}
		})
	}
}

// ----- US3: invariant holds across moves and repairs legacy rows -----

func TestUpdateTask_DescentClearsDescendantGoals(t *testing.T) {
	h, userID := newTestHandler(t)
	ctx := ctxWithUser(userID)

	gh := &handler.Goal{Queries: h.Queries, Pool: h.Pool}
	goalResp, err := gh.CreateGoal(ctx, connect.NewRequest(&goalv1.CreateGoalRequest{Name: "legacy goal"}))
	if err != nil {
		t.Fatalf("CreateGoal: %v", err)
	}
	goalID := goalResp.Msg.Goal.Id

	topID := mustCreateTask(t, ctx, h, "top", nil)
	midID := mustCreateTask(t, ctx, h, "mid", &topID)
	leafID := mustCreateTask(t, ctx, h, "leaf", &midID)
	freeID := mustCreateTask(t, ctx, h, "free top", nil)

	// Hand-craft a legacy-shaped violation by writing goal_id directly on a
	// nested task; no code path today reaches this state via the API.
	if _, err := h.Pool.Exec(context.Background(),
		`UPDATE tasks SET goal_id = $1 WHERE id = $2 AND user_id = $3`,
		goalID, leafID, userID); err != nil {
		t.Fatalf("write legacy goal_id: %v", err)
	}

	if _, err := h.UpdateTask(ctx, connect.NewRequest(&taskv1.UpdateTaskRequest{
		Id:       topID,
		Name:     "top",
		ParentId: &freeID,
	})); err != nil {
		t.Fatalf("UpdateTask (descent): %v", err)
	}
	leaf := requireTask(t, h.Pool, userID, leafID)
	if leaf.GoalID.Valid {
		t.Errorf("leaf goal_id = %v, want NULL after subtree clear", leaf.GoalID.Int64)
	}
}

func TestUpdateTask_DescentClearsDeepDescendantGoals(t *testing.T) {
	h, userID := newTestHandler(t)
	ctx := ctxWithUser(userID)

	gh := &handler.Goal{Queries: h.Queries, Pool: h.Pool}
	goalResp, err := gh.CreateGoal(ctx, connect.NewRequest(&goalv1.CreateGoalRequest{Name: "deep legacy"}))
	if err != nil {
		t.Fatalf("CreateGoal: %v", err)
	}
	goalID := goalResp.Msg.Goal.Id

	aID := mustCreateTask(t, ctx, h, "A", nil)
	bID := mustCreateTask(t, ctx, h, "B", &aID)
	cID := mustCreateTask(t, ctx, h, "C", &bID)
	dID := mustCreateTask(t, ctx, h, "D", &cID)
	freeID := mustCreateTask(t, ctx, h, "free", nil)

	if _, err := h.Pool.Exec(context.Background(),
		`UPDATE tasks SET goal_id = $1 WHERE id = $2 AND user_id = $3`,
		goalID, dID, userID); err != nil {
		t.Fatalf("write legacy goal_id on deep row: %v", err)
	}
	if _, err := h.UpdateTask(ctx, connect.NewRequest(&taskv1.UpdateTaskRequest{
		Id:       aID,
		Name:     "A",
		ParentId: &freeID,
	})); err != nil {
		t.Fatalf("UpdateTask (descent): %v", err)
	}
	deep := requireTask(t, h.Pool, userID, dID)
	if deep.GoalID.Valid {
		t.Errorf("deep leaf goal_id = %v, want NULL — ClearSubtreeGoals should recurse", deep.GoalID.Int64)
	}
}

// TestUpdateTask_DescentRollsBackOnFailure covers G6: ClearSubtreeGoals runs
// before the row UPDATE, inside the same transaction. If a later step in
// that transaction fails, the subtree clear must not be left in effect. A
// second connection holds a row lock on the moved task itself so the
// handler's own UPDATE blocks; a short context deadline then forces the
// handler's call to fail, and the descendant's goal link must survive.
func TestUpdateTask_DescentRollsBackOnFailure(t *testing.T) {
	h, userID := newTestHandler(t)
	ctx := ctxWithUser(userID)

	freeID := mustCreateTask(t, ctx, h, "free top", nil)
	movedID := mustCreateTask(t, ctx, h, "moved", nil)
	childID := mustCreateTask(t, ctx, h, "child", &movedID)
	strayGoalID := linkGoalForTest(t, ctx, h, mustCreateTask(t, ctx, h, "temp root for stray goal", nil), "Stray")
	if _, err := h.Pool.Exec(context.Background(),
		`UPDATE tasks SET goal_id = $1 WHERE id = $2 AND user_id = $3`,
		strayGoalID, childID, userID); err != nil {
		t.Fatalf("write stray goal_id on child: %v", err)
	}

	// Hold a row lock on the moved task from a separate connection so the
	// handler's own UPDATE of that row blocks until this lock is released.
	conn, err := h.Pool.Acquire(context.Background())
	if err != nil {
		t.Fatalf("Acquire: %v", err)
	}
	defer conn.Release()
	lockTx, err := conn.Begin(context.Background())
	if err != nil {
		t.Fatalf("Begin lock tx: %v", err)
	}
	defer lockTx.Rollback(context.Background())
	if _, err := lockTx.Exec(context.Background(),
		`SELECT id FROM tasks WHERE id = $1 FOR UPDATE`, movedID); err != nil {
		t.Fatalf("lock moved row: %v", err)
	}

	shortCtx, cancel := context.WithTimeout(ctx, 500*time.Millisecond)
	defer cancel()
	if _, err := h.UpdateTask(shortCtx, connect.NewRequest(&taskv1.UpdateTaskRequest{
		Id:       movedID,
		Name:     "moved",
		ParentId: &freeID,
	})); err == nil {
		t.Fatal("UpdateTask should have failed while the row was locked")
	}

	// Release the lock so the read below is not itself blocked.
	if err := lockTx.Rollback(context.Background()); err != nil {
		t.Fatalf("release lock: %v", err)
	}

	child := requireTask(t, h.Pool, userID, childID)
	if !child.GoalID.Valid || child.GoalID.Int64 != strayGoalID {
		t.Errorf("child goal_id = %v, want %d — ClearSubtreeGoals must roll back with the rest of the failed transaction", child.GoalID, strayGoalID)
	}
}

func TestUpdateTask_InvariantAfterMoveSequence(t *testing.T) {
	h, userID := newTestHandler(t)
	ctx := ctxWithUser(userID)

	// Seed: A (goal Fitness), B (goal Health) at root. C under A. D under C.
	aID := mustCreateTask(t, ctx, h, "A", nil)
	linkGoalForTest(t, ctx, h, aID, "Fitness")

	bID := mustCreateTask(t, ctx, h, "B", nil)
	linkGoalForTest(t, ctx, h, bID, "Health")

	cID := mustCreateTask(t, ctx, h, "C", &aID)
	dID := mustCreateTask(t, ctx, h, "D", &cID)

	// 1. Promote C → inherits Fitness.
	if _, err := h.UpdateTask(ctx, connect.NewRequest(&taskv1.UpdateTaskRequest{
		Id: cID, Name: "C",
	})); err != nil {
		t.Fatalf("promote C: %v", err)
	}

	// 2. Descent: A under B → Fitness carried via B; A loses its own link.
	if _, err := h.UpdateTask(ctx, connect.NewRequest(&taskv1.UpdateTaskRequest{
		Id: aID, Name: "A", ParentId: &bID,
	})); err != nil {
		t.Fatalf("move A under B: %v", err)
	}

	// 3. Promote D to root → walks up: D's parent is C (top, goal=Fitness).
	if _, err := h.UpdateTask(ctx, connect.NewRequest(&taskv1.UpdateTaskRequest{
		Id: dID, Name: "D",
	})); err != nil {
		t.Fatalf("promote D: %v", err)
	}

	if n := countGoalLinksBelow(t, h.Pool, userID); n != 0 {
		t.Errorf("invariant violations after move sequence: %d", n)
	}
	cRow := requireTask(t, h.Pool, userID, cID)
	if !cRow.GoalID.Valid {
		t.Errorf("C should retain goal_id after promotion, got NULL")
	}
}

