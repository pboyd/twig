-- name: CreateTask :one
INSERT INTO tasks (name, description, due, parent_id, user_id)
VALUES ($1, $2, $3, $4, $5)
RETURNING *;

-- name: GetTask :one
SELECT * FROM tasks WHERE id = $1 AND user_id = $2;

-- name: ListTasks :many
SELECT * FROM tasks WHERE user_id = $1 ORDER BY id;

-- name: UpdateTask :one
UPDATE tasks
SET name = $2, description = $3, due = $4, parent_id = $5
WHERE id = $1 AND user_id = $6
RETURNING *;

-- name: DeleteTask :one
DELETE FROM tasks WHERE id = $1 AND user_id = $2
RETURNING id;

-- name: TaskExists :one
SELECT EXISTS (SELECT 1 FROM tasks WHERE id = $1 AND user_id = $2) AS exists;
