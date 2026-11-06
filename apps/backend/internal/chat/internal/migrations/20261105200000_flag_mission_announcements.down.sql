DELETE FROM announcements WHERE kind = 'flag_mission' OR room <> 'world';
DROP INDEX announcements_room_announced_at;
ALTER TABLE announcements DROP COLUMN room;
