ALTER TABLE tasks ADD COLUMN position INTEGER NOT NULL DEFAULT 0;

WITH ordered AS (
    SELECT id,
           ROW_NUMBER() OVER (
               PARTITION BY user_id, parent_id
               ORDER BY id
           ) - 1 AS pos
    FROM tasks
)
UPDATE tasks t SET position = ordered.pos
FROM ordered WHERE ordered.id = t.id;

CREATE INDEX tasks_user_parent_position_idx ON tasks (user_id, parent_id, position);
