CREATE TYPE mute_kind AS ENUM ('chat', 'gifs');

-- Every mute kept before took the whole chat away.
ALTER TABLE mutes ADD COLUMN kind mute_kind NOT NULL DEFAULT 'chat';

ALTER TABLE mutes ALTER COLUMN kind DROP DEFAULT;
