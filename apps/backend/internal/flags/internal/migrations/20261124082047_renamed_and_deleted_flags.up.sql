-- Planet tells which flags a player played for from now on. The one flag opened before this ran had no tile and no
-- pledge.
ALTER TABLE flags ADD COLUMN played boolean NOT NULL DEFAULT false;

CREATE UNIQUE INDEX flags_name_key ON flags (lower(name));

ALTER TYPE outbox_event ADD VALUE 'flags.v1.FlagRenamed';
ALTER TYPE outbox_event ADD VALUE 'flags.v1.FlagDeleted';
