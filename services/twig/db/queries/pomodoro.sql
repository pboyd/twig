-- name: StartPomodoro :one
INSERT INTO pomodoros (user_id, task_id, start_at)
VALUES ($1, $2, NOW())
RETURNING *;

-- name: CancelActivePomodoro :one
UPDATE pomodoros SET end_at = NOW()
WHERE user_id = $1 AND end_at IS NULL
RETURNING *;

-- name: CompleteActivePomodoro :one
UPDATE pomodoros SET end_at = start_at + INTERVAL '25 minutes', complete = TRUE
WHERE user_id = $1 AND end_at IS NULL
RETURNING *;

-- name: GetActivePomodoro :one
SELECT * FROM pomodoros WHERE user_id = $1 AND end_at IS NULL;

-- name: ListPomodorosForTask :many
SELECT * FROM pomodoros WHERE task_id = $1 AND user_id = $2 ORDER BY start_at;

-- name: CountCompletedPomodorosForTask :one
SELECT count(*)::bigint AS count FROM pomodoros WHERE task_id = $1 AND user_id = $2 AND complete;

-- name: CountCompletedPomodorosByTask :many
SELECT task_id, count(*)::bigint AS count
FROM pomodoros
WHERE user_id = $1 AND complete
GROUP BY task_id;
