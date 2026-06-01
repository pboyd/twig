package handler

import (
	"context"
	"errors"
	"fmt"
	"strings"
	"time"
	"unicode/utf8"

	"connectrpc.com/connect"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgtype"
	"github.com/jackc/pgx/v5/pgxpool"

	planv1 "github.com/pboyd/twig/services/twig/gen/plan/v1"
	"github.com/pboyd/twig/services/twig/internal/auth"
	"github.com/pboyd/twig/services/twig/internal/db"
	"github.com/pboyd/twig/services/twig/internal/plan"
)

type Plan struct {
	Queries *db.Queries
	Pool    *pgxpool.Pool
}

func dbPlanEntryToProto(e db.PlanEntry) *planv1.PlanEntry {
	pe := &planv1.PlanEntry{
		Day:            e.Day.Time.Format("2006-01-02"),
		Id:             e.ID,
		DurationMinute: int32(e.DurationMinute),
	}
	if e.StartMinute.Valid {
		v := int32(e.StartMinute.Int16)
		pe.StartMinute = &v
	}
	if e.TaskID.Valid {
		pe.TaskId = e.TaskID.Int64
	}
	if e.Name.Valid {
		pe.Name = e.Name.String
	}
	return pe
}

func parseDay(s string) (pgtype.Date, error) {
	t, err := time.Parse("2006-01-02", s)
	if err != nil {
		return pgtype.Date{}, connect.NewError(connect.CodeInvalidArgument,
			fmt.Errorf("day must be YYYY-MM-DD, got %q", s))
	}
	return pgtype.Date{Time: t, Valid: true}, nil
}

func validatePlanName(name string) (string, error) {
	trimmed := strings.TrimSpace(name)
	if trimmed == "" {
		return "", connect.NewError(connect.CodeInvalidArgument, errors.New("name is required"))
	}
	if utf8.RuneCountInString(trimmed) > 255 {
		return "", connect.NewError(connect.CodeInvalidArgument, errors.New("name must be 255 characters or fewer"))
	}
	return trimmed, nil
}

func validateStartMinute(m int32) error {
	if m < 0 || m >= 1440 {
		return connect.NewError(connect.CodeInvalidArgument,
			fmt.Errorf("start_minute %d out of range [0, 1440)", m))
	}
	return nil
}

func (p *Plan) beginSerializableTx(ctx context.Context) (pgx.Tx, *db.Queries, error) {
	tx, err := p.Pool.BeginTx(ctx, pgx.TxOptions{IsoLevel: pgx.Serializable})
	if err != nil {
		return nil, nil, connect.NewError(connect.CodeInternal, err)
	}
	return tx, p.Queries.WithTx(tx), nil
}

// errDuplicateUntimed is returned when a second untimed entry would be created for the same (day, task).
func errDuplicateUntimed() error {
	return connect.NewError(connect.CodeFailedPrecondition,
		errors.New("That one's already parked here without a time — it can only wait in one spot."))
}

// checkOverlap returns (conflicting id, true) if any timed entry in rows overlaps [start, start+dur).
// Untimed entries (null start_minute) are skipped — they occupy no grid slot.
func checkOverlap(rows []db.PlanEntry, start, dur int32, excludeID int32) (int32, bool) {
	for _, r := range rows {
		if excludeID != 0 && r.ID == excludeID {
			continue
		}
		if !r.StartMinute.Valid {
			continue
		}
		if plan.Overlap(int(start), int(dur), int(r.StartMinute.Int16), int(r.DurationMinute)) {
			return r.ID, true
		}
	}
	return 0, false
}

func (p *Plan) ListPlanEntries(
	ctx context.Context,
	req *connect.Request[planv1.ListPlanEntriesRequest],
) (*connect.Response[planv1.ListPlanEntriesResponse], error) {
	userID := auth.UserID(ctx)

	day, err := parseDay(req.Msg.Day)
	if err != nil {
		return nil, err
	}

	rows, err := p.Queries.ListPlanEntriesForDay(ctx, db.ListPlanEntriesForDayParams{
		UserID: userID,
		Day:    day,
	})
	if err != nil {
		return nil, connect.NewError(connect.CodeInternal, err)
	}

	// Collect task IDs that need name fallback.
	taskNames := map[int64]string{}
	for _, r := range rows {
		if r.TaskID.Valid && !r.Name.Valid {
			taskNames[r.TaskID.Int64] = ""
		}
	}
	for taskID := range taskNames {
		task, err := p.Queries.GetTask(ctx, db.GetTaskParams{ID: taskID, UserID: userID})
		if err == nil {
			taskNames[taskID] = task.Name
		}
	}

	entries := make([]*planv1.PlanEntry, len(rows))
	for i, r := range rows {
		pe := &planv1.PlanEntry{
			Day:            r.Day.Time.Format("2006-01-02"),
			Id:             r.ID,
			DurationMinute: int32(r.DurationMinute),
			Completed:      r.Completed.Bool,
		}
		if r.StartMinute.Valid {
			v := int32(r.StartMinute.Int16)
			pe.StartMinute = &v
		}
		if r.TaskID.Valid {
			pe.TaskId = r.TaskID.Int64
		}
		if r.Name.Valid {
			pe.Name = r.Name.String
		}
		if r.TaskID.Valid && !r.Name.Valid {
			pe.Name = taskNames[r.TaskID.Int64]
		}
		entries[i] = pe
	}
	return connect.NewResponse(&planv1.ListPlanEntriesResponse{Entries: entries}), nil
}

func (p *Plan) AddPlanEvent(
	ctx context.Context,
	req *connect.Request[planv1.AddPlanEventRequest],
) (*connect.Response[planv1.AddPlanEventResponse], error) {
	userID := auth.UserID(ctx)

	day, err := parseDay(req.Msg.Day)
	if err != nil {
		return nil, err
	}
	if err := validateStartMinute(req.Msg.StartMinute); err != nil {
		return nil, err
	}
	if req.Msg.DurationMinute < 0 {
		return nil, connect.NewError(connect.CodeInvalidArgument, errors.New("duration_minute must be >= 0"))
	}
	dur := req.Msg.DurationMinute
	if dur == 0 {
		dur = 30
	}
	if int32(req.Msg.StartMinute)+dur > 1440 {
		return nil, connect.NewError(connect.CodeInvalidArgument,
			errors.New("start_minute + duration_minute exceeds 1440 (midnight crossing not allowed)"))
	}
	name, err := validatePlanName(req.Msg.Name)
	if err != nil {
		return nil, err
	}

	tx, txq, err := p.beginSerializableTx(ctx)
	if err != nil {
		return nil, err
	}
	defer tx.Rollback(ctx)

	locked, err := txq.LockPlanEntriesForDay(ctx, db.LockPlanEntriesForDayParams{UserID: userID, Day: day})
	if err != nil {
		return nil, connect.NewError(connect.CodeInternal, err)
	}
	if conflictID, overlaps := checkOverlap(locked, req.Msg.StartMinute, dur, 0); overlaps {
		return nil, connect.NewError(connect.CodeFailedPrecondition,
			fmt.Errorf("No room there — that bumps into entry %d.", conflictID))
	}

	nextID, err := txq.NextPlanEntryId(ctx, db.NextPlanEntryIdParams{UserID: userID, Day: day})
	if err != nil {
		return nil, connect.NewError(connect.CodeInternal, err)
	}

	row, err := txq.InsertPlanEntry(ctx, db.InsertPlanEntryParams{
		UserID:         userID,
		Day:            day,
		ID:             nextID,
		Name:           pgtype.Text{String: name, Valid: true},
		StartMinute:    pgtype.Int2{Int16: int16(req.Msg.StartMinute), Valid: true},
		DurationMinute: int16(dur),
	})
	if err != nil {
		return nil, connect.NewError(connect.CodeInternal, err)
	}

	if err := tx.Commit(ctx); err != nil {
		return nil, connect.NewError(connect.CodeInternal, err)
	}
	return connect.NewResponse(&planv1.AddPlanEventResponse{Entry: dbPlanEntryToProto(row)}), nil
}

func (p *Plan) AddPlanTask(
	ctx context.Context,
	req *connect.Request[planv1.AddPlanTaskRequest],
) (*connect.Response[planv1.AddPlanTaskResponse], error) {
	userID := auth.UserID(ctx)

	day, err := parseDay(req.Msg.Day)
	if err != nil {
		return nil, err
	}

	isTimed := req.Msg.StartMinute != nil

	if isTimed {
		if err := validateStartMinute(req.Msg.GetStartMinute()); err != nil {
			return nil, err
		}
	}
	if req.Msg.DurationMinute < 0 {
		return nil, connect.NewError(connect.CodeInvalidArgument, errors.New("duration_minute must be >= 0"))
	}

	// Verify task exists and compute default duration if needed.
	task, err := p.Queries.GetTask(ctx, db.GetTaskParams{ID: req.Msg.TaskId, UserID: userID})
	if errors.Is(err, pgx.ErrNoRows) {
		return nil, connect.NewError(connect.CodeNotFound, errors.New("task not found"))
	}
	if err != nil {
		return nil, connect.NewError(connect.CodeInternal, err)
	}

	dur := req.Msg.DurationMinute
	if dur == 0 {
		if task.Estimate > 0 {
			completed, err := p.Queries.CountCompletedPomodorosForTask(ctx, db.CountCompletedPomodorosForTaskParams{
				TaskID: req.Msg.TaskId,
				UserID: userID,
			})
			if err != nil {
				return nil, connect.NewError(connect.CodeInternal, err)
			}
			remaining := int32(task.Estimate) - int32(completed)
			if remaining > 0 {
				dur = remaining * 30
			} else {
				dur = 30
			}
		} else {
			dur = 30
		}
	}

	if isTimed && req.Msg.GetStartMinute()+dur > 1440 {
		return nil, connect.NewError(connect.CodeInvalidArgument,
			errors.New("start_minute + duration_minute exceeds 1440 (midnight crossing not allowed)"))
	}

	tx, txq, err := p.beginSerializableTx(ctx)
	if err != nil {
		return nil, err
	}
	defer tx.Rollback(ctx)

	locked, err := txq.LockPlanEntriesForDay(ctx, db.LockPlanEntriesForDayParams{UserID: userID, Day: day})
	if err != nil {
		return nil, connect.NewError(connect.CodeInternal, err)
	}
	if isTimed {
		if conflictID, overlaps := checkOverlap(locked, req.Msg.GetStartMinute(), dur, 0); overlaps {
			return nil, connect.NewError(connect.CodeFailedPrecondition,
				fmt.Errorf("No room there — that bumps into entry %d.", conflictID))
		}
	} else {
		for _, r := range locked {
			if r.TaskID.Valid && r.TaskID.Int64 == req.Msg.TaskId && !r.StartMinute.Valid {
				return nil, errDuplicateUntimed()
			}
		}
	}

	nextID, err := txq.NextPlanEntryId(ctx, db.NextPlanEntryIdParams{UserID: userID, Day: day})
	if err != nil {
		return nil, connect.NewError(connect.CodeInternal, err)
	}

	startParam := pgtype.Int2{Valid: false}
	if isTimed {
		startParam = pgtype.Int2{Int16: int16(req.Msg.GetStartMinute()), Valid: true}
	}

	row, err := txq.InsertPlanEntry(ctx, db.InsertPlanEntryParams{
		UserID:         userID,
		Day:            day,
		ID:             nextID,
		TaskID:         pgtype.Int8{Int64: req.Msg.TaskId, Valid: true},
		StartMinute:    startParam,
		DurationMinute: int16(dur),
	})
	if err != nil {
		return nil, connect.NewError(connect.CodeInternal, err)
	}

	if err := tx.Commit(ctx); err != nil {
		return nil, connect.NewError(connect.CodeInternal, err)
	}
	return connect.NewResponse(&planv1.AddPlanTaskResponse{Entry: dbPlanEntryToProto(row)}), nil
}

func (p *Plan) RemovePlanEntry(
	ctx context.Context,
	req *connect.Request[planv1.RemovePlanEntryRequest],
) (*connect.Response[planv1.RemovePlanEntryResponse], error) {
	userID := auth.UserID(ctx)

	day, err := parseDay(req.Msg.Day)
	if err != nil {
		return nil, err
	}

	_, err = p.Queries.DeletePlanEntry(ctx, db.DeletePlanEntryParams{
		UserID: userID,
		Day:    day,
		ID:     req.Msg.Id,
	})
	if errors.Is(err, pgx.ErrNoRows) {
		return nil, connect.NewError(connect.CodeNotFound, errors.New("entry not found"))
	}
	if err != nil {
		return nil, connect.NewError(connect.CodeInternal, err)
	}
	return connect.NewResponse(&planv1.RemovePlanEntryResponse{}), nil
}

func (p *Plan) RenamePlanEntry(
	ctx context.Context,
	req *connect.Request[planv1.RenamePlanEntryRequest],
) (*connect.Response[planv1.RenamePlanEntryResponse], error) {
	userID := auth.UserID(ctx)

	name, err := validatePlanName(req.Msg.Name)
	if err != nil {
		return nil, err
	}
	day, err := parseDay(req.Msg.Day)
	if err != nil {
		return nil, err
	}

	row, err := p.Queries.UpdatePlanEntryName(ctx, db.UpdatePlanEntryNameParams{
		UserID: userID,
		Day:    day,
		ID:     req.Msg.Id,
		Name:   pgtype.Text{String: name, Valid: true},
	})
	if errors.Is(err, pgx.ErrNoRows) {
		return nil, connect.NewError(connect.CodeNotFound, errors.New("entry not found"))
	}
	if err != nil {
		return nil, connect.NewError(connect.CodeInternal, err)
	}
	return connect.NewResponse(&planv1.RenamePlanEntryResponse{Entry: dbPlanEntryToProto(row)}), nil
}

func (p *Plan) MovePlanEntry(
	ctx context.Context,
	req *connect.Request[planv1.MovePlanEntryRequest],
) (*connect.Response[planv1.MovePlanEntryResponse], error) {
	userID := auth.UserID(ctx)

	day, err := parseDay(req.Msg.Day)
	if err != nil {
		return nil, err
	}
	if req.Msg.DurationMinute < 0 {
		return nil, connect.NewError(connect.CodeInvalidArgument, errors.New("duration_minute must be >= 0"))
	}

	isTimed := req.Msg.StartMinute != nil

	if isTimed {
		if err := validateStartMinute(req.Msg.GetStartMinute()); err != nil {
			return nil, err
		}
	}

	tx, txq, err := p.beginSerializableTx(ctx)
	if err != nil {
		return nil, err
	}
	defer tx.Rollback(ctx)

	existing, err := txq.GetPlanEntry(ctx, db.GetPlanEntryParams{
		UserID: userID,
		Day:    day,
		ID:     req.Msg.Id,
	})
	if errors.Is(err, pgx.ErrNoRows) {
		return nil, connect.NewError(connect.CodeNotFound, errors.New("entry not found"))
	}
	if err != nil {
		return nil, connect.NewError(connect.CodeInternal, err)
	}

	dur := req.Msg.DurationMinute
	if dur == 0 {
		dur = int32(existing.DurationMinute)
	}

	startParam := pgtype.Int2{Valid: false}
	if isTimed {
		if req.Msg.GetStartMinute()+dur > 1440 {
			return nil, connect.NewError(connect.CodeInvalidArgument,
				errors.New("start_minute + duration_minute exceeds 1440 (midnight crossing not allowed)"))
		}

		locked, err := txq.LockPlanEntriesForDay(ctx, db.LockPlanEntriesForDayParams{UserID: userID, Day: day})
		if err != nil {
			return nil, connect.NewError(connect.CodeInternal, err)
		}
		if conflictID, overlaps := checkOverlap(locked, req.Msg.GetStartMinute(), dur, req.Msg.Id); overlaps {
			return nil, connect.NewError(connect.CodeFailedPrecondition,
				fmt.Errorf("No room there — that bumps into entry %d.", conflictID))
		}
		startParam = pgtype.Int2{Int16: int16(req.Msg.GetStartMinute()), Valid: true}
	} else {
		// Clear-start (unschedule): reject if a different untimed entry already exists for this task.
		if existing.TaskID.Valid {
			locked, err := txq.LockPlanEntriesForDay(ctx, db.LockPlanEntriesForDayParams{UserID: userID, Day: day})
			if err != nil {
				return nil, connect.NewError(connect.CodeInternal, err)
			}
			for _, r := range locked {
				if r.ID == req.Msg.Id {
					continue
				}
				if r.TaskID.Valid && r.TaskID.Int64 == existing.TaskID.Int64 && !r.StartMinute.Valid {
					return nil, errDuplicateUntimed()
				}
			}
		}
	}

	row, err := txq.UpdatePlanEntryTime(ctx, db.UpdatePlanEntryTimeParams{
		UserID:         userID,
		Day:            day,
		ID:             req.Msg.Id,
		StartMinute:    startParam,
		DurationMinute: int16(dur),
	})
	if errors.Is(err, pgx.ErrNoRows) {
		return nil, connect.NewError(connect.CodeNotFound, errors.New("entry not found"))
	}
	if err != nil {
		return nil, connect.NewError(connect.CodeInternal, err)
	}

	if err := tx.Commit(ctx); err != nil {
		return nil, connect.NewError(connect.CodeInternal, err)
	}
	return connect.NewResponse(&planv1.MovePlanEntryResponse{Entry: dbPlanEntryToProto(row)}), nil
}

func (p *Plan) ClearPlan(
	ctx context.Context,
	req *connect.Request[planv1.ClearPlanRequest],
) (*connect.Response[planv1.ClearPlanResponse], error) {
	userID := auth.UserID(ctx)

	day, err := parseDay(req.Msg.Day)
	if err != nil {
		return nil, err
	}
	if err := validateStartMinute(req.Msg.StartMinute); err != nil {
		return nil, err
	}

	tx, txq, err := p.beginSerializableTx(ctx)
	if err != nil {
		return nil, err
	}
	defer tx.Rollback(ctx)

	locked, err := txq.LockPlanEntriesForDay(ctx, db.LockPlanEntriesForDayParams{UserID: userID, Day: day})
	if err != nil {
		return nil, connect.NewError(connect.CodeInternal, err)
	}

	var trimmed bool
	cutoff := req.Msg.StartMinute
	for _, r := range locked {
		if !r.TaskID.Valid {
			continue
		}
		if !r.StartMinute.Valid {
			continue // untimed entries are never cleared
		}
		start := int32(r.StartMinute.Int16)
		end := start + int32(r.DurationMinute)
		if start < cutoff && cutoff < end {
			// Straddles: shorten duration so it ends at cutoff.
			_, err := txq.TrimPlanEntryDuration(ctx, db.TrimPlanEntryDurationParams{
				UserID:         userID,
				Day:            day,
				ID:             r.ID,
				DurationMinute: int16(cutoff - start),
			})
			if err != nil {
				return nil, connect.NewError(connect.CodeInternal, err)
			}
			trimmed = true
			break
		}
	}

	deleted, err := txq.DeletePlanEntriesFromMinute(ctx, db.DeletePlanEntriesFromMinuteParams{
		UserID:      userID,
		Day:         day,
		StartMinute: pgtype.Int2{Int16: int16(cutoff), Valid: true},
	})
	if err != nil {
		return nil, connect.NewError(connect.CodeInternal, err)
	}

	if err := tx.Commit(ctx); err != nil {
		return nil, connect.NewError(connect.CodeInternal, err)
	}
	return connect.NewResponse(&planv1.ClearPlanResponse{
		DeletedCount:           int32(deleted),
		TrimmedStraddlingEntry: trimmed,
	}), nil
}
