CREATE TABLE plan_days (
    user_id   BIGINT       NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    day       DATE         NOT NULL,
    objective VARCHAR(255) NOT NULL DEFAULT '',
    notes     TEXT         NOT NULL DEFAULT '',
    PRIMARY KEY (user_id, day)
);
