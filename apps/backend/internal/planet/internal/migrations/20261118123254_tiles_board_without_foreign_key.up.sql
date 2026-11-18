-- A switch of the world's board writes a million tiles in the transaction that holds every click: the foreign key's check
-- of each row took 22 of the 26 seconds of the COPY in the rehearsal (docs/boards.md). Every tile's board is its stage's,
-- which its era names, and the boot refuses a map whose rows name another board than its era's.
ALTER TABLE tiles DROP CONSTRAINT IF EXISTS tiles_board_fkey;
