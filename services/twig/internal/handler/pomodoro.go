package handler

import (
	"context"
	"errors"

	"connectrpc.com/connect"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"
	"github.com/jackc/pgx/v5/pgtype"
	"google.golang.org/protobuf/types/known/timestamppb"

	taskv1 "github.com/pboyd/twig/api/gen/task/v1"
	"github.com/pboyd/twig/services/twig/internal/auth"
	"github.com/pboyd/twig/services/twig/internal/db"
)

func dbPomodoroToProto(p db.Pomodoro) *taskv1.Pomodoro {
	pp := &taskv1.Pomodoro{
		Id:       p.ID,
		TaskId:   p.TaskID,
		Complete: p.Complete,
		StartAt:  timestamppb.New(p.StartAt.Time),
	}
	if p.EndAt.Valid {
		pp.EndAt = timestamppb.New(p.EndAt.Time)
	}
	return pp
}

func pgErrIsUniqueViolation(err error, constraint string) bool {
	var pgErr *pgconn.PgError
	if errors.As(err, &pgErr) {
		return pgErr.Code == "23505" && pgErr.ConstraintName == constraint
	}
	return false
}

func (t *Task) GetActivePomodoro(
	ctx context.Context,
	req *connect.Request[taskv1.GetActivePomodoroRequest],
) (*connect.Response[taskv1.GetActivePomodoroResponse], error) {
	userID := auth.UserID(ctx)
	p, err := t.Queries.GetActivePomodoro(ctx, userID)
	if errors.Is(err, pgx.ErrNoRows) {
		return connect.NewResponse(&taskv1.GetActivePomodoroResponse{}), nil
	}
	if err != nil {
		return nil, connect.NewError(connect.CodeInternal, err)
	}
	return connect.NewResponse(&taskv1.GetActivePomodoroResponse{
		Pomodoro: dbPomodoroToProto(p),
	}), nil
}

func (t *Task) StartPomodoro(
	ctx context.Context,
	req *connect.Request[taskv1.StartPomodoroRequest],
) (*connect.Response[taskv1.StartPomodoroResponse], error) {
	userID := auth.UserID(ctx)

	_, err := t.Queries.GetTask(ctx, db.GetTaskParams{ID: req.Msg.TaskId, UserID: userID})
	if errors.Is(err, pgx.ErrNoRows) {
		return nil, connect.NewError(connect.CodeNotFound, errors.New("task not found"))
	}
	if err != nil {
		return nil, connect.NewError(connect.CodeInternal, err)
	}

	p, err := t.Queries.StartPomodoro(ctx, db.StartPomodoroParams{
		UserID: userID,
		TaskID: req.Msg.TaskId,
	})
	if err != nil {
		if pgErrIsUniqueViolation(err, "pomodoros_one_active_per_user") {
			active, aerr := t.Queries.GetActivePomodoro(ctx, userID)
			if aerr != nil {
				return nil, connect.NewError(connect.CodeInternal, aerr)
			}
			cerr := connect.NewError(connect.CodeAlreadyExists, errors.New("you already have an active pomodoro"))
			detail, _ := connect.NewErrorDetail(&taskv1.StartPomodoroRequest{TaskId: active.TaskID})
			cerr.AddDetail(detail)
			return nil, cerr
		}
		return nil, connect.NewError(connect.CodeInternal, err)
	}
	return connect.NewResponse(&taskv1.StartPomodoroResponse{Pomodoro: dbPomodoroToProto(p)}), nil
}

func (t *Task) CancelPomodoro(
	ctx context.Context,
	req *connect.Request[taskv1.CancelPomodoroRequest],
) (*connect.Response[taskv1.CancelPomodoroResponse], error) {
	userID := auth.UserID(ctx)
	p, err := t.Queries.CancelActivePomodoro(ctx, userID)
	if errors.Is(err, pgx.ErrNoRows) {
		return nil, connect.NewError(connect.CodeFailedPrecondition, errors.New("There's no pomodoro ticking. Start one first."))
	}
	if err != nil {
		return nil, connect.NewError(connect.CodeInternal, err)
	}
	return connect.NewResponse(&taskv1.CancelPomodoroResponse{Pomodoro: dbPomodoroToProto(p)}), nil
}

func (t *Task) CompletePomodoro(
	ctx context.Context,
	req *connect.Request[taskv1.CompletePomodoroRequest],
) (*connect.Response[taskv1.CompletePomodoroResponse], error) {
	userID := auth.UserID(ctx)
	p, err := t.Queries.CompleteActivePomodoro(ctx, userID)
	if errors.Is(err, pgx.ErrNoRows) {
		return nil, connect.NewError(connect.CodeFailedPrecondition, errors.New("There's no pomodoro ticking. Start one first."))
	}
	if err != nil {
		return nil, connect.NewError(connect.CodeInternal, err)
	}
	return connect.NewResponse(&taskv1.CompletePomodoroResponse{Pomodoro: dbPomodoroToProto(p)}), nil
}

func (t *Task) CountCompletedPomodoros(
	ctx context.Context,
	req *connect.Request[taskv1.CountCompletedPomodorosRequest],
) (*connect.Response[taskv1.CountCompletedPomodorosResponse], error) {
	if req.Msg.Start == nil {
		return nil, connect.NewError(connect.CodeInvalidArgument, errors.New("start is required"))
	}
	if req.Msg.End == nil {
		return nil, connect.NewError(connect.CodeInvalidArgument, errors.New("end is required"))
	}
	if !req.Msg.End.AsTime().After(req.Msg.Start.AsTime()) {
		return nil, connect.NewError(connect.CodeInvalidArgument, errors.New("end must be after start"))
	}

	userID := auth.UserID(ctx)
	count, err := t.Queries.CountCompletedPomodorosInRange(ctx, db.CountCompletedPomodorosInRangeParams{
		UserID:  userID,
		EndAt:   pgtype.Timestamptz{Time: req.Msg.Start.AsTime(), Valid: true},
		EndAt_2: pgtype.Timestamptz{Time: req.Msg.End.AsTime(), Valid: true},
	})
	if err != nil {
		return nil, connect.NewError(connect.CodeInternal, err)
	}
	return connect.NewResponse(&taskv1.CountCompletedPomodorosResponse{Count: count}), nil
}
