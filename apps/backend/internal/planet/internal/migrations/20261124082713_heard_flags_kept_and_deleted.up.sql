-- A flag deleted (flags.v1.FlagDeleted) is unknown again, and a late fact older than its deletion changes nothing. Kept:
-- planet told the flags module a player played for the flag (flags.v1.KeepFlag), so it is told once.
ALTER TYPE heard_flag_state ADD VALUE 'deleted';

ALTER TABLE heard_flags ADD COLUMN kept boolean NOT NULL DEFAULT false;
