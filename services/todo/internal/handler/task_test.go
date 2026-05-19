package handler_test

import (
	"context"
	"os"
	"strings"
	"testing"

	"connectrpc.com/connect"
	"github.com/golang-migrate/migrate/v4"
	_ "github.com/golang-migrate/migrate/v4/database/postgres"
	_ "github.com/golang-migrate/migrate/v4/source/file"
	"github.com/jackc/pgx/v5/pgxpool"

	taskv1 "github.com/pboyd/todo/services/todo/gen/task/v1"
	"github.com/pboyd/todo/services/todo/internal/auth"
	"github.com/pboyd/todo/services/todo/internal/db"
	"github.com/pboyd/todo/services/todo/internal/handler"
)

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
	user, err := queries.CreateUser(context.Background(), db.CreateUserParams{
		Username:     "testuser_" + t.Name(),
		PasswordHash: hash,
	})
	if err != nil {
		t.Fatalf("CreateUser: %v", err)
	}
	userID := user.ID

	t.Cleanup(func() {
		_, _ = pool.Exec(context.Background(), "DELETE FROM users WHERE id = $1", userID)
	})
	return &handler.Task{Queries: queries}, userID
}

// ctxWithUser returns a context carrying the given user_id.
func ctxWithUser(userID int64) context.Context {
	return auth.WithUserID(context.Background(), userID)
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
