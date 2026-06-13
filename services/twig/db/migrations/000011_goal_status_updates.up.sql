CREATE TABLE goal_status_updates (
    id         BIGINT GENERATED ALWAYS AS IDENTITY PRIMARY KEY,
    goal_id    BIGINT NOT NULL REFERENCES goals(id) ON DELETE CASCADE,
    body       TEXT NOT NULL CHECK (btrim(body) <> ''),
    created_at TIMESTAMPTZ NOT NULL DEFAULT now()
);

CREATE INDEX goal_status_updates_goal_created_idx
    ON goal_status_updates (goal_id, created_at DESC, id DESC);
