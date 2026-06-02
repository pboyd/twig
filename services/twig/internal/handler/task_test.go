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

	taskv1 "github.com/pboyd/twig/services/twig/gen/task/v1"
	"github.com/pboyd/twig/services/twig/internal/auth"
	"github.com/pboyd/twig/services/twig/internal/db"
	"github.com/pboyd/twig/services/twig/internal/handler"
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
	return &handler.Task{Queries: queries}, userID
}

// ctxWithUser returns a context carrying the given user_id.
func ctxWithUser(userID int64) context.Context {
	return auth.WithUserID(context.Background(), userID)
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
