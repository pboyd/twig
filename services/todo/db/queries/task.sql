-- name: CreateTask :one
INSERT INTO tasks (name, description, due, parent_id)
VALUES ($1, $2, $3, $4)
RETURNING *;

-- name: GetTask :one
SELECT * FROM tasks WHERE id = $1;

-- name: ListTasks :many
SELECT * FROM tasks ORDER BY id;

-- name: UpdateTask :one
UPDATE tasks
SET name = $2, description = $3, due = $4, parent_id = $5
WHERE id = $1
RETURNING *;

-- name: DeleteTask :one
DELETE FROM tasks WHERE id = $1
RETURNING id;

-- name: TaskExists :one
SELECT EXISTS (SELECT 1 FROM tasks WHERE id = $1) AS exists;

