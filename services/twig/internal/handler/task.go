package handler

import (
	"context"
	"errors"
	"fmt"
	"strconv"
	"strings"
	"unicode/utf8"

	"connectrpc.com/connect"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgtype"
	"github.com/jackc/pgx/v5/pgxpool"
	"google.golang.org/protobuf/types/known/timestamppb"

	taskv1 "github.com/pboyd/twig/api/gen/task/v1"
	"github.com/pboyd/twig/services/twig/internal/auth"
	"github.com/pboyd/twig/services/twig/internal/db"
)

type Task struct {
	Queries *db.Queries
	Pool    *pgxpool.Pool
}

func dbTaskToProto(t db.Task) *taskv1.Task {
	pt := &taskv1.Task{
		Id:          t.ID,
		Name:        t.Name,
		Description: t.Description,
		Estimate:    int32(t.Estimate),
		Position:    int64(t.Position),
	}
	if t.Due.Valid {
		pt.Due = timestamppb.New(t.Due.Time)
	}
	if t.ParentID.Valid {
		v := t.ParentID.Int64
		pt.ParentId = &v
	}
	if t.CompletedAt.Valid {
		pt.CompletedAt = timestamppb.New(t.CompletedAt.Time)
	}
	if t.SnoozeUntil.Valid {
		pt.SnoozeUntil = timestamppb.New(t.SnoozeUntil.Time)
	}
	if t.GoalID.Valid {
		v := t.GoalID.Int64
		pt.GoalId = &v
	}
	return pt
}

func validateName(name string) (string, error) {
	trimmed := strings.TrimSpace(name)
	if trimmed == "" {
		return "", connect.NewError(connect.CodeInvalidArgument, errors.New("name is required"))
	}
	if utf8.RuneCountInString(trimmed) > 255 {
		return "", connect.NewError(connect.CodeInvalidArgument, errors.New("name must be 255 characters or fewer"))
	}
	return trimmed, nil
}

func (t *Task) parentChainContains(ctx context.Context, userID, startID, targetID int64) (bool, error) {
	current := startID
	for {
		if current == targetID {
			return true, nil
		}
		task, err := t.Queries.GetTask(ctx, db.GetTaskParams{ID: current, UserID: userID})
		if errors.Is(err, pgx.ErrNoRows) {
			return false, nil
		}
		if err != nil {
			return false, err
		}
		if !task.ParentID.Valid {
			return false, nil
		}
		current = task.ParentID.Int64
	}
}

func (t *Task) CreateTask(
	ctx context.Context,
	req *connect.Request[taskv1.CreateTaskRequest],
) (*connect.Response[taskv1.CreateTaskResponse], error) {
	userID := auth.UserID(ctx)

	name, err := validateName(req.Msg.Name)
	if err != nil {
		return nil, err
	}

	params := db.CreateTaskParams{
		Name:        name,
		Description: req.Msg.Description,
		UserID:      userID,
	}
	if req.Msg.Due != nil {
		params.Due = pgtype.Timestamptz{Time: req.Msg.Due.AsTime(), Valid: true}
	}
	if req.Msg.SnoozeUntil != nil {
		params.SnoozeUntil = pgtype.Timestamptz{Time: req.Msg.SnoozeUntil.AsTime(), Valid: true}
	}
	if req.Msg.ParentId != nil {
		exists, err := t.Queries.TaskExists(ctx, db.TaskExistsParams{ID: *req.Msg.ParentId, UserID: userID})
		if err != nil {
			return nil, connect.NewError(connect.CodeInternal, err)
		}
		if !exists {
			return nil, connect.NewError(connect.CodeInvalidArgument, errors.New("parent task not found"))
		}
		parentCompletion, err := t.Queries.GetParentCompletion(ctx, db.GetParentCompletionParams{ID: *req.Msg.ParentId, UserID: userID})
		if err != nil {
			return nil, connect.NewError(connect.CodeInternal, err)
		}
		if parentCompletion.Valid {
			return nil, connect.NewError(connect.CodeFailedPrecondition,
				fmt.Errorf("Task %d is already crossed off — no new sub-tasks for a finished job.", *req.Msg.ParentId))
		}
		params.ParentID = pgtype.Int8{Int64: *req.Msg.ParentId, Valid: true}
	}

	row, err := t.Queries.CreateTask(ctx, params)
	if err != nil {
		return nil, connect.NewError(connect.CodeInternal, err)
	}
	return connect.NewResponse(&taskv1.CreateTaskResponse{Task: dbTaskToProto(row)}), nil
}

func (t *Task) GetTask(
	ctx context.Context,
	req *connect.Request[taskv1.GetTaskRequest],
) (*connect.Response[taskv1.GetTaskResponse], error) {
	userID := auth.UserID(ctx)
	row, err := t.Queries.GetTask(ctx, db.GetTaskParams{ID: req.Msg.Id, UserID: userID})
	if errors.Is(err, pgx.ErrNoRows) {
		return nil, connect.NewError(connect.CodeNotFound, errors.New("task not found"))
	}
	if err != nil {
		return nil, connect.NewError(connect.CodeInternal, err)
	}

	count, err := t.Queries.CountCompletedPomodorosForTask(ctx, db.CountCompletedPomodorosForTaskParams{
		TaskID: req.Msg.Id,
		UserID: userID,
	})
	if err != nil {
		return nil, connect.NewError(connect.CodeInternal, err)
	}

	poms, err := t.Queries.ListPomodorosForTask(ctx, db.ListPomodorosForTaskParams{
		TaskID: req.Msg.Id,
		UserID: userID,
	})
	if err != nil {
		return nil, connect.NewError(connect.CodeInternal, err)
	}

	protoPoms := make([]*taskv1.Pomodoro, len(poms))
	for i, p := range poms {
		protoPoms[i] = dbPomodoroToProto(p)
	}

	return connect.NewResponse(&taskv1.GetTaskResponse{
		Task:                   dbTaskToProto(row),
		CompletedPomodoroCount: count,
		Pomodoros:              protoPoms,
	}), nil
}

func (t *Task) SetEstimate(
	ctx context.Context,
	req *connect.Request[taskv1.SetEstimateRequest],
) (*connect.Response[taskv1.SetEstimateResponse], error) {
	userID := auth.UserID(ctx)

	if req.Msg.Estimate < 0 || req.Msg.Estimate > 10 {
		return nil, connect.NewError(connect.CodeInvalidArgument,
			errors.New("estimate must be between 0 and 10; tasks larger than 10 pomodoros must be broken down further"))
	}

	row, err := t.Queries.SetTaskEstimate(ctx, db.SetTaskEstimateParams{
		ID:       req.Msg.TaskId,
		Estimate: int16(req.Msg.Estimate),
		UserID:   userID,
	})
	if errors.Is(err, pgx.ErrNoRows) {
		return nil, connect.NewError(connect.CodeNotFound, errors.New("task not found"))
	}
	if err != nil {
		return nil, connect.NewError(connect.CodeInternal, err)
	}
	return connect.NewResponse(&taskv1.SetEstimateResponse{Task: dbTaskToProto(row)}), nil
}

func (t *Task) ListTasks(
	ctx context.Context,
	req *connect.Request[taskv1.ListTasksRequest],
) (*connect.Response[taskv1.ListTasksResponse], error) {
	userID := auth.UserID(ctx)
	rows, err := t.Queries.ListTasks(ctx, userID)
	if err != nil {
		return nil, connect.NewError(connect.CodeInternal, err)
	}
	counts, err := t.Queries.CountCompletedPomodorosByTask(ctx, userID)
	if err != nil {
		return nil, connect.NewError(connect.CodeInternal, err)
	}
	countMap := make(map[int64]int64, len(counts))
	for _, c := range counts {
		countMap[c.TaskID] = c.Count
	}
	tasks := make([]*taskv1.Task, len(rows))
	for i, r := range rows {
		pt := dbTaskToProto(r)
		pt.CompletedPomodoroCount = int32(countMap[r.ID])
		tasks[i] = pt
	}
	return connect.NewResponse(&taskv1.ListTasksResponse{Tasks: tasks}), nil
}

func (t *Task) UpdateTask(
	ctx context.Context,
	req *connect.Request[taskv1.UpdateTaskRequest],
) (*connect.Response[taskv1.UpdateTaskResponse], error) {
	userID := auth.UserID(ctx)

	name, err := validateName(req.Msg.Name)
	if err != nil {
		return nil, err
	}

	params := db.UpdateTaskParams{
		ID:          req.Msg.Id,
		Name:        name,
		Description: req.Msg.Description,
		UserID:      userID,
	}
	if req.Msg.Due != nil {
		params.Due = pgtype.Timestamptz{Time: req.Msg.Due.AsTime(), Valid: true}
	}
	if req.Msg.SnoozeUntil != nil {
		params.SnoozeUntil = pgtype.Timestamptz{Time: req.Msg.SnoozeUntil.AsTime(), Valid: true}
	}
	if req.Msg.ParentId != nil {
		newParentID := *req.Msg.ParentId
		exists, err := t.Queries.TaskExists(ctx, db.TaskExistsParams{ID: newParentID, UserID: userID})
		if err != nil {
			return nil, connect.NewError(connect.CodeInternal, err)
		}
		if !exists {
			return nil, connect.NewError(connect.CodeInvalidArgument, errors.New("parent task not found"))
		}
		cycle, err := t.parentChainContains(ctx, userID, newParentID, req.Msg.Id)
		if err != nil {
			return nil, connect.NewError(connect.CodeInternal, err)
		}
		if cycle {
			return nil, connect.NewError(connect.CodeInvalidArgument, errors.New("parent_id would create a cycle"))
		}
		parentCompletion, err := t.Queries.GetParentCompletion(ctx, db.GetParentCompletionParams{ID: newParentID, UserID: userID})
		if err != nil {
			return nil, connect.NewError(connect.CodeInternal, err)
		}
		if parentCompletion.Valid {
			return nil, connect.NewError(connect.CodeFailedPrecondition,
				fmt.Errorf("Task %d is already crossed off — nothing moves under a finished job.", newParentID))
		}

		// Check no-nested-goal-associations invariant.
		// If the new parent or any of its ancestors has a goal association,
		// the task being moved (or its descendants) cannot also have a goal.
		newParentTask, err := t.Queries.GetTask(ctx, db.GetTaskParams{ID: newParentID, UserID: userID})
		if err != nil {
			return nil, connect.NewError(connect.CodeInternal, err)
		}
		newParentOrAncestorHasGoal := newParentTask.GoalID.Valid
		if !newParentOrAncestorHasGoal {
			ancestorHasGoal, err := t.Queries.AncestorHasGoal(ctx, db.AncestorHasGoalParams{ID: newParentID, UserID: userID})
			if err != nil {
				return nil, connect.NewError(connect.CodeInternal, err)
			}
			newParentOrAncestorHasGoal = ancestorHasGoal
		}
		if newParentOrAncestorHasGoal {
			movingTask, err := t.Queries.GetTask(ctx, db.GetTaskParams{ID: req.Msg.Id, UserID: userID})
			if err != nil {
				return nil, connect.NewError(connect.CodeInternal, err)
			}
			taskOrDescHasGoal := movingTask.GoalID.Valid
			if !taskOrDescHasGoal {
				descHasGoal, err := t.Queries.DescendantHasGoal(ctx, db.DescendantHasGoalParams{
					ParentID: pgtype.Int8{Int64: req.Msg.Id, Valid: true},
					UserID:   userID,
				})
				if err != nil {
					return nil, connect.NewError(connect.CodeInternal, err)
				}
				taskOrDescHasGoal = descHasGoal
			}
			if taskOrDescHasGoal {
				return nil, connect.NewError(connect.CodeFailedPrecondition,
					errors.New("moving this task would nest goal associations — clear the goal link first"))
			}
		}

		params.ParentID = pgtype.Int8{Int64: newParentID, Valid: true}
	}

	row, err := t.Queries.UpdateTask(ctx, params)
	if errors.Is(err, pgx.ErrNoRows) {
		return nil, connect.NewError(connect.CodeNotFound, errors.New("task not found"))
	}
	if err != nil {
		return nil, connect.NewError(connect.CodeInternal, err)
	}

	// When parent_id changed, place the task at the end of the destination group.
	if req.Msg.ParentId != nil {
		maxPos, err := t.Queries.GetMaxSiblingPosition(ctx, db.GetMaxSiblingPositionParams{
			UserID:   userID,
			ParentID: params.ParentID,
		})
		if err != nil {
			return nil, connect.NewError(connect.CodeInternal, err)
		}
		if err := t.Queries.UpdateTaskPosition(ctx, db.UpdateTaskPositionParams{
			ID:       row.ID,
			UserID:   userID,
			Position: maxPos + 1,
		}); err != nil {
			return nil, connect.NewError(connect.CodeInternal, err)
		}
		row.Position = maxPos + 1
	}

	return connect.NewResponse(&taskv1.UpdateTaskResponse{Task: dbTaskToProto(row)}), nil
}

func (t *Task) CompleteTask(
	ctx context.Context,
	req *connect.Request[taskv1.CompleteTaskRequest],
) (*connect.Response[taskv1.CompleteTaskResponse], error) {
	userID := auth.UserID(ctx)

	// Verify task exists before checking descendants.
	_, err := t.Queries.GetTask(ctx, db.GetTaskParams{ID: req.Msg.Id, UserID: userID})
	if errors.Is(err, pgx.ErrNoRows) {
		return nil, connect.NewError(connect.CodeNotFound, errors.New("task not found"))
	}
	if err != nil {
		return nil, connect.NewError(connect.CodeInternal, err)
	}

	// Block completion when any descendant is incomplete.
	hasIncomplete, err := t.Queries.HasIncompleteDescendants(ctx, db.HasIncompleteDescendantsParams{
		ParentID: pgtype.Int8{Int64: req.Msg.Id, Valid: true},
		UserID:   userID,
	})
	if err != nil {
		return nil, connect.NewError(connect.CodeInternal, err)
	}
	if hasIncomplete {
		ids, err := t.Queries.ListIncompleteDescendantIds(ctx, db.ListIncompleteDescendantIdsParams{
			ParentID: pgtype.Int8{Int64: req.Msg.Id, Valid: true},
			UserID:   userID,
		})
		if err != nil {
			return nil, connect.NewError(connect.CodeInternal, err)
		}
		return nil, connect.NewError(connect.CodeFailedPrecondition,
			fmt.Errorf("Whoa there — sub-tasks %s still need doing first.", joinIDs(ids)))
	}

	row, err := t.Queries.CompleteTask(ctx, db.CompleteTaskParams{ID: req.Msg.Id, UserID: userID})
	if errors.Is(err, pgx.ErrNoRows) {
		return nil, connect.NewError(connect.CodeNotFound, errors.New("task not found"))
	}
	if err != nil {
		return nil, connect.NewError(connect.CodeInternal, err)
	}
	return connect.NewResponse(&taskv1.CompleteTaskResponse{Task: dbTaskToProto(row)}), nil
}

// joinIDs renders a slice of int64 IDs as a comma-separated string (e.g. "3, 4, 5").
func joinIDs(ids []int64) string {
	parts := make([]string, len(ids))
	for i, id := range ids {
		parts[i] = strconv.FormatInt(id, 10)
	}
	return strings.Join(parts, ", ")
}

func (t *Task) UncompleteTask(
	ctx context.Context,
	req *connect.Request[taskv1.UncompleteTaskRequest],
) (*connect.Response[taskv1.UncompleteTaskResponse], error) {
	userID := auth.UserID(ctx)

	task, err := t.Queries.GetTask(ctx, db.GetTaskParams{ID: req.Msg.Id, UserID: userID})
	if errors.Is(err, pgx.ErrNoRows) {
		return nil, connect.NewError(connect.CodeNotFound, errors.New("task not found"))
	}
	if err != nil {
		return nil, connect.NewError(connect.CodeInternal, err)
	}

	if task.ParentID.Valid {
		parentCompletion, err := t.Queries.GetParentCompletion(ctx, db.GetParentCompletionParams{ID: task.ParentID.Int64, UserID: userID})
		if err != nil {
			return nil, connect.NewError(connect.CodeInternal, err)
		}
		if parentCompletion.Valid {
			return nil, connect.NewError(connect.CodeFailedPrecondition,
				fmt.Errorf("Its parent (task %d) is still done. Reopen that one first.", task.ParentID.Int64))
		}
	}

	row, err := t.Queries.UncompleteTask(ctx, db.UncompleteTaskParams{ID: req.Msg.Id, UserID: userID})
	if errors.Is(err, pgx.ErrNoRows) {
		return nil, connect.NewError(connect.CodeNotFound, errors.New("task not found"))
	}
	if err != nil {
		return nil, connect.NewError(connect.CodeInternal, err)
	}
	return connect.NewResponse(&taskv1.UncompleteTaskResponse{Task: dbTaskToProto(row)}), nil
}

func (t *Task) DeleteTask(
	ctx context.Context,
	req *connect.Request[taskv1.DeleteTaskRequest],
) (*connect.Response[taskv1.DeleteTaskResponse], error) {
	userID := auth.UserID(ctx)
	_, err := t.Queries.DeleteTask(ctx, db.DeleteTaskParams{ID: req.Msg.Id, UserID: userID})
	if errors.Is(err, pgx.ErrNoRows) {
		return nil, connect.NewError(connect.CodeNotFound, errors.New("task not found"))
	}
	if err != nil {
		return nil, connect.NewError(connect.CodeInternal, err)
	}
	return connect.NewResponse(&taskv1.DeleteTaskResponse{}), nil
}

func (t *Task) ReorderTask(
	ctx context.Context,
	req *connect.Request[taskv1.ReorderTaskRequest],
) (*connect.Response[taskv1.ReorderTaskResponse], error) {
	userID := auth.UserID(ctx)

	taskID := req.Msg.TaskId
	var anchorID int64
	var insertBefore bool

	switch a := req.Msg.Anchor.(type) {
	case *taskv1.ReorderTaskRequest_BeforeTaskId:
		anchorID = a.BeforeTaskId
		insertBefore = true
	case *taskv1.ReorderTaskRequest_AfterTaskId:
		anchorID = a.AfterTaskId
		insertBefore = false
	case nil:
		return nil, connect.NewError(connect.CodeInvalidArgument, errors.New("exactly one of before_task_id or after_task_id must be set"))
	}

	if anchorID == taskID {
		return nil, connect.NewError(connect.CodeInvalidArgument, errors.New("anchor must be a different task than the one being moved"))
	}

	if t.Pool == nil {
		return nil, connect.NewError(connect.CodeInternal, errors.New("pool not configured"))
	}

	tx, err := t.Pool.Begin(ctx)
	if err != nil {
		return nil, connect.NewError(connect.CodeInternal, err)
	}
	defer tx.Rollback(ctx)

	q := t.Queries.WithTx(tx)

	task, err := q.GetTask(ctx, db.GetTaskParams{ID: taskID, UserID: userID})
	if errors.Is(err, pgx.ErrNoRows) {
		return nil, connect.NewError(connect.CodeNotFound, errors.New("task not found"))
	}
	if err != nil {
		return nil, connect.NewError(connect.CodeInternal, err)
	}

	anchor, err := q.GetTask(ctx, db.GetTaskParams{ID: anchorID, UserID: userID})
	if errors.Is(err, pgx.ErrNoRows) {
		return nil, connect.NewError(connect.CodeNotFound, errors.New("anchor task not found"))
	}
	if err != nil {
		return nil, connect.NewError(connect.CodeInternal, err)
	}

	// Anchor must be a sibling: same user, same parent_id.
	if task.ParentID != anchor.ParentID {
		return nil, connect.NewError(connect.CodeInvalidArgument, errors.New("anchor must be in the same sibling group as the task"))
	}

	// Read the full sibling group in order (FOR UPDATE acquired by ListSiblingGroup).
	siblings, err := q.ListSiblingGroup(ctx, db.ListSiblingGroupParams{
		UserID:   userID,
		ParentID: task.ParentID,
	})
	if err != nil {
		return nil, connect.NewError(connect.CodeInternal, err)
	}

	// Remove task from its current position in the group.
	ordered := make([]db.Task, 0, len(siblings))
	for _, s := range siblings {
		if s.ID != taskID {
			ordered = append(ordered, s)
		}
	}

	// Find anchor position in the remaining slice and insert.
	insertIdx := len(ordered) // default: end
	for i, s := range ordered {
		if s.ID == anchorID {
			if insertBefore {
				insertIdx = i
			} else {
				insertIdx = i + 1
			}
			break
		}
	}

	// Insert task at insertIdx.
	ordered = append(ordered, db.Task{}) // grow
	copy(ordered[insertIdx+1:], ordered[insertIdx:])
	ordered[insertIdx] = task

	// Renumber 0..n-1 and persist.
	for i, s := range ordered {
		if err := q.UpdateTaskPosition(ctx, db.UpdateTaskPositionParams{
			ID:       s.ID,
			UserID:   userID,
			Position: int32(i),
		}); err != nil {
			return nil, connect.NewError(connect.CodeInternal, err)
		}
		ordered[i].Position = int32(i)
	}

	if err := tx.Commit(ctx); err != nil {
		return nil, connect.NewError(connect.CodeInternal, err)
	}

	proto := make([]*taskv1.Task, len(ordered))
	for i, s := range ordered {
		proto[i] = dbTaskToProto(s)
	}

	return connect.NewResponse(&taskv1.ReorderTaskResponse{Siblings: proto}), nil
}

func (t *Task) SetTaskGoal(
	ctx context.Context,
	req *connect.Request[taskv1.SetTaskGoalRequest],
) (*connect.Response[taskv1.SetTaskGoalResponse], error) {
	userID := auth.UserID(ctx)

	// Verify task exists.
	_, err := t.Queries.GetTask(ctx, db.GetTaskParams{ID: req.Msg.TaskId, UserID: userID})
	if errors.Is(err, pgx.ErrNoRows) {
		return nil, connect.NewError(connect.CodeNotFound, errors.New("task not found"))
	}
	if err != nil {
		return nil, connect.NewError(connect.CodeInternal, err)
	}

	// Clearing: goal_id absent.
	if req.Msg.GoalId == nil {
		row, err := t.Queries.ClearTaskGoal(ctx, db.ClearTaskGoalParams{ID: req.Msg.TaskId, UserID: userID})
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, connect.NewError(connect.CodeNotFound, errors.New("task not found"))
		}
		if err != nil {
			return nil, connect.NewError(connect.CodeInternal, err)
		}
		return connect.NewResponse(&taskv1.SetTaskGoalResponse{Task: dbTaskToProto(row)}), nil
	}

	// Setting: verify goal exists for this user.
	goalID := *req.Msg.GoalId
	_, err = t.Queries.GetGoal(ctx, db.GetGoalParams{ID: goalID, UserID: userID})
	if errors.Is(err, pgx.ErrNoRows) {
		return nil, connect.NewError(connect.CodeNotFound, errors.New("goal not found"))
	}
	if err != nil {
		return nil, connect.NewError(connect.CodeInternal, err)
	}

	// Check ancestor constraint.
	ancestorHasGoal, err := t.Queries.AncestorHasGoal(ctx, db.AncestorHasGoalParams{ID: req.Msg.TaskId, UserID: userID})
	if err != nil {
		return nil, connect.NewError(connect.CodeInternal, err)
	}
	if ancestorHasGoal {
		return nil, connect.NewError(connect.CodeFailedPrecondition, errors.New("an ancestor task already has a goal assigned"))
	}

	// Check descendant constraint.
	descendantHasGoal, err := t.Queries.DescendantHasGoal(ctx, db.DescendantHasGoalParams{
		ParentID: pgtype.Int8{Int64: req.Msg.TaskId, Valid: true},
		UserID:   userID,
	})
	if err != nil {
		return nil, connect.NewError(connect.CodeInternal, err)
	}
	if descendantHasGoal {
		return nil, connect.NewError(connect.CodeFailedPrecondition, errors.New("a descendant task already has a goal assigned"))
	}

	row, err := t.Queries.SetTaskGoal(ctx, db.SetTaskGoalParams{
		ID:     req.Msg.TaskId,
		UserID: userID,
		GoalID: pgtype.Int8{Int64: goalID, Valid: true},
	})
	if errors.Is(err, pgx.ErrNoRows) {
		return nil, connect.NewError(connect.CodeNotFound, errors.New("task not found"))
	}
	if err != nil {
		return nil, connect.NewError(connect.CodeInternal, err)
	}
	return connect.NewResponse(&taskv1.SetTaskGoalResponse{Task: dbTaskToProto(row)}), nil
}
