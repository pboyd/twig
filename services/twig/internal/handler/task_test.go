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
		_, _ = pool.Exec(context.Background(), "DELETE FROM users WHERE id = $1", userID)
	})
	return &handler.Task{Queries: queries, Pool: pool}, userID
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

func TestUpdateTask_ReparentGoalNestingRejected(t *testing.T) {
	h, userID := newTestHandler(t)
	ctx := ctxWithUser(userID)
	gh := &handler.Goal{Queries: h.Queries, Pool: h.Pool}

	goalResp, err := gh.CreateGoal(ctx, connect.NewRequest(&goalv1.CreateGoalRequest{Name: "Reparent goal"}))
	if err != nil {
		t.Fatalf("CreateGoal: %v", err)
	}
	goalID := goalResp.Msg.Goal.Id

	// Create two separate tasks, each linked to the same goal (different subtrees).
	t1Resp, err := h.CreateTask(ctx, connect.NewRequest(&taskv1.CreateTaskRequest{Name: "task1"}))
	if err != nil {
		t.Fatalf("CreateTask task1: %v", err)
	}
	t1ID := t1Resp.Msg.Task.Id

	t2Resp, err := h.CreateTask(ctx, connect.NewRequest(&taskv1.CreateTaskRequest{Name: "task2"}))
	if err != nil {
		t.Fatalf("CreateTask task2: %v", err)
	}
	t2ID := t2Resp.Msg.Task.Id

	// Each root task gets its own goal assignment.
	goal2Resp, err := gh.CreateGoal(ctx, connect.NewRequest(&goalv1.CreateGoalRequest{Name: "Second goal"}))
	if err != nil {
		t.Fatalf("CreateGoal second: %v", err)
	}
	goal2ID := goal2Resp.Msg.Goal.Id

	_, err = h.SetTaskGoal(ctx, connect.NewRequest(&taskv1.SetTaskGoalRequest{TaskId: t1ID, GoalId: &goalID}))
	if err != nil {
		t.Fatalf("SetTaskGoal task1: %v", err)
	}
	_, err = h.SetTaskGoal(ctx, connect.NewRequest(&taskv1.SetTaskGoalRequest{TaskId: t2ID, GoalId: &goal2ID}))
	if err != nil {
		t.Fatalf("SetTaskGoal task2: %v", err)
	}

	// Moving task2 under task1 would nest two goal associations — must fail.
	_, err = h.UpdateTask(ctx, connect.NewRequest(&taskv1.UpdateTaskRequest{
		Id:       t2ID,
		Name:     "task2",
		ParentId: &t1ID,
	}))
	if err == nil {
		t.Fatal("expected FailedPrecondition for goal nesting on re-parent, got nil")
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

