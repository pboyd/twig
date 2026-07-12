-- name: CreateGoal :one
INSERT INTO goals (user_id, name, description, due, state, position)
VALUES ($1, $2, $3, $4, 'incubating',
  COALESCE((SELECT MAX(position) + 1 FROM goals WHERE user_id = $1 AND state = 'incubating'), 0))
RETURNING *;

-- name: GetGoal :one
SELECT * FROM goals WHERE id = $1 AND user_id = $2;

-- name: ListGoals :many
SELECT * FROM goals
WHERE user_id = $1
ORDER BY
  CASE state
    WHEN 'committed'  THEN 1
    WHEN 'incubating' THEN 2
    WHEN 'hold'       THEN 3
    WHEN 'completed'  THEN 4
    WHEN 'archived'   THEN 5
    ELSE 6
  END,
  position, id;

-- name: UpdateGoal :one
UPDATE goals
SET name = $3, description = $4, due = $5
WHERE id = $1 AND user_id = $2
RETURNING *;

-- name: SetGoalState :one
UPDATE goals AS g
SET state = $3,
    position = COALESCE((SELECT MAX(g2.position) + 1 FROM goals g2 WHERE g2.user_id = $2 AND g2.state = $3), 0)
WHERE g.id = $1 AND g.user_id = $2 AND g.state <> $3
RETURNING *;

-- name: ListGoalStateGroup :many
SELECT * FROM goals
WHERE user_id = $1 AND state = $2
ORDER BY position, id
FOR UPDATE;

-- name: UpdateGoalPosition :exec
UPDATE goals SET position = $3 WHERE id = $1 AND user_id = $2;

-- name: DeleteGoal :one
DELETE FROM goals WHERE id = $1 AND user_id = $2 RETURNING id;
