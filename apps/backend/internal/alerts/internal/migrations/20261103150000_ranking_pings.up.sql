-- A day's ranking, told once by its round.
CREATE TABLE ranking_pings (
    round_id text        PRIMARY KEY,
    sent_at  timestamptz NOT NULL
);
