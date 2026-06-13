-- name: ListGoalStatusUpdates :many
SELECT su.* FROM goal_status_updates su
JOIN goals g ON g.id = su.goal_id
WHERE su.goal_id = $1 AND g.user_id = $2
ORDER BY su.created_at DESC, su.id DESC;

-- name: AddGoalStatusUpdate :one
INSERT INTO goal_status_updates (goal_id, body)
SELECT $1, $3 FROM goals WHERE id = $1 AND user_id = $2
RETURNING *;

-- name: UpdateGoalStatusUpdate :one
UPDATE goal_status_updates su SET body = $3
FROM goals g
WHERE su.id = $1 AND su.goal_id = g.id AND g.user_id = $2
RETURNING su.*;

-- name: DeleteGoalStatusUpdate :one
DELETE FROM goal_status_updates su
USING goals g
WHERE su.id = $1 AND su.goal_id = g.id AND g.user_id = $2
RETURNING su.id;

-- name: GetLatestGoalStatusUpdate :one
SELECT * FROM goal_status_updates
WHERE goal_id = $1
ORDER BY created_at DESC, id DESC
LIMIT 1;

-- name: ListLatestGoalStatusUpdates :many
SELECT DISTINCT ON (goal_id) *
FROM goal_status_updates
WHERE goal_id = ANY($1::bigint[])
ORDER BY goal_id, created_at DESC, id DESC;
