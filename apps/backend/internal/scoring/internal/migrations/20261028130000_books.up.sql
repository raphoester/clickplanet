-- Scoring keeps books and rounds by the ids the gamemaster gives. Only season 0 ever ran, and the day ids below are its.
DO $$
BEGIN
    IF EXISTS (SELECT 1 FROM contributions WHERE season <> 0) OR EXISTS (SELECT 1 FROM rounds WHERE season <> 0) THEN
        RAISE EXCEPTION 'only season 0 can be converted to books';
    END IF;
END
$$;

CREATE TABLE staged_round_ids AS
SELECT r.season, r.ends_at, r.own,
       CASE
           WHEN NOT r.own THEN 'season-0/day-' ||
               (1 + round(extract(epoch FROM r.ends_at - timestamptz '2026-10-08T21:00:00Z') / 86400))::bigint
           WHEN o.entry_id IS NOT NULL THEN o.entry_id::text
           ELSE 'season-' || r.season || '/own-' || row_number() OVER (PARTITION BY r.season, r.own ORDER BY r.ends_at)
       END AS round_id
FROM rounds AS r LEFT JOIN own_rounds AS o USING (season, ends_at, own);

CREATE TABLE staged_rounds AS
SELECT i.round_id, 'season-' || r.season AS book_id, r.times, coalesce(r.starts_at, r.ends_at - interval '1 day') AS starts_at,
       r.counted_takes_below, r.closed, r.ends_at
FROM rounds AS r JOIN staged_round_ids AS i USING (season, ends_at, own);

CREATE TABLE staged_results AS
SELECT i.round_id, x.country, x.rank, x.points, x.share
FROM round_results AS x JOIN staged_round_ids AS i USING (season, ends_at, own);

CREATE TABLE staged_takes AS
SELECT i.round_id, x.country, x.tiles FROM round_takes AS x JOIN staged_round_ids AS i USING (season, ends_at, own);

CREATE TABLE staged_players AS
SELECT i.round_id, x.account_id, x.country, x.tiles
FROM round_players AS x JOIN staged_round_ids AS i USING (season, ends_at, own);

CREATE TABLE staged_closed_books AS
SELECT 'season-' || e.season AS book_id, e.ended_at, (
    SELECT i.round_id FROM rounds AS r JOIN staged_round_ids AS i USING (season, ends_at, own)
    WHERE r.season = e.season AND r.finale AND r.closed ORDER BY r.own DESC LIMIT 1
) AS tiebreak_round_id
FROM ended_seasons AS e;

DROP TABLE round_results, round_takes, round_players, own_rounds, ended_seasons, rounds, rule_sets, rule_set_choice;

CREATE TABLE books (
    book_id   text        PRIMARY KEY CHECK (book_id <> '' AND length(book_id) <= 64),
    starts_at timestamptz NOT NULL
);

-- Season 0 starts at the gamemaster's origin of it, so its OpenBook finds the book it already is.
INSERT INTO books (book_id, starts_at)
SELECT DISTINCT 'season-' || season, timestamptz '2026-10-08T00:00:00Z' FROM contributions
UNION SELECT book_id, timestamptz '2026-10-08T00:00:00Z' FROM staged_rounds
UNION SELECT book_id, timestamptz '2026-10-08T00:00:00Z' FROM staged_closed_books;

CREATE TABLE closed_books (
    book_id  text        PRIMARY KEY REFERENCES books (book_id),
    ended_at timestamptz NOT NULL
);

-- A book closed with a third key: its tiebreak round's points.
CREATE TABLE book_tiebreaks (
    book_id  text PRIMARY KEY REFERENCES closed_books (book_id),
    round_id text NOT NULL CHECK (round_id <> '' AND length(round_id) <= 64)
);

CREATE TABLE rounds (
    round_id            text        PRIMARY KEY CHECK (round_id <> '' AND length(round_id) <= 64),
    book_id             text        NOT NULL REFERENCES books (book_id),
    times               integer     NOT NULL CHECK (times BETWEEN 1 AND 10),
    starts_at           timestamptz NOT NULL,
    counted_takes_below bigint      NOT NULL DEFAULT 0 CHECK (counted_takes_below >= 0)
);

CREATE INDEX rounds_book ON rounds (book_id, starts_at);

CREATE TABLE scored_rounds (
    round_id text        PRIMARY KEY REFERENCES rounds (round_id),
    ended_at timestamptz NOT NULL
);

CREATE TABLE round_results (
    round_id text             NOT NULL REFERENCES scored_rounds (round_id),
    country  text             NOT NULL CHECK (country <> ''),
    rank     integer          NOT NULL CHECK (rank > 0),
    points   integer          NOT NULL CHECK (points >= 0),
    share    double precision NOT NULL CHECK (share >= 0 AND share <= 1),
    PRIMARY KEY (round_id, country)
);

CREATE TABLE round_takes (
    round_id text   NOT NULL REFERENCES rounds (round_id),
    country  text   NOT NULL CHECK (country <> ''),
    tiles    bigint NOT NULL CHECK (tiles > 0),
    PRIMARY KEY (round_id, country)
);

CREATE TABLE round_players (
    round_id   text   NOT NULL REFERENCES rounds (round_id),
    account_id uuid   NOT NULL,
    country    text   NOT NULL CHECK (country <> ''),
    tiles      bigint NOT NULL CHECK (tiles > 0),
    PRIMARY KEY (round_id, account_id, country)
);

CREATE INDEX round_players_account ON round_players (account_id);

INSERT INTO rounds (round_id, book_id, times, starts_at, counted_takes_below)
SELECT round_id, book_id, times, starts_at, counted_takes_below FROM staged_rounds;
INSERT INTO scored_rounds (round_id, ended_at) SELECT round_id, ends_at FROM staged_rounds WHERE closed;
INSERT INTO round_results (round_id, country, rank, points, share)
SELECT round_id, country, rank, points, share FROM staged_results;
INSERT INTO round_takes (round_id, country, tiles) SELECT round_id, country, tiles FROM staged_takes;
INSERT INTO round_players (round_id, account_id, country, tiles) SELECT round_id, account_id, country, tiles FROM staged_players;
INSERT INTO closed_books (book_id, ended_at) SELECT book_id, ended_at FROM staged_closed_books;
INSERT INTO book_tiebreaks (book_id, round_id)
SELECT book_id, tiebreak_round_id FROM staged_closed_books WHERE tiebreak_round_id IS NOT NULL;

DROP TABLE staged_round_ids, staged_rounds, staged_results, staged_takes, staged_players, staged_closed_books;

ALTER TABLE contributions ADD COLUMN book_id text;
UPDATE contributions SET book_id = 'season-' || season;
DROP INDEX contributions_main_key, contributions_board, contributions_country_board;
ALTER TABLE contributions DROP CONSTRAINT contributions_pkey;
ALTER TABLE contributions DROP COLUMN season;
ALTER TABLE contributions
    ALTER COLUMN book_id SET NOT NULL,
    ADD CONSTRAINT contributions_book_id CHECK (book_id <> '' AND length(book_id) <= 64),
    ADD PRIMARY KEY (book_id, account_id, country);
CREATE UNIQUE INDEX contributions_main_key ON contributions (book_id, account_id) WHERE main;
CREATE INDEX contributions_board ON contributions (book_id, tiles DESC, account_id) WHERE main;
CREATE INDEX contributions_country_board ON contributions (book_id, country, tiles DESC, account_id);

-- An event of the old kinds left unsent is dropped: its payload is no longer a message anyone reads.
DELETE FROM outbox WHERE event IN ('scoring.v1.RoundClosed', 'scoring.v1.SeasonClosed');
ALTER TYPE outbox_event RENAME VALUE 'scoring.v1.RoundClosed' TO 'scoring.v1.RoundScored';
ALTER TYPE outbox_event RENAME VALUE 'scoring.v1.SeasonClosed' TO 'scoring.v1.BookClosed';
