-- name: CreateTask :one
INSERT INTO tasks (name, description, due, parent_id, user_id, snooze_until, position)
VALUES ($1, $2, $3, $4, $5, $6,
  COALESCE((SELECT MAX(position) + 1 FROM tasks WHERE user_id = $5 AND parent_id IS NOT DISTINCT FROM $4), 0))
RETURNING *;

-- name: GetTask :one
SELECT * FROM tasks WHERE id = $1 AND user_id = $2;

-- name: ListTasks :many
SELECT * FROM tasks WHERE user_id = $1 ORDER BY id;

-- name: UpdateTask :one
UPDATE tasks
SET name = $2, description = $3, due = $4, parent_id = $5, snooze_until = $7
WHERE id = $1 AND user_id = $6
RETURNING *;

-- name: DeleteTask :one
DELETE FROM tasks WHERE id = $1 AND user_id = $2
RETURNING id;

-- name: TaskExists :one
SELECT EXISTS (SELECT 1 FROM tasks WHERE id = $1 AND user_id = $2) AS exists;

-- name: CompleteTask :one
UPDATE tasks
SET completed_at = COALESCE(completed_at, NOW())
WHERE id = $1 AND user_id = $2
RETURNING *;

-- name: UncompleteTask :one
UPDATE tasks
SET completed_at = NULL
WHERE id = $1 AND user_id = $2
RETURNING *;

-- name: HasIncompleteDescendants :one
WITH RECURSIVE descendants AS (
    SELECT tasks.id, tasks.completed_at
      FROM tasks
     WHERE tasks.parent_id = $1 AND tasks.user_id = $2
    UNION ALL
    SELECT t.id, t.completed_at
      FROM tasks t
      JOIN descendants d ON t.parent_id = d.id
     WHERE t.user_id = $2
)
SELECT EXISTS (
    SELECT 1 FROM descendants WHERE completed_at IS NULL
) AS has_incomplete;

-- name: SetTaskEstimate :one
UPDATE tasks SET estimate = $2 WHERE id = $1 AND user_id = $3 RETURNING *;

-- name: GetParentCompletion :one
SELECT completed_at FROM tasks WHERE id = $1 AND user_id = $2;

-- name: GetMaxSiblingPosition :one
SELECT COALESCE(MAX(position), -1)::integer AS max_pos
FROM tasks
WHERE user_id = $1 AND parent_id IS NOT DISTINCT FROM $2;

-- name: ListSiblingGroup :many
SELECT * FROM tasks
WHERE user_id = $1 AND parent_id IS NOT DISTINCT FROM $2
ORDER BY position, id
FOR UPDATE;

-- name: UpdateTaskPosition :exec
UPDATE tasks SET position = $3 WHERE id = $1 AND user_id = $2;

-- name: ListIncompleteDescendantIds :many
WITH RECURSIVE descendants AS (
    SELECT tasks.id, tasks.completed_at
      FROM tasks
     WHERE tasks.parent_id = $1 AND tasks.user_id = $2
    UNION ALL
    SELECT t.id, t.completed_at
      FROM tasks t
      JOIN descendants d ON t.parent_id = d.id
     WHERE t.user_id = $2
)
SELECT id FROM descendants WHERE completed_at IS NULL ORDER BY id;

-- name: SetTaskGoal :one
UPDATE tasks SET goal_id = $3 WHERE id = $1 AND user_id = $2 RETURNING *;

-- name: ClearTaskGoal :one
UPDATE tasks SET goal_id = NULL WHERE id = $1 AND user_id = $2 RETURNING *;

-- name: AncestorHasGoal :one
WITH RECURSIVE ancestors AS (
    SELECT t.id, t.parent_id, t.goal_id
      FROM tasks t
      JOIN tasks seed ON t.id = seed.parent_id
     WHERE seed.id = $1 AND seed.user_id = $2 AND t.user_id = $2
    UNION ALL
    SELECT t.id, t.parent_id, t.goal_id
      FROM tasks t
      JOIN ancestors ON t.id = ancestors.parent_id
     WHERE t.user_id = $2
)
SELECT EXISTS (
    SELECT 1 FROM ancestors WHERE goal_id IS NOT NULL
) AS has_goal;

-- name: DescendantHasGoal :one
WITH RECURSIVE descendants AS (
    SELECT tasks.id, tasks.goal_id
      FROM tasks
     WHERE tasks.parent_id = $1 AND tasks.user_id = $2
    UNION ALL
    SELECT t.id, t.goal_id
      FROM tasks t
      JOIN descendants d ON t.parent_id = d.id
     WHERE t.user_id = $2
)
SELECT EXISTS (
    SELECT 1 FROM descendants WHERE goal_id IS NOT NULL
) AS has_goal;
