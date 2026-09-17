-- An admin of the game. The game never sets it: an operator flips it in the database.
ALTER TABLE profiles ADD COLUMN admin boolean NOT NULL DEFAULT false;
