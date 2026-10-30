CREATE TYPE arrival_source AS ENUM ('recruit', 'youtube', 'unknown');

CREATE TABLE arrivals (
    id         uuid           PRIMARY KEY,
    account_id uuid,
    source     arrival_source NOT NULL,
    referrer   text           NOT NULL,
    arrived_at timestamptz    NOT NULL
);

CREATE INDEX arrivals_account_id ON arrivals (account_id);
