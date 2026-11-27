DROP TABLE appeals;

DROP TYPE appeal_state;

DELETE FROM outbox WHERE event = 'flags.v1.ReviewAsked';

ALTER TYPE outbox_event RENAME TO outbox_event_old;

CREATE TYPE outbox_event AS ENUM ('flags.v1.FlagOpened', 'flags.v1.FlagClosed', 'flags.v1.FlagRenamed', 'flags.v1.FlagMoved',
    'flags.v1.SubmissionApproved', 'flags.v1.SubmissionRejected', 'flags.v1.TimeRefused');

ALTER TABLE outbox ALTER COLUMN event TYPE outbox_event USING event::text::outbox_event;

DROP TYPE outbox_event_old;
