-- A submission waits on a person: its player appealed the judge's rejection, or the judge was unsure and it is paid.
ALTER TYPE outbox_event ADD VALUE 'flags.v1.ReviewAsked';

CREATE TYPE appeal_state AS ENUM ('asked', 'answered');

-- A player's ask that a person look again at the judge's rejection of its submission, once per submission: asked, then
-- answered by the operator's verdict. It keeps the rejection it asks about, which the submission waiting again drops.
CREATE TABLE appeals (
    submission_id uuid             PRIMARY KEY REFERENCES submissions (id) ON DELETE CASCADE,
    account_id    uuid             NOT NULL,
    reason        rejection_reason NOT NULL,
    note          text             NOT NULL,
    state         appeal_state     NOT NULL,
    asked_at      timestamptz      NOT NULL,
    decided_at    timestamptz      NOT NULL
);
