ALTER TABLE plan_entries ADD COLUMN position SMALLINT NOT NULL DEFAULT 0;

WITH ordered AS (
    SELECT user_id, day, id,
           ROW_NUMBER() OVER (
               PARTITION BY user_id, day
               ORDER BY id
           ) - 1 AS pos
    FROM plan_entries
)
UPDATE plan_entries p SET position = ordered.pos
FROM ordered
WHERE ordered.user_id = p.user_id AND ordered.day = p.day AND ordered.id = p.id;

CREATE INDEX plan_entries_user_day_position_idx ON plan_entries (user_id, day, position);
