ALTER TABLE tasks DROP COLUMN goal_id;

DROP INDEX goals_user_state_position_idx;

DROP TABLE goals;
