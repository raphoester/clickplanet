DELETE FROM outbox WHERE event = 'flags.v1.FlagRenamed';

ALTER TYPE outbox_event RENAME TO outbox_event_old;

CREATE TYPE outbox_event AS ENUM ('flags.v1.FlagOpened', 'flags.v1.FlagClosed');

ALTER TABLE outbox ALTER COLUMN event TYPE outbox_event USING event::text::outbox_event;

DROP TYPE outbox_event_old;

DROP INDEX flags_name_key;
