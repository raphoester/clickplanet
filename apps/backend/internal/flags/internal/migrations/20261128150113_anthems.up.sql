-- A sold flag's anthem, and the one a submission's flag takes when it opens: a recording of the game's library, named
-- by the code of the flag the game ships with that it plays for. NULL: none.
ALTER TABLE flags ADD COLUMN anthem_library text;
ALTER TABLE submissions ADD COLUMN anthem_library text;

-- The holder's change of anthem, told to the pages' streams. No statement here uses it: postgres
-- uses a new value only once it is committed.
ALTER TYPE outbox_event ADD VALUE 'flags.v1.AnthemChanged';
