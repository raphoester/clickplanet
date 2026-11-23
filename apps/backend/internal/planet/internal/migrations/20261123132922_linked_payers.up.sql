-- A mine's blast and a virus's tick take tiles with no request: a trap they spring drains the bank of the player kept
-- with them, and a guest's network's guest bank too. A row from before reads as a guest's.
ALTER TABLE mines ADD COLUMN linked boolean NOT NULL DEFAULT false;
ALTER TABLE viruses ADD COLUMN linked boolean NOT NULL DEFAULT false;
