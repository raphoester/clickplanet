-- A flag may be out on the world's map too, now that the knock-out is a rule of its own.
ALTER TABLE knockouts DROP CONSTRAINT knockouts_map_check;
