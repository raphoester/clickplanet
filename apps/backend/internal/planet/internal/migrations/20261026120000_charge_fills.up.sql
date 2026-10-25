ALTER TABLE charges ADD COLUMN last_fill text NOT NULL DEFAULT '';

CREATE TYPE charge_fill_source AS ENUM ('internal', 'operator');

CREATE TABLE charge_fills (
    request_id      text               PRIMARY KEY CHECK (request_id <> ''),
    source          charge_fill_source NOT NULL,
    started_at      timestamptz        NOT NULL,
    reached         uuid,
    accounts_filled integer            NOT NULL CHECK (accounts_filled >= 0),
    finished_at     timestamptz
);
