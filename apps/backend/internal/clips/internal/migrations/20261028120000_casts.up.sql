CREATE TYPE platform AS ENUM ('tiktok', 'youtube');

-- Which account each number of a run's cast is, for the posts of the run to name: kept 3 days.
CREATE TABLE casts (
    cast_id uuid        PRIMARY KEY,
    made_at timestamptz NOT NULL,
    since   timestamptz NOT NULL,
    until   timestamptz NOT NULL
);

CREATE INDEX casts_made_at ON casts (made_at);

-- account and name: NULL once the account is deleted, so the number stays the cast's.
CREATE TABLE cast_fighters (
    cast_id uuid    NOT NULL REFERENCES casts (cast_id) ON DELETE CASCADE,
    number  integer NOT NULL CHECK (number > 0),
    account uuid,
    name    text,
    PRIMARY KEY (cast_id, number)
);

CREATE INDEX cast_fighters_account ON cast_fighters (account);

-- A post that names a player, for the game to tell it once the post is out: kept 7 days.
CREATE TABLE appearances (
    post_id      text        NOT NULL,
    account      uuid        NOT NULL,
    title        text        NOT NULL,
    platforms    platform[]  NOT NULL,
    scheduled_at timestamptz NOT NULL,
    PRIMARY KEY (post_id, account)
);

CREATE INDEX appearances_account ON appearances (account, scheduled_at);

CREATE INDEX appearances_scheduled_at ON appearances (scheduled_at);

-- A post's address on a platform, read from its newest snapshot.
CREATE INDEX snapshot_rows_post ON snapshot_rows (post_id, platform, taken_at DESC);
