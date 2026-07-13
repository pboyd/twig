UPDATE goals SET state = 'committed' WHERE state = 'hold';

ALTER TABLE goals DROP CONSTRAINT IF EXISTS goals_state_check;

ALTER TABLE goals ADD CONSTRAINT goals_state_check
    CHECK (state IN ('incubating', 'committed', 'completed', 'archived'));
