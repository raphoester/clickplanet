CREATE TABLE convictions (
    id                 bigint      GENERATED ALWAYS AS IDENTITY PRIMARY KEY,
    convicted_at       timestamptz NOT NULL,
    scope              text        NOT NULL CHECK (scope <> ''),
    account            uuid,
    alongside          uuid,
    flags              integer     NOT NULL CHECK (flags >= 0),
    offence            integer     NOT NULL CHECK (offence >= 0),
    banned_until       timestamptz NOT NULL,
    readings           jsonb       NOT NULL,
    clicks             integer     NOT NULL CHECK (clicks >= 0),
    active_for         interval    NOT NULL,
    longest_gap        interval    NOT NULL,
    top_country        text        NOT NULL,
    top_country_clicks integer     NOT NULL CHECK (top_country_clicks >= 0)
);

CREATE INDEX convictions_convicted_at ON convictions (convicted_at);
