-- earth-2 and siege-1 ship (docs/boards.md, PR 4): each board's row ships with the migration that ships the board, its
-- name the directory in /map/boards and its hash its manifest's positionsHash in 8 hex. Nothing plays them yet: no era
-- names them until a rule set's arena.board does (siege-1) or the world switches (earth-2), and the boot builds a
-- board's ground only when a map plays it.
INSERT INTO boards (id, name, hash) VALUES
    (2, 'earth-2', '5a61c10a'),
    (3, 'siege-1', 'e219e852');
