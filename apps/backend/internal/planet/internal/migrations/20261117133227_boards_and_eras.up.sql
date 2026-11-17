-- A board is a geometry, the files of /map/boards/<name>: its id is what every row of a tile names, its name what the
-- files and the wire name, its hash the first 8 hex of its manifest's sha256 (board-<hash>.json). A board's row ships
-- with the migration that ships the board; its files never change, so neither does its hash.
CREATE TABLE boards (
    id   smallint PRIMARY KEY CHECK (id > 0),
    name text     NOT NULL UNIQUE CHECK (name <> '' AND length(name) <= 64),
    hash text     NOT NULL UNIQUE CHECK (hash ~ '^[0-9a-f]{8}$')
);

INSERT INTO boards (id, name, hash) VALUES (1, 'earth-1', 'f7db567f');

-- An era is a stretch of a map on one board, from started_at on: a map's latest era is the board it plays.
CREATE TABLE eras (
    map        text        NOT NULL CHECK (map <> '' AND length(map) <= 64),
    board      smallint    NOT NULL REFERENCES boards (id),
    started_at timestamptz NOT NULL,
    PRIMARY KEY (map, started_at)
);

-- Both remaps (2026-09-21 and 2026-11-03) rewrote all history onto today's ids, and every arena played the world's
-- board: the world played earth-1 from the start, and each arena from its open.
INSERT INTO eras (map, board, started_at) VALUES ('world', 1, 'epoch');
INSERT INTO eras (map, board, started_at) SELECT id, 1, opened_at FROM arenas;

-- A constant default is a change of the catalog alone, no row is written. It is dropped once the column exists, so
-- every write names its board.
ALTER TABLE tiles ADD COLUMN board smallint NOT NULL DEFAULT 1 REFERENCES boards (id);
ALTER TABLE tiles ALTER COLUMN board DROP DEFAULT;

ALTER TABLE landmasses ADD COLUMN board smallint NOT NULL DEFAULT 1 REFERENCES boards (id);
ALTER TABLE landmasses ALTER COLUMN board DROP DEFAULT;

ALTER TABLE mines ADD COLUMN board smallint NOT NULL DEFAULT 1 REFERENCES boards (id);
ALTER TABLE mines ALTER COLUMN board DROP DEFAULT;

ALTER TABLE viruses ADD COLUMN board smallint NOT NULL DEFAULT 1 REFERENCES boards (id);
ALTER TABLE viruses ALTER COLUMN board DROP DEFAULT;

-- The ledger holds millions of rows. Its constraints are NOT VALID: they hold for every row written from now on, and
-- the rows before hold the defaults, so no scan of the table checks them under its lock.
ALTER TABLE ledger_events
    ADD COLUMN map   text     NOT NULL DEFAULT 'world',
    ADD COLUMN board smallint NOT NULL DEFAULT 1;

ALTER TABLE ledger_events
    ADD CONSTRAINT ledger_events_board_fkey FOREIGN KEY (board) REFERENCES boards (id) NOT VALID,
    ADD CONSTRAINT ledger_events_map_check CHECK (map <> '' AND length(map) <= 64) NOT VALID;

-- Until now an arena's events were told apart by time (arenas.Stretch.Holds): after its open, and before the second it
-- closed, or still open. The one time the arenas' windows are read: from now on every event names its map.
UPDATE ledger_events e SET map = a.id
FROM arenas a
LEFT JOIN closed_arenas c ON c.id = a.id
WHERE e.taken_at > a.opened_at
    AND (c.closed_at IS NULL OR e.taken_at < date_trunc('second', c.closed_at));

ALTER TABLE ledger_events
    ALTER COLUMN map DROP DEFAULT,
    ALTER COLUMN board DROP DEFAULT;
