CREATE TABLE subscriptions (
    account_id   uuid        PRIMARY KEY,
    address      text        NOT NULL CHECK (address <> '' AND char_length(address) <= 254),
    state        text        NOT NULL CHECK (state IN ('waiting', 'active', 'withdrawn')),
    consent      text        NOT NULL CHECK (consent <> ''),
    asked_at     timestamptz NOT NULL,
    withdrawn_at timestamptz,
    CHECK ((state = 'withdrawn') = (withdrawn_at IS NOT NULL))
);

CREATE INDEX subscriptions_address ON subscriptions (address);
