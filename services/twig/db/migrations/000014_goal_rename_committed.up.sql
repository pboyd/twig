ALTER TABLE goals DROP CONSTRAINT IF EXISTS goals_state_check;

UPDATE goals SET state = 'in_progress' WHERE state = 'committed';

ALTER TABLE goals ADD CONSTRAINT goals_state_check
    CHECK (state IN ('incubating', 'in_progress', 'completed', 'archived', 'hold'));
