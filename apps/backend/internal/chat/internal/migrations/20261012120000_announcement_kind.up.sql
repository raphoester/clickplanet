CREATE TYPE announcement_kind AS ENUM ('bomb', 'mute', 'round', 'fortify');

ALTER TABLE announcements ALTER COLUMN kind TYPE announcement_kind USING kind::announcement_kind;
