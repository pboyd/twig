CREATE TABLE tasks (
    id          BIGINT GENERATED ALWAYS AS IDENTITY PRIMARY KEY,
    name        VARCHAR(255) NOT NULL CHECK (btrim(name) <> ''),
    description TEXT NOT NULL DEFAULT '',
    due         TIMESTAMPTZ,
    parent_id   BIGINT REFERENCES tasks(id) ON DELETE CASCADE
);

CREATE INDEX tasks_parent_id_idx ON tasks (parent_id);
