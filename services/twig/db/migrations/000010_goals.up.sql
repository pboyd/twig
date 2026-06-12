CREATE TABLE goals (
    id          BIGINT GENERATED ALWAYS AS IDENTITY PRIMARY KEY,
    user_id     BIGINT NOT NULL REFERENCES users(id),
    name        VARCHAR(255) NOT NULL CHECK (btrim(name) <> ''),
    description TEXT NOT NULL DEFAULT '',
    due         TIMESTAMPTZ,
    state       TEXT NOT NULL DEFAULT 'incubating' CHECK (state IN ('incubating', 'committed', 'completed', 'archived')),
    position    INTEGER
);

CREATE INDEX goals_user_state_position_idx ON goals (user_id, state, position);

ALTER TABLE tasks ADD COLUMN goal_id BIGINT REFERENCES goals(id) ON DELETE SET NULL;
