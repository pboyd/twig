package handler

import (
	"context"
	"errors"
	"strings"

	"connectrpc.com/connect"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgtype"
	"github.com/jackc/pgx/v5/pgxpool"
	"google.golang.org/protobuf/types/known/timestamppb"

	goalv1 "github.com/pboyd/twig/api/gen/goal/v1"
	"github.com/pboyd/twig/services/twig/internal/auth"
	"github.com/pboyd/twig/services/twig/internal/db"
)

// Goal implements goalv1connect.GoalServiceHandler.
type Goal struct {
	Queries *db.Queries
	Pool    *pgxpool.Pool
}

// goalStateToString converts a proto GoalState enum to the DB string.
func goalStateToString(s goalv1.GoalState) (string, error) {
	switch s {
	case goalv1.GoalState_GOAL_STATE_INCUBATING:
		return "incubating", nil
	case goalv1.GoalState_GOAL_STATE_COMMITTED:
		return "committed", nil
	case goalv1.GoalState_GOAL_STATE_COMPLETED:
		return "completed", nil
	case goalv1.GoalState_GOAL_STATE_ARCHIVED:
		return "archived", nil
	case goalv1.GoalState_GOAL_STATE_HOLD:
		return "hold", nil
	default:
		return "", connect.NewError(connect.CodeInvalidArgument, errors.New("state must not be GOAL_STATE_UNSPECIFIED"))
	}
}

// goalStateFromString converts a DB string to proto GoalState.
func goalStateFromString(s string) goalv1.GoalState {
	switch s {
	case "incubating":
		return goalv1.GoalState_GOAL_STATE_INCUBATING
	case "committed":
		return goalv1.GoalState_GOAL_STATE_COMMITTED
	case "completed":
		return goalv1.GoalState_GOAL_STATE_COMPLETED
	case "archived":
		return goalv1.GoalState_GOAL_STATE_ARCHIVED
	case "hold":
		return goalv1.GoalState_GOAL_STATE_HOLD
	default:
		return goalv1.GoalState_GOAL_STATE_UNSPECIFIED
	}
}

// dbGoalToProto converts a db.Goal to a proto Goal.
func dbGoalToProto(g db.Goal) *goalv1.Goal {
	pg := &goalv1.Goal{
		Id:          g.ID,
		Name:        g.Name,
		Description: g.Description,
		State:       goalStateFromString(g.State),
	}
	if g.Due.Valid {
		pg.Due = timestamppb.New(g.Due.Time)
	}
	if g.Position.Valid {
		pg.Position = int64(g.Position.Int32)
	}
	return pg
}

// dbStatusUpdateToProto converts a db.GoalStatusUpdate to proto.
func dbStatusUpdateToProto(su db.GoalStatusUpdate) *goalv1.StatusUpdate {
	p := &goalv1.StatusUpdate{
		Id:     su.ID,
		GoalId: su.GoalID,
		Body:   su.Body,
	}
	if su.CreatedAt.Valid {
		p.CreatedAt = timestamppb.New(su.CreatedAt.Time)
	}
	return p
}

// validateBody trims and validates a status-update body.
func validateBody(body string) (string, error) {
	trimmed := strings.TrimSpace(body)
	if trimmed == "" {
		return "", connect.NewError(connect.CodeInvalidArgument,
			errors.New("a status update needs a few words — mind jotting something down?"))
	}
	return trimmed, nil
}

func (g *Goal) CreateGoal(
	ctx context.Context,
	req *connect.Request[goalv1.CreateGoalRequest],
) (*connect.Response[goalv1.CreateGoalResponse], error) {
	userID := auth.UserID(ctx)

	name, err := validateName(req.Msg.Name)
	if err != nil {
		return nil, err
	}

	params := db.CreateGoalParams{
		UserID:      userID,
		Name:        name,
		Description: req.Msg.Description,
	}
	if req.Msg.Due != nil {
		params.Due = pgtype.Timestamptz{Time: req.Msg.Due.AsTime(), Valid: true}
	}

	row, err := g.Queries.CreateGoal(ctx, params)
	if err != nil {
		return nil, connect.NewError(connect.CodeInternal, err)
	}
	return connect.NewResponse(&goalv1.CreateGoalResponse{Goal: dbGoalToProto(row)}), nil
}

func (g *Goal) GetGoal(
	ctx context.Context,
	req *connect.Request[goalv1.GetGoalRequest],
) (*connect.Response[goalv1.GetGoalResponse], error) {
	userID := auth.UserID(ctx)

	row, err := g.Queries.GetGoal(ctx, db.GetGoalParams{ID: req.Msg.Id, UserID: userID})
	if errors.Is(err, pgx.ErrNoRows) {
		return nil, connect.NewError(connect.CodeNotFound, errors.New("goal not found"))
	}
	if err != nil {
		return nil, connect.NewError(connect.CodeInternal, err)
	}
	pg := dbGoalToProto(row)
	if su, err := g.Queries.GetLatestGoalStatusUpdate(ctx, row.ID); err == nil {
		pg.LatestStatusUpdate = dbStatusUpdateToProto(su)
	}
	return connect.NewResponse(&goalv1.GetGoalResponse{Goal: pg}), nil
}

func (g *Goal) ListGoals(
	ctx context.Context,
	req *connect.Request[goalv1.ListGoalsRequest],
) (*connect.Response[goalv1.ListGoalsResponse], error) {
	userID := auth.UserID(ctx)

	rows, err := g.Queries.ListGoals(ctx, userID)
	if err != nil {
		return nil, connect.NewError(connect.CodeInternal, err)
	}
	goals := make([]*goalv1.Goal, len(rows))
	for i, r := range rows {
		goals[i] = dbGoalToProto(r)
	}

	// Populate latest_status_update for each goal in a single round trip.
	if len(goals) > 0 {
		goalIDs := make([]int64, len(rows))
		for i, r := range rows {
			goalIDs[i] = r.ID
		}
		latestUpdates, err := g.Queries.ListLatestGoalStatusUpdates(ctx, goalIDs)
		if err != nil {
			return nil, connect.NewError(connect.CodeInternal, err)
		}
		latestByGoal := make(map[int64]*goalv1.StatusUpdate, len(latestUpdates))
		for _, su := range latestUpdates {
			latestByGoal[su.GoalID] = dbStatusUpdateToProto(su)
		}
		for _, pg := range goals {
			if su, ok := latestByGoal[pg.Id]; ok {
				pg.LatestStatusUpdate = su
			}
		}
	}

	return connect.NewResponse(&goalv1.ListGoalsResponse{Goals: goals}), nil
}

func (g *Goal) UpdateGoal(
	ctx context.Context,
	req *connect.Request[goalv1.UpdateGoalRequest],
) (*connect.Response[goalv1.UpdateGoalResponse], error) {
	userID := auth.UserID(ctx)

	name, err := validateName(req.Msg.Name)
	if err != nil {
		return nil, err
	}

	params := db.UpdateGoalParams{
		ID:          req.Msg.Id,
		UserID:      userID,
		Name:        name,
		Description: req.Msg.Description,
	}
	if req.Msg.Due != nil {
		params.Due = pgtype.Timestamptz{Time: req.Msg.Due.AsTime(), Valid: true}
	}

	row, err := g.Queries.UpdateGoal(ctx, params)
	if errors.Is(err, pgx.ErrNoRows) {
		return nil, connect.NewError(connect.CodeNotFound, errors.New("goal not found"))
	}
	if err != nil {
		return nil, connect.NewError(connect.CodeInternal, err)
	}
	return connect.NewResponse(&goalv1.UpdateGoalResponse{Goal: dbGoalToProto(row)}), nil
}

func (g *Goal) SetGoalState(
	ctx context.Context,
	req *connect.Request[goalv1.SetGoalStateRequest],
) (*connect.Response[goalv1.SetGoalStateResponse], error) {
	userID := auth.UserID(ctx)

	stateStr, err := goalStateToString(req.Msg.State)
	if err != nil {
		return nil, err
	}

	row, err := g.Queries.SetGoalState(ctx, db.SetGoalStateParams{
		ID:     req.Msg.Id,
		UserID: userID,
		State:  stateStr,
	})
	if errors.Is(err, pgx.ErrNoRows) {
		// Either the goal doesn't exist, or it was already in the target state (no-op).
		current, err := g.Queries.GetGoal(ctx, db.GetGoalParams{ID: req.Msg.Id, UserID: userID})
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, connect.NewError(connect.CodeNotFound, errors.New("goal not found"))
		}
		if err != nil {
			return nil, connect.NewError(connect.CodeInternal, err)
		}
		return connect.NewResponse(&goalv1.SetGoalStateResponse{Goal: dbGoalToProto(current)}), nil
	}
	if err != nil {
		return nil, connect.NewError(connect.CodeInternal, err)
	}
	return connect.NewResponse(&goalv1.SetGoalStateResponse{Goal: dbGoalToProto(row)}), nil
}

func (g *Goal) ReorderGoal(
	ctx context.Context,
	req *connect.Request[goalv1.ReorderGoalRequest],
) (*connect.Response[goalv1.ReorderGoalResponse], error) {
	userID := auth.UserID(ctx)

	goalID := req.Msg.GoalId
	var anchorID int64
	var insertBefore bool

	switch a := req.Msg.Anchor.(type) {
	case *goalv1.ReorderGoalRequest_BeforeGoalId:
		anchorID = a.BeforeGoalId
		insertBefore = true
	case *goalv1.ReorderGoalRequest_AfterGoalId:
		anchorID = a.AfterGoalId
		insertBefore = false
	case nil:
		return nil, connect.NewError(connect.CodeInvalidArgument, errors.New("exactly one of before_goal_id or after_goal_id must be set"))
	}

	if anchorID == goalID {
		return nil, connect.NewError(connect.CodeInvalidArgument, errors.New("anchor must be a different goal than the one being moved"))
	}

	if g.Pool == nil {
		return nil, connect.NewError(connect.CodeInternal, errors.New("pool not configured"))
	}

	tx, err := g.Pool.Begin(ctx)
	if err != nil {
		return nil, connect.NewError(connect.CodeInternal, err)
	}
	defer tx.Rollback(ctx)

	q := g.Queries.WithTx(tx)

	goal, err := q.GetGoal(ctx, db.GetGoalParams{ID: goalID, UserID: userID})
	if errors.Is(err, pgx.ErrNoRows) {
		return nil, connect.NewError(connect.CodeNotFound, errors.New("goal not found"))
	}
	if err != nil {
		return nil, connect.NewError(connect.CodeInternal, err)
	}

	anchor, err := q.GetGoal(ctx, db.GetGoalParams{ID: anchorID, UserID: userID})
	if errors.Is(err, pgx.ErrNoRows) {
		return nil, connect.NewError(connect.CodeNotFound, errors.New("anchor goal not found"))
	}
	if err != nil {
		return nil, connect.NewError(connect.CodeInternal, err)
	}

	// Anchor must be in the same state group.
	if goal.State != anchor.State {
		return nil, connect.NewError(connect.CodeInvalidArgument, errors.New("anchor must be in the same state group as the goal"))
	}

	// Read the full state group in order (FOR UPDATE acquired by ListGoalStateGroup).
	group, err := q.ListGoalStateGroup(ctx, db.ListGoalStateGroupParams{
		UserID: userID,
		State:  goal.State,
	})
	if err != nil {
		return nil, connect.NewError(connect.CodeInternal, err)
	}

	// Remove goal from its current position in the group.
	ordered := make([]db.Goal, 0, len(group))
	for _, s := range group {
		if s.ID != goalID {
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

	// Insert goal at insertIdx.
	ordered = append(ordered, db.Goal{}) // grow
	copy(ordered[insertIdx+1:], ordered[insertIdx:])
	ordered[insertIdx] = goal

	// Renumber 0..n-1 and persist.
	for i, s := range ordered {
		pos := pgtype.Int4{Int32: int32(i), Valid: true}
		if err := q.UpdateGoalPosition(ctx, db.UpdateGoalPositionParams{
			ID:       s.ID,
			UserID:   userID,
			Position: pos,
		}); err != nil {
			return nil, connect.NewError(connect.CodeInternal, err)
		}
		ordered[i].Position = pos
	}

	if err := tx.Commit(ctx); err != nil {
		return nil, connect.NewError(connect.CodeInternal, err)
	}

	proto := make([]*goalv1.Goal, len(ordered))
	for i, s := range ordered {
		proto[i] = dbGoalToProto(s)
	}

	return connect.NewResponse(&goalv1.ReorderGoalResponse{Goals: proto}), nil
}

func (g *Goal) DeleteGoal(
	ctx context.Context,
	req *connect.Request[goalv1.DeleteGoalRequest],
) (*connect.Response[goalv1.DeleteGoalResponse], error) {
	userID := auth.UserID(ctx)

	_, err := g.Queries.DeleteGoal(ctx, db.DeleteGoalParams{ID: req.Msg.Id, UserID: userID})
	if errors.Is(err, pgx.ErrNoRows) {
		return nil, connect.NewError(connect.CodeNotFound, errors.New("goal not found"))
	}
	if err != nil {
		return nil, connect.NewError(connect.CodeInternal, err)
	}
	return connect.NewResponse(&goalv1.DeleteGoalResponse{}), nil
}

func (g *Goal) ListGoalStatusUpdates(
	ctx context.Context,
	req *connect.Request[goalv1.ListGoalStatusUpdatesRequest],
) (*connect.Response[goalv1.ListGoalStatusUpdatesResponse], error) {
	userID := auth.UserID(ctx)

	// Verify goal ownership.
	_, err := g.Queries.GetGoal(ctx, db.GetGoalParams{ID: req.Msg.GoalId, UserID: userID})
	if errors.Is(err, pgx.ErrNoRows) {
		return nil, connect.NewError(connect.CodeNotFound, errors.New("goal not found"))
	}
	if err != nil {
		return nil, connect.NewError(connect.CodeInternal, err)
	}

	rows, err := g.Queries.ListGoalStatusUpdates(ctx, db.ListGoalStatusUpdatesParams{
		GoalID: req.Msg.GoalId,
		UserID: userID,
	})
	if err != nil {
		return nil, connect.NewError(connect.CodeInternal, err)
	}
	updates := make([]*goalv1.StatusUpdate, len(rows))
	for i, r := range rows {
		updates[i] = dbStatusUpdateToProto(r)
	}
	return connect.NewResponse(&goalv1.ListGoalStatusUpdatesResponse{Updates: updates}), nil
}

func (g *Goal) AddGoalStatusUpdate(
	ctx context.Context,
	req *connect.Request[goalv1.AddGoalStatusUpdateRequest],
) (*connect.Response[goalv1.AddGoalStatusUpdateResponse], error) {
	userID := auth.UserID(ctx)

	body, err := validateBody(req.Msg.Body)
	if err != nil {
		return nil, err
	}

	su, err := g.Queries.AddGoalStatusUpdate(ctx, db.AddGoalStatusUpdateParams{
		GoalID: req.Msg.GoalId,
		UserID: userID,
		Body:   body,
	})
	if errors.Is(err, pgx.ErrNoRows) {
		return nil, connect.NewError(connect.CodeNotFound, errors.New("goal not found"))
	}
	if err != nil {
		return nil, connect.NewError(connect.CodeInternal, err)
	}
	return connect.NewResponse(&goalv1.AddGoalStatusUpdateResponse{Update: dbStatusUpdateToProto(su)}), nil
}

func (g *Goal) UpdateGoalStatusUpdate(
	ctx context.Context,
	req *connect.Request[goalv1.UpdateGoalStatusUpdateRequest],
) (*connect.Response[goalv1.UpdateGoalStatusUpdateResponse], error) {
	userID := auth.UserID(ctx)

	body, err := validateBody(req.Msg.Body)
	if err != nil {
		return nil, err
	}

	su, err := g.Queries.UpdateGoalStatusUpdate(ctx, db.UpdateGoalStatusUpdateParams{
		ID:     req.Msg.Id,
		UserID: userID,
		Body:   body,
	})
	if errors.Is(err, pgx.ErrNoRows) {
		return nil, connect.NewError(connect.CodeNotFound, errors.New("status update not found"))
	}
	if err != nil {
		return nil, connect.NewError(connect.CodeInternal, err)
	}
	return connect.NewResponse(&goalv1.UpdateGoalStatusUpdateResponse{Update: dbStatusUpdateToProto(su)}), nil
}

func (g *Goal) DeleteGoalStatusUpdate(
	ctx context.Context,
	req *connect.Request[goalv1.DeleteGoalStatusUpdateRequest],
) (*connect.Response[goalv1.DeleteGoalStatusUpdateResponse], error) {
	userID := auth.UserID(ctx)

	_, err := g.Queries.DeleteGoalStatusUpdate(ctx, db.DeleteGoalStatusUpdateParams{
		ID:     req.Msg.Id,
		UserID: userID,
	})
	if errors.Is(err, pgx.ErrNoRows) {
		return nil, connect.NewError(connect.CodeNotFound, errors.New("status update not found"))
	}
	if err != nil {
		return nil, connect.NewError(connect.CodeInternal, err)
	}
	return connect.NewResponse(&goalv1.DeleteGoalStatusUpdateResponse{}), nil
}
