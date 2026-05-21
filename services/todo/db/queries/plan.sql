-- name: ListPlanEntriesForDay :many
SELECT * FROM plan_entries WHERE user_id = $1 AND day = $2 ORDER BY start_minute;

-- name: GetPlanEntry :one
SELECT * FROM plan_entries WHERE user_id = $1 AND day = $2 AND id = $3;

-- name: LockPlanEntriesForDay :many
SELECT * FROM plan_entries WHERE user_id = $1 AND day = $2 FOR UPDATE;

-- name: NextPlanEntryId :one
SELECT COALESCE(MAX(id), 0) + 1 AS next_id FROM plan_entries WHERE user_id = $1 AND day = $2;

-- name: InsertPlanEntry :one
INSERT INTO plan_entries (user_id, day, id, task_id, name, start_minute, duration_minute)
VALUES ($1, $2, $3, $4, $5, $6, $7)
RETURNING *;

-- name: UpdatePlanEntryName :one
UPDATE plan_entries SET name = $4 WHERE user_id = $1 AND day = $2 AND id = $3 RETURNING *;

-- name: UpdatePlanEntryTime :one
UPDATE plan_entries SET start_minute = $4, duration_minute = $5 WHERE user_id = $1 AND day = $2 AND id = $3 RETURNING *;

-- name: DeletePlanEntry :one
DELETE FROM plan_entries WHERE user_id = $1 AND day = $2 AND id = $3 RETURNING id;

-- name: DeletePlanEntriesFromMinute :execrows
DELETE FROM plan_entries WHERE user_id = $1 AND day = $2 AND start_minute >= $3;

-- name: TrimPlanEntryDuration :one
UPDATE plan_entries SET duration_minute = $4 WHERE user_id = $1 AND day = $2 AND id = $3 RETURNING *;
