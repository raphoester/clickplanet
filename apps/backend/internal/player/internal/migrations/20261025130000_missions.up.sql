CREATE TYPE mission_kind AS ENUM ('take_for', 'take_from', 'take_in');

CREATE TYPE mission_reward AS ENUM ('shields', 'spread_clicks', 'bomb');

CREATE TABLE mission_briefs (
    day           date           NOT NULL,
    flag          text           NOT NULL CHECK (flag <> ''),
    slot          smallint       NOT NULL CHECK (slot BETWEEN 1 AND 3),
    kind          mission_kind   NOT NULL,
    target        text           NOT NULL CHECK (target <> ''),
    goal          integer        NOT NULL CHECK (goal > 0),
    reward        mission_reward NOT NULL,
    reward_amount integer        NOT NULL CHECK (reward_amount > 0),
    PRIMARY KEY (day, flag, slot)
);

CREATE TABLE mission_sheets (
    day        date NOT NULL,
    account_id uuid NOT NULL,
    flag       text NOT NULL CHECK (flag <> ''),
    PRIMARY KEY (day, account_id)
);

CREATE INDEX mission_sheets_account ON mission_sheets (account_id);

CREATE TABLE mission_progress (
    day        date     NOT NULL,
    account_id uuid     NOT NULL,
    slot       smallint NOT NULL CHECK (slot BETWEEN 1 AND 3),
    taken      integer  NOT NULL CHECK (taken > 0),
    PRIMARY KEY (day, account_id, slot),
    FOREIGN KEY (day, account_id) REFERENCES mission_sheets ON DELETE CASCADE
);

CREATE TABLE mission_collections (
    day          date        NOT NULL,
    account_id   uuid        NOT NULL,
    slot         smallint    NOT NULL CHECK (slot BETWEEN 1 AND 3),
    collected_at timestamptz NOT NULL,
    PRIMARY KEY (day, account_id, slot)
);

CREATE INDEX mission_collections_account ON mission_collections (account_id);
