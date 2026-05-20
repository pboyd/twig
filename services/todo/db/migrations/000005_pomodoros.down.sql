DROP INDEX IF EXISTS pomodoros_one_active_per_user;
DROP INDEX IF EXISTS pomodoros_task_id_idx;
DROP TABLE IF EXISTS pomodoros;
ALTER TABLE tasks DROP COLUMN IF EXISTS estimate;
