CREATE TABLE titles (
    account_id uuid        NOT NULL,
    title      text        NOT NULL CHECK (title ~ '^[a-z]+$'),
    earned_at  timestamptz NOT NULL,
    PRIMARY KEY (account_id, title)
);

INSERT INTO titles (account_id, title, earned_at)
SELECT stats.account_id, ladder.title, now()
FROM stats
JOIN (VALUES
    ('settler', 100, 0),
    ('governor', 1000, 0),
    ('conqueror', 10000, 0),
    ('emperor', 100000, 0),
    ('loyal', 0, 7),
    ('devoted', 0, 30),
    ('unbroken', 0, 100)
) AS ladder (title, tiles, streak)
    ON stats.tiles_taken >= ladder.tiles AND stats.streak_best >= ladder.streak;
