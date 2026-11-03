ALTER TABLE messages ADD COLUMN room text NOT NULL DEFAULT 'world';
ALTER TABLE messages ALTER COLUMN room DROP DEFAULT;

CREATE INDEX messages_room_seq ON messages (room, seq);

ALTER TABLE seen ADD COLUMN room text NOT NULL DEFAULT 'world';
ALTER TABLE seen ALTER COLUMN room DROP DEFAULT;
ALTER TABLE seen DROP CONSTRAINT seen_pkey;
ALTER TABLE seen ADD PRIMARY KEY (account_id, room);
