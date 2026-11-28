DELETE FROM outbox WHERE event = 'flags.v1.AnthemChanged';

ALTER TYPE outbox_event RENAME TO outbox_event_old;

CREATE TYPE outbox_event AS ENUM (
    'flags.v1.FlagOpened', 'flags.v1.FlagClosed', 'flags.v1.FlagRenamed', 'flags.v1.FlagMoved',
    'flags.v1.SubmissionApproved', 'flags.v1.SubmissionRejected', 'flags.v1.TimeRefused', 'flags.v1.ReviewAsked'
);

ALTER TABLE outbox ALTER COLUMN event TYPE outbox_event USING event::text::outbox_event;

DROP TYPE outbox_event_old;

ALTER TABLE submissions DROP COLUMN anthem_library;
ALTER TABLE flags DROP COLUMN anthem_library;
