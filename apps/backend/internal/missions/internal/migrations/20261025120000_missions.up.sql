CREATE TABLE mission_briefs (
    day      date  NOT NULL,
    flag     text  NOT NULL CHECK (flag <> ''),
    missions jsonb NOT NULL CHECK (jsonb_typeof(missions) = 'array'),
    PRIMARY KEY (day, flag)
);

CREATE TABLE mission_sheets (
    day                 date   NOT NULL,
    account_id          uuid   NOT NULL,
    flag                text   NOT NULL CHECK (flag <> ''),
    counted_takes_below bigint NOT NULL CHECK (counted_takes_below >= 0),
    PRIMARY KEY (day, account_id)
);

CREATE INDEX mission_sheets_account ON mission_sheets (account_id);

CREATE TABLE mission_progress (
    day        date     NOT NULL,
    account_id uuid     NOT NULL,
    mission_id uuid     NOT NULL,
    progress   integer  NOT NULL CHECK (progress > 0),
    PRIMARY KEY (day, account_id, mission_id),
    FOREIGN KEY (day, account_id) REFERENCES mission_sheets ON DELETE CASCADE
);

CREATE TABLE mission_payments (
    day        date NOT NULL,
    account_id uuid NOT NULL,
    mission_id uuid NOT NULL,
    PRIMARY KEY (day, account_id, mission_id),
    FOREIGN KEY (day, account_id) REFERENCES mission_sheets ON DELETE CASCADE
);

CREATE TYPE outbox_event AS ENUM ('planet.v1.GrantCharges');

CREATE TABLE outbox (
    position bigserial    PRIMARY KEY,
    event    outbox_event NOT NULL,
    payload  bytea        NOT NULL
);
