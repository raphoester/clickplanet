-- A battle whose set plays the knock-out is ranked on survival first: judged with its mode, as it starts.
ALTER TABLE battle_modes ADD COLUMN knock_out boolean NOT NULL DEFAULT false;
