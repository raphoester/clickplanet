DELETE FROM identities WHERE provider = 'crazygames';
UPDATE arrivals SET source = 'unknown' WHERE source = 'crazygames';
