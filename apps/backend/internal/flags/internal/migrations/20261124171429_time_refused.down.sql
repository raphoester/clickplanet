DELETE FROM outbox WHERE event = 'flags.v1.TimeRefused';

ALTER TYPE outbox_event RENAME TO outbox_event_old;

CREATE TYPE outbox_event AS ENUM ('flags.v1.FlagOpened', 'flags.v1.FlagClosed', 'flags.v1.FlagRenamed', 'flags.v1.FlagMoved',
    'flags.v1.SubmissionApproved', 'flags.v1.SubmissionRejected');

ALTER TABLE outbox ALTER COLUMN event TYPE outbox_event USING event::text::outbox_event;

DROP TYPE outbox_event_old;
