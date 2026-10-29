CREATE TABLE flag_days (
    day                   date        NOT NULL,
    flag                  text        NOT NULL CHECK (flag <> ''),
    takes                 bigint      NOT NULL CHECK (takes >= 0),
    counted_takes_below   bigint      NOT NULL CHECK (counted_takes_below >= 0),
    strikes               bigint      NOT NULL CHECK (strikes >= 0),
    counted_strikes_until timestamptz NOT NULL,
    PRIMARY KEY (day, flag)
);

CREATE TYPE flag_mission_kind AS ENUM ('take_in', 'break_shields');

CREATE TABLE flag_missions (
    day        date              NOT NULL,
    flag       text              NOT NULL,
    mission_id uuid              NOT NULL UNIQUE,
    kind       flag_mission_kind NOT NULL,
    payload    jsonb             NOT NULL CHECK (jsonb_typeof(payload) = 'object'),
    points     integer           NOT NULL CHECK (points > 0),
    progress   integer           NOT NULL CHECK (progress >= 0),
    done       boolean           NOT NULL,
    PRIMARY KEY (day, flag),
    FOREIGN KEY (day, flag) REFERENCES flag_days ON DELETE CASCADE
);

ALTER TYPE outbox_event ADD VALUE IF NOT EXISTS 'scoring.v1.GrantPoints';
