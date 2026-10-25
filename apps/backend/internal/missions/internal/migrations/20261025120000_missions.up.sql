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
    taken      integer  NOT NULL CHECK (taken > 0),
    PRIMARY KEY (day, account_id, mission_id),
    FOREIGN KEY (day, account_id) REFERENCES mission_sheets ON DELETE CASCADE
);

CREATE TABLE mission_collections (
    day          date        NOT NULL,
    account_id   uuid        NOT NULL,
    mission_id   uuid        NOT NULL,
    collected_at timestamptz NOT NULL,
    PRIMARY KEY (day, account_id, mission_id)
);

CREATE INDEX mission_collections_account ON mission_collections (account_id);
