ALTER TABLE goals DROP CONSTRAINT IF EXISTS goals_state_check;

UPDATE goals SET state = 'committed' WHERE state = 'in_progress';

ALTER TABLE goals ADD CONSTRAINT goals_state_check
    CHECK (state IN ('incubating', 'committed', 'completed', 'archived', 'hold'));
