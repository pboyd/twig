package handler_test

import (
	"context"
	"fmt"
	"os"
	"strings"
	"sync/atomic"
	"testing"

	"connectrpc.com/connect"
	"github.com/golang-migrate/migrate/v4"
	"github.com/jackc/pgx/v5/pgxpool"
	goalv1 "github.com/pboyd/twig/api/gen/goal/v1"
	taskv1 "github.com/pboyd/twig/api/gen/task/v1"
	"github.com/pboyd/twig/services/twig/internal/auth"
	"github.com/pboyd/twig/services/twig/internal/db"
	"github.com/pboyd/twig/services/twig/internal/handler"
)

var testGoalUserCounter int64

// newGoalTestHandler creates a Goal handler backed by a real PostgreSQL database.
// Tests that call this are integration tests and require DATABASE_URL.
func newGoalTestHandler(t *testing.T) (*handler.Goal, int64) {
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

	hash, err := auth.HashPassword("testpass")
	if err != nil {
		t.Fatalf("HashPassword: %v", err)
	}
	n := atomic.AddInt64(&testGoalUserCounter, 1)
	user, err := queries.CreateUser(context.Background(), db.CreateUserParams{
		Username:     fmt.Sprintf("goaluser_%d_%s", n, t.Name()),
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
	return &handler.Goal{Queries: queries, Pool: pool}, userID
}

// ---- Integration tests: T007 ----

func TestCreateGoal_Defaults(t *testing.T) {
	gh, userID := newGoalTestHandler(t)
	ctx := ctxWithUser(userID)

	t.Run("first goal is incubating at position 0", func(t *testing.T) {
		resp, err := gh.CreateGoal(ctx, connect.NewRequest(&goalv1.CreateGoalRequest{
			Name: "Build a spaceship",
		}))
		if err != nil {
			t.Fatalf("CreateGoal: %v", err)
		}
		g := resp.Msg.Goal
		if g.Id <= 0 {
			t.Errorf("expected positive id, got %d", g.Id)
		}
		if g.Name != "Build a spaceship" {
			t.Errorf("name = %q, want %q", g.Name, "Build a spaceship")
		}
		if g.State != goalv1.GoalState_GOAL_STATE_INCUBATING {
			t.Errorf("state = %v, want INCUBATING", g.State)
		}
		if g.Position != 0 {
			t.Errorf("position = %d, want 0", g.Position)
		}
	})

	t.Run("second goal appends to incubating at position 1", func(t *testing.T) {
		resp, err := gh.CreateGoal(ctx, connect.NewRequest(&goalv1.CreateGoalRequest{
			Name: "Learn to juggle",
		}))
		if err != nil {
			t.Fatalf("CreateGoal second: %v", err)
		}
		g := resp.Msg.Goal
		if g.State != goalv1.GoalState_GOAL_STATE_INCUBATING {
			t.Errorf("state = %v, want INCUBATING", g.State)
		}
		if g.Position != 1 {
			t.Errorf("position = %d, want 1", g.Position)
		}
	})
}

func TestCreateGoal_NameValidation(t *testing.T) {
	gh := &handler.Goal{Queries: nil}
	cases := []struct {
		name    string
		input   string
		wantErr connect.Code
	}{
		{"empty string", "", connect.CodeInvalidArgument},
		{"whitespace only", "   ", connect.CodeInvalidArgument},
		{"too long", strings.Repeat("a", 256), connect.CodeInvalidArgument},
		{"valid name", "Learn piano", 0},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			if tc.wantErr == 0 {
				defer func() {
					if r := recover(); r != nil {
						// Panic from nil Queries is expected for valid-name cases.
					}
				}()
				req := connect.NewRequest(&goalv1.CreateGoalRequest{Name: tc.input})
				_, err := gh.CreateGoal(context.Background(), req)
				if err != nil {
					ce, ok := err.(*connect.Error)
					if ok && ce.Code() == connect.CodeInvalidArgument {
						t.Errorf("unexpected InvalidArgument for valid name %q: %v", tc.input, err)
					}
				}
				return
			}
			req := connect.NewRequest(&goalv1.CreateGoalRequest{Name: tc.input})
			_, err := gh.CreateGoal(context.Background(), req)
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

func TestSetGoalState_Transitions(t *testing.T) {
	gh, userID := newGoalTestHandler(t)
	ctx := ctxWithUser(userID)

	// Create two committed goals to test bottom-of-group placement.
	r1, err := gh.CreateGoal(ctx, connect.NewRequest(&goalv1.CreateGoalRequest{Name: "Goal A"}))
	if err != nil {
		t.Fatalf("CreateGoal A: %v", err)
	}
	goalA := r1.Msg.Goal.Id

	r2, err := gh.CreateGoal(ctx, connect.NewRequest(&goalv1.CreateGoalRequest{Name: "Goal B"}))
	if err != nil {
		t.Fatalf("CreateGoal B: %v", err)
	}
	goalB := r2.Msg.Goal.Id

	// Move A to committed.
	resp, err := gh.SetGoalState(ctx, connect.NewRequest(&goalv1.SetGoalStateRequest{
		Id:    goalA,
		State: goalv1.GoalState_GOAL_STATE_COMMITTED,
	}))
	if err != nil {
		t.Fatalf("SetGoalState incubating→committed: %v", err)
	}
	if resp.Msg.Goal.State != goalv1.GoalState_GOAL_STATE_COMMITTED {
		t.Errorf("state = %v, want COMMITTED", resp.Msg.Goal.State)
	}
	// Position in the committed group: first committed → position 0.
	if resp.Msg.Goal.Position != 0 {
		t.Errorf("position after transition to committed = %d, want 0", resp.Msg.Goal.Position)
	}

	// Move B to committed — should be at position 1 (bottom of committed group).
	resp2, err := gh.SetGoalState(ctx, connect.NewRequest(&goalv1.SetGoalStateRequest{
		Id:    goalB,
		State: goalv1.GoalState_GOAL_STATE_COMMITTED,
	}))
	if err != nil {
		t.Fatalf("SetGoalState B incubating→committed: %v", err)
	}
	if resp2.Msg.Goal.State != goalv1.GoalState_GOAL_STATE_COMMITTED {
		t.Errorf("state = %v, want COMMITTED", resp2.Msg.Goal.State)
	}
	if resp2.Msg.Goal.Position != 1 {
		t.Errorf("position after B→committed = %d, want 1", resp2.Msg.Goal.Position)
	}

	// committed→completed
	resp, err = gh.SetGoalState(ctx, connect.NewRequest(&goalv1.SetGoalStateRequest{
		Id:    goalA,
		State: goalv1.GoalState_GOAL_STATE_COMPLETED,
	}))
	if err != nil {
		t.Fatalf("SetGoalState committed→completed: %v", err)
	}
	if resp.Msg.Goal.State != goalv1.GoalState_GOAL_STATE_COMPLETED {
		t.Errorf("state = %v, want COMPLETED", resp.Msg.Goal.State)
	}

	// completed→archived
	resp, err = gh.SetGoalState(ctx, connect.NewRequest(&goalv1.SetGoalStateRequest{
		Id:    goalA,
		State: goalv1.GoalState_GOAL_STATE_ARCHIVED,
	}))
	if err != nil {
		t.Fatalf("SetGoalState completed→archived: %v", err)
	}
	if resp.Msg.Goal.State != goalv1.GoalState_GOAL_STATE_ARCHIVED {
		t.Errorf("state = %v, want ARCHIVED", resp.Msg.Goal.State)
	}

	// archived→incubating
	resp, err = gh.SetGoalState(ctx, connect.NewRequest(&goalv1.SetGoalStateRequest{
		Id:    goalA,
		State: goalv1.GoalState_GOAL_STATE_INCUBATING,
	}))
	if err != nil {
		t.Fatalf("SetGoalState archived→incubating: %v", err)
	}
	if resp.Msg.Goal.State != goalv1.GoalState_GOAL_STATE_INCUBATING {
		t.Errorf("state = %v, want INCUBATING", resp.Msg.Goal.State)
	}
}

func TestSetGoalState_SameState_NoOp(t *testing.T) {
	gh, userID := newGoalTestHandler(t)
	ctx := ctxWithUser(userID)

	r, err := gh.CreateGoal(ctx, connect.NewRequest(&goalv1.CreateGoalRequest{Name: "Steady"}))
	if err != nil {
		t.Fatalf("CreateGoal: %v", err)
	}
	goalID := r.Msg.Goal.Id

	// Setting the same state is a no-op — must succeed and return the current goal.
	resp, err := gh.SetGoalState(ctx, connect.NewRequest(&goalv1.SetGoalStateRequest{
		Id:    goalID,
		State: goalv1.GoalState_GOAL_STATE_INCUBATING,
	}))
	if err != nil {
		t.Fatalf("SetGoalState same state: %v", err)
	}
	if resp.Msg.Goal.State != goalv1.GoalState_GOAL_STATE_INCUBATING {
		t.Errorf("state = %v, want INCUBATING", resp.Msg.Goal.State)
	}
	if resp.Msg.Goal.Id != goalID {
		t.Errorf("id changed: %d != %d", resp.Msg.Goal.Id, goalID)
	}
}

func TestReorderGoal_WithinGroup(t *testing.T) {
	gh, userID := newGoalTestHandler(t)
	ctx := ctxWithUser(userID)

	ra, _ := gh.CreateGoal(ctx, connect.NewRequest(&goalv1.CreateGoalRequest{Name: "Alpha"}))
	rb, _ := gh.CreateGoal(ctx, connect.NewRequest(&goalv1.CreateGoalRequest{Name: "Beta"}))
	rc, _ := gh.CreateGoal(ctx, connect.NewRequest(&goalv1.CreateGoalRequest{Name: "Gamma"}))

	aID, bID, cID := ra.Msg.Goal.Id, rb.Msg.Goal.Id, rc.Msg.Goal.Id

	// Move Gamma before Alpha → expected order: Gamma, Alpha, Beta.
	resp, err := gh.ReorderGoal(ctx, connect.NewRequest(&goalv1.ReorderGoalRequest{
		GoalId: cID,
		Anchor: &goalv1.ReorderGoalRequest_BeforeGoalId{BeforeGoalId: aID},
	}))
	if err != nil {
		t.Fatalf("ReorderGoal: %v", err)
	}
	if len(resp.Msg.Goals) != 3 {
		t.Fatalf("expected 3 goals in response, got %d", len(resp.Msg.Goals))
	}
	wantOrder := []int64{cID, aID, bID}
	for i, g := range resp.Msg.Goals {
		if g.Id != wantOrder[i] {
			t.Errorf("goals[%d].Id = %d, want %d", i, g.Id, wantOrder[i])
		}
		if g.Position != int64(i) {
			t.Errorf("goals[%d].Position = %d, want %d", i, g.Position, i)
		}
	}
}

func TestReorderGoal_CrossGroupRejected(t *testing.T) {
	gh, userID := newGoalTestHandler(t)
	ctx := ctxWithUser(userID)

	r1, _ := gh.CreateGoal(ctx, connect.NewRequest(&goalv1.CreateGoalRequest{Name: "Incubating goal"}))
	r2, _ := gh.CreateGoal(ctx, connect.NewRequest(&goalv1.CreateGoalRequest{Name: "Also incubating"}))

	// Move r2 to committed.
	_, err := gh.SetGoalState(ctx, connect.NewRequest(&goalv1.SetGoalStateRequest{
		Id:    r2.Msg.Goal.Id,
		State: goalv1.GoalState_GOAL_STATE_COMMITTED,
	}))
	if err != nil {
		t.Fatalf("SetGoalState: %v", err)
	}

	// Attempt to reorder r1 (incubating) before r2 (committed) — must fail.
	_, err = gh.ReorderGoal(ctx, connect.NewRequest(&goalv1.ReorderGoalRequest{
		GoalId: r1.Msg.Goal.Id,
		Anchor: &goalv1.ReorderGoalRequest_BeforeGoalId{BeforeGoalId: r2.Msg.Goal.Id},
	}))
	if err == nil {
		t.Fatal("expected cross-group rejection, got nil")
	}
	ce, ok := err.(*connect.Error)
	if !ok || ce.Code() != connect.CodeInvalidArgument {
		t.Errorf("expected CodeInvalidArgument, got %v", err)
	}
}

func TestDeleteGoal_ClearsTaskGoalId(t *testing.T) {
	gh, userID := newGoalTestHandler(t)
	ctx := ctxWithUser(userID)

	// Use the same queries/pool to build a task handler for the same user.
	th := &handler.Task{Queries: gh.Queries, Pool: gh.Pool}

	// Create a goal and a task.
	goalResp, err := gh.CreateGoal(ctx, connect.NewRequest(&goalv1.CreateGoalRequest{Name: "Delete me"}))
	if err != nil {
		t.Fatalf("CreateGoal: %v", err)
	}
	goalID := goalResp.Msg.Goal.Id

	taskResp, err := th.CreateTask(ctx, connect.NewRequest(&taskv1.CreateTaskRequest{Name: "Linked task"}))
	if err != nil {
		t.Fatalf("CreateTask: %v", err)
	}
	taskID := taskResp.Msg.Task.Id

	// Associate task with goal.
	_, err = th.SetTaskGoal(ctx, connect.NewRequest(&taskv1.SetTaskGoalRequest{
		TaskId: taskID,
		GoalId: &goalID,
	}))
	if err != nil {
		t.Fatalf("SetTaskGoal: %v", err)
	}

	// Verify goal_id is set.
	gr, err := th.GetTask(ctx, connect.NewRequest(&taskv1.GetTaskRequest{Id: taskID}))
	if err != nil {
		t.Fatalf("GetTask before delete: %v", err)
	}
	if gr.Msg.Task.GoalId == nil || *gr.Msg.Task.GoalId != goalID {
		t.Errorf("expected goal_id = %d, got %v", goalID, gr.Msg.Task.GoalId)
	}

	// Delete the goal.
	_, err = gh.DeleteGoal(ctx, connect.NewRequest(&goalv1.DeleteGoalRequest{Id: goalID}))
	if err != nil {
		t.Fatalf("DeleteGoal: %v", err)
	}

	// Task must still exist with goal_id = NULL.
	gr2, err := th.GetTask(ctx, connect.NewRequest(&taskv1.GetTaskRequest{Id: taskID}))
	if err != nil {
		t.Fatalf("GetTask after delete: %v", err)
	}
	if gr2.Msg.Task.GoalId != nil {
		t.Errorf("expected goal_id = nil after goal delete, got %v", gr2.Msg.Task.GoalId)
	}
}

func TestGoalCrossUserIsolation(t *testing.T) {
	ghA, userA := newGoalTestHandler(t)
	ghB, userB := newGoalTestHandler(t)

	ctxA := ctxWithUser(userA)
	ctxB := ctxWithUser(userB)

	// User A creates a goal.
	resp, err := ghA.CreateGoal(ctxA, connect.NewRequest(&goalv1.CreateGoalRequest{Name: "User A goal"}))
	if err != nil {
		t.Fatalf("CreateGoal as user A: %v", err)
	}
	goalID := resp.Msg.Goal.Id

	t.Run("user B cannot get user A's goal", func(t *testing.T) {
		_, err := ghB.GetGoal(ctxB, connect.NewRequest(&goalv1.GetGoalRequest{Id: goalID}))
		if err == nil {
			t.Fatal("expected not found, got nil")
		}
		ce, ok := err.(*connect.Error)
		if !ok || ce.Code() != connect.CodeNotFound {
			t.Errorf("expected CodeNotFound, got %v", err)
		}
	})

	t.Run("user B list does not include user A's goals", func(t *testing.T) {
		listResp, err := ghB.ListGoals(ctxB, connect.NewRequest(&goalv1.ListGoalsRequest{}))
		if err != nil {
			t.Fatalf("ListGoals as user B: %v", err)
		}
		for _, g := range listResp.Msg.Goals {
			if g.Id == goalID {
				t.Error("user B's list should not include user A's goal")
			}
		}
	})

	t.Run("user B cannot update user A's goal", func(t *testing.T) {
		_, err := ghB.UpdateGoal(ctxB, connect.NewRequest(&goalv1.UpdateGoalRequest{
			Id:   goalID,
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

	t.Run("user B cannot delete user A's goal", func(t *testing.T) {
		_, err := ghB.DeleteGoal(ctxB, connect.NewRequest(&goalv1.DeleteGoalRequest{Id: goalID}))
		if err == nil {
			t.Fatal("expected error, got nil")
		}
		ce, ok := err.(*connect.Error)
		if !ok || ce.Code() != connect.CodeNotFound {
			t.Errorf("expected CodeNotFound, got %v", err)
		}
	})

	t.Run("user B cannot change state of user A's goal", func(t *testing.T) {
		_, err := ghB.SetGoalState(ctxB, connect.NewRequest(&goalv1.SetGoalStateRequest{
			Id:    goalID,
			State: goalv1.GoalState_GOAL_STATE_COMMITTED,
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

// ---- Integration tests: T005 (status updates) ----

func TestGoalStatusUpdate_ValidateBody(t *testing.T) {
	gh := &handler.Goal{Queries: nil}
	cases := []struct {
		name  string
		body  string
		isErr bool
	}{
		{"empty string", "", true},
		{"whitespace only", "   \n\t", true},
		{"valid body", "Making progress on the roadmap.", false},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			if tc.isErr {
				req := connect.NewRequest(&goalv1.AddGoalStatusUpdateRequest{Body: tc.body})
				_, err := gh.AddGoalStatusUpdate(context.Background(), req)
				if err == nil {
					t.Fatal("expected error, got nil")
				}
				ce, ok := err.(*connect.Error)
				if !ok || ce.Code() != connect.CodeInvalidArgument {
					t.Errorf("expected CodeInvalidArgument, got %v", err)
				}
			} else {
				// Valid body passes validation; nil Queries will panic — that's OK.
				defer func() { recover() }()
				req := connect.NewRequest(&goalv1.AddGoalStatusUpdateRequest{GoalId: 1, Body: tc.body})
				_, _ = gh.AddGoalStatusUpdate(context.Background(), req)
			}
		})
	}
}

func TestGoalStatusUpdate_CRUD(t *testing.T) {
	gh, userID := newGoalTestHandler(t)
	ctx := ctxWithUser(userID)

	// Create a goal to attach updates to.
	gresp, err := gh.CreateGoal(ctx, connect.NewRequest(&goalv1.CreateGoalRequest{Name: "Status test goal"}))
	if err != nil {
		t.Fatalf("CreateGoal: %v", err)
	}
	goalID := gresp.Msg.Goal.Id

	t.Run("add status update", func(t *testing.T) {
		resp, err := gh.AddGoalStatusUpdate(ctx, connect.NewRequest(&goalv1.AddGoalStatusUpdateRequest{
			GoalId: goalID,
			Body:   "First update — things are moving!",
		}))
		if err != nil {
			t.Fatalf("AddGoalStatusUpdate: %v", err)
		}
		su := resp.Msg.Update
		if su.Id <= 0 {
			t.Errorf("expected positive id, got %d", su.Id)
		}
		if su.GoalId != goalID {
			t.Errorf("goal_id = %d, want %d", su.GoalId, goalID)
		}
		if su.Body != "First update — things are moving!" {
			t.Errorf("body = %q", su.Body)
		}
		if su.CreatedAt == nil {
			t.Error("created_at should be set")
		}
	})

	t.Run("list returns newest first", func(t *testing.T) {
		// Add a second update.
		_, err := gh.AddGoalStatusUpdate(ctx, connect.NewRequest(&goalv1.AddGoalStatusUpdateRequest{
			GoalId: goalID,
			Body:   "Second update — almost there.",
		}))
		if err != nil {
			t.Fatalf("AddGoalStatusUpdate second: %v", err)
		}

		resp, err := gh.ListGoalStatusUpdates(ctx, connect.NewRequest(&goalv1.ListGoalStatusUpdatesRequest{
			GoalId: goalID,
		}))
		if err != nil {
			t.Fatalf("ListGoalStatusUpdates: %v", err)
		}
		updates := resp.Msg.Updates
		if len(updates) < 2 {
			t.Fatalf("want at least 2 updates, got %d", len(updates))
		}
		// Newest first: second update should be first.
		if !strings.Contains(updates[0].Body, "Second") {
			t.Errorf("expected newest first, got %q", updates[0].Body)
		}
	})

	t.Run("update body preserves created_at", func(t *testing.T) {
		addResp, err := gh.AddGoalStatusUpdate(ctx, connect.NewRequest(&goalv1.AddGoalStatusUpdateRequest{
			GoalId: goalID,
			Body:   "Original body.",
		}))
		if err != nil {
			t.Fatalf("AddGoalStatusUpdate: %v", err)
		}
		su := addResp.Msg.Update
		originalCreatedAt := su.CreatedAt.AsTime()

		updResp, err := gh.UpdateGoalStatusUpdate(ctx, connect.NewRequest(&goalv1.UpdateGoalStatusUpdateRequest{
			Id:   su.Id,
			Body: "Revised body.",
		}))
		if err != nil {
			t.Fatalf("UpdateGoalStatusUpdate: %v", err)
		}
		updated := updResp.Msg.Update
		if updated.Body != "Revised body." {
			t.Errorf("body = %q, want %q", updated.Body, "Revised body.")
		}
		if !updated.CreatedAt.AsTime().Equal(originalCreatedAt) {
			t.Errorf("created_at changed: was %v, now %v", originalCreatedAt, updated.CreatedAt.AsTime())
		}
	})

	t.Run("delete removes update", func(t *testing.T) {
		addResp, err := gh.AddGoalStatusUpdate(ctx, connect.NewRequest(&goalv1.AddGoalStatusUpdateRequest{
			GoalId: goalID,
			Body:   "To be deleted.",
		}))
		if err != nil {
			t.Fatalf("AddGoalStatusUpdate: %v", err)
		}
		suID := addResp.Msg.Update.Id

		_, err = gh.DeleteGoalStatusUpdate(ctx, connect.NewRequest(&goalv1.DeleteGoalStatusUpdateRequest{
			Id: suID,
		}))
		if err != nil {
			t.Fatalf("DeleteGoalStatusUpdate: %v", err)
		}

		// Verify it's gone.
		listResp, err := gh.ListGoalStatusUpdates(ctx, connect.NewRequest(&goalv1.ListGoalStatusUpdatesRequest{
			GoalId: goalID,
		}))
		if err != nil {
			t.Fatalf("ListGoalStatusUpdates after delete: %v", err)
		}
		for _, su := range listResp.Msg.Updates {
			if su.Id == suID {
				t.Errorf("deleted update %d still present", suID)
			}
		}
	})

	t.Run("ListGoals embeds latest_status_update", func(t *testing.T) {
		// Add a known latest update.
		_, err := gh.AddGoalStatusUpdate(ctx, connect.NewRequest(&goalv1.AddGoalStatusUpdateRequest{
			GoalId: goalID,
			Body:   "The very latest.",
		}))
		if err != nil {
			t.Fatalf("AddGoalStatusUpdate: %v", err)
		}

		listResp, err := gh.ListGoals(ctx, connect.NewRequest(&goalv1.ListGoalsRequest{}))
		if err != nil {
			t.Fatalf("ListGoals: %v", err)
		}
		var found *goalv1.Goal
		for _, g := range listResp.Msg.Goals {
			if g.Id == goalID {
				found = g
				break
			}
		}
		if found == nil {
			t.Fatal("goal not found in ListGoals response")
		}
		if found.LatestStatusUpdate == nil {
			t.Fatal("latest_status_update should be set")
		}
		if found.LatestStatusUpdate.Body != "The very latest." {
			t.Errorf("latest body = %q, want %q", found.LatestStatusUpdate.Body, "The very latest.")
		}
	})
}

func TestGoalStatusUpdate_CrossUserIsolation(t *testing.T) {
	ghA, userAID := newGoalTestHandler(t)
	ctxA := ctxWithUser(userAID)

	ghB, userBID := newGoalTestHandler(t)
	ctxB := ctxWithUser(userBID)

	// User A creates a goal.
	gresp, err := ghA.CreateGoal(ctxA, connect.NewRequest(&goalv1.CreateGoalRequest{Name: "User A's goal"}))
	if err != nil {
		t.Fatalf("CreateGoal: %v", err)
	}
	goalID := gresp.Msg.Goal.Id

	// User A adds an update.
	addResp, err := ghA.AddGoalStatusUpdate(ctxA, connect.NewRequest(&goalv1.AddGoalStatusUpdateRequest{
		GoalId: goalID,
		Body:   "User A's secret plans.",
	}))
	if err != nil {
		t.Fatalf("AddGoalStatusUpdate: %v", err)
	}
	suID := addResp.Msg.Update.Id

	_ = userBID // referenced via ctxB

	t.Run("user B cannot list updates on user A's goal", func(t *testing.T) {
		_, err := ghB.ListGoalStatusUpdates(ctxB, connect.NewRequest(&goalv1.ListGoalStatusUpdatesRequest{
			GoalId: goalID,
		}))
		if err == nil {
			t.Fatal("expected error, got nil")
		}
		ce, ok := err.(*connect.Error)
		if !ok || ce.Code() != connect.CodeNotFound {
			t.Errorf("expected CodeNotFound, got %v", err)
		}
	})

	t.Run("user B cannot add update to user A's goal", func(t *testing.T) {
		_, err := ghB.AddGoalStatusUpdate(ctxB, connect.NewRequest(&goalv1.AddGoalStatusUpdateRequest{
			GoalId: goalID,
			Body:   "User B sneaking in.",
		}))
		if err == nil {
			t.Fatal("expected error, got nil")
		}
		ce, ok := err.(*connect.Error)
		if !ok || ce.Code() != connect.CodeNotFound {
			t.Errorf("expected CodeNotFound, got %v", err)
		}
	})

	t.Run("user B cannot update user A's status update", func(t *testing.T) {
		_, err := ghB.UpdateGoalStatusUpdate(ctxB, connect.NewRequest(&goalv1.UpdateGoalStatusUpdateRequest{
			Id:   suID,
			Body: "Tampered.",
		}))
		if err == nil {
			t.Fatal("expected error, got nil")
		}
		ce, ok := err.(*connect.Error)
		if !ok || ce.Code() != connect.CodeNotFound {
			t.Errorf("expected CodeNotFound, got %v", err)
		}
	})

	t.Run("user B cannot delete user A's status update", func(t *testing.T) {
		_, err := ghB.DeleteGoalStatusUpdate(ctxB, connect.NewRequest(&goalv1.DeleteGoalStatusUpdateRequest{
			Id: suID,
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

func TestSetGoalState_Hold_RoundTrip(t *testing.T) {
	gh, userID := newGoalTestHandler(t)
	ctx := ctxWithUser(userID)

	// Create an incubating goal.
	createResp, err := gh.CreateGoal(ctx, connect.NewRequest(&goalv1.CreateGoalRequest{Name: "Hold me"}))
	if err != nil {
		t.Fatalf("CreateGoal: %v", err)
	}
	goalID := createResp.Msg.Goal.Id
	if createResp.Msg.Goal.State != goalv1.GoalState_GOAL_STATE_INCUBATING {
		t.Fatalf("initial state = %v, want INCUBATING", createResp.Msg.Goal.State)
	}

	// incubating → hold
	resp, err := gh.SetGoalState(ctx, connect.NewRequest(&goalv1.SetGoalStateRequest{
		Id:    goalID,
		State: goalv1.GoalState_GOAL_STATE_HOLD,
	}))
	if err != nil {
		t.Fatalf("SetGoalState incubating→hold: %v", err)
	}
	if resp.Msg.Goal.State != goalv1.GoalState_GOAL_STATE_HOLD {
		t.Errorf("state = %v, want HOLD", resp.Msg.Goal.State)
	}

	// hold → committed
	resp, err = gh.SetGoalState(ctx, connect.NewRequest(&goalv1.SetGoalStateRequest{
		Id:    goalID,
		State: goalv1.GoalState_GOAL_STATE_COMMITTED,
	}))
	if err != nil {
		t.Fatalf("SetGoalState hold→committed: %v", err)
	}
	if resp.Msg.Goal.State != goalv1.GoalState_GOAL_STATE_COMMITTED {
		t.Errorf("state = %v, want COMMITTED", resp.Msg.Goal.State)
	}

	// committed → hold (re-enter hold from committed)
	resp, err = gh.SetGoalState(ctx, connect.NewRequest(&goalv1.SetGoalStateRequest{
		Id:    goalID,
		State: goalv1.GoalState_GOAL_STATE_HOLD,
	}))
	if err != nil {
		t.Fatalf("SetGoalState committed→hold: %v", err)
	}
	if resp.Msg.Goal.State != goalv1.GoalState_GOAL_STATE_HOLD {
		t.Errorf("state = %v, want HOLD", resp.Msg.Goal.State)
	}
}

func TestSetGoalState_Hold_TwoGoals(t *testing.T) {
	gh, userID := newGoalTestHandler(t)
	ctx := ctxWithUser(userID)

	ra, _ := gh.CreateGoal(ctx, connect.NewRequest(&goalv1.CreateGoalRequest{Name: "Alpha"}))
	rb, _ := gh.CreateGoal(ctx, connect.NewRequest(&goalv1.CreateGoalRequest{Name: "Beta"}))

	// Move both to hold — second one should be ranked below the first.
	_, err := gh.SetGoalState(ctx, connect.NewRequest(&goalv1.SetGoalStateRequest{
		Id:    ra.Msg.Goal.Id,
		State: goalv1.GoalState_GOAL_STATE_HOLD,
	}))
	if err != nil {
		t.Fatalf("SetGoalState Alpha→hold: %v", err)
	}

	_, err = gh.SetGoalState(ctx, connect.NewRequest(&goalv1.SetGoalStateRequest{
		Id:    rb.Msg.Goal.Id,
		State: goalv1.GoalState_GOAL_STATE_HOLD,
	}))
	if err != nil {
		t.Fatalf("SetGoalState Beta→hold: %v", err)
	}

	// List goals and verify hold group ordering.
	listResp, err := gh.ListGoals(ctx, connect.NewRequest(&goalv1.ListGoalsRequest{}))
	if err != nil {
		t.Fatalf("ListGoals: %v", err)
	}

	// Find the two hold goals and check their relative positions.
	var holdGoals []*goalv1.Goal
	for _, g := range listResp.Msg.Goals {
		if g.State == goalv1.GoalState_GOAL_STATE_HOLD {
			holdGoals = append(holdGoals, g)
		}
	}
	if len(holdGoals) != 2 {
		t.Fatalf("expected 2 hold goals, got %d", len(holdGoals))
	}
	if holdGoals[0].Position >= holdGoals[1].Position {
		t.Errorf("hold group order wrong: first pos=%d, second pos=%d", holdGoals[0].Position, holdGoals[1].Position)
	}
}

func TestSetGoalState_Unspecified_Rejected(t *testing.T) {
	gh, userID := newGoalTestHandler(t)
	ctx := ctxWithUser(userID)

	resp, _ := gh.CreateGoal(ctx, connect.NewRequest(&goalv1.CreateGoalRequest{Name: "Test"}))
	goalID := resp.Msg.Goal.Id

	_, err := gh.SetGoalState(ctx, connect.NewRequest(&goalv1.SetGoalStateRequest{
		Id:    goalID,
		State: goalv1.GoalState_GOAL_STATE_UNSPECIFIED,
	}))
	if err == nil {
		t.Fatal("SetGoalState(UNSPECIFIED): expected error, got nil")
	}
}
