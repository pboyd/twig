package handler

import (
	"context"
	"errors"
	"strings"
	"unicode/utf8"

	"connectrpc.com/connect"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgtype"
	"google.golang.org/protobuf/types/known/timestamppb"

	taskv1 "github.com/pboyd/todo/services/todo/gen/task/v1"
	"github.com/pboyd/todo/services/todo/internal/db"
)

type Task struct {
	Queries *db.Queries
}

// dbTaskToProto converts a db.Task to its proto representation.
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
	return pt
}

// validateName trims and validates a task name, returning the trimmed name.
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

// parentChainContains walks the ancestor chain starting at startID and
// reports whether targetID appears in it (including startID itself).
// Used to detect cycles before re-parenting.
func (t *Task) parentChainContains(ctx context.Context, startID, targetID int64) (bool, error) {
	current := startID
	for {
		if current == targetID {
			return true, nil
		}
		task, err := t.Queries.GetTask(ctx, current)
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
	name, err := validateName(req.Msg.Name)
	if err != nil {
		return nil, err
	}

	params := db.CreateTaskParams{
		Name:        name,
		Description: req.Msg.Description,
	}
	if req.Msg.Due != nil {
		params.Due = pgtype.Timestamptz{Time: req.Msg.Due.AsTime(), Valid: true}
	}
	if req.Msg.ParentId != nil {
		exists, err := t.Queries.TaskExists(ctx, *req.Msg.ParentId)
		if err != nil {
			return nil, connect.NewError(connect.CodeInternal, err)
		}
		if !exists {
			return nil, connect.NewError(connect.CodeInvalidArgument, errors.New("parent task not found"))
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
	row, err := t.Queries.GetTask(ctx, req.Msg.Id)
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
	rows, err := t.Queries.ListTasks(ctx)
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
	name, err := validateName(req.Msg.Name)
	if err != nil {
		return nil, err
	}

	params := db.UpdateTaskParams{
		ID:          req.Msg.Id,
		Name:        name,
		Description: req.Msg.Description,
	}
	if req.Msg.Due != nil {
		params.Due = pgtype.Timestamptz{Time: req.Msg.Due.AsTime(), Valid: true}
	}
	if req.Msg.ParentId != nil {
		newParentID := *req.Msg.ParentId
		exists, err := t.Queries.TaskExists(ctx, newParentID)
		if err != nil {
			return nil, connect.NewError(connect.CodeInternal, err)
		}
		if !exists {
			return nil, connect.NewError(connect.CodeInvalidArgument, errors.New("parent task not found"))
		}
		cycle, err := t.parentChainContains(ctx, newParentID, req.Msg.Id)
		if err != nil {
			return nil, connect.NewError(connect.CodeInternal, err)
		}
		if cycle {
			return nil, connect.NewError(connect.CodeInvalidArgument, errors.New("parent_id would create a cycle"))
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

func (t *Task) DeleteTask(
	ctx context.Context,
	req *connect.Request[taskv1.DeleteTaskRequest],
) (*connect.Response[taskv1.DeleteTaskResponse], error) {
	_, err := t.Queries.DeleteTask(ctx, req.Msg.Id)
	if errors.Is(err, pgx.ErrNoRows) {
		return nil, connect.NewError(connect.CodeNotFound, errors.New("task not found"))
	}
	if err != nil {
		return nil, connect.NewError(connect.CodeInternal, err)
	}
	return connect.NewResponse(&taskv1.DeleteTaskResponse{}), nil
}
