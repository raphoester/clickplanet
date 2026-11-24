-- The time each payment granted, named by the payment: the shop tells a payment again, a grant is kept once. A refund
-- takes back what it added.
CREATE TABLE time_grants (
    id         text        PRIMARY KEY,
    -- The flag the time went to: a submission's, once the payment opened it.
    flag       text        NOT NULL REFERENCES flags (id),
    -- How much later the flag ended for it, or how long it opened for.
    added      interval    NOT NULL,
    granted_at timestamptz NOT NULL
);

-- A payment refunded or lost in a dispute. One taken back before its grant was heard grants nothing.
CREATE TABLE time_takebacks (
    grant_id   text        PRIMARY KEY,
    decided_at timestamptz NOT NULL
);

CREATE TYPE time_period AS ENUM ('week', 'month', 'year');

-- A payment for a submission that waits for its review: its time is held, and starts at its approval, when it is
-- granted too. A paid submission is never replaced, so it is never deleted.
CREATE TABLE time_held (
    grant_id      text        PRIMARY KEY,
    submission_id uuid        NOT NULL REFERENCES submissions (id),
    period        time_period NOT NULL,
    held_at       timestamptz NOT NULL
);

CREATE INDEX time_held_submission ON time_held (submission_id);

-- The verdicts: an approval opens a flag paid for, a rejection is refunded by the shop.
ALTER TYPE outbox_event ADD VALUE 'flags.v1.SubmissionApproved';
ALTER TYPE outbox_event ADD VALUE 'flags.v1.SubmissionRejected';
