ALTER TABLE tasks ADD COLUMN estimate SMALLINT NOT NULL DEFAULT 0 CHECK (estimate BETWEEN 0 AND 10);

CREATE TABLE pomodoros (
    id        BIGINT GENERATED ALWAYS AS IDENTITY PRIMARY KEY,
    user_id   BIGINT      NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    task_id   BIGINT      NOT NULL REFERENCES tasks(id) ON DELETE CASCADE,
    start_at  TIMESTAMPTZ NOT NULL,
    end_at    TIMESTAMPTZ,
    complete  BOOLEAN     NOT NULL DEFAULT FALSE,
    CHECK (end_at IS NULL OR end_at >= start_at),
    CHECK (NOT complete OR end_at IS NOT NULL)
);

CREATE UNIQUE INDEX pomodoros_one_active_per_user ON pomodoros (user_id) WHERE end_at IS NULL;

CREATE INDEX pomodoros_task_id_idx ON pomodoros (task_id);
