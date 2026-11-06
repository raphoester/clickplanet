ALTER TYPE announcement_kind ADD VALUE IF NOT EXISTS 'flag_mission';

ALTER TABLE announcements ADD COLUMN room text NOT NULL DEFAULT 'world';
ALTER TABLE announcements ALTER COLUMN room DROP DEFAULT;

CREATE INDEX announcements_room_announced_at ON announcements (room, announced_at);
