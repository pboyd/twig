ALTER TABLE plan_entries ALTER COLUMN start_minute DROP NOT NULL;
ALTER TABLE plan_entries
  ADD CONSTRAINT plan_entries_event_timed
  CHECK (task_id IS NOT NULL OR start_minute IS NOT NULL);
