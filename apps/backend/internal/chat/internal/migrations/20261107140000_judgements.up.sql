CREATE TYPE judgement_verdict AS ENUM ('acted', 'dismissed', 'left_open');

-- The judge's judgement of each report support told: one per report, and the first that judged a message stands for
-- every report of it. Pruned with the messages: a report of a message past the retention is dismissed unread.
CREATE TABLE judgements (
    report_id  uuid              PRIMARY KEY,
    -- NULL for a report of a player, from its card.
    message_id text,
    verdict    judgement_verdict NOT NULL,
    judged     boolean           NOT NULL,
    flagged    boolean           NOT NULL,
    reason     text              NOT NULL,
    judged_at  timestamptz       NOT NULL
);

CREATE INDEX judgements_message_idx ON judgements (message_id, judged_at) WHERE judged;

CREATE INDEX judgements_judged_at_idx ON judgements (judged_at);

-- What the chat tells or asks of the others, kept in the transaction of what it decides, and sent by the relay: the
-- judge's support.v1.DecideReport, and the AccountMuted of a mute it gave.
CREATE TYPE outbox_event AS ENUM ('support.v1.DecideReport', 'chat.v1.AccountMuted');

CREATE TABLE outbox (
    position bigserial    PRIMARY KEY,
    event    outbox_event NOT NULL,
    payload  bytea        NOT NULL
);
