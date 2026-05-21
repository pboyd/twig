CREATE TABLE plan_entries (
    user_id         BIGINT   NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    day             DATE     NOT NULL,
    id              INT      NOT NULL,
    task_id         BIGINT            REFERENCES tasks(id) ON DELETE CASCADE,
    name            VARCHAR(255),
    start_minute    SMALLINT NOT NULL CHECK (start_minute BETWEEN 0 AND 1439),
    duration_minute SMALLINT NOT NULL CHECK (duration_minute > 0 AND duration_minute <= 1440),
    CHECK (start_minute + duration_minute <= 1440),
    CHECK (task_id IS NOT NULL OR name IS NOT NULL),
    PRIMARY KEY (user_id, day, id)
);

CREATE INDEX plan_entries_task_id_idx ON plan_entries (task_id);
