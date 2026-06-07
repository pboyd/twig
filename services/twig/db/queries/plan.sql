-- name: ListPlanEntriesForDay :many
SELECT plan_entries.*,
  (plan_entries.task_id IS NOT NULL AND tasks.completed_at IS NOT NULL) AS completed
FROM plan_entries
LEFT JOIN tasks ON plan_entries.task_id = tasks.id AND tasks.user_id = plan_entries.user_id
WHERE plan_entries.user_id = $1 AND plan_entries.day = $2
ORDER BY plan_entries.start_minute ASC NULLS FIRST, plan_entries.id ASC;

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
DELETE FROM plan_entries WHERE user_id = $1 AND day = $2 AND start_minute >= $3 AND task_id IS NOT NULL;

-- name: TrimPlanEntryDuration :one
UPDATE plan_entries SET duration_minute = $4 WHERE user_id = $1 AND day = $2 AND id = $3 RETURNING *;

-- name: ListScheduledDaysForTasks :many
SELECT DISTINCT task_id, day
FROM plan_entries
WHERE user_id = $1
  AND task_id IS NOT NULL
  AND day >= $2
ORDER BY task_id, day;
