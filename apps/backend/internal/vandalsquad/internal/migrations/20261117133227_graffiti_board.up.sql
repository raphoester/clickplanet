-- The board a graffiti's tiles are on, by its name, as planet's ledger names it: a tile id means a place only on its
-- board. Every graffiti before was painted on earth-1. The default is dropped once the column exists, so every write
-- names its board.
ALTER TABLE graffiti ADD COLUMN board text NOT NULL DEFAULT 'earth-1' CHECK (board <> '' AND length(board) <= 64);
ALTER TABLE graffiti ALTER COLUMN board DROP DEFAULT;
