-- Player kept these four tables until progression was split from it. They move, rows, indexes and constraints with
-- them, so nothing is copied and nothing is lost. A fresh database moves the ones player's own migrations just made.
DO $$
DECLARE
    moved text;
BEGIN
    FOREACH moved IN ARRAY ARRAY['stats', 'titles', 'worn_titles', 'finale_fights'] LOOP
        IF to_regclass('player.' || moved) IS NOT NULL THEN
            EXECUTE format('ALTER TABLE player.%I SET SCHEMA %I', moved, current_schema());
        END IF;
    END LOOP;
END $$;

CREATE TABLE IF NOT EXISTS stats (
    account_id                uuid        PRIMARY KEY,
    tiles_taken               bigint      NOT NULL CHECK (tiles_taken >= 0),
    streak_current            bigint      NOT NULL CHECK (streak_current >= 0),
    streak_best               bigint      NOT NULL CHECK (streak_best >= streak_current),
    streak_last_day           date,
    messages_sent             bigint      NOT NULL DEFAULT 0 CHECK (messages_sent >= 0),
    bombs_dropped             bigint      NOT NULL DEFAULT 0 CHECK (bombs_dropped >= 0),
    tiles_fortified           bigint      NOT NULL DEFAULT 0 CHECK (tiles_fortified >= 0),
    bonuses_claimed           bigint      NOT NULL DEFAULT 0 CHECK (bonuses_claimed >= 0),
    right_answers             bigint      NOT NULL DEFAULT 0 CHECK (right_answers >= 0),
    counted_takes_below       bigint      NOT NULL DEFAULT 0 CHECK (counted_takes_below >= 0),
    counted_bombs_below       bigint      NOT NULL DEFAULT 0 CHECK (counted_bombs_below >= 0),
    counted_fortifies_below   bigint      NOT NULL DEFAULT 0 CHECK (counted_fortifies_below >= 0),
    counted_messages_until    timestamptz,
    counted_bonuses_until     timestamptz,
    counted_answers_until     timestamptz,
    missions_done             bigint      NOT NULL DEFAULT 0 CHECK (missions_done >= 0),
    counted_takes_below_index integer     NOT NULL DEFAULT 0 CHECK (counted_takes_below_index >= 0)
);

CREATE TABLE IF NOT EXISTS titles (
    account_id uuid        NOT NULL,
    title      text        NOT NULL CHECK (title ~ '^[a-z0-9_]+$'),
    earned_at  timestamptz NOT NULL,
    PRIMARY KEY (account_id, title)
);

CREATE TABLE IF NOT EXISTS worn_titles (
    account_id uuid        PRIMARY KEY,
    title      text        NOT NULL CHECK (title ~ '^[a-z0-9_]+$'),
    worn_at    timestamptz NOT NULL
);

CREATE TABLE IF NOT EXISTS finale_fights (
    season     integer NOT NULL CHECK (season >= 0),
    account_id uuid    NOT NULL,
    country    text    NOT NULL CHECK (country <> ''),
    -- 0: the flag is not on the season's table.
    flag_rank  integer NOT NULL CHECK (flag_rank >= 0),
    -- 0: recorded before the rank was kept.
    tiles_rank integer NOT NULL DEFAULT 0 CHECK (tiles_rank >= 0),
    PRIMARY KEY (season, account_id)
);

CREATE INDEX IF NOT EXISTS finale_fights_account ON finale_fights (account_id);

-- An account with a username: player.v1.PlayerCreated.
CREATE TABLE nameplates (
    account_id uuid PRIMARY KEY
);

-- An account its player asked to delete: player.v1.PlayerLeaving and PlayerReturned, the latest decision wins.
CREATE TABLE departures (
    account_id uuid        PRIMARY KEY,
    leaving    boolean     NOT NULL,
    decided_at timestamptz NOT NULL
);

-- What player held when progression was made, once; player's events keep both from then on.
DO $$
BEGIN
    IF to_regclass('player.profiles') IS NOT NULL THEN
        INSERT INTO nameplates (account_id) SELECT account_id FROM player.profiles ON CONFLICT DO NOTHING;
    END IF;
    IF to_regclass('player.departures') IS NOT NULL THEN
        INSERT INTO departures (account_id, leaving, decided_at)
        SELECT account_id, leaving, decided_at FROM player.departures ON CONFLICT DO NOTHING;
    END IF;
END $$;
