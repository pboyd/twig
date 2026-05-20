package handler

import (
	"context"
	"errors"
	"fmt"
	"strings"
	"unicode/utf8"

	"connectrpc.com/connect"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgtype"
	"google.golang.org/protobuf/types/known/timestamppb"

	taskv1 "github.com/pboyd/todo/services/todo/gen/task/v1"
	"github.com/pboyd/todo/services/todo/internal/auth"
	"github.com/pboyd/todo/services/todo/internal/db"
)

type Task struct {
	Queries *db.Queries
}

func dbTaskToProto(t db.Task) *taskv1.Task {
	pt := &taskv1.Task{
		Id:          t.ID,
		Name:        t.Name,
		Description: t.Description,
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
				fmt.Errorf("cannot add a subtask under task %d: parent is complete", *req.Msg.ParentId))
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
	return connect.NewResponse(&taskv1.GetTaskResponse{Task: dbTaskToProto(row)}), nil
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
	tasks := make([]*taskv1.Task, len(rows))
	for i, r := range rows {
		tasks[i] = dbTaskToProto(r)
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
				fmt.Errorf("cannot move task under task %d: parent is complete", newParentID))
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
			fmt.Errorf("incomplete descendants: %v", ids))
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
