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

	planv1 "github.com/pboyd/twig/api/gen/plan/v1"
	"github.com/pboyd/twig/services/twig/internal/auth"
	"github.com/pboyd/twig/services/twig/internal/db"
	"github.com/pboyd/twig/services/twig/internal/plan"
)

type Plan struct {
	Queries *db.Queries
	Pool    *pgxpool.Pool
}

// dbPlanDayToProto turns a stored PlanDay row into the wire message.
// `day` is `e.Day.Time.Format(...)` so callers can always echo the stored value.
func dbPlanDayToProto(e db.PlanDay) *planv1.PlanDay {
	return &planv1.PlanDay{
		Day:       e.Day.Time.Format("2006-01-02"),
		Objective: e.Objective,
		Notes:     e.Notes,
	}
}

// loadPlanDay returns the stored PlanDay for (userID, day), or a zero-value
// PlanDay (with Day echoed in YYYY-MM-DD) if no row exists. It never returns
// pgx.ErrNoRows.
func (p *Plan) loadPlanDay(ctx context.Context, userID int64, day pgtype.Date) (*planv1.PlanDay, error) {
	row, err := p.Queries.GetPlanDay(ctx, db.GetPlanDayParams{UserID: userID, Day: day})
	if errors.Is(err, pgx.ErrNoRows) {
		return &planv1.PlanDay{Day: day.Time.Format("2006-01-02")}, nil
	}
	if err != nil {
		return nil, connect.NewError(connect.CodeInternal, err)
	}
	return dbPlanDayToProto(row), nil
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

	dayMsg, err := p.loadPlanDay(ctx, userID, day)
	if err != nil {
		return nil, err
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
	return connect.NewResponse(&planv1.ListPlanEntriesResponse{Entries: entries, Day: dayMsg}), nil
}

func (p *Plan) SetPlanObjective(
	ctx context.Context,
	req *connect.Request[planv1.SetPlanObjectiveRequest],
) (*connect.Response[planv1.SetPlanObjectiveResponse], error) {
	userID := auth.UserID(ctx)

	day, err := parseDay(req.Msg.Day)
	if err != nil {
		return nil, err
	}
	trimmed := strings.TrimSpace(req.Msg.Objective)
	if utf8.RuneCountInString(trimmed) > 255 {
		return nil, connect.NewError(connect.CodeInvalidArgument,
			errors.New("That objective is a bit long — keep it under 255 characters."))
	}

	row, err := p.Queries.UpsertPlanDayObjective(ctx, db.UpsertPlanDayObjectiveParams{
		UserID:    userID,
		Day:       day,
		Objective: trimmed,
	})
	if err != nil {
		return nil, connect.NewError(connect.CodeInternal, err)
	}
	return connect.NewResponse(&planv1.SetPlanObjectiveResponse{Day: dbPlanDayToProto(row)}), nil
}

func (p *Plan) SetPlanNotes(
	ctx context.Context,
	req *connect.Request[planv1.SetPlanNotesRequest],
) (*connect.Response[planv1.SetPlanNotesResponse], error) {
	userID := auth.UserID(ctx)

	day, err := parseDay(req.Msg.Day)
	if err != nil {
		return nil, err
	}
	trimmed := strings.TrimSpace(req.Msg.Notes)

	row, err := p.Queries.UpsertPlanDayNotes(ctx, db.UpsertPlanDayNotesParams{
		UserID: userID,
		Day:    day,
		Notes:  trimmed,
	})
	if err != nil {
		return nil, connect.NewError(connect.CodeInternal, err)
	}
	return connect.NewResponse(&planv1.SetPlanNotesResponse{Day: dbPlanDayToProto(row)}), nil
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
	var setPosition bool
	var newPosition int16
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
		// Clear-start (unschedule).
		locked, err := txq.LockPlanEntriesForDay(ctx, db.LockPlanEntriesForDayParams{UserID: userID, Day: day})
		if err != nil {
			return nil, connect.NewError(connect.CodeInternal, err)
		}

		if existing.StartMinute.Valid {
			// Timed→untimed transition: append to end of untimed group.
			var maxPos int16 = -1
			for _, r := range locked {
				if r.Position > maxPos {
					maxPos = r.Position
				}
			}
			newPosition = maxPos + 1
			setPosition = true
		}

		if existing.TaskID.Valid {
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

	var row db.PlanEntry
	if setPosition {
		row, err = txq.UpdatePlanEntryTimeAndPosition(ctx, db.UpdatePlanEntryTimeAndPositionParams{
			UserID:         userID,
			Day:            day,
			ID:             req.Msg.Id,
			StartMinute:    startParam,
			DurationMinute: int16(dur),
			Position:       newPosition,
		})
	} else {
		row, err = txq.UpdatePlanEntryTime(ctx, db.UpdatePlanEntryTimeParams{
			UserID:         userID,
			Day:            day,
			ID:             req.Msg.Id,
			StartMinute:    startParam,
			DurationMinute: int16(dur),
		})
	}
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

func (p *Plan) ReorderPlanEntry(
	ctx context.Context,
	req *connect.Request[planv1.ReorderPlanEntryRequest],
) (*connect.Response[planv1.ReorderPlanEntryResponse], error) {
	userID := auth.UserID(ctx)

	day, err := parseDay(req.Msg.Day)
	if err != nil {
		return nil, err
	}

	movedID := req.Msg.Id
	var anchorID int32
	var insertBefore bool

	switch a := req.Msg.Anchor.(type) {
	case *planv1.ReorderPlanEntryRequest_BeforeId:
		anchorID = a.BeforeId
		insertBefore = true
	case *planv1.ReorderPlanEntryRequest_AfterId:
		anchorID = a.AfterId
		insertBefore = false
	case nil:
		return nil, connect.NewError(connect.CodeInvalidArgument, errors.New("exactly one of before_id or after_id must be set"))
	}

	if anchorID == movedID {
		return nil, connect.NewError(connect.CodeInvalidArgument, errors.New("anchor must be a different entry than the one being moved"))
	}

	tx, txq, err := p.beginSerializableTx(ctx)
	if err != nil {
		return nil, err
	}
	defer tx.Rollback(ctx)

	moved, err := txq.GetPlanEntry(ctx, db.GetPlanEntryParams{
		UserID: userID,
		Day:    day,
		ID:     movedID,
	})
	if errors.Is(err, pgx.ErrNoRows) {
		return nil, connect.NewError(connect.CodeNotFound, errors.New("entry not found"))
	}
	if err != nil {
		return nil, connect.NewError(connect.CodeInternal, err)
	}

	anchor, err := txq.GetPlanEntry(ctx, db.GetPlanEntryParams{
		UserID: userID,
		Day:    day,
		ID:     anchorID,
	})
	if errors.Is(err, pgx.ErrNoRows) {
		return nil, connect.NewError(connect.CodeNotFound, errors.New("anchor entry not found"))
	}
	if err != nil {
		return nil, connect.NewError(connect.CodeInternal, err)
	}

	// Validate both are untimed.
	if moved.StartMinute.Valid {
		return nil, connect.NewError(connect.CodeInvalidArgument, errors.New("only untimed entries can be reordered"))
	}
	if anchor.StartMinute.Valid {
		return nil, connect.NewError(connect.CodeInvalidArgument, errors.New("anchor must be an untimed entry"))
	}

	// Lock the day's untimed entries and read them.
	untimed, err := txq.LockUntimedPlanEntriesForDay(ctx, db.LockUntimedPlanEntriesForDayParams{
		UserID: userID,
		Day:    day,
	})
	if err != nil {
		return nil, connect.NewError(connect.CodeInternal, err)
	}

	// Build the ordered slice (excluding the moved entry).
	ordered := make([]db.PlanEntry, 0, len(untimed))
	for _, e := range untimed {
		if e.ID != movedID {
			ordered = append(ordered, e)
		}
	}

	// Find anchor position and insert.
	insertIdx := len(ordered)
	for i, e := range ordered {
		if e.ID == anchorID {
			if insertBefore {
				insertIdx = i
			} else {
				insertIdx = i + 1
			}
			break
		}
	}

	ordered = append(ordered, db.PlanEntry{})
	copy(ordered[insertIdx+1:], ordered[insertIdx:])
	ordered[insertIdx] = moved

	// Renumber 0..n-1 and persist.
	for i, e := range ordered {
		if err := txq.UpdatePlanEntryPosition(ctx, db.UpdatePlanEntryPositionParams{
			UserID:   userID,
			Day:      day,
			ID:       e.ID,
			Position: int16(i),
		}); err != nil {
			return nil, connect.NewError(connect.CodeInternal, err)
		}
	}

	if err := tx.Commit(ctx); err != nil {
		return nil, connect.NewError(connect.CodeInternal, err)
	}

	proto := make([]*planv1.PlanEntry, len(ordered))
	for i, e := range ordered {
		proto[i] = dbPlanEntryToProto(e)
	}

	return connect.NewResponse(&planv1.ReorderPlanEntryResponse{Untimed: proto}), nil
}

func (p *Plan) ListScheduledDays(
	ctx context.Context,
	req *connect.Request[planv1.ListScheduledDaysRequest],
) (*connect.Response[planv1.ListScheduledDaysResponse], error) {
	userID := auth.UserID(ctx)

	fromDay, err := parseDay(req.Msg.FromDay)
	if err != nil {
		return nil, err
	}

	rows, err := p.Queries.ListScheduledDaysForTasks(ctx, db.ListScheduledDaysForTasksParams{
		UserID: userID,
		Day:    fromDay,
	})
	if err != nil {
		return nil, connect.NewError(connect.CodeInternal, err)
	}

	days := make([]*planv1.ScheduledDay, len(rows))
	for i, r := range rows {
		days[i] = &planv1.ScheduledDay{
			TaskId: r.TaskID.Int64,
			Day:    r.Day.Time.Format("2006-01-02"),
		}
	}
	return connect.NewResponse(&planv1.ListScheduledDaysResponse{Days: days}), nil
}

