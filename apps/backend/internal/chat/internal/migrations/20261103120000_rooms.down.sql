DELETE FROM seen WHERE room <> 'world';
ALTER TABLE seen DROP CONSTRAINT seen_pkey;
ALTER TABLE seen ADD PRIMARY KEY (account_id);
ALTER TABLE seen DROP COLUMN room;

DELETE FROM messages WHERE room <> 'world';
DROP INDEX messages_room_seq;
ALTER TABLE messages DROP COLUMN room;
