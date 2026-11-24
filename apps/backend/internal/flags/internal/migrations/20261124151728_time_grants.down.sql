DELETE FROM outbox WHERE event IN ('flags.v1.SubmissionApproved', 'flags.v1.SubmissionRejected');

ALTER TYPE outbox_event RENAME TO outbox_event_old;

CREATE TYPE outbox_event AS ENUM ('flags.v1.FlagOpened', 'flags.v1.FlagClosed', 'flags.v1.FlagRenamed', 'flags.v1.FlagMoved');

ALTER TABLE outbox ALTER COLUMN event TYPE outbox_event USING event::text::outbox_event;

DROP TYPE outbox_event_old;
DROP TABLE time_held;
DROP TYPE time_period;
DROP TABLE time_takebacks;
DROP TABLE time_grants;
