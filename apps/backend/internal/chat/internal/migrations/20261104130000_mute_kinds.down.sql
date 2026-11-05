DELETE FROM mutes WHERE kind <> 'chat';

ALTER TABLE mutes DROP COLUMN kind;

DROP TYPE mute_kind;
