DROP TABLE outbox;
DROP TYPE outbox_event;
DROP TABLE closed_arenas;
DROP TABLE arenas;

DELETE FROM landmasses WHERE map <> 'world';
ALTER TABLE landmasses DROP CONSTRAINT landmasses_pkey, ADD PRIMARY KEY (asset, id);
ALTER TABLE landmasses DROP COLUMN map;

DELETE FROM tiles WHERE map <> 'world';
ALTER TABLE tiles DROP CONSTRAINT tiles_pkey, ADD PRIMARY KEY (id);
ALTER TABLE tiles DROP COLUMN map;
