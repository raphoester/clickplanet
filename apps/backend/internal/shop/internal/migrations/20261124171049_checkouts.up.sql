-- Each Stripe page the shop opened, for a flag rejected while its buyer is on it: the shop closes the page first.
CREATE TABLE checkouts (
    session_id  text        PRIMARY KEY,
    target_kind target_kind NOT NULL,
    target      text        NOT NULL,
    opened_at   timestamptz NOT NULL
);

CREATE INDEX checkouts_target ON checkouts (target_kind, target);
