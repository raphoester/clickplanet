DELETE FROM outbox WHERE event IN ('scoring.v1.RoundScored', 'scoring.v1.BookClosed');
ALTER TYPE outbox_event RENAME VALUE 'scoring.v1.RoundScored' TO 'scoring.v1.RoundClosed';
ALTER TYPE outbox_event RENAME VALUE 'scoring.v1.BookClosed' TO 'scoring.v1.SeasonClosed';

ALTER TABLE contributions ADD COLUMN season integer;
UPDATE contributions SET season = substring(book_id FROM '^season-([0-9]+)$')::integer;
DELETE FROM contributions WHERE season IS NULL;
DROP INDEX contributions_main_key, contributions_board, contributions_country_board;
ALTER TABLE contributions DROP CONSTRAINT contributions_pkey;
ALTER TABLE contributions DROP COLUMN book_id;
ALTER TABLE contributions
    ALTER COLUMN season SET NOT NULL,
    ADD CONSTRAINT contributions_season_check CHECK (season >= 0),
    ADD PRIMARY KEY (season, account_id, country);
CREATE UNIQUE INDEX contributions_main_key ON contributions (season, account_id) WHERE main;
CREATE INDEX contributions_board ON contributions (season, tiles DESC, account_id) WHERE main;
CREATE INDEX contributions_country_board ON contributions (season, country, tiles DESC, account_id);

CREATE TABLE staged_rounds AS
SELECT r.round_id, substring(r.book_id FROM '^season-([0-9]+)$')::integer AS season,
       coalesce(s.ended_at, r.starts_at + interval '1 day') AS ends_at, r.round_id NOT LIKE '%/day-%' AS own,
       r.times, r.starts_at, r.counted_takes_below, s.round_id IS NOT NULL AS closed,
       coalesce(t.round_id = r.round_id, false) AS finale
FROM rounds AS r
LEFT JOIN scored_rounds AS s USING (round_id)
LEFT JOIN book_tiebreaks AS t ON t.book_id = r.book_id;
DELETE FROM staged_rounds WHERE season IS NULL;

CREATE TABLE staged_results AS SELECT * FROM round_results;
CREATE TABLE staged_takes AS SELECT * FROM round_takes;
CREATE TABLE staged_players AS SELECT * FROM round_players;
CREATE TABLE staged_ended AS
SELECT substring(book_id FROM '^season-([0-9]+)$')::integer AS season, ended_at FROM closed_books;

DROP TABLE round_results, round_takes, round_players, scored_rounds, rounds, book_tiebreaks, closed_books, books;

CREATE TABLE rounds (
    season              integer     NOT NULL CHECK (season >= 0),
    ends_at             timestamptz NOT NULL,
    finale              boolean     NOT NULL,
    closed              boolean     NOT NULL DEFAULT false,
    own                 boolean     NOT NULL,
    starts_at           timestamptz,
    times               integer     NOT NULL CHECK (times BETWEEN 1 AND 10),
    counted_takes_below bigint      NOT NULL DEFAULT 0 CHECK (counted_takes_below >= 0),
    CONSTRAINT rounds_own_starts CHECK (own = (starts_at IS NOT NULL)),
    PRIMARY KEY (season, ends_at, own)
);
CREATE INDEX rounds_unclosed ON rounds (ends_at) WHERE NOT closed;

CREATE TABLE round_results (
    season  integer          NOT NULL,
    ends_at timestamptz      NOT NULL,
    own     boolean          NOT NULL,
    country text             NOT NULL CHECK (country <> ''),
    rank    integer          NOT NULL CHECK (rank > 0),
    points  integer          NOT NULL CHECK (points >= 0),
    share   double precision NOT NULL CONSTRAINT round_results_share CHECK (share >= 0 AND share <= 1),
    PRIMARY KEY (season, ends_at, own, country),
    FOREIGN KEY (season, ends_at, own) REFERENCES rounds (season, ends_at, own)
);

CREATE TABLE round_takes (
    season  integer     NOT NULL,
    ends_at timestamptz NOT NULL,
    own     boolean     NOT NULL CHECK (own),
    country text        NOT NULL CHECK (country <> ''),
    tiles   bigint      NOT NULL CHECK (tiles > 0),
    PRIMARY KEY (season, ends_at, own, country),
    FOREIGN KEY (season, ends_at, own) REFERENCES rounds (season, ends_at, own)
);

CREATE TABLE round_players (
    season     integer     NOT NULL,
    ends_at    timestamptz NOT NULL,
    own        boolean     NOT NULL CHECK (own),
    account_id uuid        NOT NULL,
    country    text        NOT NULL CHECK (country <> ''),
    tiles      bigint      NOT NULL CHECK (tiles > 0),
    PRIMARY KEY (season, ends_at, own, account_id, country),
    FOREIGN KEY (season, ends_at, own) REFERENCES rounds (season, ends_at, own)
);
CREATE INDEX round_players_account ON round_players (account_id);

CREATE TABLE ended_seasons (
    season   integer     PRIMARY KEY CHECK (season >= 0),
    ended_at timestamptz NOT NULL
);

CREATE TABLE own_rounds (
    season   integer     NOT NULL,
    ends_at  timestamptz NOT NULL,
    own      boolean     NOT NULL CHECK (own),
    entry_id uuid        NOT NULL,
    PRIMARY KEY (season, ends_at, own),
    FOREIGN KEY (season, ends_at, own) REFERENCES rounds (season, ends_at, own)
);

CREATE TABLE rule_sets (
    name  text  PRIMARY KEY CHECK (name <> ''),
    rules jsonb NOT NULL CHECK (jsonb_typeof(rules) = 'object')
);
INSERT INTO rule_sets (name, rules) VALUES ('default', '{"ownRound": {"enabled": false, "times": 1}}');

CREATE TABLE rule_set_choice (
    single     boolean     PRIMARY KEY DEFAULT true CHECK (single),
    rule_set   text        NOT NULL CHECK (rule_set <> ''),
    starts_at  timestamptz,
    ends_at    timestamptz,
    decided_at timestamptz NOT NULL,
    entry_id   uuid,
    CHECK ((starts_at IS NULL) = (ends_at IS NULL)),
    CONSTRAINT rule_set_choice_entry CHECK ((entry_id IS NULL) = (ends_at IS NULL))
);

INSERT INTO rounds (season, ends_at, finale, closed, own, starts_at, times, counted_takes_below)
SELECT season, ends_at, finale, closed, own, CASE WHEN own THEN starts_at END, times, counted_takes_below FROM staged_rounds
ON CONFLICT DO NOTHING;
INSERT INTO round_results (season, ends_at, own, country, rank, points, share)
SELECT r.season, r.ends_at, r.own, x.country, x.rank, x.points, x.share
FROM staged_results AS x JOIN staged_rounds AS r USING (round_id) ON CONFLICT DO NOTHING;
INSERT INTO round_takes (season, ends_at, own, country, tiles)
SELECT r.season, r.ends_at, r.own, x.country, x.tiles
FROM staged_takes AS x JOIN staged_rounds AS r USING (round_id) WHERE r.own ON CONFLICT DO NOTHING;
INSERT INTO round_players (season, ends_at, own, account_id, country, tiles)
SELECT r.season, r.ends_at, r.own, x.account_id, x.country, x.tiles
FROM staged_players AS x JOIN staged_rounds AS r USING (round_id) WHERE r.own ON CONFLICT DO NOTHING;
INSERT INTO own_rounds (season, ends_at, own, entry_id)
SELECT season, ends_at, own, round_id::uuid FROM staged_rounds
WHERE own AND round_id ~ '^[0-9a-f]{8}-[0-9a-f]{4}-[0-9a-f]{4}-[0-9a-f]{4}-[0-9a-f]{12}$' ON CONFLICT DO NOTHING;
INSERT INTO ended_seasons (season, ended_at) SELECT season, ended_at FROM staged_ended WHERE season IS NOT NULL;

DROP TABLE staged_rounds, staged_results, staged_takes, staged_players, staged_ended;
