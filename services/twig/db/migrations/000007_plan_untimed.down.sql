ALTER TABLE plan_entries DROP CONSTRAINT plan_entries_event_timed;
-- Will fail if untimed rows exist; acceptable for a rollback.
ALTER TABLE plan_entries ALTER COLUMN start_minute SET NOT NULL;
